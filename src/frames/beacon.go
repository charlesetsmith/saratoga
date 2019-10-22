package frames

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"sarflags"
)

// Beacon -- Holds Beacon frame information
type Beacon struct {
	flags     uint32
	freespace uint64
	eid       string
}

// BeaconMake - Construct a beacon - return byte slice of frame
// Flags is of format "flagname1=flagval1,flagname2=flagval2..."
func BeaconMake(flags string, eid string, freespace uint64) ([]byte, error) {

	var header uint32
	var err error
	var reportfreespace bool

	flags = strings.Replace(flags, " ", "", -1) // Get rid of extra spaces in flags

	if header, err = sarflags.Set(header, "version", "v1"); err != nil {
		return nil, err
	}

	if header, err = sarflags.Set(header, "frametype", "beacon"); err != nil {
		return nil, err
	}

	// Grab the flags and set the frame header
	flag := strings.Split(flags, ",") // The name=val of the flag
	for fl := range flag {
		f := strings.Split(flag[fl], "=") // f[0]=name f[1]=val
		switch f[0] {
		case "descriptor", "stream", "txwilling", "rxwilling", "udplite", "freespaced":
			if header, err = sarflags.Set(header, f[0], f[1]); err != nil {
				return nil, err
			}
		case "freespace":
			if header, err = sarflags.Set(header, f[0], f[1]); err != nil {
				return nil, err
			}
			if f[1] == "yes" {
				reportfreespace = true
			} else {
				reportfreespace = false
			}
		default:
			e := "Invalid Flag " + f[0] + " for Beacon Frame"
			return nil, errors.New(e)
		}
	}

	if reportfreespace == true {
		if freespace <= sarflags.MaxUint16 {
			if header, err = sarflags.Set(header, "freespaced", "d16"); err != nil {
				return nil, err
			}
			frame := make([]byte, 4+2+len(eid))
			binary.BigEndian.PutUint32(frame[:4], header)
			binary.BigEndian.PutUint16(frame[4:6], uint16(freespace))
			copy(frame[6:], []byte(eid))
			return frame, nil
		} else if freespace <= sarflags.MaxUint32 {
			if header, err = sarflags.Set(header, "freespaced", "d32"); err != nil {
				return nil, err
			}
			frame := make([]byte, 4+4+len(eid))
			binary.BigEndian.PutUint32(frame[:4], header)
			binary.BigEndian.PutUint32(frame[4:8], uint32(freespace))
			copy(frame[8:], []byte(eid))
			return frame, nil
		} else {
			if header, err = sarflags.Set(header, "freespaced", "d64"); err != nil {
				return nil, err
			}
			frame := make([]byte, 4+8+len(eid))
			binary.BigEndian.PutUint32(frame[:4], header)
			binary.BigEndian.PutUint64(frame[4:12], uint64(freespace))
			copy(frame[12:], []byte(eid))
			return frame, nil
		}
	} else {
		if header, err = sarflags.Set(header, "freespaced", "d16"); err != nil {
			return nil, err
		}
		frame := make([]byte, 4+len(eid))
		binary.BigEndian.PutUint32(frame[:4], header)
		copy(frame[4:], []byte(eid))
		return frame, nil
	}
}

// BeaconGet -- Decode Beacon byte slice frame into Beacon struct
func BeaconGet(frame []byte) (Beacon, error) {
	var b Beacon

	b.flags = binary.BigEndian.Uint32(frame[:4])
	if sarflags.GetStr(b.flags, "freespace") == "yes" {
		switch sarflags.GetStr(b.flags, "freespaced") {
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
			return b, errors.New("Invalid Beacon Frame")
		}
		return b, nil
	}
	// No freespace to be reported
	b.freespace = 0
	b.eid = string(frame[:4])
	return b, nil
}

// BeaconPrint - Print out details of Beacon struct
func BeaconPrint(b Beacon) string {
	sflag := fmt.Sprintf("Beacon: 0x%x\n", b.flags)
	bflags := sarflags.Frame("beacon")
	for f := range bflags {
		n := sarflags.GetStr(b.flags, bflags[f])
		sflag += fmt.Sprintf("  %s:%s\n", bflags[f], n)
	}
	if sarflags.GetStr(b.flags, "freespace") == "yes" {
		sflag += fmt.Sprintf("  free:%dkB\n", b.freespace)
	}
	sflag += fmt.Sprintf("  EID:%s\n", b.eid)
	return sflag
}
