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

// New - Add a new transfer to the Transfers list
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
	return fmt.Sprintf("%s %s %s %s", t.ttype, t.peer.String(), t.filename, t.flags)
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
		for _, i := range tinfo {
			screen.Fprintln(g, "msg", "green_black", i.Print())
		}
	} else {
		msg := fmt.Sprintf("No %s transfers currently in progress", ttype)
		screen.Fprintln(g, "msg", "green_black", msg)
	}
}

/*
 *************************************************************************************************
 * CLIENT TRANSFER HANDLERS
 *************************************************************************************************
 */

// ClientPut - put a file
func (t *Transfer) ClientPut(g *gocui.Gui, errflag chan string) {
	var err error
	var wframe []byte // The frame to write

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
	if !t.blind {
		var req request.Request
		r := &req
		if err = r.New(t.flags, t.session, t.filename, nil); err != nil {
			screen.Fprintln(g, "msg", "red_black", "Cannot create request", err.Error())
			errflag <- "badrequest"
			return
		}
		if wframe, err = r.Put(); err != nil {
			errflag <- "badrequest"
			return
		}
		screen.Fprintln(g, "msg", "red_black", "ping 3")
	} else { // Create the metadata & make a frame for blind put startup
		var met metadata.MetaData
		m := &met
		if err = m.New(t.flags, t.session, t.filename); err != nil {
			screen.Fprintln(g, "msg", "red_black", "Cannot create metadata", err.Error())
			errflag <- "badrequest"
			return
		}
		if wframe, err = m.Put(); err != nil {
			errflag <- "badrequest"
			return
		}
	}

	// Send the initial request or metadata frame
	_, err = conn.Write(wframe)
	screen.Fprintln(g, "msg", "red_black", "ping 4")
	if err != nil {
		errflag <- "cantsend"
		return
	}
	screen.Fprintln(g, "msg", "green_black", "Sent:", t.Print())
	if !t.blind {
		screen.Fprintf(g, "msg", "green_black", "Transfer Request Sent to %s\n",
			t.peer.String())
	} else {
		screen.Fprintf(g, "msg", "green_black", "Transfer Metadata Sent for blind put to %s\n",
			t.peer.String())
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
		   		switch sarflags.GetStr(header, "frametype") {
		   		case "status":
		   			errf := sarflags.GetStr(header, "errcode")
		   			switch errf {
		   			case "success": // All good process status

		   			case "metadatarequired": // Flag to send a metadata all good process status
		   				sendmetadata = true

		   			case "badoffset", // Send warning & Drop the frame (dont process status)
		   				"badpacket",
		   				"badstatus",
		   				"didnotdelete":
		   				goto next

		   			case "internaltimeout", // Kill the Transfer
		   				"rxnotinterested",
		   				"fileinuse",
		   				"rxtimeout",
		   				"filetobig",
		   				"accessdenied",
		   				"cantsend",
		   				"cantreceive",
		   				"filenotfound",
		   				"unknownid",
		   				"unspecified",
		   				"badrequest",
		   				"baddataflag":
		   				errflag <- errf
		   				return

		   			default: // Invlid code Send warning & Drop the frame (dont process status)
		   				goto next
		   			}

		   			if sarflags.GetStr(header, "metadatarecvd") == "no" || sendmetadata {
		   				// No metadata has been received or it has been re-requested so send/resend it
		   				sendmetadata = false
		   				var met metadata.MetaData
		   				m := &met
		   				if err = m.New(t.flags, t.session, t.filename); err != nil {
		   					screen.Fprintln(g, "msg", "red_black", "Cannot create metadata", err.Error())
		   					errflag <- "badrequest"
		   					return
		   				}
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

		   		default:
		   			errflag <- "badpacket"
		   			return
		   		}
		   		// Send a data frame
		   	}
	*/
}

// ClientPutrm - put a file then remove the local copy of that file
func (t *Transfer) ClientPutrm(g *gocui.Gui, errflag chan string) {
	rmerrflag := make(chan string, 1) // The return channel holding the saratoga errflag
	defer close(rmerrflag)

	t.ClientPut(g, rmerrflag)
	errcode := <-rmerrflag
	errflag <- errcode
	// MORE TO GO HERE TO REMOVE THE LOCAL FILE
	return
}
