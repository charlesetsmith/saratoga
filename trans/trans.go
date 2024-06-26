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
	"github.com/charlesetsmith/saratoga/status"
	"github.com/charlesetsmith/saratoga/timestamp"
	"github.com/jroimartin/gocui"
)

type Transfer struct {
	Peer      net.UDPAddr         // Remote Host
	Session   uint32              // Session + peer is the unique key
	Ttype     string              // Transfer type: get,take,put,give,delete,getdir
	Rxwilling string              // Are we willing to receive files (ignored): no,invalid,capable,yes
	Txwilling string              // Are we willing to send files (ignored): no,invalid,capable,yes
	Filename  string              // Remote File name to get/put (if null send error status)
	Timetype  string              // Timestamp type: posix32, posix64, posix32_32, posix64_32, epock2000_32, ""
	Tstamp    timestamp.Timestamp // Timestamp used in transfer
	Stflags   string              // Status Flags currently set WORK ON THIS!!!!!
	Csumtype  string              // What type of checksum are we using
	Checksum  []byte              // Checksum of the remote file to be get/put if requested
	Havemeta  bool                // Have we recieved a metadata yet
	Dir       dirent.DirEnt       // Directory entry info of the file to get/put
	Fp        *os.File            // Local File to write to/read from
	Data      []byte              // Buffered data
	Dcount    int                 // Count of Data frames so we can schedule status
	Progress  uint64              // Current Progress indicator
	Inrespto  uint64              // In respose to indicator
	CurFills  holes.Holes         // What has been received
}

// Trmu - Protect transfer
var Trmu sync.Mutex

// Transfers - Slice of transfers currently in progress
var Transfers = []Transfer{}

// We have received a request frame from a remote host
// Create the transfer associated with the received request
// Send status frame back via the tx channel upon failure or success
func AddRxTran(g *gocui.Gui, r *request.Request, from *net.UDPAddr, tx chan interface{}) bool {
	ttype := sarflags.GetStr(r.Header, "request")

	// If a bad version received then send back a Status errcode to the initiator
	var st status.Status
	sinfo := status.Sinfo{Session: 0, Progress: 0, Inrespto: 0, Holes: nil}

	if sarflags.GetStr(r.Header, "version") != "v1" {
		if st.New("errcode=badrequest", &sinfo) != nil {
			sarwin.ErrPrintln(g, "red_black", "Cannot create badrequest status")
			return false
		}
		tx <- st.Val(from)
		return false
	}

	if sarflags.GetStr(r.Header, "udplite") != "no" {
		sarwin.ErrPrintln(g, "red_black", "UDP Lite not supported")
		if st.New("errcode=badrequest", &sinfo) != nil {
			sarwin.ErrPrintln(g, "red_black", "Cannot create badrequest status")
			return false
		}
		tx <- st.Val(from)
		return false
	}

	if sarflags.GetStr(r.Header, "stream") != "no" {
		sarwin.ErrPrintln(g, "red_black", "Streams not supported")
		if st.New("errcode=badrequest", &sinfo) != nil {
			sarwin.ErrPrintln(g, "red_black", "Cannot create badrequest status")
			return false
		}
		tx <- st.Val(from)
		return false
	}

	// ONLY create a transfer if both rxwilling and txwilling are set to "yes"
	// Thehese are silly flags as if I am the sender I a already know if I ca can receive files
	// and if I am the receiver it is up to me to just ignore requests that have these flags set to
	// capable, no or invalid
	// RECOMMEND THESE FLAGS BE REMOVED FROM THE DRAFT
	// If I receive a request and cannot do it then send back a STATUS with the corresponding error
	// code should be the action, which is what I am going to do in this implementation
	var rxwilling string
	if rxwilling = sarflags.GetStr(r.Header, "rxwilling"); rxwilling == "invalid" {
		// Create STATUS and set errcode to "cantreceive"
		sarwin.ErrPrintln(g, "red_black", "Invalid Rxwilling")
		if st.New("errcode=cantreceive", &sinfo) != nil {
			sarwin.ErrPrintln(g, "red_black", "Cannot create cantreceive status")
			return false
		}
		tx <- st.Val(from)
		return false
	}
	var txwilling string
	if txwilling = sarflags.GetStr(r.Header, "txwilling"); txwilling == "invalid" {
		sarwin.ErrPrintln(g, "red_black", "Invalid Txwilling")
		// Create STATUS and set errcode to "cantsend"
		if st.New("errcode=cantsend", &sinfo) != nil {
			sarwin.ErrPrintln(g, "red_black", "Cannot create cantsend status")
			return false
		}
		tx <- st.Val(from)
		return false
	}

	// See if the file exists on our local system
	exists := fileio.FileExists(r.Fname)
	switch ttype {
	// Open the local file to read from
	case "get", "getdir", "take":
		if rxwilling != "yes" {
			sarwin.ErrPrintln(g, "red_black", "Cannot get as rxwilling set to ", rxwilling)
			// Create STATUS and set errcode to "cantreceive"
			if st.New("errcode=cantreceive", &sinfo) != nil {
				sarwin.ErrPrintln(g, "red_black", "Cannot create rxwilling status")
				return false
			}
			tx <- st.Val(from)
			return false
		}
		if !exists {
			sarwin.ErrPrintln(g, "red_black", "Local File ", r.Fname, "does not exist for ", ttype)
			// Create STATUS and set errcode to "filenotfound"
			if st.New("errcode=filenotfound", &sinfo) != nil {
				sarwin.ErrPrintln(g, "red_black", "Cannot create filenotfound status")
				return false
			}
			tx <- st.Val(from)
			return false
		}
		// Open the file and transfer here

	// Delete the local file
	case "delete":
		if !exists {
			sarwin.ErrPrintln(g, "red_black", "Local File ", r.Fname, "does not exist for ", ttype)
			// Create STATUS and set errcode to "filenotfound"
			if st.New("errcode=filenotfound", &sinfo) != nil {
				sarwin.ErrPrintln(g, "red_black", "Cannot create filenotfound status")
				return false
			}
			tx <- st.Val(from)
			return false
		}
		// Delete the file
		if err := fileio.FileRm(r.Fname); err != nil {
			sarwin.ErrPrintln(g, "red_black", "Unable to remove  ", r.Fname)
			if st.New("errcode=didnotdelete", &sinfo) != nil {
				sarwin.ErrPrintln(g, "red_black", "Cannot create didnotdelete status")
				return false
			}
			tx <- st.Val(from)
			return false
		}
		// Create STATUS and set errcode to "success" signalling deletion of the file
		if st.New("errcode=success", &sinfo) != nil {
			sarwin.ErrPrintln(g, "red_black", "Cannot create success status")
			return false
		}
		tx <- st.Val(from)
		return true

	// Open the local file to write to
	case "put", "give":
		if txwilling != "yes" {
			sarwin.ErrPrintln(g, "red_black", "Cannot put as txwilling set to ", txwilling)
			// Create STATUS and set errcode to "cantsend"
			if st.New("errcode=cantsend", &sinfo) != nil {
				sarwin.ErrPrintln(g, "red_black", "Cannot create cantsend status")
				return false
			}
			tx <- st.Val(from)
			return false
		}
		if exists {
			sarwin.ErrPrintln(g, "red_black", "Local File ", r.Fname, "already exists for ", ttype)
			// Create STATUS and set errcode to "fileinuse"
			if st.New("errcode=fileinuse", &sinfo) != nil {
				sarwin.ErrPrintln(g, "red_black", "Cannot create fileinuse status")
				return false
			}
			tx <- st.Val(from)
			return false
		}
		// Create the file and transfer here
	default:
		sarwin.ErrPrintln(g, "red_black", "Invalid request")
		// Create STATUS and set errcode to "badrequest"
		if st.New("errcode=badrequest", &sinfo) != nil {
			sarwin.ErrPrintln(g, "red_black", "Cannot create badrequest status")
			return false
		}
		tx <- st.Val(from)
		return false
	}
	timetype := "epoch_2000_32"
	ts := new(timestamp.Timestamp)
	ts.Now(timetype)
	dent := new(dirent.DirEnt)

	// Do some error checking to see that the transfer is not already underway

	// Initialise the Transfer with the information we have in the request
	t := Transfer{Peer: *from, Session: r.Session, Ttype: ttype,
		Filename: r.Fname,
		Timetype: timetype, Tstamp: *ts,
		Stflags: "", Csumtype: "none", Checksum: nil,
		Havemeta: false, Dir: *dent, Fp: nil,
		Data: nil, Dcount: 0,
		Progress: 0, Inrespto: 0, CurFills: nil}

	// Add the transfer to our current slice
	Trmu.Lock()
	Transfers = append(Transfers, t)
	Trmu.Unlock()
	// Create STATUS and set errcode to "success"
	if st.New("errcode=success", &sinfo) != nil {
		sarwin.ErrPrintln(g, "red_black", "Cannot create success status")
		return false
	}
	tx <- st.Val(from)
	return true
}
