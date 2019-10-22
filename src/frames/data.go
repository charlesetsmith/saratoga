package frames

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"sarflags"
)

// Data -- Holds Data frame information
type Data struct {
	flags   uint32
	session uint32
	tstamp  Timestamp
	offset  uint64
	payload []byte
}

// DataMake - Construct a data frame - return byte slice of frame
// Flags is of format "flagname1=flagval1,flagname2=flagval2...
// The timestamp type to use is also in the flags as "timestamp=flagval"
func DataMake(flags string, session uint32, offset uint64, payload []byte) ([]byte, error) {

	var header uint32
	var err error
	var havetstamp = false
	var tstamp []byte

	flags = strings.Replace(flags, " ", "", -1) // Get rid of extra spaces in flags

	if header, err = sarflags.Set(header, "version", "v1"); err != nil {
		return nil, err
	}

	if header, err = sarflags.Set(header, "frametype", "data"); err != nil {
		return nil, err
	}

	// Grab the flags and set the frame header
	flag := strings.Split(flags, ",") // The name=val of the flag
	for fl := range flag {
		f := strings.Split(flag[fl], "=") // f[0]=name f[1]=val
		switch f[0] {
		case "descriptor", "reqtstamp", "reqstatus", "eod":
			if header, err = sarflags.Set(header, f[0], f[1]); err != nil {
				return nil, err
			}
		case "transfer":
			if f[1] == "bundle" {
				return nil, errors.New("Bundle Transfers not supported")
			}
		case "localinterp", "posix32", "posix64", "posix32_32", "posix64_32":
			if tstamp, err = TimeStampNow(f[0]); err != nil {
				return nil, err
			}
			havetstamp = true
		default:
			e := "Invalid Flag " + f[0] + " for Data Frame"
			return nil, errors.New(e)
		}
	}

	// Create the frame slice
	framelen := 4 + 4 // Header + Session

	if sarflags.GetStr(header, "reqtstamp") == "yes" {
		if havetstamp == false {
			return nil, errors.New("Data Timestamps Requested but type not specified")
		}
		framelen += 16 // Timestamp
		havetstamp = true
	}

	switch sarflags.GetStr(header, "descriptor") { // Offset
	case "d16":
		framelen += 2
	case "d32":
		framelen += 4
	case "d64":
		framelen += 8
	default:
		return nil, errors.New("Invalid descriptor in Data frame")
	}
	framelen += len(payload)

	if framelen > sarflags.MaxFrameSize {
		return nil, errors.New("Data - Maximum Frame Size Exceeded")
	}
	frame := make([]byte, framelen)

	binary.BigEndian.PutUint32(frame[:4], header)
	binary.BigEndian.PutUint32(frame[4:8], session)
	if havetstamp == true {
		copy(frame[8:24], tstamp[:16])
		switch sarflags.GetStr(header, "descriptor") {
		case "d16":
			binary.BigEndian.PutUint16(frame[24:26], uint16(offset))
			copy(frame[26:], payload)
		case "d32":
			binary.BigEndian.PutUint32(frame[24:28], uint32(offset))
			copy(frame[28:], payload)
		case "d64":
			binary.BigEndian.PutUint64(frame[24:32], uint64(offset))
			copy(frame[32:], payload)
		default:
			return nil, errors.New("Malformed Data frame")
		}
		return frame, nil
	}
	switch sarflags.GetStr(header, "descriptor") {
	case "d16":
		binary.BigEndian.PutUint16(frame[8:10], uint16(offset))
		copy(frame[10:], payload)
	case "d32":
		binary.BigEndian.PutUint32(frame[8:12], uint32(offset))
		copy(frame[12:], payload)
	case "d64":
		binary.BigEndian.PutUint64(frame[8:16], uint64(offset))
		copy(frame[16:], payload)
	default:
		return nil, errors.New("Malformed Data frame")
	}
	return frame, nil
}

// DataGet -- Decode Data byte slice frame into Data struct
func DataGet(frame []byte) (Data, error) {
	var d Data

	if len(frame) < 10 {
		return d, errors.New("data.Get - Frame too short")
	}
	d.flags = binary.BigEndian.Uint32(frame[:4])
	d.session = binary.BigEndian.Uint32(frame[4:8])
	if sarflags.GetStr(d.flags, "reqtstamp") == "yes" {
		var ts Timestamp
		var err error
		if ts, err = TimeStampGet(frame[8:24]); err != nil {
			return d, err
		}
		d.tstamp = ts
		switch sarflags.GetStr(d.flags, "descriptor") {
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
			return d, errors.New("Invalid Data Frame")
		}
		return d, nil
	}
	switch sarflags.GetStr(d.flags, "descriptor") {
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
		return d, errors.New("Invalid Data Frame")
	}
	return d, nil
}

// DataPrint - Print out details of Beacon struct
func DataPrint(d Data) string {
	sflag := fmt.Sprintf("Data: 0x%x\n", d.flags)
	dflags := sarflags.Frame("data")
	for f := range dflags {
		n := sarflags.GetStr(d.flags, dflags[f])
		sflag += fmt.Sprintf("  %s:%s\n", dflags[f], n)
	}
	if sarflags.GetStr(d.flags, "reqtstamp") == "yes" {
		sflag += fmt.Sprintf("  timestamp:%s\n", TimeStampPrint(d.tstamp))
	}
	sflag += fmt.Sprintf("  session:%d", d.session)
	sflag += fmt.Sprintf("  offset:%d", d.offset)
	sflag += fmt.Sprintf("  Payload :<%d>\n", len(d.payload))
	return sflag
}
