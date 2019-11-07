package request

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"sarflags"
)

// Request -- Holds Request frame information
type Request struct {
	header  uint32
	session uint32
	fname   string
	auth    []byte
}

// New - Construct a request frame - return byte slice of frame
// Flags is of format "flagname1=flagval1,flagname2=flagval2...
func (r *Request) New(flags string, session uint32, fname string, auth []byte) error {

	var err error

	if r.header, err = sarflags.Set(r.header, "version", "v1"); err != nil {
		return err
	}

	if r.header, err = sarflags.Set(r.header, "frametype", "request"); err != nil {
		return err
	}

	flags = strings.Replace(flags, " ", "", -1) // Get rid of extra spaces in flags
	// Grab the flags and set the frame header
	flag := strings.Split(flags, ",") // The name=val of the flag
	for fl := range flag {
		f := strings.Split(flag[fl], "=") // f[0]=name f[1]=val
		switch f[0] {
		case "descriptor", "stream", "reqtype", "fileordir", "udplite":
			if r.header, err = sarflags.Set(r.header, f[0], f[1]); err != nil {
				return err
			}
		default:
			e := "Invalid Flag <" + f[0] + "> for Request Frame"
			return errors.New(e)
		}
	}

	r.session = session
	r.fname = fname
	if auth != nil {
		r.auth = make([]byte, len(auth))
		copy(r.auth, auth)
	} else {
		r.auth = nil
	}
	return nil
}

// Put -- Encode the Saratoga Request buffer
func (r Request) Put() ([]byte, error) {

	// Create the frame slice
	framelen := 4 + 4 + len(r.fname) + 1 + len(r.auth) // Header + Session
	if framelen > sarflags.MaxFrameSize {
		return nil, errors.New("Request - Maximum Frame Size Exceeded")
	}
	frame := make([]byte, framelen)

	binary.BigEndian.PutUint32(frame[:4], r.header)
	binary.BigEndian.PutUint32(frame[4:8], r.session)
	pos := 8 + len(r.fname)
	copy(frame[8:pos], r.fname)
	copy(frame[pos:], "\x00") // The null at end of fname
	if r.auth != nil {
		copy(frame[pos+1:], r.auth)
	}
	return frame, nil
}

// Get -- Decode Data byte slice frame into Data struct
func (r *Request) Get(frame []byte) error {

	if len(frame) < 9 {
		return errors.New("request.Get - Frame too short")
	}
	r.header = binary.BigEndian.Uint32(frame[:4])
	r.session = binary.BigEndian.Uint32(frame[4:8])
	pos := 8
	for i := range frame[8:] { // Filename in frame is null terminated string
		if frame[pos+i] == '\x00' { // Hit null
			break
		}
		r.fname += string(frame[pos+i])
	}
	pos = 8 + 1 + len(r.fname)
	if pos == len(frame) { // No following auth field
		r.auth = nil
		return nil
	}
	// Rest of frame is auth field
	r.auth = make([]byte, len(frame[pos:]))
	copy(r.auth, frame[pos:])
	return nil
}

// Print - Print out details of Beacon struct
func (r Request) Print() string {
	sflag := fmt.Sprintf("Request: 0x%x\n", r.header)
	rflags := sarflags.Values("request")
	for f := range rflags {
		n := sarflags.GetStr(r.header, rflags[f])
		sflag += fmt.Sprintf("  %s:%s\n", rflags[f], n)
	}
	sflag += fmt.Sprintf("  session:%d", r.session)
	sflag += fmt.Sprintf("  filename:<%s>", r.fname)
	sflag += fmt.Sprintf("  auth:<%s>\n", r.auth)
	return sflag
}
