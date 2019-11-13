package beacon

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"sarflags"

	"github.com/charlesetsmith/saratoga/src/sarnet"
)

// Beacon -- Holds Beacon frame information
type Beacon struct {
	Header    uint32
	Freespace uint64
	Eid       string
}

// New - Construct a beacon - return byte slice of frame
func (b *Beacon) New(flags string, eid string, freespace uint64) error {
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
	b.Freespace = freespace
	b.Eid = eid
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

// Put -- Encode the Saratoga Data Frame buffer
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
