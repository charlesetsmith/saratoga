package transfer

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/charlesetsmith/saratoga/src/request"
	"github.com/charlesetsmith/saratoga/src/sarflags"
	"github.com/charlesetsmith/saratoga/src/screen"
	"github.com/charlesetsmith/saratoga/src/status"
	"github.com/jroimartin/gocui"
)

// STransfer Server Transfer Info
type STransfer struct {
	direction string   // "client|server"
	ttype     string   // STransfer type "get,getrm,put,putrm,blindput,rm"
	tstamp    string   // Timestamp type used in transfer
	session   uint32   // Session ID - This is the unique key
	peer      net.IP   // Remote Host
	filename  string   // File name to get from remote host
	fp        *os.File // Local File name to write to
}

var strmu sync.Mutex

// STransfers - Server transfers in progress
var STransfers = []STransfer{}

// compose process status frames
// We read from and seek within fp
// Our connection to the client is conn
// We assemble Status using sflags
// We transmit status as required
// We send back a string holding the status error code "success" keeps transfer alive
func writestatus(g *gocui.Gui, t *STransfer, sflags string, conn *net.UDPConn,
	progresp chan [2]uint64,
	hole chan []status.Hole,
	inerrflag chan string,
	errflag chan string) {

	// var filelen uint64
	// var fi os.FileInfo
	// var err error

	// Grab the file informaion
	//if fi, err = t.fp.Stat(); err != nil {
	//	errflag <- "filenotfound"
	//	return
	//}
	// filelen = uint64(fi.Size())
	var maxholes = stpaylen(sflags) // Work out maximum # holes we can put in a single status frame
	for {
		prval := <-progresp // Read in the current progress & inrespto
		progress := prval[0]
		inrespto := prval[1]
		holes := <-hole // Read in the current holes we need to process

		inerr := <-inerrflag // Errflag to send to the client
		errf := "errflag=" + inerr
		flags := replaceflag(sflags, errf)
		var lasthole int
		if errf == "success" {
			lasthole = len(holes) // How many holes do we have
		} else {
			lasthole = 0 // We have no holes if an error is being sent
		}

		var framecnt int // Number of status frames we will need (at least 1)
		if lasthole <= maxholes {
			framecnt = 1
			flags = replaceflag(sflags, "allholes=yes")
		} else {
			framecnt = len(holes)/maxholes + 1
			flags = replaceflag(sflags, "allholes=no")
		}

		// Loop through creating and sending the status frames with the holes in them
		for fc := 0; fc < framecnt; fc++ {
			starthole := fc * maxholes
			endhole := fc*maxholes + maxholes
			if endhole > lasthole {
				endhole = lasthole
			}

			var st status.Status
			if err := st.New(flags, t.session, progress, inrespto, holes[starthole:endhole]); err != nil {
				errflag <- "badstatus"
				return
			}
			var wframe []byte
			var err error
			if wframe, err = st.Put(); err != nil {
				errflag <- "badstatus"
				return
			}
			_, err = conn.Write(wframe)
			if err != nil {
				errflag <- "cantsend"
				return
			}
		}
		errflag <- "success"
		return
	}
}

func readdata(g *gocui.Gui, t *STransfer, sflags string, conn *net.UDPConn,
	progresp chan [2]uint64,
	hole chan []status.Hole,
	inerrflag chan string,
	errflag chan string) {
}

// SMatch - Return a pointer to the transfer if we find it, nil otherwise
func SMatch(ttype string, ip string, session uint32) *STransfer {
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

	for _, i := range STransfers {
		if ttype == i.ttype && addr.Equal(i.peer) && session == i.session {
			return &i
		}
	}
	return nil
}

// SNew - Add a new transfer to the STransfers list
func SNew(g *gocui.Gui, ttype string, r request.Request, ip string, session uint32) error {

	var t STransfer
	// screen.Fprintln(g, "msg", "red_black", "Addtran for", ip, fname, flags)
	if addr := net.ParseIP(ip); addr != nil { // We have a valid IP Address
		for _, i := range STransfers { // Don't add duplicates
			if addr.Equal(i.peer) && session == i.session {
				emsg := fmt.Sprintf("STransfer for session %d to %s is currently in progress, cannnot add transfer",
					session, i.peer.String())
				screen.Fprintln(g, "msg", "red_black", emsg)
				return errors.New(emsg)
			}
		}

		// Lock it as we are going to add a new transfer slice
		strmu.Lock()
		defer strmu.Unlock()
		t.direction = "server"
		t.ttype = ttype
		t.tstamp = sarflags.Cli.Timestamp
		t.session = session
		t.peer = addr
		// t.filename = fname
		var msg string

		msg = fmt.Sprintf("Added %s Transfer to %s session %d",
			t.ttype, t.peer.String(), t.session)
		STransfers = append(STransfers, t)
		screen.Fprintln(g, "msg", "green_black", msg)
		return nil
	}
	screen.Fprintln(g, "msg", "red_black", "CTransfer not added, invalid IP address", ip)
	return errors.New("Invalid IP Address")
}

// Remove - Remove a CTransfer from the CTransfers
func (t *STransfer) Remove() error {
	strmu.Lock()
	defer strmu.Unlock()
	for i := len(STransfers) - 1; i >= 0; i-- {
		if t.peer.Equal(CTransfers[i].peer) && t.session == STransfers[i].session {
			STransfers = append(STransfers[:i], STransfers[i+1:]...)
			return nil
		}
	}
	emsg := fmt.Sprintf("Cannot remove %s Transfer for session %d to %s",
		t.ttype, t.session, t.peer.String())
	return errors.New(emsg)
}

// FmtPrint - String of relevant transfer info
func (t *STransfer) FmtPrint(sfmt string) string {
	return fmt.Sprintf(sfmt, t.direction,
		t.ttype,
		t.peer.String(),
		t.session)
}

// Print - String of relevant transfer info
func (t *STransfer) Print() string {
	return fmt.Sprintf("%s|%s|%s|%d", t.direction,
		t.ttype,
		t.peer.String(),
		t.session)
}
