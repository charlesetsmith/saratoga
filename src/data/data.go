package data

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/charlesetsmith/saratoga/src/sarflags"
	"github.com/charlesetsmith/saratoga/src/screen"
	"github.com/charlesetsmith/saratoga/src/timestamp"
	"github.com/jroimartin/gocui"
)

// Data -- Holds Data frame information
type Data struct {
	Header  uint32
	Session uint32
	Tstamp  timestamp.Timestamp
	Offset  uint64
	Payload []byte
}

// New - Construct a data frame - return byte slice of frame and Data structure
// Flags is of format "flagname1=flagval1,flagname2=flagval2...
// The timestamp type to use is also in the flags as "timestamp=flagval"
func (d *Data) New(flags string, session uint32, offset uint64, payload []byte) error {

	var err error

	if d.Header, err = sarflags.Set(d.Header, "version", "v1"); err != nil {
		return err
	}
	if d.Header, err = sarflags.Set(d.Header, "frametype", "data"); err != nil {
		return err
	}

	flags = strings.Replace(flags, " ", "", -1) // Get rid of extra spaces in flags

	// Grab the flags and set the frame header
	flag := strings.Split(flags, ",") // The name=val of the flag
	for fl := range flag {
		f := strings.Split(flag[fl], "=") // f[0]=name f[1]=val
		switch f[0] {
		case "descriptor", "reqtstamp", "reqstatus", "eod":
			if d.Header, err = sarflags.Set(d.Header, f[0], f[1]); err != nil {
				return err
			}
		case "transfer":
			if f[1] == "bundle" {
				return errors.New("Bundle Transfers not supported")
			}
		case "localinterp", "posix32", "posix64", "posix32_32", "posix64_32":
			if err = d.Tstamp.Now(f[0]); err != nil { // Set the timestamp to right now
				return err
			}
			d.Header, err = sarflags.Set(d.Header, "reqtstamp", "yes")
		default:
			e := "Invalid Flag " + f[0] + " for Data Frame"
			return errors.New(e)
		}
	}
	d.Session = session
	d.Offset = offset
	d.Payload = make([]byte, len(payload))
	copy(d.Payload, payload)
	return nil
}

// Make - Construct a data frame with a given header - return byte slice of frame and Data structure
func (d *Data) Make(header uint32, session uint32, offset uint64, payload []byte) error {

	var err error

	if header, err = sarflags.Set(header, "version", "v1"); err != nil {
		return err
	}
	if header, err = sarflags.Set(header, "frametype", "data"); err != nil {
		return err
	}

	d.Header = header
	d.Session = session
	d.Offset = offset
	d.Payload = make([]byte, len(payload))
	copy(d.Payload, payload)
	return nil
}

// Get -- Decode Data byte slice frame into Data struct
func (d *Data) Get(frame []byte) error {

	if len(frame) < 10 {
		return errors.New("data.Get - Frame too short")
	}
	d.Header = binary.BigEndian.Uint32(frame[:4])
	d.Session = binary.BigEndian.Uint32(frame[4:8])
	if sarflags.GetStr(d.Header, "reqtstamp") == "yes" {
		if err := d.Tstamp.Get(frame[8:24]); err != nil {
			return err
		}
		switch sarflags.GetStr(d.Header, "descriptor") {
		case "d16":
			d.Offset = uint64(binary.BigEndian.Uint16(frame[24:26]))
			d.Payload = make([]byte, len(frame[26:]))
			copy(d.Payload, frame[26:])
		case "d32":
			d.Offset = uint64(binary.BigEndian.Uint32(frame[24:28]))
			d.Payload = make([]byte, len(frame[28:]))
			copy(d.Payload, frame[26:])
		case "d64":
			d.Offset = binary.BigEndian.Uint64(frame[24:32])
			d.Payload = make([]byte, len(frame[32:]))
			copy(d.Payload, frame[32:])
		default:
			return errors.New("Invalid Data Frame")
		}
		return nil
	}
	switch sarflags.GetStr(d.Header, "descriptor") {
	case "d16":
		d.Offset = uint64(binary.BigEndian.Uint16(frame[8:10]))
		d.Payload = make([]byte, len(frame[10:]))
		copy(d.Payload, frame[10:])
	case "d32":
		d.Offset = uint64(binary.BigEndian.Uint32(frame[8:12]))
		d.Payload = make([]byte, len(frame[12:]))
		copy(d.Payload, frame[12:])
	case "d64":
		d.Offset = binary.BigEndian.Uint64(frame[8:16])
		d.Payload = make([]byte, len(frame[16:]))
		copy(d.Payload, frame[16:])
	default:
		return errors.New("Invalid Data Frame")
	}
	return nil
}

// Put -- Encode the Saratoga Data Frame buffer
func (d *Data) Put() ([]byte, error) {

	havetstamp := false

	framelen := 4 + 4 // Header + Session

	if sarflags.GetStr(d.Header, "reqtstamp") == "yes" {
		framelen += 16 // Timestamp
		havetstamp = true
	}

	switch sarflags.GetStr(d.Header, "descriptor") { // Offset
	case "d16":
		framelen += 2
	case "d32":
		framelen += 4
	case "d64":
		framelen += 8
	default:
		return nil, errors.New("Invalid descriptor in Data frame")
	}
	framelen += len(d.Payload)

	frame := make([]byte, framelen)

	binary.BigEndian.PutUint32(frame[:4], d.Header)
	binary.BigEndian.PutUint32(frame[4:8], d.Session)
	pos := 8
	if havetstamp {
		copy(frame[pos:24], d.Tstamp.Put())
		pos = 24
	}
	switch sarflags.GetStr(d.Header, "descriptor") {
	case "d16":
		binary.BigEndian.PutUint16(frame[pos:pos+2], uint16(d.Offset))
		pos += 2
	case "d32":
		binary.BigEndian.PutUint32(frame[pos:pos+4], uint32(d.Offset))
		pos += 4
	case "d64":
		binary.BigEndian.PutUint64(frame[pos:pos+8], uint64(d.Offset))
		pos += 8
	default:
		return nil, errors.New("Malformed Data frame")
	}
	copy(frame[pos:], d.Payload)
	return frame, nil
}

// Print - Print out details of Beacon struct
func (d Data) Print() string {
	sflag := fmt.Sprintf("Data: 0x%x\n", d.Header)
	dflags := sarflags.Values("data")
	for _, f := range dflags {
		n := sarflags.GetStr(d.Header, f)
		sflag += fmt.Sprintf("  %s:%s\n", f, n)
	}
	if sarflags.GetStr(d.Header, "reqtstamp") == "yes" {
		sflag += fmt.Sprintf("  timestamp:%s\n", d.Tstamp.Print())
	}
	sflag += fmt.Sprintf("  session:%d", d.Session)
	sflag += fmt.Sprintf("  offset:%d", d.Offset)
	sflag += fmt.Sprintf("  Payload :<%d>\n", len(d.Payload))
	return sflag
}

// Handler - We have some incoming data for a session. Add the data to the session
// For this implementation of saratoga if no session exists then just dump the data
func (d *Data) Handler(g *gocui.Gui, from *net.UDPAddr, session uint32) string {
	screen.Fprintln(g, "msg", "yellow_black", d.Print())
	// Return an errcode string
	return "success"
}
