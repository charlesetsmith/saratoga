package request

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"sarflags"
	"screen"

	"github.com/charlesetsmith/saratoga/src/sarnet"
	"github.com/jroimartin/gocui"
)

// Request -- Holds Request frame information
type Request struct {
	Header  uint32
	Session uint32
	Fname   string
	Auth    []byte
}

// Make - Construct a request frame with a given header - return byte slice of frame
// Flags is of format "flagname1=flagval1,flagname2=flagval2...
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
func (r Request) Put() ([]byte, error) {

	// Create the frame slice
	framelen := 4 + 4 + len(r.Fname) + 1 + len(r.Auth) // Header + Session
	frame := make([]byte, framelen)

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
func (r *Request) Get(frame []byte) error {

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
func (r Request) Print() string {
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

// Handler - We have a request starting up a new session or updating the session info
// Create a new, or change an existing sessions information
func (r *Request) Handler(g *gocui.Gui, from *net.UDPAddr, session uint32) string {

	// screen.Fprintln(g, "msg", "green_black", r.Print())

	switch sarflags.GetStr(r.Header, "reqtype") {
	case "noaction":
		screen.Fprintln(g, "msg", "green_black", "Request Noaction from", sarnet.UDPinfo(from), " session ", session)
	case "get":
		screen.Fprintln(g, "msg", "green_black", "Request Get from", sarnet.UDPinfo(from), " session ", session)
	case "put":
		screen.Fprintln(g, "msg", "green_black", "Request Put from", sarnet.UDPinfo(from), " session ", session)
	case "getdelete":
		screen.Fprintln(g, "msg", "green_black", "Request GetDelete from", sarnet.UDPinfo(from), " session ", session)
	case "putdelete":
		screen.Fprintln(g, "msg", "green_black", "Request PutDelete from", sarnet.UDPinfo(from), " session ", session)
	case "delete":
		screen.Fprintln(g, "msg", "green_black", "Request Delete from", sarnet.UDPinfo(from), " session ", session)
	case "getdir":
		screen.Fprintln(g, "msg", "green_black", "Request GetDir from", sarnet.UDPinfo(from), " session ", session)
	default:
		screen.Fprintln(g, "msg", "red_black", "Invalid Request from", sarnet.UDPinfo(from), " session ", session)
		return "badrequest"
	}
	return "success"
	// Return an errcode string
}
