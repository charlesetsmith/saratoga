// Timestamp Handlers

package timestamp

import (
	"encoding/binary"
	"errors"
	"strings"
	"time"

	"github.com/charlesetsmith/saratoga/sarflags"
)

// Timestamp -- Holds Beacon tstamp information
type Timestamp struct {
	header uint8
	secs   uint64
	nsecs  uint64
	local  string // If we dont have a defined timestamp type then just return what is in flag string
	ttype  string // What are we posix32,posix64...
}

// New - Construct a timestamp - return byte slice of time
// flag is string of format "xxxxx"
func (t *Timestamp) New(flag string, ts time.Time) error {

	var header uint8
	var err error

	flag = strings.Replace(flag, " ", "", -1) // Get rid of extra spaces in flags

	if header, err = sarflags.SetT(header, flag); err != nil {
		return err
	}
	t.header = header

	switch flag {
	case "posix32":
		secs := ts.Unix()
		t.secs = uint64(secs)
		t.ttype = flag
		if t.secs > sarflags.MaxUint32 {
			return errors.New("posix32:Seconds exceed 32 bits")
		}
		t.nsecs = 0
		t.local = ""
	case "posix64":
		secs := ts.Unix()
		t.secs = uint64(secs)
		t.ttype = flag
		t.nsecs = 0
		t.local = ""
	case "posix32_32":
		nsecs := ts.UnixNano()
		t.secs = uint64(nsecs / 1e9)
		t.ttype = flag
		t.nsecs = uint64(nsecs % 1e9)
		if t.secs > sarflags.MaxUint32 {
			return errors.New("posix32_32:Seconds exceed 32 bits")
		}
		t.local = ""
	case "posix64_32":
		nsecs := ts.UnixNano()
		t.secs = uint64(nsecs / 1e9)
		t.ttype = flag
		t.nsecs = uint64(nsecs % 1e9)
		if t.nsecs > sarflags.MaxUint32 {
			return errors.New("posix64_32:Remainder exceed 32 bits")
		}
		t.local = ""
	case "epoch2000_32":
		epoch2k, _ := time.Parse(time.RFC3339, "2000-01-01T00:00:00Z")
		secs := ts.Unix()
		secs -= epoch2k.Unix()
		t.ttype = flag
		t.secs = uint64(secs)
		if t.secs > sarflags.MaxUint32 {
			return errors.New("epoch2000_32:Seconds out of bounds")
		}
		t.nsecs = 0
		t.local = ""
	default: // localinterp Dont know this timestamp type so whatever is in flag is used
		t.local = flag
		t.ttype = flag
	}
	return nil
}

// Put - Return byte sequence in saratoga format ready to transmit
func (t Timestamp) Put() []byte {

	tstamp := make([]byte, 16)

	tstamp[0] = byte(t.header)
	switch sarflags.NameT(t.header) {
	case "posix32":
		binary.BigEndian.PutUint32(tstamp[1:5], uint32(t.secs))
		copy(tstamp[5:], "")
	case "posix64":
		binary.BigEndian.PutUint64(tstamp[1:9], uint64(t.secs))
		copy(tstamp[9:], "")
	case "posix32_32":
		binary.BigEndian.PutUint32(tstamp[1:5], uint32(t.secs))
		binary.BigEndian.PutUint32(tstamp[5:9], uint32(t.nsecs))
		copy(tstamp[9:], "")
	case "posix64_32":
		binary.BigEndian.PutUint64(tstamp[1:9], uint64(t.secs))
		binary.BigEndian.PutUint32(tstamp[9:13], uint32(t.nsecs))
		copy(tstamp[13:], "")
	case "epoch2000_32":
		binary.BigEndian.PutUint32(tstamp[1:5], uint32(t.secs))
		copy(tstamp[5:], "")
	default: // Dont know this timestamp type so is local but only first 15 bytes
		copy(tstamp[1:15], t.local)
	}
	return tstamp
}

// Now -- Create a timestamp of the current time
func (t *Timestamp) Now(flag string) error {

	if err := t.New(flag, time.Now()); err != nil {
		e := "TimeStampNow invalid flag:" + flag
		return errors.New(e)
	}
	return nil
}

// Get -- Get  timestamp from buffer
func (t *Timestamp) Get(tstamp []byte) error {

	t.header = tstamp[0]
	ttype := sarflags.GetTStr(t.header)
	switch ttype {
	case "posix32":
		t.secs = uint64(binary.BigEndian.Uint32(tstamp[1:5]))
		t.nsecs = 0
		t.local = ""
		t.ttype = ttype
	case "posix64":
		t.secs = uint64(binary.BigEndian.Uint64(tstamp[1:9]))
		t.nsecs = 0
		t.local = ""
		t.ttype = ttype
	case "posix32_32":
		t.secs = uint64(binary.BigEndian.Uint32(tstamp[1:5]))
		t.nsecs = uint64(binary.BigEndian.Uint32(tstamp[5:9]))
		t.local = ""
		t.ttype = ttype
	case "posix64_32":
		t.secs = uint64(binary.BigEndian.Uint64(tstamp[1:9]))
		t.nsecs = uint64(binary.BigEndian.Uint32(tstamp[9:13]))
		t.local = ""
		t.ttype = ttype
	case "epoch2000_32":
		t.secs = uint64(binary.BigEndian.Uint32(tstamp[1:5]))
		t.nsecs = 0
		t.local = ""
		t.ttype = ttype
	default:
		t.secs = 0
		t.nsecs = 0
		t.local = string(tstamp[1:15])
		t.ttype = ttype
		return errors.New("timestamp.Get: Invalid Timestamp")
	}
	return nil
}

// Secs -- How Many whole seconds in the timestamp
func (t Timestamp) Secs() uint64 {
	return t.secs
}

// Print - Print out the UTC
func (t Timestamp) Print() string {
	switch sarflags.GetTStr(t.header) {
	case "posix32", "posix64":
		ti := time.Unix(int64(t.secs), int64(t.nsecs))
		ti = ti.UTC()
		return ti.Format("Mon Jan _2 15:04:05 2006 UTC")
	case "posix32_32", "posix64_32":
		ti := time.Unix(int64(t.secs), int64(t.nsecs))
		ti = ti.UTC()
		return ti.Format("Mon Jan _2 15:04:05.000 2006 UTC")
	case "epoch2000_32":
		epoch2k, _ := time.Parse(time.RFC3339, "2000-01-01T00:00:00Z")
		ut := uint64(epoch2k.Unix())
		ut += t.secs
		ti := time.Unix(int64(ut), int64(t.nsecs))
		ti = ti.UTC()
		return ti.Format("Mon Jan _2 15:04:05 2006 UTC")
	default:
		return t.local
	}
}
