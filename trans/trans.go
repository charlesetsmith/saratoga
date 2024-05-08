package trans

import (
	"net"
	"os"
	"sync"

	"github.com/charlesetsmith/saratoga/dirent"
	"github.com/charlesetsmith/saratoga/fileio"
	"github.com/charlesetsmith/saratoga/holes"
	"github.com/charlesetsmith/saratoga/request"
	"github.com/charlesetsmith/saratoga/sarflags"
	"github.com/charlesetsmith/saratoga/sarwin"
	"github.com/charlesetsmith/saratoga/timestamp"
	"github.com/jroimartin/gocui"
)

type Transfer struct {
	Peer     net.UDPAddr         // Remote Host
	Session  uint32              // Session + peer is the unique key
	Ttype    string              // Transfer type: get,take,put,give,delete,getdir
	Filename string              // Remote File name to get/put
	Timetype string              // Timestamp type: posix32, posix64, posix32_32, posix64_32, epock2000_32, ""
	Tstamp   timestamp.Timestamp // Timestamp used in transfer
	Stflags  string              // Status Flags currently set WORK ON THIS!!!!!
	Csumtype string              // What type of checksum are we using
	Checksum []byte              // Checksum of the remote file to be get/put if requested
	Havemeta bool                // Have we recieved a metadata yet
	Dir      dirent.DirEnt       // Directory entry info of the file to get/put
	Fp       *os.File            // Local File to write to/read from
	Data     []byte              // Buffered data
	Dcount   int                 // Count of Data frames so we can schedule status
	Progress uint64              // Current Progress indicator
	Inrespto uint64              // In respose to indicator
	CurFills holes.Holes         // What has been received
}

// Trmu - Protect transfer
var Trmu sync.Mutex

// Transfers - Slice of transfers currently in progress
var Transfers = []Transfer{}

// We have received a request frame from a remote host
// Create the transfer associated with the received request
func AddRx(g *gocui.Gui, r *request.Request, from *net.UDPAddr) bool {
	ttype := sarflags.GetStr(r.Header, "request")
	if sarflags.GetStr(r.Header, "udplite") != "no" {
		sarwin.ErrPrintln(g, "red_black", "UDP Lite not supported")
		return false
	}
	if sarflags.GetStr(r.Header, "stream") != "no" {
		sarwin.ErrPrintln(g, "red_black", "Streams not supported")
		return false
	}

	// See if the file exists on our local system
	exists := fileio.FileExists(r.Fname)
	switch ttype {
	// Open the local file to read from
	case "get", "getdir", "delete", "take":
		if !exists {
			sarwin.ErrPrintln(g, "red_black", "Local File ", r.Fname, "does not exist for ", ttype)
			return false
		}
	// Open the local file to write to
	case "put", "give":
		if exists {
			sarwin.ErrPrintln(g, "red_black", "Local File ", r.Fname, "already exists for ", ttype)
			return false
		}
	default:
		sarwin.ErrPrintln(g, "red_black", "Invalid request")
		return false
	}
	timetype := "epoch_2000_32"
	ts := new(timestamp.Timestamp)
	ts.Now(timetype)
	dent := new(dirent.DirEnt)

	// DO some error checking to see that the transfer is not already underway

	// Initialise the Transfer
	t := Transfer{Peer: *from, Session: r.Session, Ttype: ttype,
		Filename: r.Fname,
		Timetype: timetype, Tstamp: *ts,
		Stflags: "", Csumtype: "none", Checksum: nil,
		Havemeta: false, Dir: *dent, Fp: nil,
		Data: nil, Dcount: 0,
		Progress: 0, Inrespto: 0, CurFills: nil}
	Trmu.Lock()
	Transfers = append(Transfers, t)
	Trmu.Unlock()
	return true
}
