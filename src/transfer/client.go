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

	"github.com/charlesetsmith/saratoga/src/data"
	"github.com/charlesetsmith/saratoga/src/metadata"
	"github.com/charlesetsmith/saratoga/src/request"
	"github.com/charlesetsmith/saratoga/src/sarflags"
	"github.com/charlesetsmith/saratoga/src/sarnet"
	"github.com/charlesetsmith/saratoga/src/screen"
	"github.com/charlesetsmith/saratoga/src/status"
	"github.com/jroimartin/gocui"
)

// CTransfer Information
type CTransfer struct {
	direction string // "client|server"
	ttype     string // CTransfer type "get,getrm,put,putrm,blindput,rm"
	tstamp    string // TImestamp type used in transfer
	session   uint32 // Session ID - This is the unique key
	peer      net.IP // Remote Host
	filename  string // File name to get from remote host
	fp        *os.File
	frames    [][]byte // Frame queue
	holes     []Hole   // Holes
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
func readstatus(g *gocui.Gui, t *CTransfer, dflags string, conn *net.UDPConn,
	m *metadata.MetaData, errflag chan string) {

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

	timeout := time.Duration(sarflags.Cli.Timeout.Status) * time.Second
	for {
		screen.Fprintln(g, "msg", "blue_black", "Waiting to Read a Status Frame on",
			conn.LocalAddr().String())
		conn.SetReadDeadline(time.Now().Add(timeout))
		rlen, err := conn.Read(rbuf)
		if err != nil {
			screen.Fprintln(g, "msg", "blue_black", "Timeout on Status Read",
				":", err.Error())
			errflag <- "cantreceive"
			return
		}
		// We have a status so grab it
		screen.Fprintln(g, "msg", "blue_black", "Read a Frame len", rlen, "bytes")
		rframe := make([]byte, rlen)
		copy(rframe, rbuf[:rlen])
		header := binary.BigEndian.Uint32(rframe[:4])
		if sarflags.GetStr(header, "version") != "v1" { // Make sure we are Version 1
			screen.Fprintln(g, "msg", "red_black", "Not Saratoga Version 1 Frame from ",
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
		} else { // Not a status frame
			errflag <- "badpacket"
			return
		}

		// Process the status header
		if sarflags.GetStr(header, "metadatarecvd") == "no" {
			// No metadata has been received yet so send/resend it
			var wframe []byte
			var err error
			if wframe, err = m.Put(); err != nil {
				errflag <- "badrequest"
				return
			}
			_, err = conn.Write(wframe)
			if err != nil {
				errflag <- "cantsend"
				return
			}
		}
		// We have "success" so Decode into a Status
		var st status.Status
		if st.Get(rframe) != nil {
			errflag <- "badstatus"
			return
		}

		// Send back the current progress & inrespto over the channel so we can process
		// then in the transfer
		errflag <- fmt.Sprintf("%d %d", st.Progress, st.Inrespto)
		if st.Progress == filelen {
			screen.Fprintln(g, "msg", "blue_black", "File",
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
				errflag <- "badoffset"
				return
			}
			var pend int

			plen := dpaylen(dflags) // Work out maximum payload for data frame
			dframes := rlen / plen  // Work out how many frames we need to re-send
			// Loop around re-sending data frames for the hole
			for fc := 0; fc < dframes; fc++ { // Bump frame counter
				var df data.Data
				pstart := rlen - (fc * plen)
				if pstart+plen > int(h.End) {
					pend = int(h.End) // Last frame may be shorter
				} else {
					pend = pstart + plen
				}

				df.New(dflags, t.session, uint64(pstart), buf[pstart:pend]) // Create the Data
				if wframe, err := df.Put(); err != nil {
					errflag <- "badoffset"
					return
				} else if _, err = conn.Write(wframe); err != nil { // And send it
					errflag <- "cantsend"
					return
				}
			}
		}
		errflag <- "success"
		return
	}
}

// senddata - Read from fp and send the data to the server
// Handle datapos as recieved from the channel - THIS NOT DONE IN THIS SIMPLE VERSION YET!!!
// Send out errflag on its channel if failure or success (done)
func senddata(g *gocui.Gui, t *CTransfer, dflags string, conn *net.UDPConn,
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
	rbuf := make([]byte, dpaylen(dflags))
	screen.Fprintln(g, "msg", "yellow_black", "Data Maxpayload=", dpaylen(dflags))
	for { // Just blast away and send the complete file asking for a status every 100 frames sent
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
		var d data.Data

		if d.New(flags, t.session, curpos, rbuf[:nread]) != nil {
			errflag <- "badpacket"
			return
		}
		if wframe, err := d.Put(); err != nil {
			errflag <- "badpacket"
			return
		} else if _, err = conn.Write(wframe); err != nil { // And send it
			errflag <- "cantsend"
			return
		}
		curpos += uint64(nread)
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
				errstr <- retcode
				return
			}
		}
	}
	errstr <- "undefined"
}

// New - Add a new transfer to the CTransfers list
func (t *CTransfer) New(g *gocui.Gui, ttype string, ip string, fname string) error {

	// screen.Fprintln(g, "msg", "red_black", "Addtran for", ip, fname, flags)
	if addr := net.ParseIP(ip); addr != nil { // We have a valid IP Address
		for _, i := range CTransfers { // Don't add duplicates
			if addr.Equal(i.peer) && fname == i.filename {
				emsg := fmt.Sprintf("CTransfer for %s to %s is currently in progress, cannnot add transfer",
					fname, i.peer.String())
				screen.Fprintln(g, "msg", "red_black", emsg)
				return errors.New(emsg)
			}
		}

		// Lock it as we are going to add a new transfer slice
		ctrmu.Lock()
		defer ctrmu.Unlock()
		t.direction = "client"
		t.ttype = ttype
		t.tstamp = sarflags.Cli.Timestamp
		t.session = newsession()
		t.peer = addr
		t.filename = fname
		var msg string

		msg = fmt.Sprintf("Added %s CTransfer to %s %s",
			t.ttype, t.peer.String(), t.filename)
		CTransfers = append(CTransfers, *t)
		screen.Fprintln(g, "msg", "green_black", msg)
		return nil
	}
	screen.Fprintln(g, "msg", "red_black", "CTransfer not added, invalid IP address", ip)
	return errors.New("Invalid IP Address")
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
	emsg := fmt.Sprintf("Cannot remove %s CTransfer for %s to %s",
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
	var wframe []byte // The frame to write
	var fp *os.File
	var pos int64

	// Open the local data file for reading only
	fname := os.Getenv("SARDIR") +
		string(os.PathSeparator) +
		strings.TrimLeft(t.filename, string(os.PathSeparator))
	if t.fp, err = os.Open(fname); err != nil {
		t.fp = fp
		screen.Fprintln(g, "msg", "red_black", "Cannot open", fname)
		errflag <- "filenotfound"
		return
	}
	defer t.fp.Close()
	tdesc := filedescriptor(fname) // CTransfer descriptor to be used

	if pos, err = t.fp.Seek(0, io.SeekStart); err != nil {
		screen.Fprintln(g, "msg", "red_black", "Cannot seek to", pos)
		errflag <- "badoffset"
		return
	}

	// Set up the connection
	var udpad string
	var udpaddr *net.UDPAddr

	if t.peer.To4() == nil { // IPv6
		udpad = "[" + t.peer.String() + "]" + ":" + strconv.Itoa(sarnet.Port())
	} else { // IPv4
		udpad = t.peer.String() + ":" + strconv.Itoa(sarnet.Port())
	}
	if udpaddr, err = net.ResolveUDPAddr("udp", udpad); err != nil {
		errflag <- "cantsend"
		return
	}
	conn, err := net.DialUDP("udp", nil, udpaddr)
	if err != nil {
		conn.Close()
		errflag <- "cantsend"
		return
	}

	// Create the request & make a frame for normal request/status exchange startup
	var req request.Request
	r := &req
	rflags := "reqtype=put,fileordir=file,"
	rflags += sarflags.Setglobal("request")
	rflags = replaceflag(rflags, tdesc)
	screen.Fprintln(g, "msg", "magenta_black", "Request Flags <", rflags, ">")
	if err = r.New(rflags, t.session, t.filename, nil); err != nil {
		screen.Fprintln(g, "msg", "red_black", "Cannot create request", err.Error())
		conn.Close()
		errflag <- "badrequest"
		return
	}
	if wframe, err = r.Put(); err != nil {
		conn.Close()
		errflag <- "badrequest"
		return
	}
	// Send the request frame
	_, err = conn.Write(wframe)
	if err != nil {
		conn.Close()
		errflag <- "cantsend"
		return
	}
	screen.Fprintln(g, "msg", "green_black", "Sent:", t.Print())
	screen.Fprintln(g, "msg", "green_black", "CTransfer Request Sent to",
		t.peer.String())

	// Create the metadata & send
	var met metadata.MetaData
	m := &met
	mflags := "transfer=file,progress=inprogress,"
	mflags += sarflags.Setglobal("metadata")
	mflags = replaceflag(mflags, tdesc)
	screen.Fprintln(g, "msg", "magenta_black", "Metadata Flags <", mflags, ">")
	if err = m.New(mflags, t.session, t.filename); err != nil {
		screen.Fprintln(g, "msg", "red_black", "Cannot create metadata", err.Error())
		conn.Close()
		errflag <- "badrequest"
		return
	}
	if wframe, err = m.Put(); err != nil {
		conn.Close()
		errflag <- "badrequest"
		return
	}
	// Send the initial metadata frame
	_, err = conn.Write(wframe)
	if err != nil {
		conn.Close()
		errflag <- "cantsend"
		return
	}

	// Prime the data header flags for the transfer
	// during the transfer we only play with "eod" after this
	// For retransmitting holes we also need to know the data flags to use
	dflags := "transfer=file,eod=no,"
	dflags += sarflags.Setglobal("data")
	dflags = replaceflag(dflags, tdesc)
	screen.Fprintln(g, "msg", "magenta_black", "Data Flags <", dflags, ">")

	statuserr := make(chan string, 1)  // The return channel holding the saratoga errflag
	dataerr := make(chan string, 1)    // The return channel holding the saratoga errflag
	datapos := make(chan [2]uint64, 1) // input channel from readstatus with progress & inrespto

	// This is the guts of handling status. It sits in a loop reading away and processing
	// the status when received. It sends metadata & data (to fill holes) as required
	go readstatus(g, t, dflags, conn, m, statuserr)
	go senddata(g, t, dflags, conn, datapos, dataerr)
	for { // Multiplex between writing data & reading status when we have messages coming back
		select {
		case serr := <-statuserr:
			var progress, inrespto uint64
			if n, _ := fmt.Sscanf(serr, "%d %d", &progress, &inrespto); n == 2 {
				screen.Fprintln(g, "msg", "magenta_black", "Progress=", progress, "Inrespto=", inrespto)
				var dpos [2]uint64
				// Send on datapos channel to senddata the latest progress & inrespto indicators
				dpos[0] = progress
				dpos[1] = inrespto
				datapos <- dpos
			} else if serr != "success" {
				conn.Close()
				errflag <- serr
				return
			}
		case derr := <-dataerr:
			if derr != "success" {
				conn.Close()
				errflag <- derr
				return
			}
		}
	}
}

// client blind put a file
func cputblind(t *CTransfer, g *gocui.Gui, errflag chan string) {
	var err error
	var wframe []byte // The frame to write
	var pos int64

	// Open the local data file for reading only
	fname := os.Getenv("SARDIR") +
		string(os.PathSeparator) +
		strings.TrimLeft(t.filename, string(os.PathSeparator))
	if t.fp, err = os.Open(fname); err != nil {
		screen.Fprintln(g, "msg", "red_black", "Cannot open", fname)
		errflag <- "filenotfound"
		return
	}
	defer t.fp.Close()
	tdesc := filedescriptor(fname) // CTransfer descriptor to be used

	if pos, err = t.fp.Seek(0, io.SeekStart); err != nil {
		screen.Fprintln(g, "msg", "red_black", "Cannot seek to", pos)
		errflag <- "badoffset"
		return
	}

	// Set up the connection
	var udpad string
	if t.peer.To4() == nil { // IPv6
		udpad = "[" + t.peer.String() + "]" + ":" + strconv.Itoa(sarnet.Port())
	} else { // IPv4
		udpad = t.peer.String() + ":" + strconv.Itoa(sarnet.Port())
	}
	conn, err := net.Dial("udp", udpad)
	defer conn.Close()
	if err != nil {
		errflag <- "cantsend"
		return
	}

	// Create the request & make a frame for normal request/status exchange startup
	var met metadata.MetaData
	m := &met
	mflags := "transfer=file,progress=inprogress,"
	mflags += sarflags.Setglobal("metadata")
	mflags = replaceflag(mflags, tdesc)
	if err = m.New(mflags, t.session, t.filename); err != nil {
		screen.Fprintln(g, "msg", "red_black", "Cannot create metadata", err.Error())
		errflag <- "badrequest"
		return
	}
	if wframe, err = m.Put(); err != nil {
		errflag <- "badrequest"
		return
	}

	// Send the initial metadata frame
	_, err = conn.Write(wframe)
	if err != nil {
		errflag <- "cantsend"
		return
	}
	screen.Fprintln(g, "msg", "green_black", "Sent:", t.Print())
	screen.Fprintln(g, "msg", "green_black", "CTransfer Metadata Sent for blind put to",
		t.peer.String())
	errflag <- "success"
	return
}

// client put a file then remove the local copy of that file
func cputrm(t *CTransfer, g *gocui.Gui, errflag chan string) {
	rmerrflag := make(chan string, 1) // The return channel holding the saratoga errflag
	defer close(rmerrflag)

	cput(t, g, rmerrflag)
	errcode := <-rmerrflag
	if errcode == "success" {
		fname := strings.TrimRight(os.Getenv("SARDIR"), "/") + "/" + t.filename
		screen.Fprintln(g, "msg", "green_black", "Successfully put file", fname)
		// All good so remove the local file
		if os.Remove(fname) != nil {
			screen.Fprintln(g, "msg", "red_black", "Cannot remove local file", fname)
			errflag <- "didnotdelete"
			return
		}
		screen.Fprintln(g, "msg", "red_black", "Local file", fname, "removed")
	}
	errflag <- errcode
	return
}
