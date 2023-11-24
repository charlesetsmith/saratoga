package transfer

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/jroimartin/gocui"

	"github.com/charlesetsmith/saratoga/dirent"
	"github.com/charlesetsmith/saratoga/frames"
	"github.com/charlesetsmith/saratoga/holes"
	"github.com/charlesetsmith/saratoga/metadata"
	"github.com/charlesetsmith/saratoga/request"
	"github.com/charlesetsmith/saratoga/sarflags"
	"github.com/charlesetsmith/saratoga/sarwin"
	"github.com/charlesetsmith/saratoga/status"
	"github.com/charlesetsmith/saratoga/timestamp"
)

// Ttypes - Transfer types
var Ttypes = []string{"get", "getrm", "getdir", "put", "putblind", "putrm", "rm", "rmdir"}

// Transfer direction we are a sender or receiver
const Initiator bool = true
const Responder bool = false

var Directions = map[bool]string{true: "Initiator", false: "Responder"}

// current protected session number
var smu sync.Mutex
var sessionid uint32

type Transfer struct {
	Direction  bool                // Am I the Initiator or Responder end of the connection
	Session    uint32              // Session ID - This is the unique key
	Peer       net.IP              // IP Address of the peer
	Conn       *net.UDPConn        // The connection to the remote peer
	Ttype      string              // Transfer type "get,getrm,put,putrm,putblind,rm"
	Tstamp     timestamp.Timestamp // Latest timestamp received from Data
	Tstamptype string              // Timestamp type "localinterp,posix32,posix64,posix32_32,posix64_32,epoch2000_32"
	Filename   string              // Local File name to receive or remove from remote host or send from local host
	Fp         *os.File            // File pointer for local file
	// frames    [][]byte           // Frames to process
	// holes     holes.Holes        // Holes to process
	Version    string               // Flag
	Fileordir  string               // Flag
	Udplite    string               // Flag
	Descriptor string               // Flag
	Stream     string               // Flag
	Csumtype   string               // What type of checksum are we using
	Havemeta   bool                 // Have we recieved a metadata yet
	Checksum   []byte               // Checksum of the remote file to be get/put if requested
	Dir        *dirent.DirEnt       // Directory entry info of the file to get/put
	Fileinfo   *dirent.FileMetaData // File metadata of the local file
	Data       []byte               // Buffered data
	Dcount     uint64               // Number Data frames sent/recieved
	Framecount uint64               // Total number frames received in this transfer (so we can schedule status)
	Progress   uint64               // Current Progress indicator
	Inrespto   uint64               // In respose to indicator
	Curfills   holes.Holes          // What has been received
	Cliflags   *sarflags.Cliflags   // Global flags used in this transfer
}

// Transfers - protected transfers in progress
var Trmu sync.Mutex
var Transfers = []Transfer{}

// Lookup - Return a pointer to the transfer if we find it in Transfers, nil otherwise
func Lookup(direction bool, session uint32, peer string) *Transfer {
	var addr net.IP
	if addr = net.ParseIP(peer); addr == nil { // Do we have a valid IP Address
		return nil
	}
	for _, i := range Transfers {
		remaddr := net.ParseIP(i.Conn.RemoteAddr().String())
		// Check if direction (Initiator or Responder), session # and IP address match in our current
		// list of transfers (if so then return a pointer to it)
		if direction == i.Direction && session == i.Session && addr.Equal(remaddr) {
			return &i
		}
	}
	return nil
}

// Lookup a host & session and return transfer pointer or nil if it does not exist
func Match(addr, session string) *Transfer {
	ses, err := strconv.Atoi(session)
	for i := len(Transfers) - 1; i >= 0; i-- {
		if err == nil && addr == Transfers[i].Conn.RemoteAddr().String() && uint32(ses) == Transfers[i].Session {
			return &Transfers[i]
		}
	}
	return nil
}

// WriteErrStatus - Send an error status
func WriteErrStatus(g *gocui.Gui, flags string, session uint32, conn *net.UDPConn, remoteAddr *net.UDPAddr) string {
	if sarflags.FlagValue(flags, "errcode") == "success" { // Dont send success that is silly
		return "success"
	}
	var st status.Status
	sinfo := status.Sinfo{Session: session, Progress: 0, Inrespto: 0, Holes: nil}
	if err := frames.New(&st, flags, &sinfo); err != nil {
		// if err := st.New(flags, session, 0, 0, nil); err != nil {
		return "badstatus"
	}
	err := frames.UDPWrite(&st, conn)
	sarwin.PacketPrintln(g, "cyan_black", "Tx ", st.ShortPrint())
	return err
}

// CNew - Add a new transfer to the Transfers list
func NewInitiator(g *gocui.Gui, ttype string, ip string, fname string, c *sarflags.Cliflags) (*Transfer, error) {
	// screen.Fprintln(g,  "red_black", "Addtran for ", ip, " ", fname, " ", flags)
	if addr := net.ParseIP(ip); addr != nil { // We have a valid IP Address
		for _, i := range Transfers { // Don't add duplicates (ie dont try act on same fname)
			if addr.Equal(i.Peer) && fname == i.Filename { // We can't write to same file
				emsg := fmt.Sprintf("Initiator Transfer for %s to %s is currently in progress, cannnot add transfer",
					fname, i.Peer.String())
				sarwin.ErrPrintln(g, "red_black", emsg)
				return nil, errors.New(emsg)
			}
		}

		// Lock it as we are going to add a new transfer slice
		Trmu.Lock()
		defer Trmu.Unlock()
		t := new(Transfer)
		t.Direction = Initiator
		t.Ttype = ttype
		t.Tstamptype = c.Timestamp
		t.Session = newsession()
		t.Peer = addr
		t.Filename = fname

		// Copy the FLAGS to t.cliflags
		var err error
		if t.Cliflags, err = c.CopyCliflags(); err != nil {
			panic(err)
		}
		msg := fmt.Sprintf("Initiator Added %s Transfer to %s %s",
			t.Ttype, t.Peer.String(), t.Filename)
		Transfers = append(Transfers, *t)
		sarwin.MsgPrintln(g, "green_black", msg)
		return t, nil
	}
	sarwin.ErrPrintln(g, "red_black", "Initiator Transfer not added, invalid IP address ", ip)
	return nil, errors.New("invalid IP Address")
}

// New - Add a new transfer to the Transfers list upon receipt of a request
// when we receive a request we are therefore a "server"
func NewResponder(g *gocui.Gui, r request.Request, ip string) error {

	var err error
	var addr net.IP
	if addr = net.ParseIP(ip); addr == nil { // Do we have a valid IP Address
		sarwin.ErrPrintln(g, "red_black", "Transfer not added, invalid IP address ", ip)
		return errors.New(" invalid IP Address")
	}
	if Lookup(Responder, r.Session, ip) != nil {
		emsg := fmt.Sprintf("Transfer %s for session %d to %s is currently in progress, cannnot duplicate transfer",
			Directions[Responder], r.Session, ip)
		sarwin.ErrPrintln(g, "red_black", emsg)
		return errors.New(emsg)
	}
	// Lock it as we are going to add a new transfer
	Trmu.Lock()
	defer Trmu.Unlock()
	t := new(Transfer)
	t.Direction = Responder // We are the Responder
	t.Session = r.Session
	// The Header flags set for the transfer
	t.Version = sarflags.GetStr(r.Header, "version")       // What version of saratoga
	t.Ttype = sarflags.GetStr(r.Header, "reqtype")         // What is the request type "get,getrm,put,putrm,putblind,rm"
	t.Fileordir = sarflags.GetStr(r.Header, "fileordir")   // Are we handling a file or directory entry
	t.Udplite = sarflags.GetStr(r.Header, "udplite")       // Should always be "no"
	t.Stream = sarflags.GetStr(r.Header, "stream")         // Denotes a named pipe
	t.Descriptor = sarflags.GetStr(r.Header, "descriptor") // What descriptor we use for the transfer

	t.Peer = addr
	t.Havemeta = false
	t.Framecount = 0 // No data yet. count of data frames
	t.Csumtype = ""  // We don't know checksum type until we get a metadata
	t.Checksum = nil // Nor do we know what it is
	t.Filename = r.Fname
	t.Tstamptype = "" // Filled out with status or data frame "localinterp,posix32,posix64,posix32_32,posix64_32,epoch2000_32"
	t.Progress = 0    // Current progress indicator
	t.Inrespto = 0    // Cururent In response to indicator
	t.Dir = nil       //

	// flags
	// property - normalfile, normaldirectory, specialfile, specialdirectory
	// descriptor - d16, d32, d64, d128
	// reliability - yes, no
	var flags string

	switch t.Ttype {
	case "get", "getrm", "rm": // We are acting on a file local to this system
		// Find the file metadata to get it's properties
		if t.Fileinfo, err = dirent.FileMeta(t.Filename); err != nil {
			return err
		}
		if t.Fileinfo.IsDir {
			flags = sarflags.AddFlag("", "property", "normaldirectory")
			flags = sarflags.AddFlag(flags, "reliability", "yes")
		} else if t.Fileinfo.IsRegular {
			flags = sarflags.AddFlag("", "property", "normalfile")
			flags = sarflags.AddFlag(flags, "reliability", "yes")
		} else { // specialfile (no such thing as a "specialdirectory")
			flags = sarflags.AddFlag("", "property", "specialfile")
			flags = sarflags.AddFlag(flags, "reliability", "no")
		}
		flags = sarflags.AddFlag(flags, "descriptor", t.Descriptor)

	case "put", "putrm": // FIX THIS!!!!!!! We are creating/deleting a file local to this system
		flags = sarflags.AddFlag(flags, "reliability", "yes")
	case "putblind": // We are putting a file onto this system
		flags = sarflags.AddFlag(flags, "reliability", "no")
	}
	if t.Dir, err = dirent.New(flags, t.Filename); err != nil {
		return err
	}
	t.Curfills = nil
	if t.Cliflags, err = sarwin.Cmdptr.CopyCliflags(); err != nil {
		return errors.New("Cannot copy CLI flags for transfer")
	}

	// conn * net.UDPConn // The connection to the remote peer
	// fp * os.File       // File pointer for local file
	// frames    [][]byte           // Frames to process
	// holes     holes.Holes        // Holes to process
	t.Data = nil // Buffered data

	msg := fmt.Sprintf("Added %s Transfer to %s session %d",
		Directions[t.Direction], ip, r.Session)
	Transfers = append(Transfers, *t)
	sarwin.MsgPrintln(g, "green_black", msg)
	return nil
}

// Info - List transfers in progress to msg window
func Info(g *gocui.Gui, ttype string) {
	var tinfo []Transfer

	for i := range Transfers {
		if ttype == "" || Transfers[i].Ttype == ttype {
			tinfo = append(tinfo, Transfers[i])
		}
	}
	if len(tinfo) > 0 {
		var maxaddrlen, maxfname int // Work out the width for the table
		for key := range tinfo {
			if len(tinfo[key].Conn.RemoteAddr().String()) > maxaddrlen {
				maxaddrlen = len(tinfo[key].Conn.RemoteAddr().String())
			}
			if len(tinfo[key].Filename) > maxfname {
				maxfname = len(tinfo[key].Conn.RemoteAddr().String())
			}
		}
		// Table format
		sfmt := fmt.Sprintf("|%%6s|%%8s|%%%ds|%%%ds|\n", maxaddrlen, maxfname)
		sborder := fmt.Sprintf(sfmt, strings.Repeat("-", 6), strings.Repeat("-", 8),
			strings.Repeat("-", maxaddrlen), strings.Repeat("-", maxfname))

		var sslice sort.StringSlice
		for key := range tinfo {
			sslice = append(sslice, tinfo[key].FmtPrint(sfmt))
		}
		sort.Sort(sslice)

		sbuf := sborder
		sbuf += fmt.Sprintf(sfmt, "Direct", "Tran Typ", "IP", "Fname")
		sbuf += sborder
		for key := 0; key < len(sslice); key++ {
			sbuf += sslice[key]
		}
		sbuf += sborder
		sarwin.MsgPrintln(g, "magenta_black", sbuf)
	} else {
		msg := fmt.Sprintf("No %s transfers currently in progress", ttype)
		sarwin.MsgPrintln(g, "green_black", msg)
	}
}

// WriteStatus -- compose & send status frames
// Our connection to the client is conn
// We assemble Status using sflags
// We transmit status immediately
// We send back a string holding the status error code or "success" keeps transfer alive
func (t *Transfer) WriteStatus(g *gocui.Gui, sflags string) string {

	if t.Conn != nil {
		sarwin.MsgPrintln(g, "cyan_black", "Responder Connection from ", t.Conn.RemoteAddr().String())
	}
	sarwin.MsgPrintln(g, "cyan_black", "Responder Assemble & Send status to ", t.Conn.RemoteAddr().String())
	var maxholes = stpaylen(sflags) // Work out maximum # holes we can put in a single status frame

	errf := sarflags.FlagValue(sflags, "errcode")
	var lasthole int
	h := t.Curfills.Getholes()
	if errf == "success" {
		lasthole = len(h) // How many holes do we have
	}

	var framecnt int // Number of status frames we will need (at least 1)
	flags := sflags
	if lasthole <= maxholes {
		framecnt = 1
		flags = sarflags.ReplaceFlag(sflags, "allholes", "yes")
	} else {
		h := t.Curfills.Getholes()
		framecnt = len(h)/maxholes + 1
		flags = sarflags.ReplaceFlag(sflags, "allholes", "no")
	}

	// Loop through creating and sending the status frames with the holes in them
	for fc := 0; fc < framecnt; fc++ {
		starthole := fc * maxholes
		endhole := starthole + maxholes
		if endhole > lasthole {
			endhole = lasthole
		}

		var st status.Status
		h := t.Curfills.Getholes()
		sinfo := status.Sinfo{Session: t.Session, Progress: t.Progress, Inrespto: t.Inrespto, Holes: h}
		if err := frames.New(&st, flags, &sinfo); err != nil {
			sarwin.MsgPrintln(g, "cyan_black", "Responder Bad Status:", err, frames.Print(&st))
			return "badstatus"
		}
		if e := frames.UDPWrite(&st, t.Conn); e != "success" {
			//sarwin.MsgPrintln(g, "cyan_black", "Responder cant write Status:", e, frames.Print(&st),
			//	"to", conn.RemoteAddr().String())
			return e
		} else {
			sarwin.PacketPrintln(g, "cyan_black", "Tx ", st.ShortPrint())
			sarwin.MsgPrintln(g, "cyan_black", "Responder Sent Status:", frames.Print(&st),
				" to ", t.Conn.RemoteAddr().String())
		}
	}
	return "success"
}

// Change - Add metadata information to the Transfer in Transfers list upon receipt of a metadata
func (t *Transfer) Change(g *gocui.Gui, m metadata.MetaData) error {
	// Lock it as we are going to add a new transfer slice
	Trmu.Lock()
	defer Trmu.Unlock()
	t.Csumtype = sarflags.GetStr(m.Header, "csumtype")
	t.Checksum = make([]byte, len(m.Checksum))
	copy(t.Checksum, m.Checksum)
	t.Dir = m.Dir.Copy()
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
	return nil
}

// Remove - Remove a Transfer from the Transfers
func (t *Transfer) Remove() error {
	Trmu.Lock()
	defer Trmu.Unlock()

	for i := len(Transfers) - 1; i >= 0; i-- {
		if Lookup(t.Direction, t.Session, t.Conn.RemoteAddr().String()) != nil {
			Transfers = append(Transfers[:i], Transfers[i+1:]...)
			return nil
		}
	}
	emsg := fmt.Sprintf("Cannot remove %s Transfer for session %d to %s",
		Directions[t.Direction], t.Session, t.Conn.RemoteAddr().String())
	return errors.New(emsg)
}

// FmtPrint - String of relevant transfer info
func (t *Transfer) FmtPrint(sfmt string) string {
	return fmt.Sprintf(sfmt, "Initiator",
		t.Ttype,
		t.Conn.RemoteAddr().String(),
		t.Filename)
}

// Print - String of relevant transfer info
func (t *Transfer) Print() string {
	return fmt.Sprintf("%s|%s|%s|%s", Directions[t.Direction],
		t.Ttype,
		t.Conn.RemoteAddr().String(),
		t.Filename)
}

func (t *Transfer) Do(g *gocui.Gui, chan e error) {
	sarwin.MsgPrintln(g, "Doing command")
	e = nil
	<- e
}