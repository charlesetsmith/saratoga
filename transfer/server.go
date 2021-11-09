// Server Transfer

package transfer

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/charlesetsmith/saratoga/dirent"
	"github.com/charlesetsmith/saratoga/frames"
	"github.com/charlesetsmith/saratoga/holes"
	"github.com/charlesetsmith/saratoga/metadata"
	"github.com/charlesetsmith/saratoga/request"
	"github.com/charlesetsmith/saratoga/sarflags"
	"github.com/charlesetsmith/saratoga/sarwin"
	"github.com/charlesetsmith/saratoga/status"
	"github.com/charlesetsmith/saratoga/timestamp"
	"github.com/jroimartin/gocui"
)

// STransfer Server Transfer Info
type STransfer struct {
	Direction string              // "client|server"
	Ttype     string              // STransfer type "get,getrm,put,putrm,blindput,rm"
	Tstamp    timestamp.Timestamp // Timestamp type used in transfer
	Peer      net.IP              // Remote Host
	Session   uint32              // Session + peer is the unique key
	Stflags   string              // Status Flags currently set WORK ON THIS!!!!!
	Filename  string              // Remote File name to get/put
	Csumtype  string              // What type of checksum are we using
	Havemeta  bool                // Have we recieved a metadata yet
	Checksum  []byte              // Checksum of the remote file to be get/put if requested
	Dir       dirent.DirEnt       // Directory entry info of the remote file to be get/put
	Fp        *os.File            // Local File to write to/read from
	Data      []byte              // Buffered data
	Dcount    int                 // Count of Data frames so we can schedule status
	Progress  uint64              // Current Progress indicator
	Inrespto  uint64              // In respose to indicator
	CurFills  holes.Holes         // What has been received
}

// Strmu - Protect transfer
var Strmu sync.Mutex

// STransfers - Slice of Server transfers in progress
var STransfers = []STransfer{}

// Dcount - Data frmae counter
var Dcount int

// WriteErrStatus - Send an error status

func WriteErrStatus(g *gocui.Gui, flags string, session uint32, conn *net.UDPConn, remoteAddr *net.UDPAddr) string {
	if flagvalue(flags, "errcode") == "success" { // Dont send success that is silly
		return "success"
	}
	var st status.Status
	sinfo := status.Sinfo{Session: session, Progress: 0, Inrespto: 0, Holes: nil}
	if err := frames.New(&st, flags, &sinfo); err != nil {
		// if err := st.New(flags, session, 0, 0, nil); err != nil {
		return "badstatus"
	}
	err := frames.UDPWrite(&st, conn, remoteAddr)
	sarwin.PacketPrintln(g, "cyan_black", "Tx ", st.ShortPrint())
	return err
}

// WriteStatus -- compose & send status frames
// Our connection to the client is conn
// We assemble Status using sflags
// We transmit status immediately
// We send back a string holding the status error code or "success" keeps transfer alive
func WriteStatus(g *gocui.Gui, t *STransfer, sflags string, conn *net.UDPConn, remoteAddr *net.UDPAddr) string {

	if conn != nil {
		sarwin.MsgPrintln(g, "cyan_black", "Server Connection from ", remoteAddr.String())
	}
	sarwin.MsgPrintln(g, "cyan_black", "Server Assemble & Send status to ", remoteAddr.String())
	var maxholes = stpaylen(sflags) // Work out maximum # holes we can put in a single status frame

	errf := flagvalue(sflags, "errcode")
	var lasthole int
	h := t.CurFills.Getholes()
	if errf == "success" {
		lasthole = len(h) // How many holes do we have
	}

	var framecnt int // Number of status frames we will need (at least 1)
	flags := sflags
	if lasthole <= maxholes {
		framecnt = 1
		flags = replaceflag(sflags, "allholes=yes")
	} else {
		h := t.CurFills.Getholes()
		framecnt = len(h)/maxholes + 1
		flags = replaceflag(sflags, "allholes=no")
	}

	// Loop through creating and sending the status frames with the holes in them
	for fc := 0; fc < framecnt; fc++ {
		starthole := fc * maxholes
		endhole := starthole + maxholes
		if endhole > lasthole {
			endhole = lasthole
		}

		var st status.Status
		h := t.CurFills.Getholes()
		sinfo := status.Sinfo{Session: t.Session, Progress: t.Progress, Inrespto: t.Inrespto, Holes: h}
		if err := frames.New(&st, flags, &sinfo); err != nil {
			sarwin.MsgPrintln(g, "cyan_black", "Server Bad Status:", err, frames.Print(&st))
			return "badstatus"
		}
		if e := frames.UDPWrite(&st, conn, remoteAddr); e != "success" {
			//sarwin.MsgPrintln(g, "cyan_black", "Server cant write Status:", e, frames.Print(&st),
			//	"to", conn.RemoteAddr().String())
			return e
		} else {
			sarwin.PacketPrintln(g, "cyan_black", "Tx ", st.ShortPrint())
			sarwin.MsgPrintln(g, "cyan_black", "Server Sent Status:", frames.Print(&st),
				" to ", conn.RemoteAddr().String())
		}
	}
	return "success"
}

// SMatch - Return a pointer to the STransfer if we find it, nil otherwise
func SMatch(ip string, session uint32) *STransfer {

	// Check that ip address is valid
	var addr net.IP
	if addr = net.ParseIP(ip); addr == nil { // We have a valid IP Address
		return nil
	}

	for _, i := range STransfers {
		if addr.Equal(i.Peer) && session == i.Session {
			return &i
		}
	}
	return nil
}

// SNew - Add a new transfer to the STransfers list upon receipt of a request
func SNew(g *gocui.Gui, ttype string, r request.Request, ip string, session uint32) error {

	var t STransfer
	// screen.Fprintln(g,  "red_black", "Addtran for ", ip, " ",, fname, " ", flags)
	if addr := net.ParseIP(ip); addr != nil { // We have a valid IP Address
		for _, i := range STransfers { // Don't add duplicates
			if addr.Equal(i.Peer) && session == i.Session {
				emsg := fmt.Sprintf("STransfer for session %d to %s is currently in progress, cannnot add transfer",
					session, i.Peer.String())
				sarwin.MsgPrintln(g, "red_black", emsg)
				return errors.New(emsg)
			}
		}

		// Lock it as we are going to add a new transfer slice
		Strmu.Lock()
		defer Strmu.Unlock()
		t.Direction = "server"
		t.Ttype = ttype
		t.Session = session
		t.Peer = addr
		t.Havemeta = false
		t.Dcount = 0
		// t.filename = fname

		msg := fmt.Sprintf("Added %s Transfer to %s session %d",
			t.Ttype, t.Peer.String(), t.Session)
		STransfers = append(STransfers, t)
		sarwin.MsgPrintln(g, "green_black", msg)
		return nil
	}
	sarwin.MsgPrintln(g, "red_black", "CTransfer not added, invalid IP address ", ip)
	return errors.New(" invalid IP Address")
}

// SChange - Add metadata information to the STransfer in STransfers list upon receipt of a metadata
func (t *STransfer) SChange(g *gocui.Gui, m metadata.MetaData) error {
	// Lock it as we are going to add a new transfer slice
	Strmu.Lock()
	t.Csumtype = sarflags.GetStr(m.Header, "csumtype")
	t.Checksum = make([]byte, len(m.Checksum))
	copy(t.Checksum, m.Checksum)
	t.Dir = m.Dir
	// Create the file buffer for the transfer
	// AT THE MOMENT WE ARE HOLDING THE WHOLE FILE IN A MEMORY BUFFER!!!!
	// OF COURSE WE NEED TO SORT THIS OUT LATER
	if len(t.Data) == 0 { // Create the buffer only once
		t.Data = make([]byte, t.Dir.Size)
	}
	if len(t.Data) != (int)(m.Dir.Size) {
		emsg := fmt.Sprintf("Size of File Differs - Old=%d New=%d",
			len(t.Data), m.Dir.Size)
		return errors.New(emsg)
	}
	t.Havemeta = true
	sarwin.MsgPrintln(g, "yellow_black", "Added metadata to transfer and file buffer size ", len(t.Data))
	Strmu.Unlock()
	return nil
}

// Remove - Remove a STransfer from the STransfers
func (t *STransfer) Remove() error {
	Strmu.Lock()
	defer Strmu.Unlock()
	for i := len(STransfers) - 1; i >= 0; i-- {
		if t.Peer.Equal(CTransfers[i].peer) && t.Session == STransfers[i].Session {
			STransfers = append(STransfers[:i], STransfers[i+1:]...)
			return nil
		}
	}
	emsg := fmt.Sprintf("Cannot remove %s Transfer for session %d to %s",
		t.Ttype, t.Session, t.Peer.String())
	return errors.New(emsg)
}

// FmtPrint - String of relevant STransfer info
func (t *STransfer) FmtPrint(sfmt string) string {
	return fmt.Sprintf(sfmt, t.Direction,
		t.Ttype,
		t.Peer.String(),
		t.Session)
}

// Print - String of relevant STransfer info
func (t *STransfer) Print() string {
	return fmt.Sprintf("%s|%s|%s|%d|%s\n\t%s", t.Direction,
		t.Ttype,
		t.Peer.String(),
		t.Session,
		t.Csumtype,
		t.Dir.Print())
}
