package frames

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"sarflags"
)

// Hole -- Beggining and End of a hole
type Hole struct {
	Start uint64
	End   uint64
}

// Status -- Status of the transfer and holes
type Status struct {
	flags    uint32
	session  uint32
	tstamp   Timestamp
	progress uint64
	inrespto uint64
	holes    []Hole
}

// StatusMake - Construct a Status frame - return byte slice of frame
// Flags is of format "flagname1=flagval1,flagname2=flagval2...
// The timestamp type to use is also in the flags as "timestamp=flagval"
func StatusMake(flags string, session uint32, progress uint64, inrespto uint64, holes []Hole) ([]byte, error) {

	var header uint32
	var err error
	var havetstamp = false
	var tstamp []byte

	flags = strings.Replace(flags, " ", "", -1) // Get rid of extra spaces in flags

	if header, err = sarflags.Set(header, "version", "v1"); err != nil {
		return nil, err
	}

	if header, err = sarflags.Set(header, "frametype", "status"); err != nil {
		return nil, err
	}

	// Grab the flags and set the frame header
	flag := strings.Split(flags, ",") // The name=val of the flag
	for fl := range flag {
		f := strings.Split(flag[fl], "=") // f[0]=name f[1]=val
		switch f[0] {
		case "descriptor", "stream", "reqtstamp", "metadatarecvd", "allholes", "reqholes", "errcode":
			if header, err = sarflags.Set(header, f[0], f[1]); err != nil {
				return nil, err
			}
		case "localinterp", "posix32", "posix64", "posix32_32", "posix64_32":
			if tstamp, err = TimeStampNow(f[0]); err != nil {
				return nil, err
			}
			havetstamp = true
		default:
			e := "Invalid Flag " + f[0] + " for Status Frame"
			return nil, errors.New(e)
		}
	}

	// Create the frame slice
	framelen := 4 + 4 // Header + Session

	if sarflags.GetStr(header, "reqtstamp") == "yes" {
		if havetstamp == false {
			return nil, errors.New("Status Timestamps Requested but type not specified")
		}
		framelen += 16 // Timestamp
	}

	var dsize int

	switch sarflags.GetStr(header, "descriptor") { // Offset
	case "d16":
		dsize = 2
	case "d32":
		dsize = 4
	case "d64":
		dsize = 8
	default:
		return nil, errors.New("Invalid descriptor in Status frame")
	}
	framelen += (dsize*2 + (dsize * len(holes) * 2)) // progress + inrespto + holes
	if framelen > sarflags.MaxFrameSize {
		return nil, errors.New("Status - Maximum Frame Size Exceeded")
	}
	frame := make([]byte, framelen)

	binary.BigEndian.PutUint32(frame[:4], header)
	binary.BigEndian.PutUint32(frame[4:8], session)

	pos := 8
	if havetstamp == true {
		copy(frame[pos:24], tstamp[:16])
		pos = 24
	}
	switch dsize {
	case 2:
		binary.BigEndian.PutUint16(frame[pos:pos+dsize], uint16(progress))
		pos += dsize
		binary.BigEndian.PutUint16(frame[pos:pos+dsize], uint16(inrespto))
		pos += dsize
		for i := range holes {
			binary.BigEndian.PutUint16(frame[pos:pos+dsize], uint16(holes[i].Start))
			pos += dsize
			binary.BigEndian.PutUint16(frame[pos:pos+dsize], uint16(holes[i].End))
			pos += dsize
		}
	case 4:
		binary.BigEndian.PutUint32(frame[pos:pos+dsize], uint32(progress))
		pos += dsize
		binary.BigEndian.PutUint32(frame[pos:pos+4], uint32(inrespto))
		pos += dsize
		for i := range holes {
			binary.BigEndian.PutUint32(frame[pos:pos+dsize], uint32(holes[i].Start))
			pos += dsize
			binary.BigEndian.PutUint32(frame[pos:pos+dsize], uint32(holes[i].End))
			pos += dsize
		}
	case 8:
		binary.BigEndian.PutUint64(frame[pos:pos+dsize], uint64(progress))
		pos += dsize
		binary.BigEndian.PutUint64(frame[pos:pos+dsize], uint64(inrespto))
		pos += dsize
		for i := range holes {
			binary.BigEndian.PutUint64(frame[pos:pos+dsize], uint64(holes[i].Start))
			pos += dsize
			binary.BigEndian.PutUint64(frame[pos:pos+dsize], uint64(holes[i].End))
			pos += dsize
		}
	}
	return frame, nil
}

// StatusGet -- Decode Data byte slice frame into Data struct
func StatusGet(frame []byte) (Status, error) {
	var s Status

	if len(frame) < 16 {
		return s, errors.New("Status Frame too short")
	}
	s.flags = binary.BigEndian.Uint32(frame[:4])
	s.session = binary.BigEndian.Uint32(frame[4:8])
	pos := 8
	if sarflags.GetStr(s.flags, "reqtstamp") == "yes" {
		var ts Timestamp
		var err error
		if ts, err = TimeStampGet(frame[pos:24]); err != nil {
			return s, err
		}
		s.tstamp = ts
		pos = 24
	}

	var dsize int

	switch sarflags.GetStr(s.flags, "descriptor") {
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
		return s, errors.New("Invalid descriptor in Status frame")
	}
	return s, nil
}

// StatusPrint - Print out details of Beacon struct
func StatusPrint(s Status) string {
	sflag := fmt.Sprintf("Status: 0x%x\n", s.flags)
	sflags := sarflags.Frame("status")
	for f := range sflags {
		n := sarflags.GetStr(s.flags, sflags[f])
		sflag += fmt.Sprintf("  %s:%s\n", sflags[f], n)
	}
	sflag += fmt.Sprintf("  session:%d\n", s.session)
	if sarflags.GetStr(s.flags, "reqtstamp") == "yes" {
		sflag += fmt.Sprintf("  timestamp:%s\n", TimeStampPrint(s.tstamp))
	}
	sflag += fmt.Sprintf("  progress:%d", s.progress)
	sflag += fmt.Sprintf("  inresponseto:%d\n", s.inrespto)
	for i := range s.holes {
		sflag += fmt.Sprintf("  Hole[%d]: Start:%d End:%d\n", i, s.holes[i].Start, s.holes[i].End)
	}
	return sflag
}
