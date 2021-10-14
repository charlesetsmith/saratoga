// Request Frame Handing

package request

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/charlesetsmith/saratoga/sarflags"
)

// Request -- Holds Request frame information
type Request struct {
	Header  uint32
	Session uint32
	Fname   string
	Auth    []byte
}

// New - Construct a request - Fill in the request struct
func (r *Request) New(flags string, session uint32, fname string, auth []byte) error {
	var err error

	// Always present in a Request
	if r.Header, err = sarflags.Set(r.Header, "version", "v1"); err != nil {
		return err
	}
	// And yes we are a Request Frame
	if r.Header, err = sarflags.Set(r.Header, "frametype", "request"); err != nil {
		return err
	}

	flags = strings.Replace(flags, " ", "", -1) // Get rid of extra spaces in flags
	flags = strings.TrimRight(flags, ",")       // And the last comma if its there

	// Grab the flags and set the frame header
	flag := strings.Split(flags, ",") // The name=val of the flag
	for fl := range flag {
		f := strings.Split(flag[fl], "=") // f[0]=name f[1]=val
		switch f[0] {
		case "frametype", "version", "descriptor", "stream", "fileordir", "reqtype", "udplite":
			if r.Header, err = sarflags.Set(r.Header, f[0], f[1]); err != nil {
				return err
			}
		default:
			e := "Request.New: Invalid Flag " + f[0] + "=" + f[1] + "<" + flags + ">"
			return errors.New(e)
		}
	}
	r.Session = session
	r.Fname = fname
	r.Auth = auth
	return nil
}

// Make - Construct a request frame with a given header
func (r *Request) Make(header uint32, session uint32, fname string, auth []byte) error {

	var err error

	if header, err = sarflags.Set(header, "version", "v1"); err != nil {
		return err
	}

	if header, err = sarflags.Set(header, "frametype", "request"); err != nil {
		return err
	}

	r.Header = header
	r.Session = session
	r.Fname = fname
	if auth != nil {
		r.Auth = make([]byte, len(auth))
		copy(r.Auth, auth)
	} else {
		r.Auth = nil
	}
	return nil
}

// Put -- Encode the Saratoga Request buffer
func (r *Request) Encode() ([]byte, error) {

	// Create the frame slice
	framelen := 4 + 4 + len(r.Fname) + 1 + len(r.Auth) // Header + Session + Fname + NULL + Auth
	frame := make([]byte, framelen)                    // Allocate the frame buffer

	binary.BigEndian.PutUint32(frame[:4], r.Header)
	binary.BigEndian.PutUint32(frame[4:8], r.Session)
	pos := 8 + len(r.Fname)
	copy(frame[8:pos], r.Fname)
	copy(frame[pos:], "\x00") // The null at end of fname
	if r.Auth != nil {
		copy(frame[pos+1:], r.Auth)
	}
	return frame, nil
}

// Get -- Decode Data byte slice frame into Data struct
func (r *Request) Decode(frame []byte) error {

	if len(frame) < 9 {
		return errors.New("request.Get - Frame too short")
	}
	r.Header = binary.BigEndian.Uint32(frame[:4])
	r.Session = binary.BigEndian.Uint32(frame[4:8])
	pos := 8
	for i := range frame[8:] { // Filename in frame is null terminated string
		if frame[pos+i] == '\x00' { // Hit null
			break
		}
		r.Fname += string(frame[pos+i])
	}
	pos = 8 + 1 + len(r.Fname)
	if pos == len(frame) { // No following auth field
		r.Auth = nil
		return nil
	}
	// Rest of frame is auth field
	r.Auth = make([]byte, len(frame[pos:]))
	copy(r.Auth, frame[pos:])
	return nil
}

// Print - Print out details of Beacon struct
func (r *Request) Print() string {
	sflag := fmt.Sprintf("Request: 0x%x\n", r.Header)
	rflags := sarflags.Values("request")
	for f := range rflags {
		n := sarflags.GetStr(r.Header, rflags[f])
		sflag += fmt.Sprintf("  %s:%s\n", rflags[f], n)
	}
	sflag += fmt.Sprintf("  session:%d", r.Session)
	sflag += fmt.Sprintf("  filename:<%s>", r.Fname)
	sflag += fmt.Sprintf("  auth:<%s>\n", r.Auth)
	return sflag
}
