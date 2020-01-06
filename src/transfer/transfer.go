package transfer

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"

	"github.com/charlesetsmith/saratoga/src/metadata"
	"github.com/charlesetsmith/saratoga/src/request"
	"github.com/charlesetsmith/saratoga/src/sarflags"
	"github.com/charlesetsmith/saratoga/src/sarnet"
	"github.com/charlesetsmith/saratoga/src/screen"
	"github.com/jroimartin/gocui"
)

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
	ttype    string   // Transfer type "get,getrm,put,putrm,blindput,rm"
	session  uint32   // Session ID - This is the unique key
	peer     net.IP   // Remote Host
	filename string   // File name to get from remote host
	flags    string   // Flag Header to be used
	blind    bool     // Is this a blind transfer (no initial request/status exchange)
	frames   [][]byte // Frame queue
	holes    []Hole   // Holes
}

var trmu sync.Mutex

// Transfers - Get list used in get,getrm,getdir,put,putrm & delete
var Transfers = []Transfer{}

// New - Add a new transfer to the Transfers list and return pointer to it
func (t *Transfer) New(g *gocui.Gui, ttype string, ip string, fname string, blind bool, flags string) error {

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
		t.ttype = ttype
		t.session = newsession()
		t.peer = addr
		t.filename = fname
		t.blind = blind
		var msg string

		if !blind { // request/status exchange
			t.flags = flags + "," + sarflags.Setglobal("request", t.filename)
			msg = fmt.Sprintf("Added %s Transfer to %s %s %s",
				t.ttype, t.peer.String(), t.filename, t.flags)
		} else { // no request/status required just metadata then data
			t.flags = flags + "," + sarflags.Setglobal("metadata", t.filename)
			msg = fmt.Sprintf("Added Blind %s Transfer to %s %s %s",
				t.ttype, t.peer.String(), t.filename, t.flags)
		}
		Transfers = append(Transfers, *t)
		screen.Fprintln(g, "msg", "green_black", msg)
		return nil
	}
	screen.Fprintln(g, "msg", "red_black", "Transfer not added, invalid IP address", ip)
	return errors.New("Invalid IP Address")
}

// Print - String of relevant transfer info
func (t *Transfer) Print() string {
	return fmt.Sprintf("%s %s %s %s", t.ttype, t.peer.String(), t.filename, t.flags)
}

// Info - List transfers in progress to msg window
func Info(g *gocui.Gui, ttype string) {
	var tinfo []Transfer

	for _, i := range Transfers {
		switch ttype {
		case "get":
			if i.ttype == "get" {
				tinfo = append(tinfo, i)
			}
		case "getrm":
			if i.ttype == "getrm" {
				tinfo = append(tinfo, i)
			}

		case "getdir":
			if i.ttype == "getdir" {
				tinfo = append(tinfo, i)
			}
		case "put":
			if i.ttype == "put" {
				tinfo = append(tinfo, i)
			}
		case "putrm":
			if i.ttype == "putrm" {
				tinfo = append(tinfo, i)
			}
		case "putblind":
			if i.ttype == "putblind" {
				tinfo = append(tinfo, i)
			}
		case "rm":
			if i.ttype == "rm" {
				tinfo = append(tinfo, i)
			}
		case "rmdir":
			if i.ttype == "rmdir" {
				tinfo = append(tinfo, i)
			}
		default:
			tinfo = append(tinfo, i)
		}
	}
	if len(tinfo) > 0 {
		for _, i := range tinfo {
			screen.Fprintln(g, "msg", "green_black", i.Print())
		}
	} else {
		msg := fmt.Sprintf("No %s transfers currently in progress", ttype)
		screen.Fprintln(g, "msg", "green_black", msg)
	}
}

// Client - Go routine to setup a client connection to a peer to get/put/delete/getdir files
func (t *Transfer) Client(g *gocui.Gui, errflag chan string) {

	var err error

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
	var frame []byte
	if !t.blind {
		var req request.Request
		r := &req
		if err = r.New(t.flags, t.session, t.filename, nil); err != nil {
			screen.Fprintln(g, "msg", "red_black", "Cannot create request", err.Error())
			errflag <- "badrequest"
			return
		}
		if frame, err = r.Put(); err != nil {
			errflag <- "badrequest"
			return
		}
	} else { // Create the metadata & make a frame for blind put startup
		var met metadata.MetaData
		m := &met
		if err = m.New(t.flags, t.session, t.filename); err != nil {
			screen.Fprintln(g, "msg", "red_black", "Cannot create metadata", err.Error())
			errflag <- "badrequest"
			return
		}
		if frame, err = m.Put(); err != nil {
			errflag <- "badrequest"
			return
		}
	}

	// Send the frame
	_, err = conn.Write(frame)
	if err != nil {
		errflag <- "cantsend"
		return
	}
	// screen.Fprintln(g, "msg", "green_black", "Sent:", txb.Print())
	if !t.blind {
		screen.Fprintf(g, "msg", "green_black", "Transfer Request Sent to %s\n",
			t.peer.String())
	} else {
		screen.Fprintf(g, "msg", "green_black", "Transfer Metadata Sent for blind put to %s\n",
			t.peer.String())
	}
	errflag <- "success"
}
