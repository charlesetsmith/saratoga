package status

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/charlesetsmith/saratoga/src/holes"
	"github.com/charlesetsmith/saratoga/src/sarflags"
	"github.com/charlesetsmith/saratoga/src/timestamp"
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

// New - Construct a Status frame - return byte slice of frame
// Flags is of format "flagname1=flagval1,flagname2=flagval2...
// The timestamp type to use is also in the flags as "timestamp=flagval"
func (s *Status) New(flags string, session uint32, progress uint64, inrespto uint64, holes holes.Holes) error {

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
		case "descriptor", "stream", "reqtstamp", "metadatarecvd", "allholes", "reqholes", "errcode":
			if s.Header, err = sarflags.Set(s.Header, f[0], f[1]); err != nil {
				return err
			}
		case "localinterp", "posix32", "posix64", "posix32_32", "posix64_32":
			if err = s.Tstamp.Now(f[0]); err != nil {
				return err
			}
			s.Header, err = sarflags.Set(s.Header, "reqtsamp", "yes")
		default:
			e := "Invalid Flag " + f[0] + " for Status Frame"
			return errors.New(e)
		}
	}
	s.Session = session
	s.Progress = progress
	s.Inrespto = inrespto
	for i := range holes {
		s.Holes[i].Start = holes[i].Start
		s.Holes[i].End = holes[i].End
	}

	return nil
}

// Put - Encode the Saratoga Status frame
func (s Status) Put() ([]byte, error) {

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
	default:
		return nil, errors.New("Invalid descriptor in Status frame")
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
	case 2:
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
	case 4:
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
	case 8:
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
	}
	return frame, nil
}

// Get -- Decode Status byte slice frame into Status struct
func (s *Status) Get(frame []byte) error {

	if len(frame) < 12 {
		return errors.New("Status Frame too short")
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
	default:
		return errors.New("Invalid descriptor in Status frame")
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
