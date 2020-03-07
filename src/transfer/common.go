package transfer

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/charlesetsmith/saratoga/src/sarflags"
	"github.com/charlesetsmith/saratoga/src/screen"
	"github.com/jroimartin/gocui"
)

// Ttypes - Transfer types
var Ttypes = []string{"get", "getrm", "getdir", "put", "putblind", "putrm", "rm", "rmdir"}

// current protected session number
var smu sync.Mutex
var sessionid uint32

// Info - List client & server transfers in progress to msg window matching ttype or all if ""
func Info(g *gocui.Gui, ttype string) {
	var tinfo []CTransfer

	for i := range CTransfers {
		if ttype == "" {
			tinfo = append(tinfo, CTransfers[i])
		} else if CTransfers[i].ttype == ttype {
			tinfo = append(tinfo, CTransfers[i])
		}
	}
	if len(tinfo) > 0 {
		var maxaddrlen, maxfname int // Work out the width for the table
		for key := range tinfo {
			if len(tinfo[key].peer.String()) > maxaddrlen {
				maxaddrlen = len(tinfo[key].peer.String())
			}
			if len(tinfo[key].filename) > maxfname {
				maxfname = len(tinfo[key].peer.String())
			}
		}
		// Table format
		sfmt := fmt.Sprintf("|%%6s|%%8s|%%%ds|%%%ds|\n", maxaddrlen, maxfname)
		sborder := fmt.Sprintf(sfmt, strings.Repeat("-", 6), strings.Repeat("-", 8),
			strings.Repeat("-", maxaddrlen), strings.Repeat("-", maxfname))

		var sslice sort.StringSlice
		for key := range tinfo {
			sslice = append(sslice, fmt.Sprintf("%s", tinfo[key].FmtPrint(sfmt)))
		}
		sort.Sort(sslice)

		sbuf := sborder
		sbuf += fmt.Sprintf(sfmt, "Direct", "Tran Typ", "IP", "Fname")
		sbuf += sborder
		for key := 0; key < len(sslice); key++ {
			sbuf += fmt.Sprintf("%s", sslice[key])
		}
		sbuf += sborder
		screen.Fprintln(g, "msg", "magenta_black", sbuf)
	} else {
		msg := fmt.Sprintf("No %s transfers currently in progress", ttype)
		screen.Fprintln(g, "msg", "green_black", msg)
	}
}

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

// FileDescriptor - Get the appropriate descriptor flag size based on file length
func filedescriptor(fname string) string {
	if fi, err := os.Stat(fname); err == nil {
		size := uint64(fi.Size())
		if size <= sarflags.MaxUint16 {
			return "descriptor=d16"
		}
		if size <= sarflags.MaxUint32 {
			return "descriptor=d32"
		}
		if size <= sarflags.MaxUint64 {
			return "descriptor=d64"
		}
	}
	// Just send back the maximum supported descriptor
	if sarflags.MaxUint <= sarflags.MaxUint16 {
		return "descriptor=d16"
	}
	if sarflags.MaxUint <= sarflags.MaxUint32 {
		return "descriptor=d32"
	}
	return "descriptor=d64"
}

// Replace an existing flag or add it
func replaceflag(curflags string, newflag string) string {
	var fs string
	var replaced bool

	for _, curflag := range strings.Split(curflags, ",") {
		if strings.Split(curflag, "=")[0] == strings.Split(newflag, "=")[0] {
			replaced = true
			fs += newflag + ","
		} else {
			fs += curflag + ","
		}
	}
	if !replaced {
		fs += newflag
	}
	return strings.TrimRight(fs, ",")
}

// Look for and return value of a particular flag in flags
// e.g flags:descriptor=d32,timestamp=off flag:timestamp return:off
func flagvalue(flags, flag string) string {
	flags = strings.Replace(flags, " ", "", -1) // Get rid of extra spaces in flags
	// Grab the flags and set the frame header
	flagslice := strings.Split(flags, ",") // The name=val of the flag
	for fl := range flagslice {
		f := strings.Split(flagslice[fl], "=") // f[0]=name f[1]=val
		if f[0] == flag {
			return f[1]
		}
	}
	return ""
}

// Work out the maximum payload in data.Data frame given flags
func dpaylen(flags string) int {

	plen := sarflags.MTU - 60 - 8 // 60 for IP header, 8 for UDP header
	plen -= 8                     // Saratoga Header + Offset

	flags = strings.Replace(flags, " ", "", -1) // Get rid of extra spaces in flags
	// Grab the flags and set the frame header
	flag := strings.Split(flags, ",") // The name=val of the flag
	for fl := range flag {
		f := strings.Split(flag[fl], "=") // f[0]=name f[1]=val
		switch f[0] {
		case "descriptor":
			switch f[1] {
			case "d16":
				plen -= 2
			case "d32":
				plen -= 4
			case "d64":
				plen -= 8
			default:
				return 0
			}
		case "reqtstamp":
			if f[1] == "yes" {
				plen -= 16
			}
		default:
		}
	}
	return plen
}

// Work out the maximum payload in status.Status frame given flags
func stpaylen(flags string) int {
	var hsize int
	var plen int

	plen = sarflags.MTU - 60 - 8 // 60 for IP header, 8 for UDP header
	plen -= 8                    // Saratoga Header + Session

	flags = strings.Replace(flags, " ", "", -1) // Get rid of extra spaces in flags
	// Grab the flags and set the frame header
	flag := strings.Split(flags, ",") // The name=val of the flag
	for fl := range flag {
		f := strings.Split(flag[fl], "=") // f[0]=name f[1]=val
		switch f[0] {
		case "descriptor":
			switch f[1] {
			case "d16":
				hsize = 4
			case "d32":
				hsize = 8
			case "d64":
				hsize = 16
			default:
				return 0
			}
		case "reqtstamp":
			if f[1] == "yes" {
				plen -= 16
			}
		default:
		}
	}
	plen -= hsize       // For Progress & Inrespto
	return plen / hsize // Max holes we can now have in the frame
}

// ****************************************************************************************
// THE HOLE HANDLER
// ****************************************************************************************

// Hole -- Beggining and End of a hole
// e.g. [0,1] Hole starts at index 0 up to 1 so is 1 byte long
// e.g. [5,7] Hole starts at index 5 up to 7 so is 2 bytes long
// Start is "from" and End is "up to but not including"
type Hole struct {
	Start int
	End   int
}

// Holes - Slices of Hole or Fill
type Holes []Hole

// Removes an entry from Holes slice
// Please note it is not exported for good reason - you NEVER call this yourself
// it is only called by optimise
func (fills Holes) remove(i int) Holes {
	copy(fills[i:], fills[i+1:])
	return fills[:len(fills)-1]
}

// Optimises the Holes slice
// Please note it is not exported for good reason - you NEVER call this yourself
// it is only called by Add
func (fills Holes) optimise() Holes {
	if len(fills) <= 1 { // Pretty hard to optimise a single fill slice
		return fills
	}
	for i := 0; i < len(fills); i++ {
		if i == len(fills)-1 { // We got to the end pop the optimise
			return fills
		}
		if fills[i+1].Start <= fills[i].End {
			if fills[i+1].End <= fills[i].End {
				// The next fill is inside current fill so just remove it
				fills = fills.remove(i + 1)
			} else if fills[i+1].End >= fills[i].End {
				// The next fill spans into the current so extend the current and remove the next
				fills[i].End = fills[i+1].End
				fills = fills.remove(i + 1)
			}
			// fmt.Println("E", Fills)
			// And here is the secret sauce to optimise - the beuty of recursion
			fills = fills.optimise()
		}
	}
	return fills
}

// Add - Add an entry to Holes (actually fills) slice
// We can only ever Add to the fill there is no Delete as it optimises the min # entries in the slice
// So in the End we have a slice that has a single entry that contains the complete block [0,n]
// that means we have everything and there are no holes
func (fills Holes) Add(start int, end int) Holes {
	if end <= start { // Error check it jsut in case
		return fills
	}
	fill := Hole{start, end}

	// fmt.Println("Appending", fill)
	fills = append(fills, fill)
	if len(fills) > 1 {
		sort.Slice(fills, func(i, j int) bool {
			return fills[i].Start < fills[j].Start
		})
		fills = fills.optimise()
	}
	return fills
	// fmt.Println("Fills=", Fills)
}

// Getholes - return the slice of actual holes from Fills
// THis is used to construct the Holes in Status Frames
func (fills Holes) Getholes() Holes {
	var holes []Hole

	lenfills := len(fills)
	if lenfills == 0 {
		return holes
	}
	for f := range fills {
		if f == 0 && fills[f].Start != 0 {
			start := 0
			end := fills[f].Start
			holes = append(holes, Hole{start, end})
		}
		if f < lenfills-1 {
			start := fills[f].End
			end := fills[f+1].Start
			holes = append(holes, Hole{start, end})
		}
	}
	return holes
}
