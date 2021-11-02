// Client Transfer

package transfer

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charlesetsmith/saratoga/data"
	"github.com/charlesetsmith/saratoga/frames"
	"github.com/charlesetsmith/saratoga/metadata"
	"github.com/charlesetsmith/saratoga/request"
	"github.com/charlesetsmith/saratoga/sarflags"
	"github.com/charlesetsmith/saratoga/sarwin"
	"github.com/charlesetsmith/saratoga/status"
	"github.com/jroimartin/gocui"
)

// CTransfer Information
type CTransfer struct {
	direction string // "client|server"
	ttype     string // CTransfer type "get,getrm,put,putrm,blindput,rm"
	tstamp    string // Timestamp type "localinterp,posix32,posix64,posix32_32,posix64_32,epoch2000_32"
	session   uint32 // Session ID - This is the unique key
	peer      net.IP // Remote Host
	filename  string // File name to get from remote host
	fp        *os.File
	// frames    [][]byte           // Frame queue
	// holes     holes.Holes        // Holes
	cliflags *sarflags.Cliflags // Global flags for this transfer
}

var ctrmu sync.Mutex

// CTransfers - Client transfers in progress
var CTransfers = []CTransfer{}

// read and process status frames
// We read from and seek within fp
// Our connection to the server is conn
// We assemble Data using dflags
// We transmit metadata as required
// We send back a string holding the status error code "success" keeps transfer alive
func readstatus(g *gocui.Gui, t *CTransfer, dflags string, conn *net.UDPConn, addr *net.UDPAddr,
	m *metadata.MetaData, pos chan [2]uint64, errflag chan string) {

	var filelen uint64
	var fi os.FileInfo
	var err error

	// Grab the file informaion
	if fi, err = t.fp.Stat(); err != nil {
		errflag <- "filenotfound"
		return
	}
	filelen = uint64(fi.Size())

	// Allocate a recieve buffer for a status frame
	rbuf := make([]byte, sarflags.MaxBuff)

	timeout := time.Duration(t.cliflags.Timeout.Status) * time.Second
	for {
		sarwin.MsgPrintln(g, "blue_black", "Client Waiting to Read a Status Frame on",
			conn.LocalAddr().String())
		conn.SetReadDeadline(time.Now().Add(timeout))
		rlen, err := conn.Read(rbuf)
		if err != nil {
			sarwin.MsgPrintln(g, "blue_black", "Client Timeout on Status Read",
				":", err.Error())
			errflag <- "cantreceive"
			return
		}
		// We have a status so grab it
		sarwin.MsgPrintln(g, "blue_black", "Client Read a Frame len", rlen, "bytes")
		rframe := make([]byte, rlen)
		copy(rframe, rbuf[:rlen])
		header := binary.BigEndian.Uint32(rframe[:4])
		if sarflags.GetStr(header, "version") != "v1" { // Make sure we are Version 1
			sarwin.MsgPrintln(g, "red_black", "Client Not Saratoga Version 1 Frame from ",
				t.peer.String())
			errflag <- "badpacket"
			return
		}
		// Process the received frame and make sure it is a status
		if sarflags.GetStr(header, "frametype") == "status" {
			errf := sarflags.GetStr(header, "errcode")
			if errf != "success" { // We have a error from the server
				errflag <- errf
				return
			}
			sarwin.MsgPrintln(g, "blue_black", "Client Status Read:", errf)
		} else { // Not a status frame
			sarwin.MsgPrintln(g, "blue_black", "Client Expected status but received ",
				sarflags.GetStr(header, "frametpye"), "frame")
			errflag <- "badpacket"
			return
		}

		// Process the status header
		if sarflags.GetStr(header, "metadatarecvd") == "no" {
			// No metadata has been received yet so send/resend it
			if retcode := frames.UDPWrite(m, conn, addr); retcode != "success" {
				errflag <- retcode
				return
			}
			sarwin.PacketPrintln(g, "cyan_black", "Tx", m.ShortPrint())
		}
		// We have "success" so Decode into a Status
		var st status.Status
		if err := frames.Decode(&st, rframe); err != nil {
			sarwin.MsgPrintln(g, "red_black", "Client read Bad Status with error:", err)
			errflag <- "badstatus"
			return
		} // else {
		// sarwin.MsgPrintln(g,  "blue_black", "Status Frame IS GOOD:", st.Print())
		// }

		// Send back to the caller the current progress & inrespto over the channel so we can process
		// them in the transfer as well as a success status
		var proins [2]uint64
		proins[0] = st.Progress
		proins[1] = st.Inrespto
		pos <- proins

		if st.Progress == filelen {
			sarwin.MsgPrintln(g, "blue_black", "Client File",
				t.filename, "length", filelen, "successfully transferred")
			errflag <- "success"
		}
		// Handle Holes
		for _, h := range st.Holes {
			// We re-read in from fp all of the holes
			//Allocate a buffer to hold all of the hole data
			buf := make([]byte, h.End-h.Start)
			var rlen int
			var err error
			// Seek to the hole start and read it all into buf
			if rlen, err = t.fp.ReadAt(buf, int64(h.Start)); err != nil {
				sarwin.MsgPrintln(g, "blue_black", "Client we have a bad Hole:", h.Start, h.End)
				errflag <- "badoffset"
				return
			}
			var pend int

			plen := maxpaylen(dflags) // Work out maximum payload for data frame
			dframes := rlen / plen    // Work out how many frames we need to re-send
			// Loop around re-sending data frames for the hole
			for fc := 0; fc < dframes; fc++ { // Bump frame counter
				var df data.Data
				d := &df
				pstart := rlen - (fc * plen)
				if pstart+plen > int(h.End) {
					pend = int(h.End) // Last frame may be shorter
				} else {
					pend = pstart + plen
				}
				dinfo := data.Dinfo{Session: t.session, Offset: uint64(pstart), Payload: buf[pstart:pend]}
				if frames.New(d, dflags, &dinfo) != nil {
					errflag <- "badpacket"
					return
				} // Create the Data
				if retcode := frames.UDPWrite(d, conn, addr); retcode != "success" {
					errflag <- retcode
					return
				}
				sarwin.PacketPrintln(g, "cyan_black", "Tx", d.ShortPrint())
			}
		}
		sarwin.MsgPrintln(g, "blue_black", "Client File",
			t.filename, "length", filelen, "successfully processed status")
		errflag <- "success"
		// MMMMM should this be a forever looping for !!!!!!
	}
}

// senddata - Read from fp and send the data to the server
// Handle datapos as recieved from the channel - THIS NOT DONE IN THIS SIMPLE VERSION YET!!!
// Send out errflag on its channel if failure or success (done)
func senddata(g *gocui.Gui, t *CTransfer, dflags string, conn *net.UDPConn, addr *net.UDPAddr,
	datapos chan [2]uint64, errflag chan string) {

	var curpos uint64

	fcount := 0
	eod := false
	flags := replaceflag(dflags, "eod=no")
	if fv := flagvalue(dflags, "reqtstamp"); fv != "no" && fv != "" {
		timetype := "reqtstamp=" + fv // Set it to the appropriate timestamp type to use
		flags = replaceflag(flags, timetype)
	}

	// Allocate a read buffer for a data frame
	rbuf := make([]byte, maxpaylen(dflags))
	sarwin.MsgPrintln(g, "yellow_black", "Client Max Data Payload Len=", len(rbuf))
	for { // Just blast away and send the complete file asking for a status every 100 frames sent
		if t.fp == nil {
			sarwin.MsgPrintln(g, "yellow_black", "File Pointer is nil", len(rbuf))
			errflag <- "accessdenied"
		}
		nread, err := t.fp.ReadAt(rbuf, int64(curpos))
		if err != nil && err != io.EOF {
			errflag <- "accessdenied"
			return
		}
		if err == io.EOF { // We have read in the whole file
			flags = replaceflag(flags, "eod=yes")
			eod = true
		}
		fcount++
		if fcount == 100 || eod { // We want a status back after every 100 frames sent and at the end
			flags = replaceflag(flags, "reqstatus=yes")
			fcount = 0
		} else {
			flags = replaceflag(flags, "reqstatus=no")
		}

		// OK so create the data frame and send it
		var df data.Data
		d := &df

		dinfo := data.Dinfo{Session: t.session, Offset: curpos, Payload: rbuf[:nread]}
		if frames.New(d, flags, &dinfo) != nil {
			errflag <- "badpacket"
			return
		}
		// sarwin.MsgPrintln(g, "white_black", "New Data (", len(rbuf[:nread]), ")", d.Print())
		if retcode := frames.UDPWrite(d, conn, addr); retcode != "success" {
			errflag <- retcode
		}
		sarwin.PacketPrintln(g, "cyan_black", "Tx", d.ShortPrint())
		curpos += uint64(nread)
		// sarwin.MsgPrintln(g,  "yellow_black", "Data Frame Written is:", d.Print(), "nread=", nread, "curpos=", curpos)
		if eod { // All read and sent so we are done with the senddata loop
			break
		}
	}
	errflag <- "success"
}

// CMatch - Return a pointer to the transfer if we find it, nil otherwise
func CMatch(ttype string, ip string, fname string) *CTransfer {
	ttypeok := false
	addrok := false

	// Check that transfer type is valid
	for _, tt := range Ttypes {
		if tt == ttype {
			ttypeok = true
			break
		}
	}
	// Check that ip address is valid
	var addr net.IP
	if addr = net.ParseIP(ip); addr != nil { // We have a valid IP Address
		addrok = true
	}
	if !ttypeok || !addrok {
		return nil
	}

	for _, i := range CTransfers {
		if ttype == i.ttype && addr.Equal(i.peer) && fname == i.filename {
			return &i
		}
	}
	return nil
}

// Doclient -- Execute the command entered
// Function pointer to the go routine for the transaction type
// Spawns a go thread for the command to execute
func Doclient(t *CTransfer, g *gocui.Gui, errstr chan string) {
	for _, i := range Ttypes {
		if i == t.ttype {
			fn, ok := clienthandler[i]
			if ok {
				errflag := make(chan string, 1) // The return channel holding the saratoga errflag
				go fn(t, g, errflag)
				retcode := <-errflag
				close(errflag)
				sarwin.MsgPrintln(g, "yellow_black", "Client Doclient completed with errstr:", retcode)
				errstr <- retcode
				return
			}
		}
	}
	errstr <- "undefined"
}

// CNew - Add a new transfer to the CTransfers list
func (t *CTransfer) CNew(g *gocui.Gui, ttype string, ip string, fname string, c *sarflags.Cliflags) error {

	// screen.Fprintln(g,  "red_black", "Addtran for", ip, fname, flags)
	if addr := net.ParseIP(ip); addr != nil { // We have a valid IP Address
		for _, i := range CTransfers { // Don't add duplicates
			if addr.Equal(i.peer) && fname == i.filename {
				emsg := fmt.Sprintf("Client CTransfer for %s to %s is currently in progress, cannnot add transfer",
					fname, i.peer.String())
				sarwin.MsgPrintln(g, "red_black", emsg)
				return errors.New(emsg)
			}
		}

		// Lock it as we are going to add a new transfer slice
		ctrmu.Lock()
		defer ctrmu.Unlock()
		t.direction = "client"
		t.ttype = ttype
		t.tstamp = c.Timestamp
		t.session = newsession()
		t.peer = addr
		t.filename = fname

		// NOW COPY THE FLAGS to t.cliflags
		t.cliflags = new(sarflags.Cliflags)
		if err := sarflags.CopyCliflags(t.cliflags, c); err != nil {
			panic(err)
		}

		msg := fmt.Sprintf("Client Added %s CTransfer to %s %s",
			t.ttype, t.peer.String(), t.filename)
		CTransfers = append(CTransfers, *t)
		sarwin.MsgPrintln(g, "green_black", msg)
		return nil
	}
	sarwin.MsgPrintln(g, "red_black", "Client CTransfer not added, invalid IP address", ip)
	return errors.New("invalid IP Address")
}

// Remove - Remove a CTransfer from the CTransfers
func (t *CTransfer) Remove() error {
	ctrmu.Lock()
	defer ctrmu.Unlock()
	for i := len(CTransfers) - 1; i >= 0; i-- {
		if t.peer.Equal(CTransfers[i].peer) && t.filename == CTransfers[i].filename {
			CTransfers = append(CTransfers[:i], CTransfers[i+1:]...)
			return nil
		}
	}
	emsg := fmt.Sprintf("Client Cannot remove %s CTransfer for %s to %s",
		t.ttype, t.filename, t.peer.String())
	return errors.New(emsg)
}

// FmtPrint - String of relevant transfer info
func (t *CTransfer) FmtPrint(sfmt string) string {
	return fmt.Sprintf(sfmt, t.direction,
		t.ttype,
		t.peer.String(),
		t.filename)
}

// Print - String of relevant transfer info
func (t *CTransfer) Print() string {
	return fmt.Sprintf("%s|%s|%s|%s", t.direction,
		t.ttype,
		t.peer.String(),
		t.filename)
}

/*
 *************************************************************************************************
 * CLIENT TRANSFER HANDLERS
 *************************************************************************************************
 */

type clientfunc func(*CTransfer, *gocui.Gui, chan string)

// Client Commands and function pointers to handle them
var clienthandler = map[string]clientfunc{
	// "get":	cget,
	// "getrm":	cgetrm,
	// "getdir":	cgetdir,
	"put":      cput,
	"putrm":    cputrm,
	"putblind": cputblind,
	// "rm":		crm,
	// "rmdir":	crmdir,
}

/*
 *************************************************************************************************
 * CLIENT PUT HANDLERS put, putrm, putblind, rm & rmdir
 *************************************************************************************************
 */

// client put a file to server
// Engine - Send Request, Send Metadata, Wait for Status
// 		Loop Sending Data and receiving intermittant Status
// 		Resend Metadata if Requested in Status
// 		Status is requested in Data every status timer secs
//		or Datacnt Data frames sent, whichever comes first
//		Abort with error if Rx Status errcode != "success"
//
func cput(t *CTransfer, g *gocui.Gui, errflag chan string) {
	var err error
	var fp *os.File
	var pos int64

	// Open the local data file for reading only
	fname := os.Getenv("SARDIR") +
		string(os.PathSeparator) +
		strings.TrimLeft(t.filename, string(os.PathSeparator))
	if t.fp, err = os.Open(fname); err != nil {
		t.fp = fp
		sarwin.MsgPrintln(g, "red_black", "Client Cannot open", fname)
		errflag <- "filenotfound"
		return
	}
	defer t.fp.Close()
	tdesc := filedescriptor(fname) // CTransfer descriptor to be used

	if pos, err = t.fp.Seek(0, io.SeekStart); err != nil {
		sarwin.MsgPrintln(g, "red_black", "Client Cannot seek to", pos)
		errflag <- "badoffset"
		return
	}

	// Set up the connection
	var udpad string
	var udpaddr *net.UDPAddr

	if t.peer.To4() == nil { // IPv6
		udpad = "[" + t.peer.String() + "]" + ":" + strconv.Itoa(t.cliflags.Port)
	} else { // IPv4
		udpad = t.peer.String() + ":" + strconv.Itoa(t.cliflags.Port)
	}
	if udpaddr, err = net.ResolveUDPAddr("udp", udpad); err != nil {
		errflag <- "cantsend"
		return
	}
	sarwin.MsgPrintln(g, "magenta_black", "Client Dialing ", udpaddr.String())
	conn, err := net.DialUDP("udp", nil, udpaddr)
	if err != nil {
		s := "Could not Dial " + udpaddr.String() + ":" + err.Error()
		sarwin.MsgPrintln(g, "red_black", s)
		conn.Close()
		errflag <- "cantsend"
		return
	}

	// Create the request & make a frame for normal request/status exchange startup
	var req request.Request
	r := &req
	rflags := "reqtype=put,fileordir=file,"
	rflags += sarflags.Setglobal("request", t.cliflags)
	rflags = replaceflag(rflags, tdesc)
	sarwin.MsgPrintln(g, "magenta_black", "Client Request Flags <", rflags, ">")
	rinfo := request.Rinfo{Session: t.session, Fname: t.filename, Auth: nil}
	if err := frames.New(r, rflags, &rinfo); err != nil {
		sarwin.MsgPrintln(g, "red_black", "Client Cannot create request", err.Error())
		conn.Close()
		errflag <- "badrequest"
		return
	}
	if retcode := frames.UDPWrite(r, conn, udpaddr); retcode != "success" {
		conn.Close()
		errflag <- retcode
		return
	}
	sarwin.PacketPrintln(g, "cyan_black", "Tx", r.ShortPrint())

	sarwin.MsgPrintln(g, "green_black", "Sent:", t.Print())
	sarwin.MsgPrintln(g, "green_black", "Client CTransfer Request Sent to",
		t.peer.String())

	// Create the metadata & send
	var met metadata.MetaData
	m := &met
	mflags := "transfer=file,progress=inprogress,"
	mflags += sarflags.Setglobal("metadata", t.cliflags)
	mflags = replaceflag(mflags, tdesc)
	sarwin.MsgPrintln(g, "magenta_black", "Client Metadata Flags <", mflags, ">")
	minfo := metadata.Minfo{Session: t.session, Fname: t.filename}
	if err = frames.New(m, mflags, &minfo); err != nil {
		sarwin.MsgPrintln(g, "red_black", "Client Cannot create metadata", err.Error())
		conn.Close()
		errflag <- "badrequest"
		return
	}
	if retcode := frames.UDPWrite(m, conn, udpaddr); retcode != "success" {
		conn.Close()
		errflag <- retcode
		return
	}
	sarwin.PacketPrintln(g, "cyan_black", "Tx", m.ShortPrint())

	// Prime the data header flags for the transfer
	// during the transfer we only play with "eod" after this
	// For retransmitting holes we also need to know the data flags to use
	dflags := "transfer=file,eod=no,"
	dflags += sarflags.Setglobal("data", t.cliflags)
	dflags = replaceflag(dflags, tdesc)
	sarwin.MsgPrintln(g, "magenta_black", "Client Data Flags <", dflags, ">")

	statuserr := make(chan string, 1)    // The return channel holding the saratoga errflag
	statuspos := make(chan [2]uint64, 1) // The return channel from readstatus with progress & inrespto
	datapos := make(chan [2]uint64, 1)
	dataerr := make(chan string, 1)

	// ISSUE CAUSING HANG SOMEWHERE IN HERE!!!!

	// This is the guts of handling status. It sits in a loop reading away and processing
	// the status when received. It sends metadata & data (to fill holes) as required
	go readstatus(g, t, dflags, conn, udpaddr, m, statuspos, statuserr)
	go senddata(g, t, dflags, conn, udpaddr, datapos, dataerr)
	for { // Multiplex between writing data & reading status when we have messages coming back
		select {
		case serr := <-statuserr:
			var progress, inrespto uint64
			if n, _ := fmt.Sscanf(serr, "%d %d", &progress, &inrespto); n == 2 {
				sarwin.MsgPrintln(g, "magenta_black", "Client Progress=", progress, "Inrespto=", inrespto)
				var dpos [2]uint64
				// Send on datapos channel to senddata the latest progress & inrespto indicators
				dpos[0] = progress
				dpos[1] = inrespto
				datapos <- dpos
			} else if serr != "success" {
				sarwin.MsgPrintln(g, "red_black", "Client Status Error in go senddata:", serr)
				conn.Close()
				errflag <- serr
				return
			}
		case derr := <-dataerr:
			if derr != "success" {
				// Close the connection
				conn.Close()
				sarwin.MsgPrintln(g, "red_black", "Client Data Error in go senddata:", derr)
				errflag <- derr
				return
			}
		case spos := <-statuspos:
			sarwin.MsgPrintln(g, "magenta_black", "Client Read Status Pos=", spos)
		case dpos := <-datapos:
			sarwin.MsgPrintln(g, "magenta_black", "Client Read Data Pos=", dpos)
			// default: // the select is non-blocking, fall through
			// screen.Fprintf(g,  "magenta_black", "*")
		}
	}
}

// client blind put a file
func cputblind(t *CTransfer, g *gocui.Gui, errflag chan string) {
	var err error
	var pos int64

	// Open the local data file for reading only
	fname := os.Getenv("SARDIR") +
		string(os.PathSeparator) +
		strings.TrimLeft(t.filename, string(os.PathSeparator))
	if t.fp, err = os.Open(fname); err != nil {
		sarwin.MsgPrintln(g, "red_black", "Client cputblind Cannot open", fname)
		errflag <- "filenotfound"
		return
	}
	defer t.fp.Close()
	tdesc := filedescriptor(fname) // CTransfer descriptor to be used

	if pos, err = t.fp.Seek(0, io.SeekStart); err != nil {
		sarwin.MsgPrintln(g, "red_black", "Client cputblind Cannot seek to", pos)
		errflag <- "badoffset"
		return
	}

	// Set up the connection
	var udpad string
	if t.peer.To4() == nil { // IPv6
		udpad = "[" + t.peer.String() + "]" + ":" + strconv.Itoa(t.cliflags.Port)
	} else { // IPv4
		udpad = t.peer.String() + ":" + strconv.Itoa(t.cliflags.Port)
	}
	var udpaddr *net.UDPAddr
	if udpaddr, err = net.ResolveUDPAddr("udp", udpad); err != nil {
		errflag <- "cantsend"
		return
	}
	var conn *net.UDPConn
	if conn, err = net.DialUDP("udp", nil, udpaddr); err != nil {
		errflag <- "cantsend"
		return
	}
	defer conn.Close()

	// Create the request & make a frame for normal request/status exchange startup
	var met metadata.MetaData
	m := &met
	mflags := "transfer=file,progress=inprogress,"
	mflags += sarflags.Setglobal("metadata", t.cliflags)
	mflags = replaceflag(mflags, tdesc)
	minfo := metadata.Minfo{Session: t.session, Fname: t.filename}
	if err = frames.New(m, mflags, &minfo); err != nil {
		sarwin.MsgPrintln(g, "red_black", "Client cputblind Cannot create metadata", err.Error())
		errflag <- "badrequest"
		return
	}
	if retcode := frames.UDPWrite(m, conn, udpaddr); retcode != "success" {
		errflag <- retcode
		return
	}
	sarwin.PacketPrintln(g, "cyan_black", "Tx", m.ShortPrint())

	sarwin.MsgPrintln(g, "green_black", "Client cputblind Sent:", t.Print())
	sarwin.MsgPrintln(g, "green_black", "Client cputblind CTransfer Metadata Sent for blind put to",
		t.peer.String())
	errflag <- "success"
}

// client put a file then remove the local copy of that file
func cputrm(t *CTransfer, g *gocui.Gui, errflag chan string) {
	rmerrflag := make(chan string, 1) // The return channel holding the saratoga errflag
	defer close(rmerrflag)

	cput(t, g, rmerrflag)
	errcode := <-rmerrflag
	if errcode == "success" {
		fname := strings.TrimRight(os.Getenv("SARDIR"), "/") + "/" + t.filename
		sarwin.MsgPrintln(g, "green_black", "Client cputrm Successfully put file", fname)
		// All good so remove the local file
		if os.Remove(fname) != nil {
			sarwin.MsgPrintln(g, "red_black", "Client cputrm Cannot remove local file", fname)
			errflag <- "didnotdelete"
			return
		}
		sarwin.MsgPrintln(g, "red_black", "Client cputrm Local file", fname, "removed")
	}
	errflag <- errcode
}
