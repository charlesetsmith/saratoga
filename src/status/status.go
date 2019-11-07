package status

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"sarflags"
	"timestamp"
)

// Hole -- Beggining and End of a hole
type Hole struct {
	Start uint64
	End   uint64
}

// Status -- Status of the transfer and holes
type Status struct {
	header   uint32
	session  uint32
	tstamp   timestamp.Timestamp
	progress uint64
	inrespto uint64
	holes    []Hole
}

// New - Construct a Status frame - return byte slice of frame
// Flags is of format "flagname1=flagval1,flagname2=flagval2...
// The timestamp type to use is also in the flags as "timestamp=flagval"
func (s *Status) New(flags string, session uint32, progress uint64, inrespto uint64, holes []Hole) error {

	var err error

	flags = strings.Replace(flags, " ", "", -1) // Get rid of extra spaces in flags

	if s.header, err = sarflags.Set(s.header, "version", "v1"); err != nil {
		return err
	}

	if s.header, err = sarflags.Set(s.header, "frametype", "status"); err != nil {
		return err
	}

	// Grab the flags and set the frame header
	flag := strings.Split(flags, ",") // The name=val of the flag
	for fl := range flag {
		f := strings.Split(flag[fl], "=") // f[0]=name f[1]=val
		switch f[0] {
		case "descriptor", "stream", "reqtstamp", "metadatarecvd", "allholes", "reqholes", "errcode":
			if s.header, err = sarflags.Set(s.header, f[0], f[1]); err != nil {
				return err
			}
		case "localinterp", "posix32", "posix64", "posix32_32", "posix64_32":
			if err = s.tstamp.Now(f[0]); err != nil {
				return err
			}
			s.header, err = sarflags.Set(s.header, "reqtsamp", "yes")
		default:
			e := "Invalid Flag " + f[0] + " for Status Frame"
			return errors.New(e)
		}
	}
	s.session = session
	s.progress = progress
	s.inrespto = inrespto
	for i := range holes {
		s.holes[i].Start = holes[i].Start
		s.holes[i].End = holes[i].End
	}

	return nil
}

// Put - Encode the Saratoga Status frame
func (s Status) Put() ([]byte, error) {

	havetstamp := false

	// Create the frame slice
	framelen := 4 + 4 // Header + Session

	if sarflags.GetStr(s.header, "reqtstamp") == "yes" {
		framelen += 16 // Timestamp
	}

	var dsize int

	switch sarflags.GetStr(s.header, "descriptor") { // Offset
	case "d16":
		dsize = 2
	case "d32":
		dsize = 4
	case "d64":
		dsize = 8
	default:
		return nil, errors.New("Invalid descriptor in Status frame")
	}
	framelen += (dsize*2 + (dsize * len(s.holes) * 2)) // progress + inrespto + holes
	if framelen > sarflags.MaxFrameSize {
		return nil, errors.New("Status - Maximum Frame Size Exceeded")
	}
	frame := make([]byte, framelen)

	binary.BigEndian.PutUint32(frame[:4], s.header)
	binary.BigEndian.PutUint32(frame[4:8], s.session)

	pos := 8
	if havetstamp {
		ts := s.tstamp.Put()
		copy(frame[pos:24], ts)
		pos = 24
	}
	switch dsize {
	case 2:
		binary.BigEndian.PutUint16(frame[pos:pos+dsize], uint16(s.progress))
		pos += dsize
		binary.BigEndian.PutUint16(frame[pos:pos+dsize], uint16(s.inrespto))
		pos += dsize
		for i := range s.holes {
			binary.BigEndian.PutUint16(frame[pos:pos+dsize], uint16(s.holes[i].Start))
			pos += dsize
			binary.BigEndian.PutUint16(frame[pos:pos+dsize], uint16(s.holes[i].End))
			pos += dsize
		}
	case 4:
		binary.BigEndian.PutUint32(frame[pos:pos+dsize], uint32(s.progress))
		pos += dsize
		binary.BigEndian.PutUint32(frame[pos:pos+4], uint32(s.inrespto))
		pos += dsize
		for i := range s.holes {
			binary.BigEndian.PutUint32(frame[pos:pos+dsize], uint32(s.holes[i].Start))
			pos += dsize
			binary.BigEndian.PutUint32(frame[pos:pos+dsize], uint32(s.holes[i].End))
			pos += dsize
		}
	case 8:
		binary.BigEndian.PutUint64(frame[pos:pos+dsize], uint64(s.progress))
		pos += dsize
		binary.BigEndian.PutUint64(frame[pos:pos+dsize], uint64(s.inrespto))
		pos += dsize
		for i := range s.holes {
			binary.BigEndian.PutUint64(frame[pos:pos+dsize], uint64(s.holes[i].Start))
			pos += dsize
			binary.BigEndian.PutUint64(frame[pos:pos+dsize], uint64(s.holes[i].End))
			pos += dsize
		}
	}
	return frame, nil
}

// Get -- Decode Data byte slice frame into Data struct
func (s *Status) Get(frame []byte) error {

	if len(frame) < 16 {
		return errors.New("Status Frame too short")
	}
	s.header = binary.BigEndian.Uint32(frame[:4])
	s.session = binary.BigEndian.Uint32(frame[4:8])
	pos := 8
	if sarflags.GetStr(s.header, "reqtstamp") == "yes" {
		var err error

		if err = s.tstamp.Get(frame[pos:24]); err != nil {
			return err
		}
		pos = 24
	}

	var dsize int

	switch sarflags.GetStr(s.header, "descriptor") {
	case "d16":
		dsize = 2
		s.progress = uint64(binary.BigEndian.Uint16(frame[pos : pos+dsize]))
		pos += dsize
		s.inrespto = uint64(binary.BigEndian.Uint16(frame[pos : pos+dsize]))
		pos += dsize
		hlen := len(frame[pos:])
		// log.Fatal("Holes in frame len", hlen, "number of holes", hlen/2/dsize)
		for i := 0; i < hlen/2/dsize; i++ {
			start := uint64(binary.BigEndian.Uint16(frame[pos : pos+dsize]))
			pos += dsize
			end := uint64(binary.BigEndian.Uint16(frame[pos : pos+dsize]))
			pos += dsize
			ah := Hole{Start: start, End: end}
			s.holes = append(s.holes, ah)
		}
	case "d32":
		dsize = 4
		s.progress = uint64(binary.BigEndian.Uint32(frame[pos : pos+dsize]))
		pos += dsize
		s.inrespto = uint64(binary.BigEndian.Uint32(frame[pos : pos+dsize]))
		pos += dsize
		hlen := len(frame[pos:])
		for i := 0; i < hlen/2/dsize; i++ {
			start := uint64(binary.BigEndian.Uint32(frame[pos : pos+dsize]))
			pos += dsize
			end := uint64(binary.BigEndian.Uint32(frame[pos : pos+dsize]))
			pos += dsize
			ah := Hole{Start: start, End: end}
			s.holes = append(s.holes, ah)
		}
	case "d64":
		dsize = 8
		s.progress = uint64(binary.BigEndian.Uint64(frame[pos : pos+dsize]))
		pos += dsize
		s.inrespto = uint64(binary.BigEndian.Uint64(frame[pos : pos+dsize]))
		pos += dsize
		hlen := len(frame[pos:])
		for i := 0; i < hlen/2/dsize; i++ {
			start := uint64(binary.BigEndian.Uint64(frame[pos : pos+dsize]))
			pos += dsize
			end := uint64(binary.BigEndian.Uint64(frame[pos : pos+dsize]))
			pos += dsize
			ah := Hole{Start: start, End: end}
			s.holes = append(s.holes, ah)
		}
	default:
		return errors.New("Invalid descriptor in Status frame")
	}
	return nil
}

// Print - Print out details of Beacon struct
func (s Status) Print() string {
	sflag := fmt.Sprintf("Status: 0x%x\n", s.header)
	sflags := sarflags.Values("status")
	for f := range sflags {
		n := sarflags.GetStr(s.header, sflags[f])
		sflag += fmt.Sprintf("  %s:%s\n", sflags[f], n)
	}
	sflag += fmt.Sprintf("  session:%d\n", s.session)
	if sarflags.GetStr(s.header, "reqtstamp") == "yes" {
		sflag += fmt.Sprintf("  timestamp:%s\n", s.tstamp.Print())
	}
	sflag += fmt.Sprintf("  progress:%d", s.progress)
	sflag += fmt.Sprintf("  inresponseto:%d\n", s.inrespto)
	for i := range s.holes {
		sflag += fmt.Sprintf("  Hole[%d]: Start:%d End:%d\n", i, s.holes[i].Start, s.holes[i].End)
	}
	return sflag
}
