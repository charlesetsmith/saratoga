package trans

import (
	"net"
	"os"
	"sync"

	"github.com/charlesetsmith/saratoga/dirent"
	"github.com/charlesetsmith/saratoga/holes"
	"github.com/charlesetsmith/saratoga/request"
	"github.com/charlesetsmith/saratoga/sarflags"
	"github.com/charlesetsmith/saratoga/timestamp"
	"github.com/jroimartin/gocui"
)

type Transfer struct {
	Peer     net.UDPAddr         // Remote Host
	Session  uint32              // Session + peer is the unique key
	Ttype    string              // Transfer type: get,getrm,put,putrm,blindput,rm
	ForD     string              // File or Directory: file,dirextory
	Filename string              // Remote File name to get/put
	Udplite  string              // UDPLite: no,yes (Always NO!)
	Timetype string              // Timestamp type: posix32, posix64, posix32_32, posix64_32, epock2000_32, ""
	Tstamp   timestamp.Timestamp // Timestamp used in transfer
	Stflags  string              // Status Flags currently set WORK ON THIS!!!!!
	Csumtype string              // What type of checksum are we using
	Checksum []byte              // Checksum of the remote file to be get/put if requested
	Havemeta bool                // Have we recieved a metadata yet
	Dir      dirent.DirEnt       // Directory entry info of the remote file to be get/put
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

func Add(g *gocui.Gui, r *request.Request, from *net.UDPAddr) bool {
	ttype := sarflags.GetStr(r.Header, "request")
	ford := sarflags.GetStr(r.Header, "fileordir")
	udplite := sarflags.GetStr(r.Header, "udplite")
	timetype := "epoch_2000_32"
	ts := new(timestamp.Timestamp)
	ts.Now(timetype)
	dent := new(dirent.DirEnt)

	// DO some error checking to see that the transfer is not already underway

	// Initialise the Transfer
	t := Transfer{Peer: *from, Session: r.Session, Ttype: ttype, ForD: ford,
		Filename: r.Fname, Udplite: udplite,
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
