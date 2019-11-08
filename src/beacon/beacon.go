package beacon

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"sarflags"
)

// Beacon -- Holds Beacon frame information
type Beacon struct {
	header    uint32
	freespace uint64
	eid       string
}

// New - Construct a beacon - return byte slice of frame
func (b *Beacon) New(flags string, eid string, freespace uint64) error {
	var err error

	// Always present in a Beacon
	if b.header, err = sarflags.Set(b.header, "version", "v1"); err != nil {
		return err
	}
	// And yes we are a Beacon Frame
	if b.header, err = sarflags.Set(b.header, "frametype", "beacon"); err != nil {
		return err
	}

	flags = strings.Replace(flags, " ", "", -1) // Get rid of extra spaces in flags

	// Grab the flags and set the frame header
	flag := strings.Split(flags, ",") // The name=val of the flag
	for fl := range flag {
		f := strings.Split(flag[fl], "=") // f[0]=name f[1]=val
		switch f[0] {
		case "descriptor", "stream", "txwilling", "rxwilling", "udplite", "freespace":
			if b.header, err = sarflags.Set(b.header, f[0], f[1]); err != nil {
				return err
			}
		case "freespaced":
			// Make sure freespace has been turned on
			if b.header, err = sarflags.Set(b.header, "freespace", "yes"); err != nil {
				return err
			}
			switch f[1] {
			case "d16", "d32", "d64":
				if b.header, err = sarflags.Set(b.header, f[0], f[1]); err != nil {
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
	b.freespace = freespace
	b.eid = eid
	return nil
}

// Put -- Encode the Saratoga Data Frame buffer
func (b Beacon) Put() ([]byte, error) {

	var frame []byte

	framelen := 4 + len(b.eid)
	if sarflags.GetStr(b.header, "freespace") == "yes" {
		switch sarflags.GetStr(b.header, "freespaced") {
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

	if framelen > sarflags.MaxFrameSize {
		return nil, errors.New("Data - Maximum Frame Size Exceeded")
	}
	frame = make([]byte, framelen)

	pos := 4
	binary.BigEndian.PutUint32(frame[:pos], b.header)
	if sarflags.GetStr(b.header, "freespace") == "yes" {
		switch sarflags.GetStr(b.header, "freespaced") {
		case "d16":
			binary.BigEndian.PutUint16(frame[pos:6], uint16(b.freespace))
			pos += 2
		case "d32":
			binary.BigEndian.PutUint32(frame[pos:8], uint32(b.freespace))
			pos += 4
		case "d64":
			binary.BigEndian.PutUint64(frame[pos:12], uint64(b.freespace))
			pos += 8
		default:
			return nil, errors.New("Invalid Beacon Frame")
		}
	}
	copy(frame[pos:], []byte(b.eid))
	return frame, nil
}

// Get -- Decode Beacon byte slice frame into Beacon struct
func (b *Beacon) Get(frame []byte) error {

	b.header = binary.BigEndian.Uint32(frame[:4])
	if sarflags.GetStr(b.header, "freespace") == "yes" {
		switch sarflags.GetStr(b.header, "freespaced") {
		case "d16":
			b.freespace = uint64(binary.BigEndian.Uint16(frame[4:6]))
			b.eid = string(frame[6:])
		case "d32":
			b.freespace = uint64(binary.BigEndian.Uint32(frame[4:8]))
			b.eid = string(frame[8:])
		case "d64":
			b.freespace = binary.BigEndian.Uint64(frame[4:12])
			b.eid = string(frame[12:])
		default:
			b.freespace = 0
			b.eid = string(frame[4:])
			return errors.New("Invalid Beacon Frame")
		}
		return nil
	}
	// No freespace to be reported
	b.freespace = 0
	b.eid = string(frame[4:])
	return nil
}

// Print - Print out details of Beacon struct
func (b Beacon) Print() string {
	sflag := fmt.Sprintf("Beacon: 0x%x\n", b.header)
	bflags := sarflags.Values("beacon")
	for _, f := range bflags {
		n := sarflags.GetStr(b.header, f)
		sflag += fmt.Sprintf("  %s:%s\n", f, n)
	}
	if sarflags.GetStr(b.header, "freespace") == "yes" {
		sflag += fmt.Sprintf("  free:%dkB\n", b.freespace)
	}
	sflag += fmt.Sprintf("  EID:%s\n", b.eid)
	return sflag
}
