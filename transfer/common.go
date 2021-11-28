// Common Transfer Routines

package transfer

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/charlesetsmith/saratoga/sarflags"
	"github.com/charlesetsmith/saratoga/sarwin"
	"github.com/jroimartin/gocui"
)

// Ttypes - Transfer types
var Ttypes = []string{"get", "getrm", "getdir", "put", "putblind", "putrm", "rm", "rmdir"}

// Direction of Transfer
const (
	Send    bool = true
	Receive bool = false
)

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
func maxpaylen(flags string) int {

	plen := sarflags.Mtu() - 60 - 8 // 60 for IP header, 8 for UDP header
	plen -= 8                       // Saratoga Header + Offset

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
			case "d128":
				plen -= 16
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

	plen = sarflags.Mtu() - 60 - 8 // 60 for IP header, 8 for UDP header
	plen -= 8                      // Saratoga Header + Session

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
			case "d128":
				hsize = 32
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
