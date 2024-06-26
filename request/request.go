// Request Frame Handing

package request

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"reflect"
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
type Rinfo struct {
	Session uint32
	Fname   string
	Auth    []byte
}

type Packet struct {
	Addr net.UDPAddr
	Info Request
}

// No pointers in return as used in channels
func (r *Request) Val(addr *net.UDPAddr) Packet {
	return Packet{Addr: *addr,
		Info: Request{Header: r.Header, Session: r.Session, Fname: r.Fname, Auth: r.Auth}}
}

// New - Construct a request - Fill in the request struct
// func (r *Request) New(flags string, session uint32, fname string, auth []byte) error {
func (r *Request) New(flags string, info interface{}) error {
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
		case "frametype", "version", "descriptor", "stream", "txwilling",
			"rxwilling", "fileordir", "reqtype", "udplite":
			if r.Header, err = sarflags.Set(r.Header, f[0], f[1]); err != nil {
				return err
			}
		default:
			return errors.New("Request.New: Invalid Flag " + f[0] + "=" + f[1] + "<" + flags + ">")
		}
	}
	e := reflect.ValueOf(info).Elem()
	r.Session = uint32(e.FieldByName("Session").Uint())
	r.Fname = e.FieldByName("Fname").String()
	copy(r.Auth, e.FieldByName("Auth").Bytes())

	return nil
}

// Make - Construct a request frame with a given header
// func (r *Request) Make(header uint32, session uint32, fname string, auth []byte) error {
func (r *Request) Make(header uint32, info interface{}) error {

	var err error

	if header, err = sarflags.Set(header, "version", "v1"); err != nil {
		return err
	}

	if header, err = sarflags.Set(header, "frametype", "request"); err != nil {
		return err
	}

	r.Header = header

	e := reflect.ValueOf(info).Elem()
	r.Session = uint32(e.Elem().FieldByName("Session").Uint())
	r.Fname = e.FieldByName("Fname").String()
	copy(r.Auth, e.FieldByName("Auth").Bytes())

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
		return errors.New("request.Decode - request frame too short")
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

// Print - Print out details of Request struct
func (r Request) Print() string {
	sflag := fmt.Sprintf("Request: 0x%x\n", r.Header)
	rflags := sarflags.Values("request")
	for f := range rflags {
		n := sarflags.GetStr(r.Header, rflags[f])
		sflag += fmt.Sprintf("  %s:%s\n", rflags[f], n)
	}
	sflag += fmt.Sprintf("  session:%d\n", r.Session)
	sflag += fmt.Sprintf("  filename:%s\n", r.Fname)
	sflag += fmt.Sprintf("  auth:%s", r.Auth)
	return sflag
}

// Print - Print out details of Request struct
func (r Request) ShortPrint() string {
	sflag := fmt.Sprintf("Request: 0x%x\n", r.Header)
	sflag += fmt.Sprintf("  session:%d\n", r.Session)
	sflag += fmt.Sprintf("  filename:%s\n", r.Fname)
	sflag += fmt.Sprintf("  auth:%s", r.Auth)
	return sflag
}

/* Send a request to an address on the connection */
func (r Request) Send(conn *net.UDPConn, to *net.UDPAddr) error {
	var err error
	var buf []byte
	var wlen int

	if buf, err = r.Encode(); err != nil {
		return err
	}
	if wlen, err = conn.WriteTo(buf, to); err != nil {
		return err
	}
	if wlen != len(buf) {
		return fmt.Errorf("Request sent (%d) to %s != frame size (%d)", wlen, to.String(), len(buf))
	}
	return nil
}
