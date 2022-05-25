// Common Transfer Routines

package transfer

import (
	"os"
	"strings"

	"github.com/charlesetsmith/saratoga/sarflags"
)

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
