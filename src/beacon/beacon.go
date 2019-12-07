package beacon

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"

	"sarflags"

	"github.com/charlesetsmith/saratoga/src/sarnet"
	"github.com/charlesetsmith/saratoga/src/screen"
)

// Beacon -- Holds Beacon frame information
type Beacon struct {
	Header    uint32
	Freespace uint64 // Set in b.New
	Eid       string // This changes depending upon Host IP Address. Set in "b.Send"
}

// New - Construct a beacon - Fill in the Beacon struct - Not EID we do that in Send
func (b *Beacon) New(flags string) error {
	var err error

	// Always present in a Beacon
	if b.Header, err = sarflags.Set(b.Header, "version", "v1"); err != nil {
		return err
	}
	// And yes we are a Beacon Frame
	if b.Header, err = sarflags.Set(b.Header, "frametype", "beacon"); err != nil {
		return err
	}

	flags = strings.Replace(flags, " ", "", -1) // Get rid of extra spaces in flags

	// Grab the flags and set the frame header
	flag := strings.Split(flags, ",") // The name=val of the flag
	for fl := range flag {
		f := strings.Split(flag[fl], "=") // f[0]=name f[1]=val
		switch f[0] {
		case "descriptor", "stream", "txwilling", "rxwilling", "udplite", "freespace":
			if b.Header, err = sarflags.Set(b.Header, f[0], f[1]); err != nil {
				return err
			}
		case "freespaced":
			// Make sure freespace has been turned on
			if b.Header, err = sarflags.Set(b.Header, "freespace", "yes"); err != nil {
				return err
			}
			switch f[1] {
			case "d16", "d32", "d64":
				if b.Header, err = sarflags.Set(b.Header, f[0], f[1]); err != nil {
					return err
				}

			default:
				es := "Beacon.New: Invalid freespaced " + f[1]
				return errors.New(es)
			}
		default:
			e := "Beacon.New: Invalid Flag " + f[0] + "=" + f[1] + " for Data Frame"
			return errors.New(e)
		}
	}
	b.Eid = ""

	if sarflags.GetStr(b.Header, "freespace") == "yes" {
		// Ignore if freespaced is set, just set it to the correct size
		var fs syscall.Statfs_t
		var sardir string

		// Where we put/get files from/to
		if sardir, err = os.Getwd(); err != nil {
			b.Header, _ = sarflags.Set(b.Header, "freespace", "no")
			b.Freespace = 0
			return err
		}

		if err := syscall.Statfs(sardir, &fs); err != nil {
			b.Header, _ = sarflags.Set(b.Header, "freespace", "no")
			b.Freespace = 0
			return nil
		}
		// Freespace is number of Kilobytes (1024 bytes) left on disk
		b.Freespace = (uint64(fs.Bsize) * fs.Bavail) / 1024
		if b.Freespace*1024 < sarflags.MaxUint16 {
			b.Header, _ = sarflags.Set(b.Header, "freespaced", "d16")
			return nil
		}
		if b.Freespace*1024 < sarflags.MaxUint32 {
			b.Header, _ = sarflags.Set(b.Header, "freespaced", "d32")
			return nil
		}
		if b.Freespace*1024 < sarflags.MaxUint64 {
			b.Header, _ = sarflags.Set(b.Header, "freespaced", "d64")
			return nil
		}
		e := "beacon.New: More than uint64 can hold freespace left - We dont do d128 yet!"
		return errors.New(e)
	}

	return nil
}

// Make - Construct a beacon with a given header - return byte slice of frame
func (b *Beacon) Make(header uint32, eid string, freespace uint64) error {
	var err error

	// Always present in a Beacon
	if header, err = sarflags.Set(header, "version", "v1"); err != nil {
		return err
	}
	// And yes we are a Beacon Frame
	if header, err = sarflags.Set(header, "frametype", "beacon"); err != nil {
		return err
	}

	b.Header = header
	b.Freespace = freespace
	b.Eid = eid
	return nil
}

// Put -- Encode the Saratoga Beacon into a Frame buffer
func (b Beacon) Put() ([]byte, error) {

	var frame []byte

	framelen := 4 + len(b.Eid)
	if sarflags.GetStr(b.Header, "freespace") == "yes" {
		switch sarflags.GetStr(b.Header, "freespaced") {
		case "d16":
			framelen += 2
		case "d32":
			framelen += 4
		case "d64":
			framelen += 8
		default:
			return nil, errors.New("Invalid Beacon Frame")
		}
	}

	if framelen > sarnet.MaxFrameSize {
		return nil, errors.New("Data - Maximum Frame Size Exceeded")
	}
	frame = make([]byte, framelen)

	pos := 4
	binary.BigEndian.PutUint32(frame[:pos], b.Header)
	if sarflags.GetStr(b.Header, "freespace") == "yes" {
		switch sarflags.GetStr(b.Header, "freespaced") {
		case "d16":
			binary.BigEndian.PutUint16(frame[pos:6], uint16(b.Freespace))
			pos += 2
		case "d32":
			binary.BigEndian.PutUint32(frame[pos:8], uint32(b.Freespace))
			pos += 4
		case "d64":
			binary.BigEndian.PutUint64(frame[pos:12], uint64(b.Freespace))
			pos += 8
		default:
			return nil, errors.New("Invalid Beacon Frame")
		}
	}
	copy(frame[pos:], []byte(b.Eid))
	return frame, nil
}

// Get -- Decode Beacon byte slice frame into Beacon struct
func (b *Beacon) Get(frame []byte) error {

	b.Header = binary.BigEndian.Uint32(frame[:4])
	if sarflags.GetStr(b.Header, "freespace") == "yes" {
		switch sarflags.GetStr(b.Header, "freespaced") {
		case "d16":
			b.Freespace = uint64(binary.BigEndian.Uint16(frame[4:6]))
			b.Eid = string(frame[6:])
		case "d32":
			b.Freespace = uint64(binary.BigEndian.Uint32(frame[4:8]))
			b.Eid = string(frame[8:])
		case "d64":
			b.Freespace = binary.BigEndian.Uint64(frame[4:12])
			b.Eid = string(frame[12:])
		default:
			b.Freespace = 0
			b.Eid = string(frame[4:])
			return errors.New("Invalid Beacon Frame")
		}
		return nil
	}
	// No freespace to be reported
	b.Freespace = 0
	b.Eid = string(frame[4:])
	return nil
}

// Print - Print out details of Beacon struct
func (b Beacon) Print() string {
	sflag := fmt.Sprintf("Beacon: 0x%x\n", b.Header)
	bflags := sarflags.Values("beacon")
	for _, f := range bflags {
		n := sarflags.GetStr(b.Header, f)
		sflag += fmt.Sprintf("  %s:%s\n", f, n)
	}
	if sarflags.GetStr(b.Header, "freespace") == "yes" {
		sflag += fmt.Sprintf("  free:%dkB\n", b.Freespace)
	}
	sflag += fmt.Sprintf("  EID:%s\n", b.Eid)
	return sflag
}

// SendV4Mcast - Send a IPv4 Multicast beacon
func SendV4Mcast(b *Beacon, errflag chan uint32) {

	addr := sarnet.IPv4Multicast

	screen.Fprintln(screen.Msg, "blue_black", "Sending Multicast beacons to ",
		sarnet.IPv4Multicast, sarnet.Port())
	var frame []byte
	var err error

	if frame, err = b.Put(); err != nil {
		ret, _ := sarflags.Set(0, "errcode", "badpacket")
		errflag <- ret
	}
	screen.Fprintln(screen.Msg, "yellow_black", "Sending Beacon to ", addr, ":", b.Print())

	udpad := addr + ":" + strconv.Itoa(sarnet.Port())
	conn, err := net.Dial("udp", udpad)
	defer conn.Close()
	if err != nil {
		log.Fatalf("Cannot open UDP Socket to %s %v", udpad, err)
		return
	}

	_, err = conn.Write(frame)
	errflag <- uint32(sarflags.Value("errcode", "success"))
}

// SendV6Mcast - Send a IPv6 Multicast beacon
func SendV6Mcast(b *Beacon, errflag chan uint32) {

	addr := sarnet.IPv6Multicast

	screen.Fprintln(screen.Msg, "blue_black", "Sending Multicast beacons to ",
		sarnet.IPv6Multicast, sarnet.Port())
	var frame []byte
	var err error

	if frame, err = b.Put(); err != nil {
		ret, _ := sarflags.Set(0, "errcode", "badpacket")
		errflag <- ret
	}
	screen.Fprintln(screen.Msg, "yellow_black", "Sending Beacon to ", addr, ":", b.Print())

	udpad := addr + ":" + strconv.Itoa(sarnet.Port())
	conn, err := net.Dial("udp", udpad)
	defer conn.Close()
	if err != nil {
		log.Fatalf("Cannot open UDP Socket to %s %v", udpad, err)
		return
	}

	_, err = conn.Write(frame)
	errflag <- uint32(sarflags.Value("errcode", "success"))
}

// Send - Send a IPv4 or IPv6 beacon to a server
func Send(b *Beacon, addr string, errflag chan uint32) {

	var eid string

	// If our destination is IPv4 host then set this host's IPv4 Address in the EID
	// If our destination is IPv6 host then set this host's IPv6 address in the EID
	if net.ParseIP(addr) != nil {
		pstr := strconv.Itoa(sarnet.Port())
		if strings.Contains(addr, ".") { // IPv4
			eid = fmt.Sprintf("%s:%s %d", sarnet.OutboundIP("IPv4").String(), pstr, os.Getpid())
		} else if strings.Contains(addr, ":") { // IPv6
			eid = fmt.Sprintf("[%s]:%s %d", sarnet.OutboundIP("IPv6").String(), pstr, os.Getpid())
		} else {
			ret, _ := sarflags.Set(0, "errcode", "badpacket")
			errflag <- ret
			return
		}
	}

	// Copy this back into the beacons Eid
	// It is the outbound "IP:Socket PID"
	b.Eid = eid

	// Assemble the beacon frame from the beacon struct
	var frame []byte
	var err error

	if frame, err = b.Put(); err != nil {
		ret, _ := sarflags.Set(0, "errcode", "badpacket")
		errflag <- ret
	}
	screen.Fprintln(screen.Msg, "yellow_black", "Sending Beacon to ", addr, ":", b.Print())

	// Set up the connection
	udpad := addr + ":" + strconv.Itoa(sarnet.Port())
	conn, err := net.Dial("udp", udpad)
	defer conn.Close()
	if err != nil {
		log.Fatalf("Cannot open UDP Socket to %s %v", udpad, err)
		return
	}

	// Send it off
	_, err = conn.Write(frame)
	errflag <- uint32(sarflags.Value("errcode", "success"))
}
