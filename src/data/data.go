package data

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"sarflags"
	"timestamp"
)

// Data -- Holds Data frame information
type Data struct {
	header  uint32
	session uint32
	tstamp  timestamp.Timestamp
	offset  uint64
	payload []byte
}

// New - Construct a data frame - return byte slice of frame and Data structure
// Flags is of format "flagname1=flagval1,flagname2=flagval2...
// The timestamp type to use is also in the flags as "timestamp=flagval"
func (d *Data) New(flags string, session uint32, offset uint64, payload []byte) error {

	var err error

	if d.header, err = sarflags.Set(d.header, "version", "v1"); err != nil {
		return err
	}
	if d.header, err = sarflags.Set(d.header, "frametype", "data"); err != nil {
		return err
	}

	flags = strings.Replace(flags, " ", "", -1) // Get rid of extra spaces in flags

	// Grab the flags and set the frame header
	flag := strings.Split(flags, ",") // The name=val of the flag
	for fl := range flag {
		f := strings.Split(flag[fl], "=") // f[0]=name f[1]=val
		switch f[0] {
		case "descriptor", "reqtstamp", "reqstatus", "eod":
			if d.header, err = sarflags.Set(d.header, f[0], f[1]); err != nil {
				return err
			}
		case "transfer":
			if f[1] == "bundle" {
				return errors.New("Bundle Transfers not supported")
			}
		case "localinterp", "posix32", "posix64", "posix32_32", "posix64_32":
			if err = d.tstamp.Now(f[0]); err != nil { // Set the timestamp to right now
				return err
			}
			d.header, err = sarflags.Set(d.header, "reqtstamp", "yes")
		default:
			e := "Invalid Flag " + f[0] + " for Data Frame"
			return errors.New(e)
		}
	}
	d.session = session
	d.offset = offset
	d.payload = make([]byte, len(payload))
	copy(d.payload, payload)
	return nil
}

// Get -- Decode Data byte slice frame into Data struct
func (d *Data) Get(frame []byte) error {

	if len(frame) < 10 {
		return errors.New("data.Get - Frame too short")
	}
	d.header = binary.BigEndian.Uint32(frame[:4])
	d.session = binary.BigEndian.Uint32(frame[4:8])
	if sarflags.GetStr(d.header, "reqtstamp") == "yes" {
		if err := d.tstamp.Get(frame[8:24]); err != nil {
			return err
		}
		switch sarflags.GetStr(d.header, "descriptor") {
		case "d16":
			d.offset = uint64(binary.BigEndian.Uint16(frame[24:26]))
			d.payload = make([]byte, len(frame[26:]))
			copy(d.payload, frame[26:])
		case "d32":
			d.offset = uint64(binary.BigEndian.Uint32(frame[24:28]))
			d.payload = make([]byte, len(frame[28:]))
			copy(d.payload, frame[26:])
		case "d64":
			d.offset = binary.BigEndian.Uint64(frame[24:32])
			d.payload = make([]byte, len(frame[32:]))
			copy(d.payload, frame[32:])
		default:
			return errors.New("Invalid Data Frame")
		}
		return nil
	}
	switch sarflags.GetStr(d.header, "descriptor") {
	case "d16":
		d.offset = uint64(binary.BigEndian.Uint16(frame[8:10]))
		d.payload = make([]byte, len(frame[10:]))
		copy(d.payload, frame[10:])
	case "d32":
		d.offset = uint64(binary.BigEndian.Uint32(frame[8:12]))
		d.payload = make([]byte, len(frame[12:]))
		copy(d.payload, frame[12:])
	case "d64":
		d.offset = binary.BigEndian.Uint64(frame[8:16])
		d.payload = make([]byte, len(frame[16:]))
		copy(d.payload, frame[16:])
	default:
		return errors.New("Invalid Data Frame")
	}
	return nil
}

// Put -- Encode the Saratoga Data Frame buffer
func (d *Data) Put() ([]byte, error) {

	havetstamp := false

	framelen := 4 + 4 // Header + Session

	if sarflags.GetStr(d.header, "reqtstamp") == "yes" {
		framelen += 16 // Timestamp
		havetstamp = true
	}

	switch sarflags.GetStr(d.header, "descriptor") { // Offset
	case "d16":
		framelen += 2
	case "d32":
		framelen += 4
	case "d64":
		framelen += 8
	default:
		return nil, errors.New("Invalid descriptor in Data frame")
	}
	framelen += len(d.payload)

	if framelen > sarflags.MaxFrameSize {
		return nil, errors.New("Data - Maximum Frame Size Exceeded")
	}

	frame := make([]byte, framelen)

	binary.BigEndian.PutUint32(frame[:4], d.header)
	binary.BigEndian.PutUint32(frame[4:8], d.session)
	pos := 8
	if havetstamp {
		copy(frame[pos:24], d.tstamp.Put())
		pos = 24
	}
	switch sarflags.GetStr(d.header, "descriptor") {
	case "d16":
		binary.BigEndian.PutUint16(frame[pos:pos+2], uint16(d.offset))
		pos += 2
	case "d32":
		binary.BigEndian.PutUint32(frame[pos:pos+4], uint32(d.offset))
		pos += 4
	case "d64":
		binary.BigEndian.PutUint64(frame[pos:pos+8], uint64(d.offset))
		pos += 8
	default:
		return nil, errors.New("Malformed Data frame")
	}
	copy(frame[pos:], d.payload)
	return frame, nil
}

// Print - Print out details of Beacon struct
func (d Data) Print() string {
	sflag := fmt.Sprintf("Data: 0x%x\n", d.header)
	dflags := sarflags.Values("data")
	for f := range dflags {
		n := sarflags.GetStr(d.header, dflags[f])
		sflag += fmt.Sprintf("  %s:%s\n", dflags[f], n)
	}
	if sarflags.GetStr(d.header, "reqtstamp") == "yes" {
		sflag += fmt.Sprintf("  timestamp:%s\n", d.tstamp.Print())
	}
	sflag += fmt.Sprintf("  session:%d", d.session)
	sflag += fmt.Sprintf("  offset:%d", d.offset)
	sflag += fmt.Sprintf("  Payload :<%d>\n", len(d.payload))
	return sflag
}
