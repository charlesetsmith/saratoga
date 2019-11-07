package beacon

import (
	"encoding/binary"
	"errors"
	"fmt"

	"sarflags"
)

// Beacon -- Holds Beacon frame information
type Beacon struct {
	header    uint32
	freespace uint64
	eid       string
}

// Make - Construct a beacon - return byte slice of frame
func (b *Beacon) Make(header uint32, eid string, freespace uint64) ([]byte, error) {
	var err error

	// Always present in a Beacon
	if header, err = sarflags.Set(header, "version", "v1"); err != nil {
		return nil, err
	}
	// And yes we are a Beacon Frame
	if header, err = sarflags.Set(header, "frametype", "beacon"); err != nil {
		return nil, err
	}

	var frame []byte

	pos := 0
	b.header = header
	b.freespace = freespace
	b.eid = eid
	if sarflags.GetStr(header, "freespace") == "yes" {
		switch sarflags.GetStr(header, "freespaced") {
		case "d16":
			frame = make([]byte, 4+2+len(eid))
			binary.BigEndian.PutUint32(frame[:4], header)
			binary.BigEndian.PutUint16(frame[4:6], uint16(freespace))
			pos = 6
		case "d32":
			frame = make([]byte, 4+4+len(eid))
			binary.BigEndian.PutUint32(frame[:4], header)
			binary.BigEndian.PutUint32(frame[4:8], uint32(freespace))
			pos = 8
		case "d64":
			frame = make([]byte, 4+4+len(eid))
			binary.BigEndian.PutUint32(frame[:4], header)
			binary.BigEndian.PutUint64(frame[4:12], uint64(freespace))
			pos = 12
		default:
			return frame, errors.New("Invalid Beacon Frame")
		}
		copy(frame[pos:], []byte(eid))
		return frame, nil
	}
	frame = make([]byte, 4+len(eid))
	binary.BigEndian.PutUint32(frame[:4], header)
	copy(frame[4:], []byte(eid))
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
	b.eid = string(frame[:4])
	return nil
}

// Print - Print out details of Beacon struct
func (b Beacon) Print() string {
	sflag := fmt.Sprintf("Beacon: 0x%x\n", b.header)
	bflags := sarflags.Values("beacon")
	for f := range bflags {
		n := sarflags.GetStr(b.header, bflags[f])
		sflag += fmt.Sprintf("  %s:%s\n", bflags[f], n)
	}
	if sarflags.GetStr(b.header, "freespace") == "yes" {
		sflag += fmt.Sprintf("  free:%dkB\n", b.freespace)
	}
	sflag += fmt.Sprintf("  EID:%s\n", b.eid)
	return sflag
}
