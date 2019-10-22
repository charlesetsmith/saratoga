package frames

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"sarflags"
)

// Request -- Holds Request frame information
type Request struct {
	flags   uint32
	session uint32
	fname   string
	auth    []byte
}

// RequestMake - Construct a request frame - return byte slice of frame
// Flags is of format "flagname1=flagval1,flagname2=flagval2...
// The timestamp type to use is also in the flags as "timestamp=flagval"
func RequestMake(flags string, session uint32, fname string, auth []byte) ([]byte, error) {

	var header uint32
	var err error

	flags = strings.Replace(flags, " ", "", -1) // Get rid of extra spaces in flags

	if header, err = sarflags.Set(header, "version", "v1"); err != nil {
		return nil, err
	}

	if header, err = sarflags.Set(header, "frametype", "request"); err != nil {
		return nil, err
	}

	// Grab the flags and set the frame header
	flag := strings.Split(flags, ",") // The name=val of the flag
	for fl := range flag {
		f := strings.Split(flag[fl], "=") // f[0]=name f[1]=val
		switch f[0] {
		case "descriptor", "stream", "reqtype", "fileordir", "udplite":
			if header, err = sarflags.Set(header, f[0], f[1]); err != nil {
				return nil, err
			}
		default:
			e := "Invalid Flag <" + f[0] + "> for Request Frame"
			return nil, errors.New(e)
		}
	}

	// Create the frame slice
	framelen := 4 + 4 + len(fname) + 1 + len(auth) // Header + Session
	if framelen > sarflags.MaxFrameSize {
		return nil, errors.New("Request - Maximum Frame Size Exceeded")
	}
	frame := make([]byte, framelen)

	binary.BigEndian.PutUint32(frame[:4], header)
	binary.BigEndian.PutUint32(frame[4:8], session)
	pos := 8 + len(fname)
	copy(frame[8:pos], fname)
	copy(frame[pos:], "\x00")
	if auth != nil {
		copy(frame[pos+1:], auth)
	}
	return frame, nil
}

// RequestGet -- Decode Data byte slice frame into Data struct
func RequestGet(frame []byte) (Request, error) {
	var r Request

	if len(frame) < 9 {
		return r, errors.New("request.Get - Frame too short")
	}
	r.flags = binary.BigEndian.Uint32(frame[:4])
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
		return r, nil
	}
	// Rest of frame is auth field
	r.auth = make([]byte, len(frame[pos:]))
	copy(r.auth, frame[pos:])
	return r, nil
}

// RequestPrint - Print out details of Beacon struct
func RequestPrint(r Request) string {
	sflag := fmt.Sprintf("Request: 0x%x\n", r.flags)
	rflags := sarflags.Frame("request")
	for f := range rflags {
		n := sarflags.GetStr(r.flags, rflags[f])
		sflag += fmt.Sprintf("  %s:%s\n", rflags[f], n)
	}
	sflag += fmt.Sprintf("  session:%d", r.session)
	sflag += fmt.Sprintf("  filename:<%s>", r.fname)
	sflag += fmt.Sprintf("  auth:<%s>\n", r.auth)
	return sflag
}
