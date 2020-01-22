package transfer

import (
	"net"
	"os"
	"sync"

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
	fp        *os.File // Local FIle name to write to
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
	progresp chan [2]uint64, hole chan []status.Hole, errflag chan string) {

	// var filelen uint64
	// var fi os.FileInfo
	// var err error

	// Grab the file informaion
	//if fi, err = t.fp.Stat(); err != nil {
	//	errflag <- "filenotfound"
	//	return
	//}
	// filelen = uint64(fi.Size())
	prval := <-progresp
	progress := prval[0]
	inrespto := prval[1]
	holes := <-hole
	for {
		var maxholes = stpaylen(sflags) // Work out maximum # holes we can put in a single status frame
		var lasthole = len(holes)       // How many holes do we have

		var flags string
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
