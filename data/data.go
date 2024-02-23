// Data Frame Handling

package data

import (
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/charlesetsmith/saratoga/sarflags"
	"github.com/charlesetsmith/saratoga/timestamp"
)

// Data -- Holds Data frame information
type Data struct {
	Header  uint32
	Session uint32
	Tstamp  timestamp.Timestamp // This is set at the time of creation of the Data stuct i.e. Now
	Offset  uint64
	Payload []byte
}

// Data info struct we send to Interface for creation in New and Make
type Dinfo struct {
	Session uint32
	Offset  uint64
	Payload []byte
}

// New - Construct a data frame - return byte slice of frame and Data structure
// Flags is of format "flagname1=flagval1,flagname2=flagval2...
// The timestamp type to use is also in the flags as "timestamp=flagval"
// func (d *Data) New(flags string, session uint32, offset uint64, payload []byte) error {
func (d *Data) New(flags string, info interface{}) error {

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
		case "version", "descriptor", "reqstatus", "eod":
			if d.Header, err = sarflags.Set(d.Header, f[0], f[1]); err != nil {
				return err
			}
		case "transfer":
			if f[1] == "bundle" {
				return errors.New("bundle transfers not supported")
			}
		case "reqtstamp":
			switch f[1] {
			case "no":
				if d.Header, err = sarflags.Set(d.Header, f[0], f[1]); err != nil {
					return err
				}
			case "localinterp", "posix32", "posix64", "posix32_32", "posix64_32", "epoch2000_32":
				if d.Header, err = sarflags.Set(d.Header, f[0], "yes"); err != nil {
					return err
				}
				if err = d.Tstamp.Now(f[1]); err != nil { // Set the timestamp to right now
					return err
				}
			case "yes":
				d.Header, _ = sarflags.Set(d.Header, f[0], "yes")
				if err = d.Tstamp.Now("posix32"); err != nil { // Set the timestamp to right now
					return err
				}
			default:
				return errors.New(f[1] + " invalid timestamp type in data")
			}
		default:
			return errors.New(f[0] + " invalid flag in data")
		}
	}

	e := reflect.ValueOf(info).Elem()
	// Assign the values from the interface Dinfo structure
	d.Session = uint32(e.FieldByName("Session").Uint())
	d.Offset = e.FieldByName("Offset").Uint()
	d.Payload = make([]byte, e.FieldByName("Payload").Len())
	copy(d.Payload, e.FieldByName("Payload").Bytes())
	return nil
}

// Make - Construct a data frame with a given header - return byte slice of frame and Data structure
// func (d *Data) Make(header uint32, session uint32, offset uint64, payload []byte) error {
func (d *Data) Make(header uint32, info interface{}) error {

	var err error

	d.Header = header
	if d.Header, err = sarflags.Set(d.Header, "version", "v1"); err != nil {
		return err
	}
	if d.Header, err = sarflags.Set(d.Header, "frametype", "data"); err != nil {
		return err
	}

	tstamp := sarflags.GetStr(d.Header, "reqtstamp")
	switch tstamp {
	case "no":
		// Don't do anything it's already set to no
	case "localinterp", "posix32", "posix64", "posix32_32", "posix64_32", "epoch2000_32":
		if err = d.Tstamp.Now(tstamp); err != nil { // Set the timestamp to right now
			return err
		}
	case "yes":
		if err = d.Tstamp.Now("posix32"); err != nil { // Set the timestamp to right now
			return err
		}
	default:
		return errors.New(sarflags.GetStr(d.Header, "reqtstamp") + "invalid timestamp type")
	}

	e := reflect.ValueOf(info).Elem()
	// Assign the values from the interface Dinfo structure
	d.Session = uint32(e.FieldByName("Session").Uint())
	d.Offset = e.FieldByName("Offset").Uint()
	dlen := e.FieldByName("Payload").Len()
	d.Payload = make([]byte, dlen)
	copy(d.Payload, e.FieldByName("Payload").Bytes())

	return nil
}

// Get -- Decode Data byte slice frame into Data struct
func (d *Data) Decode(frame []byte) error {

	if len(frame) < 10 {
		return errors.New("data.Get - data frame too short")
	}
	d.Header = binary.BigEndian.Uint32(frame[:4])
	d.Session = binary.BigEndian.Uint32(frame[4:8])
	if sarflags.GetStr(d.Header, "reqtstamp") == "yes" {
		if err := d.Tstamp.Get(frame[8:24]); err != nil {
			return err
		}
		switch sarflags.GetStr(d.Header, "descriptor") {
		case "d16":
			d.Offset = uint64(binary.BigEndian.Uint16(frame[24:26])) // 2 bytes
			d.Payload = make([]byte, len(frame[26:]))
			copy(d.Payload, frame[26:])
			if len(d.Payload) == 0 {
				return errors.New("length of data payload is 0")
			}
		case "d32":
			d.Offset = uint64(binary.BigEndian.Uint32(frame[24:28])) // 4 bytes
			d.Payload = make([]byte, len(frame[28:]))
			copy(d.Payload, frame[28:])
		case "d64":
			d.Offset = binary.BigEndian.Uint64(frame[24:32]) // 8 bytes
			d.Payload = make([]byte, len(frame[32:]))
			copy(d.Payload, frame[32:])
		case "d128": // KLUDGE!!!!
			return errors.New(sarflags.GetStr(d.Header, "descriptor") + "d128 not supported in data")
			// d.Offset = binary.BigEndian.Uint64(frame[24+8 : 40]) // 16 bytes
			// d.Payload = make([]byte, len(frame[64:]))
			// copy(d.Payload, frame[64:])
		default:
			return errors.New(sarflags.GetStr(d.Header, "descriptor") + " invalid descriptor in data")
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
		d.Offset = uint64(binary.BigEndian.Uint64(frame[8:16]))
		d.Payload = make([]byte, len(frame[16:]))
		copy(d.Payload, frame[16:])
	case "d128": // KLUDGE!!!!
		return errors.New(sarflags.GetStr(d.Header, "descriptor") + "d128 not supported in data")
		// d.Offset = uint64(binary.BigEndian.Uint64(frame[8+8 : 32]))
		// d.Payload = make([]byte, len(frame[32:]))
		// copy(d.Payload, frame[32:])
	default:
		return errors.New(sarflags.GetStr(d.Header, "descriptor") + "invalid descriptor in data")
	}
	return nil
}

// Put -- Encode the Saratoga Data Frame buffer
func (d *Data) Encode() ([]byte, error) {

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
	case "d128":
		return nil, errors.New(sarflags.GetStr(d.Header, "descriptor") + "d128 not supported in data")
		// framelen += 16
	default:
		return nil, errors.New(sarflags.GetStr(d.Header, "descriptor") + "invalid descriptor in data")
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
	case "d128": // KLUDGE!!!!!
		return nil, errors.New(sarflags.GetStr(d.Header, "descriptor") + "d128 not supported in data")
		// pos += 16
	default:
		return nil, errors.New(sarflags.GetStr(d.Header, "descriptor") + " invalid descriptor in data")
	}
	copy(frame[pos:], d.Payload)
	return frame, nil
}

// Print - Print out full details of Data struct
func (d Data) Print() string {
	sflag := fmt.Sprintf("Data: 0x%x\n", d.Header)
	dflags := sarflags.Values("data")
	for _, f := range dflags {
		n := sarflags.GetStr(d.Header, f)
		sflag += fmt.Sprintf("  %s:%s\n", f, n)
	}
	if sarflags.GetStr(d.Header, "reqtstamp") == "yes" {
		sflag += fmt.Sprintf("  tstamp:%s\n", d.Tstamp.Print())
	}
	sflag += fmt.Sprintf("  session:%d", d.Session)
	sflag += fmt.Sprintf("  offset:%d", d.Offset)
	sflag += fmt.Sprintf("  paylen : %d", len(d.Payload))
	return sflag
}

// ShortPrint - Print out minimal details of Data struct
func (d Data) ShortPrint() string {
	sflag := fmt.Sprintf("Data: 0x%x\n", d.Header)
	if sarflags.GetStr(d.Header, "reqtstamp") == "yes" {
		sflag += fmt.Sprintf("  tstamp:%s\n", d.Tstamp.Print())
	}
	sflag += fmt.Sprintf("  session:%d,", d.Session)
	sflag += fmt.Sprintf("  offset:%d,", d.Offset)
	sflag += fmt.Sprintf("  paylen:%d", len(d.Payload))
	return sflag
}
