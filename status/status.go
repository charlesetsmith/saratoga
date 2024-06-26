// Status Frame Handler

package status

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"reflect"
	"strings"

	"github.com/charlesetsmith/saratoga/holes"
	"github.com/charlesetsmith/saratoga/sarflags"
	"github.com/charlesetsmith/saratoga/timestamp"
)

// Status -- Status of the transfer and holes
type Status struct {
	Header   uint32
	Session  uint32
	Tstamp   timestamp.Timestamp
	Progress uint64
	Inrespto uint64
	Holes    holes.Holes
}

type Sinfo struct {
	Session  uint32
	Progress uint64
	Inrespto uint64
	Holes    holes.Holes
}

type Packet struct {
	Addr net.UDPAddr
	Info Status
}

// Assemble the Packet with to/from address
// No pointers in return as used in channel
func (s *Status) Val(addr *net.UDPAddr) Packet {
	// We have to copy across and send all of the holes!!!!!
	return Packet{
		Addr: *addr,
		Info: Status{Header: s.Header, Session: s.Session, Tstamp: s.Tstamp,
			Progress: s.Progress, Inrespto: s.Inrespto, Holes: s.Holes}}
}

func (s *Status) Values() Status {
	return Status{Header: s.Header, Session: s.Session, Tstamp: s.Tstamp, Progress: s.Progress,
		Inrespto: s.Inrespto, Holes: s.Holes}
}

// New - Construct a Status frame - return byte slice of frame
// Flags is of format "flagname1=flagval1,flagname2=flagval2...
// The timestamp type to use is also in the flags as "timestamp=flagval"
// func (s *Status) New(flags string, session uint32, progress uint64, inrespto uint64, holes holes.Holes) error {
func (s *Status) New(flags string, info interface{}) error {

	var err error

	flags = strings.Replace(flags, " ", "", -1) // Get rid of extra spaces in flags

	if s.Header, err = sarflags.Set(s.Header, "version", "v1"); err != nil {
		return err
	}

	if s.Header, err = sarflags.Set(s.Header, "frametype", "status"); err != nil {
		return err
	}

	// Grab the flags and set the frame header
	flag := strings.Split(flags, ",") // The name=val of the flag
	for fl := range flag {
		f := strings.Split(flag[fl], "=") // f[0]=name f[1]=val
		switch f[0] {
		case "descriptor", "stream", "metadatarecvd", "allholes", "reqholes", "errcode":
			if s.Header, err = sarflags.Set(s.Header, f[0], f[1]); err != nil {
				return err
			}
		case "reqtstamp":
			if s.Header, err = sarflags.Set(s.Header, f[0], f[1]); err != nil {
				return err
			}
			// If we requested a timestamp then default it to posix32
			if f[1] == "yes" {
				if err = s.Tstamp.Now("posix32"); err != nil {
					return err
				}
			}
		// If we have not requested a timestamp (reqtstamp=no) but we have set the type to one of below then
		// get the time and date and set (reqtstamp=yes)
		case "localinterp", "posix32", "posix64", "posix32_32", "posix64_32":
			if err = s.Tstamp.Now(f[0]); err != nil {
				return err
			}
			if s.Header, err = sarflags.Set(s.Header, "reqtstamp", "yes"); err != nil {
				return err
			}
		default:
			return errors.New("Invalid Flag " + f[0] + " for Status Frame")
		}
	}
	e := reflect.ValueOf(info).Elem()
	s.Session = uint32(e.FieldByName("Session").Uint())
	s.Progress = e.FieldByName("Progress").Uint()
	s.Inrespto = e.FieldByName("Inrespto").Uint()
	for i := 0; i < e.FieldByName("Holes").Len(); i++ {
		var h holes.Hole
		// Get the Start and End from within the Holes Structure
		h.Start = int(e.FieldByName("Holes").Index(i).FieldByName("Start").Int())
		h.End = int(e.FieldByName("Holes").Index(i).FieldByName("End").Int())
		s.Holes = append(s.Holes, h)
	}
	return nil
}

// Make - Construct a status frame with a given header
// func (s *Status) Make(header uint32, session uint32, progress uint64, inrespto uint64, holes holes.Holes) error {
func (s *Status) Make(header uint32, info interface{}) error {

	var err error

	if header, err = sarflags.Set(header, "version", "v1"); err != nil {
		return err
	}
	if header, err = sarflags.Set(header, "frametype", "status"); err != nil {
		return err
	}

	s.Header = header
	e := reflect.ValueOf(info).Elem()
	s.Session = uint32(e.FieldByName("Session").Uint())
	s.Progress = e.FieldByName("Progress").Uint()
	s.Inrespto = e.FieldByName("Inrespto").Uint()
	for i := 0; i < e.FieldByName("Holes").Len(); i++ {
		var h holes.Hole
		// Get the Start and End from within the Holes Structure
		h.Start = int(e.FieldByName("Holes").Index(i).FieldByName("Start").Int())
		h.End = int(e.FieldByName("Holes").Index(i).FieldByName("End").Int())
		s.Holes = append(s.Holes, h)
	}
	return nil
}

// Put - Encode the Saratoga Status frame
func (s Status) Encode() ([]byte, error) {

	havetstamp := false

	// Create the frame slice
	framelen := 4 + 4 // Header + Session

	if sarflags.GetStr(s.Header, "reqtstamp") == "yes" {
		framelen += 16 // Timestamp
	}

	var dsize int

	switch sarflags.GetStr(s.Header, "descriptor") { // Offset
	case "d16":
		dsize = 2
	case "d32":
		dsize = 4
	case "d64":
		dsize = 8
	case "d128":
		return nil, errors.New("d128 not supported in status")
		// dsize = 16
	default:
		return nil, errors.New("invalid descriptor in status")
	}
	framelen += (dsize*2 + (dsize * len(s.Holes) * 2)) // progress + inrespto + holes
	frame := make([]byte, framelen)

	binary.BigEndian.PutUint32(frame[:4], s.Header)
	binary.BigEndian.PutUint32(frame[4:8], s.Session)

	pos := 8
	if havetstamp {
		ts := s.Tstamp.Put()
		copy(frame[pos:24], ts)
		pos = 24
	}
	switch dsize {
	case 2: // d16 bit descriptor
		binary.BigEndian.PutUint16(frame[pos:pos+dsize], uint16(s.Progress))
		pos += dsize
		binary.BigEndian.PutUint16(frame[pos:pos+dsize], uint16(s.Inrespto))
		pos += dsize
		for i := range s.Holes {
			binary.BigEndian.PutUint16(frame[pos:pos+dsize], uint16(s.Holes[i].Start))
			pos += dsize
			binary.BigEndian.PutUint16(frame[pos:pos+dsize], uint16(s.Holes[i].End))
			pos += dsize
		}
	case 4: // d32 bit descriptor
		binary.BigEndian.PutUint32(frame[pos:pos+dsize], uint32(s.Progress))
		pos += dsize
		binary.BigEndian.PutUint32(frame[pos:pos+4], uint32(s.Inrespto))
		pos += dsize
		for i := range s.Holes {
			binary.BigEndian.PutUint32(frame[pos:pos+dsize], uint32(s.Holes[i].Start))
			pos += dsize
			binary.BigEndian.PutUint32(frame[pos:pos+dsize], uint32(s.Holes[i].End))
			pos += dsize
		}
	case 8: // d64 bit descriptor
		binary.BigEndian.PutUint64(frame[pos:pos+dsize], uint64(s.Progress))
		pos += dsize
		binary.BigEndian.PutUint64(frame[pos:pos+dsize], uint64(s.Inrespto))
		pos += dsize
		for i := range s.Holes {
			binary.BigEndian.PutUint64(frame[pos:pos+dsize], uint64(s.Holes[i].Start))
			pos += dsize
			binary.BigEndian.PutUint64(frame[pos:pos+dsize], uint64(s.Holes[i].End))
			pos += dsize
		}
		/*
			case 16: // d128 bit descriptor when we get there!
				binary.BigEndian.PutUint128(frame[pos:pos+dsize], uint128(s.Progress))
				pos += dsize
				binary.BigEndian.PutUint128(frame[pos:pos+dsize], uint128(s.Inrespto))
				pos += dsize
				for i := range s.Holes {
					binary.BigEndian.PutUint128(frame[pos:pos+dsize], uint128(s.Holes[i].Start))
					pos += dsize
					binary.BigEndian.PutUint128(frame[pos:pos+dsize], uint128(s.Holes[i].End))
					pos += dsize
				}
		*/
	}
	return frame, nil
}

// Get -- Decode Status byte slice frame into Status struct
func (s *Status) Decode(frame []byte) error {

	if len(frame) < 12 {
		return errors.New("status frame too short")
	}
	s.Header = binary.BigEndian.Uint32(frame[:4])
	s.Session = binary.BigEndian.Uint32(frame[4:8])
	pos := 8
	if sarflags.GetStr(s.Header, "reqtstamp") == "yes" {
		var err error

		if err = s.Tstamp.Get(frame[pos:24]); err != nil {
			return err
		}
		pos = 24
	}

	var dsize int

	switch sarflags.GetStr(s.Header, "descriptor") {
	case "d16":
		dsize = 2
		s.Progress = uint64(binary.BigEndian.Uint16(frame[pos : pos+dsize]))
		pos += dsize
		s.Inrespto = uint64(binary.BigEndian.Uint16(frame[pos : pos+dsize]))
		pos += dsize
		hlen := len(frame[pos:])
		// log.Fatal("Holes in frame len", hlen, "number of holes", hlen/2/dsize)
		for i := 0; i < hlen/2/dsize; i++ {
			start := int(binary.BigEndian.Uint16(frame[pos : pos+dsize]))
			pos += dsize
			end := int(binary.BigEndian.Uint16(frame[pos : pos+dsize]))
			pos += dsize
			s.Holes = append(s.Holes, holes.Hole{Start: start, End: end})
		}
	case "d32":
		dsize = 4
		s.Progress = uint64(binary.BigEndian.Uint32(frame[pos : pos+dsize]))
		pos += dsize
		s.Inrespto = uint64(binary.BigEndian.Uint32(frame[pos : pos+dsize]))
		pos += dsize
		hlen := len(frame[pos:])
		for i := 0; i < hlen/2/dsize; i++ {
			start := int(binary.BigEndian.Uint32(frame[pos : pos+dsize]))
			pos += dsize
			end := int(binary.BigEndian.Uint32(frame[pos : pos+dsize]))
			pos += dsize
			s.Holes = append(s.Holes, holes.Hole{Start: start, End: end})
		}
	case "d64":
		dsize = 8
		s.Progress = uint64(binary.BigEndian.Uint64(frame[pos : pos+dsize]))
		pos += dsize
		s.Inrespto = uint64(binary.BigEndian.Uint64(frame[pos : pos+dsize]))
		pos += dsize
		hlen := len(frame[pos:])
		for i := 0; i < hlen/2/dsize; i++ {
			start := int(binary.BigEndian.Uint64(frame[pos : pos+dsize]))
			pos += dsize
			end := int(binary.BigEndian.Uint64(frame[pos : pos+dsize]))
			pos += dsize
			s.Holes = append(s.Holes, holes.Hole{Start: start, End: end})
		}
	case "d128":
		return errors.New("d128 not supported in status")
		/* when we get there!
		dsize = 16
		s.Progress = uint128(binary.BigEndian.Uint128(frame[pos : pos+dsize]))
		pos += dsize
		s.Inrespto = uint128(binary.BigEndian.Uint128(frame[pos : pos+dsize]))
		pos += dsize
		hlen := len(frame[pos:])
		for i := 0; i < hlen/2/dsize; i++ {
			start := int(binary.BigEndian.Uint128(frame[pos : pos+dsize]))
			pos += dsize
			end := int(binary.BigEndian.Uint128(frame[pos : pos+dsize]))
			pos += dsize
			s.Holes = append(s.Holes, holes.Hole{Start: start, End: end})
		}
		*/
	default:
		return errors.New("invalid descriptor in status")
	}
	return nil
}

// Print - Print out details of Status struct
func (s Status) Print() string {
	sflag := fmt.Sprintf("Status: 0x%x\n", s.Header)
	sflags := sarflags.Values("status")
	for f := range sflags {
		n := sarflags.GetStr(s.Header, sflags[f])
		sflag += fmt.Sprintf("  %s:%s\n", sflags[f], n)
	}
	sflag += fmt.Sprintf("  session:%d\n", s.Session)
	if sarflags.GetStr(s.Header, "reqtstamp") == "yes" {
		sflag += fmt.Sprintf("  timestamp:%s\n", s.Tstamp.Print())
	}
	sflag += fmt.Sprintf("  progress:%d", s.Progress)
	sflag += fmt.Sprintf("  inresponseto:%d\n", s.Inrespto)
	for i := range s.Holes {
		sflag += fmt.Sprintf("  Hole[%d]: Start:%d End:%d\n", i, s.Holes[i].Start, s.Holes[i].End)
	}
	return sflag
}

// ShortPrint - Quick print out of Status struct
func (s Status) ShortPrint() string {
	sflag := fmt.Sprintf("Status: 0x%x\n", s.Header)
	sflag += fmt.Sprintf("  session:%d\n", s.Session)
	if sarflags.GetStr(s.Header, "reqtstamp") == "yes" {
		sflag += fmt.Sprintf("  timestamp:%s\n", s.Tstamp.Print())
	}
	sflag += fmt.Sprintf("  errcode:%s\n", sarflags.GetStr(s.Header, "errcode"))
	sflag += fmt.Sprintf("  progress:%d\n", s.Progress)
	sflag += fmt.Sprintf("  inrespto:%d\n", s.Inrespto)
	sflag += fmt.Sprintf("  numb holes:%d", len(s.Holes))
	return sflag
}

/* Send a data to an address on the connection */
func (s Status) Send(conn *net.UDPConn, to *net.UDPAddr) error {
	var err error
	var buf []byte
	var wlen int

	if buf, err = s.Encode(); err != nil {
		return err
	}
	if wlen, err = conn.WriteTo(buf, to); err != nil {
		return err
	}
	if wlen != len(buf) {
		return fmt.Errorf("status sent (%d) to %s != frame size (%d)", wlen, to.String(), len(buf))
	}
	return nil
}
