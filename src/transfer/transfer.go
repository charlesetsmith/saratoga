package transfer

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/charlesetsmith/saratoga/src/metadata"
	"github.com/charlesetsmith/saratoga/src/request"
	"github.com/charlesetsmith/saratoga/src/sarflags"
	"github.com/charlesetsmith/saratoga/src/sarnet"
	"github.com/charlesetsmith/saratoga/src/screen"
	"github.com/jroimartin/gocui"
)

// Ttypes - Transfer types
var Ttypes = []string{"get", "getrm", "getdir", "put", "putblind", "putrm", "rm", "rmdir"}

// current protected session number
var smu sync.Mutex
var sessionid uint32

// Create new Session number
func newsession() uint32 {

	smu.Lock()
	defer smu.Unlock()

	if sessionid == 0 {
		sessionid = uint32(os.Getpid()) + 1
	} else {
		sessionid++
	}
	return sessionid
}

// Hole - Beginning and end of a Hole
type Hole struct {
	start uint64
	end   uint64
}

// Transfer Information
type Transfer struct {
	direction string   // "client|server"
	ttype     string   // Transfer type "get,getrm,put,putrm,blindput,rm"
	session   uint32   // Session ID - This is the unique key
	peer      net.IP   // Remote Host
	filename  string   // File name to get from remote host
	frames    [][]byte // Frame queue
	holes     []Hole   // Holes
}

var trmu sync.Mutex

// Transfers - Get list used in get,getrm,getdir,put,putrm & delete
var Transfers = []Transfer{}

// New - Add a new transfer to the Transfers list
func (t *Transfer) New(g *gocui.Gui, ttype string, ip string, fname string) error {

	// screen.Fprintln(g, "msg", "red_black", "Addtran for", ip, fname, flags)
	if addr := net.ParseIP(ip); addr != nil { // We have a valid IP Address
		for _, i := range Transfers { // Don't add duplicates
			if addr.Equal(i.peer) && fname == i.filename {
				emsg := fmt.Sprintf("Transfer for %s to %s is currently in progress, cannnot add transfer",
					fname, i.peer.String())
				screen.Fprintln(g, "msg", "red_black", emsg)
				return errors.New(emsg)
			}
		}

		// Lock it as we are going to add a new transfer slice
		trmu.Lock()
		defer trmu.Unlock()
		t.direction = "client"
		t.ttype = ttype
		t.session = newsession()
		t.peer = addr
		t.filename = fname
		var msg string

		msg = fmt.Sprintf("Added %s Transfer to %s %s",
			t.ttype, t.peer.String(), t.filename)
		Transfers = append(Transfers, *t)
		screen.Fprintln(g, "msg", "green_black", msg)
		return nil
	}
	screen.Fprintln(g, "msg", "red_black", "Transfer not added, invalid IP address", ip)
	return errors.New("Invalid IP Address")
}

// Match - Return a pointer to the transfer if we find it, nil otherwise
func Match(ttype string, ip string, fname string) *Transfer {
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

	for _, i := range Transfers {
		if ttype == i.ttype && addr.Equal(i.peer) && fname == i.filename {
			return &i
		}
	}
	return nil
}

// Remove - Remove a Transfer from the Transfers
func (t *Transfer) Remove() error {
	trmu.Lock()
	defer trmu.Unlock()
	for i := len(Transfers) - 1; i >= 0; i-- {
		if t.peer.Equal(Transfers[i].peer) && t.filename == Transfers[i].filename {
			Transfers = append(Transfers[:i], Transfers[i+1:]...)
			return nil
		}
	}
	emsg := fmt.Sprintf("Cannot remove %s Transfer for %s to %s",
		t.ttype, t.filename, t.peer.String())
	return errors.New(emsg)
}

// Print - String of relevant transfer info
func (t *Transfer) Print() string {
	return fmt.Sprintf("%s %s %s %s", t.direction,
		t.ttype,
		t.peer.String(),
		t.filename)
}

// Info - List transfers in progress to msg window matching ttype or all if ""
func Info(g *gocui.Gui, ttype string) {
	var tinfo []Transfer

	for _, i := range Transfers {
		if ttype == "" {
			tinfo = append(tinfo, i)
		} else {
			for _, t := range Ttypes {
				if t == ttype {
					tinfo = append(tinfo, i)
				}
			}
		}
	}
	if len(tinfo) > 0 {
		var colour string
		for _, i := range tinfo {
			if i.direction == "client" {
				colour = "green_black"
			} else {
				colour = "yellow_black"
			}
			screen.Fprintln(g, "msg", colour, i.Print())
		}
	} else {
		msg := fmt.Sprintf("No %s transfers currently in progress", ttype)
		screen.Fprintln(g, "msg", "green_black", msg)
	}
}

// FileDescriptor - Get the appropriate descriptor flag size based on file length
func filedescriptor(fname string) string {
	if fi, err := os.Stat(fname); err == nil {
		size := uint64(fi.Size())
		if size <= sarflags.MaxUint16 {
			return "descriptor=d16"
		}
		if size <= sarflags.MaxUint32 {
			return "descriptor=d32"
		}
		if size <= sarflags.MaxUint64 {
			return "descriptor=d64"
		}
	}
	// Just send back the maximum supported descriptor
	if sarflags.MaxUint <= sarflags.MaxUint16 {
		return "descriptor=d16"
	}
	if sarflags.MaxUint <= sarflags.MaxUint32 {
		return "descriptor=d32"
	}
	return "descriptor=d64"
}

// Replace an existing flag or add it
func replaceflag(curflags string, newflag string) string {
	var fs string
	var replaced bool

	for _, curflag := range strings.Split(curflags, ",") {
		if strings.Split(curflag, "=")[0] == strings.Split(newflag, "=")[0] {
			replaced = true
			fs += newflag + ","
		} else {
			fs += curflag + ","
		}
	}
	if !replaced {
		fs += newflag
	}
	return strings.TrimRight(fs, ",")
}

// Doclient -- Execute the command entered
// Function pointer to the go routine for the transaction type
// Spawns a go thread for the command to execute
func Doclient(t *Transfer, g *gocui.Gui, errstr chan string) {
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

/*
 *************************************************************************************************
 * CLIENT TRANSFER HANDLERS
 *************************************************************************************************
 */

type clientfunc func(*Transfer, *gocui.Gui, chan string)

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

// cput - put a file from client to server
// Engine - Send Request, Send Metadata, Wait for Status
// 		Loop Sending Data and receiving intermittant Status
// 		Resend Metadata if Requested in Status
// 		Status is requested in Data every status timer secs
//		or statuscount Data frames sent, whichever comes first
//		Abort with error if Rx Status errcode != "success"
//
func cput(t *Transfer, g *gocui.Gui, errflag chan string) {
	var err error
	var wframe []byte // The frame to write
	var fp *os.File
	var pos int64

	// Open the local data file for reading only
	fname := os.Getenv("SARDIR") +
		string(os.PathSeparator) +
		strings.TrimLeft(t.filename, string(os.PathSeparator))
	if fp, err = os.Open(fname); err != nil {
		screen.Fprintln(g, "msg", "red_black", "Cannot open", fname)
		errflag <- "filenotfound"
		return
	}
	defer fp.Close()
	tdesc := filedescriptor(fname) // Transfer descriptor to be used

	if pos, err = fp.Seek(0, io.SeekStart); err != nil {
		screen.Fprintln(g, "msg", "red_black", "Cannot seek to", pos)
		errflag <- "badoffset"
		return
	}

	screen.Fprintln(g, "msg", "red_black", "ping 1")
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
	screen.Fprintln(g, "msg", "red_black", "ping 2")

	// Create the request & make a frame for normal request/status exchange startup
	var req request.Request
	r := &req
	rflags := "reqtype=put,fileordir=file,"
	rflags += sarflags.Setglobal("request")
	rflags = replaceflag(rflags, tdesc)
	screen.Fprintln(g, "msg", "magenta_black", "Request Flags <", rflags, ">")
	if err = r.New(rflags, t.session, t.filename, nil); err != nil {
		screen.Fprintln(g, "msg", "red_black", "Cannot create request", err.Error())
		errflag <- "badrequest"
		return
	}
	if wframe, err = r.Put(); err != nil {
		errflag <- "badrequest"
		return
	}
	screen.Fprintln(g, "msg", "red_black", "ping 3")
	// Send the request frame
	_, err = conn.Write(wframe)
	screen.Fprintln(g, "msg", "red_black", "ping 4")
	if err != nil {
		errflag <- "cantsend"
		return
	}
	screen.Fprintln(g, "msg", "green_black", "Sent:", t.Print())
	screen.Fprintln(g, "msg", "green_black", "Transfer Request Sent to",
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
		errflag <- "badrequest"
		return
	}
	if wframe, err = m.Put(); err != nil {
		errflag <- "badrequest"
		return
	}
	// Send the initial metadata frame
	_, err = conn.Write(wframe)
	screen.Fprintln(g, "msg", "red_black", "ping 4")
	if err != nil {
		errflag <- "cantsend"
		return
	}
	errflag <- "success"
	return

	/*
		rbuf := make([]byte, sarflags.MaxBuff)
		sendmetadata := false

		   next:
		   	for { // Sit in a loop reading status and writing data and maybe metadata frames
		   		screen.Fprintln(g, "msg", "yellow_black", "Waiting to Read a Frame")
		   		rlen, err := conn.Read(rbuf)
		   		if err != nil {
		   			errflag <- "cantreceive"
		   			return
		   		}
		   		screen.Fprintln(g, "msg", "yellow_black", "Read a Frame len", rlen, "bytes")
		   		rframe := make([]byte, rlen)
		   		copy(rframe, rbuf[:rlen])
		   		header := binary.BigEndian.Uint32(rframe[:4])
		   		if sarflags.GetStr(header, "version") != "v1" { // Make sure we are Version 1
		   			screen.Fprintln(g, "msg", "red_black", "Not Saratoga Version 1 Frame from ",
		   				t.peer.String())
		   			errflag <- "badpacket"
		   			return
		   		}
		   		// Process the received frame
		   		if sarflags.GetStr(header, "frametype") == "status" {
					   errf := sarflags.GetStr(header, "errcode")
					   if errf != "success" {
						   errflag <- errf
						   return
					   }
					// Process the status header
		   			if sarflags.GetStr(header, "metadatarecvd") == "no" {
		   				// No metadata has been received yet so send/resend it
		   				if wframe, err = m.Put(); err != nil {
		   					errflag <- "badrequest"
		   					return
		   				}
		   				_, err = conn.Write(wframe)
		   				if err != nil {
		   					errflag <- "cantsend"
		   					return
		   				}
		   			} else {
		   				errflag <- "badpacket"
						   return
					   }
		   		}
		   		// Send a data frame
		   	}
	*/
}

// ClientBlindPut - blind put a file
func cputblind(t *Transfer, g *gocui.Gui, errflag chan string) {
	var err error
	var wframe []byte // The frame to write
	var fp *os.File
	var pos int64

	// Open the local data file for reading only
	fname := os.Getenv("SARDIR") +
		string(os.PathSeparator) +
		strings.TrimLeft(t.filename, string(os.PathSeparator))
	if fp, err = os.Open(fname); err != nil {
		screen.Fprintln(g, "msg", "red_black", "Cannot open", fname)
		errflag <- "filenotfound"
		return
	}
	defer fp.Close()
	tdesc := filedescriptor(fname) // Transfer descriptor to be used

	if pos, err = fp.Seek(0, io.SeekStart); err != nil {
		screen.Fprintln(g, "msg", "red_black", "Cannot seek to", pos)
		errflag <- "badoffset"
		return
	}

	screen.Fprintln(g, "msg", "red_black", "ping 1")
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
	screen.Fprintln(g, "msg", "red_black", "ping 2")

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
	screen.Fprintln(g, "msg", "red_black", "ping 4")
	if err != nil {
		errflag <- "cantsend"
		return
	}
	screen.Fprintln(g, "msg", "green_black", "Sent:", t.Print())
	screen.Fprintln(g, "msg", "green_black", "Transfer Metadata Sent for blind put to",
		t.peer.String())
	errflag <- "success"
	return
}

// ClientPutrm - put a file then remove the local copy of that file
func cputrm(t *Transfer, g *gocui.Gui, errflag chan string) {
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
