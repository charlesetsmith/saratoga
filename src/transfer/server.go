package transfer

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/charlesetsmith/saratoga/src/data"
	"github.com/charlesetsmith/saratoga/src/dirent"
	"github.com/charlesetsmith/saratoga/src/metadata"
	"github.com/charlesetsmith/saratoga/src/request"
	"github.com/charlesetsmith/saratoga/src/sarflags"
	"github.com/charlesetsmith/saratoga/src/screen"
	"github.com/charlesetsmith/saratoga/src/status"
	"github.com/charlesetsmith/saratoga/src/timestamp"
	"github.com/jroimartin/gocui"
)

// STransfer Server Transfer Info
type STransfer struct {
	direction string              // "client|server"
	ttype     string              // STransfer type "get,getrm,put,putrm,blindput,rm"
	tstamp    timestamp.Timestamp // Timestamp type used in transfer
	peer      net.IP              // Remote Host
	session   uint32              // Session + peer is the unique key
	stflags   string              // Status Flags currently set WORK ON THIS!!!!!
	filename  string              // Remote File name to get/put
	csumtype  string              // What type of checksum are we using
	havemeta  bool                // Have we recieved a metadata yet
	checksum  []byte              // Checksum of the remote file to be get/put if requested
	dir       dirent.DirEnt       // Directory entry info of the remote file to be get/put
	fp        *os.File            // Local File to write to/read from
	data      []byte              // Buffered data
	progress  uint64              // Current Progress indicator
	inrespto  uint64              // In respose to indicator
	holes     []status.Hole       // What holes I need to fill
}

var strmu sync.Mutex

// STransfers - Slice of Server transfers in progress
var STransfers = []STransfer{}

// Dcount - Data frmae counter
var Dcount int

// Writestatus -- compose & semd status frames
// Our connection to the client is conn
// We assemble Status using sflags
// We transmit status immediately
// We send back a string holding the status error code or "success" keeps transfer alive
func Writestatus(g *gocui.Gui, t *STransfer, sflags string, conn *net.UDPConn, remoteAddr *net.UDPAddr) string {

	var maxholes = stpaylen(sflags) // Work out maximum # holes we can put in a single status frame

	errf := flagvalue(sflags, "errcode")
	var lasthole int
	if errf == "success" {
		lasthole = len(t.holes) // How many holes do we have
	} else {
		lasthole = 0 // We have no holes if an error is being sent
	}

	var framecnt int // Number of status frames we will need (at least 1)
	flags := sflags
	if lasthole <= maxholes {
		framecnt = 1
		flags = replaceflag(sflags, "allholes=yes")
	} else {
		framecnt = len(t.holes)/maxholes + 1
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
		if err := st.New(flags, t.session, t.progress, t.inrespto, t.holes[starthole:endhole]); err != nil {
			return "badstatus"
		}
		var wframe []byte
		var err error
		if wframe, err = st.Put(); err != nil {
			return "badstatus"
		}
		_, err = conn.WriteToUDP(wframe, remoteAddr)
		if err != nil {
			return "cantsend"
		}
	}
	return "success"
}

func readdata(g *gocui.Gui, t *STransfer, sflags string, conn *net.UDPConn,
	progresp chan [2]uint64,
	hole chan []status.Hole,
	inerrflag chan string,
	errflag chan string) {
}

// SMatch - Return a pointer to the STransfer if we find it, nil otherwise
func SMatch(ip string, session uint32) *STransfer {

	// Check that ip address is valid
	var addr net.IP
	if addr = net.ParseIP(ip); addr == nil { // We have a valid IP Address
		return nil
	}

	for _, i := range STransfers {
		if addr.Equal(i.peer) && session == i.session {
			return &i
		}
	}
	return nil
}

// SNew - Add a new transfer to the STransfers list upon receipt of a request
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
		t.session = session
		t.peer = addr
		t.havemeta = false
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

// SChange - Add metadata information to the STransfer in STransfers list upon receipt of metadata
func (t *STransfer) SChange(g *gocui.Gui, m metadata.MetaData) {
	// Lock it as we are going to add a new transfer slice
	strmu.Lock()
	t.csumtype = sarflags.GetStr(m.Header, "csumtype")
	t.checksum = make([]byte, len(m.Checksum))
	copy(t.checksum, m.Checksum)
	t.dir = m.Dir
	t.havemeta = true
	screen.Fprintln(g, "msg", "yellow_black", "Added metadata info to transfer", t.Print())
	strmu.Unlock()
}

// SData - Add data information to the STransfer in STransfers list upon receipt of data
func (t *STransfer) SData(g *gocui.Gui, d data.Data, conn *net.UDPConn, remoteAddr *net.UDPAddr) {
	// Lock it as we are going to add a new transfer slice
	strmu.Lock()
	if sarflags.GetStr(d.Header, "reqtstamp") == "yes" { // Grab the timestamp from data
		t.tstamp = d.Tstamp
	}
	Dcount++
	if Dcount%100 == 0 { // Send back a status every 100 data frames recieved
		Dcount = 0
	}
	if Dcount == 0 || sarflags.GetStr(d.Header, "reqstatus") == "yes" || !t.havemeta { // Send a status back
		stheader := "descriptor=" + sarflags.GetStr(d.Header, "descriptor") // echo the descriptor
		stheader += ",allholes=yes,reqholes=requested,errcode=success,"
		if !t.havemeta {
			stheader += "metadatarecvd=no"
		} else {
			stheader += "metadatarecvd=yes"
		}
		// Send back a status to the client to tell it a success with creating the transfer
		Writestatus(g, t, stheader, conn, remoteAddr)
	}

	// copy(t.data[d.Offset:], d.Payload)
	strmu.Unlock()
}

// Remove - Remove a STransfer from the STransfers
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

// FmtPrint - String of relevant STransfer info
func (t *STransfer) FmtPrint(sfmt string) string {
	return fmt.Sprintf(sfmt, t.direction,
		t.ttype,
		t.peer.String(),
		t.session)
}

// Print - String of relevant STransfer info
func (t *STransfer) Print() string {
	return fmt.Sprintf("%s|%s|%s|%d|%s\n\t%s", t.direction,
		t.ttype,
		t.peer.String(),
		t.session,
		t.csumtype,
		t.dir.Print())
}
