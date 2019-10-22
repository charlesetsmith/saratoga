package frames

import (
	"encoding/binary"
	"errors"
	"strings"
	"time"

	"sarflags"
)

// Timestamp -- Holds Beacon tstamp information
type Timestamp struct {
	flags uint8
	secs  uint64
	nsecs uint64
	local []byte
}

// TimeStampMake - Construct a timestamp - return byte slice of time
// Flags is of format "flagval"
func TimeStampMake(flag string, t time.Time) ([]byte, error) {

	var header uint8
	var err error

	flag = strings.Replace(flag, " ", "", -1) // Get rid of extra spaces in flags

	tstamp := make([]byte, 16)

	switch flag {
	case "localinterp":
		if header, err = sarflags.SetT(header, flag); err != nil {
			return nil, err
		}
		tstamp[0] = byte(header)
		copy(tstamp[1:], "LOCALINTERPXXXX")
	case "posix32":
		if header, err = sarflags.SetT(header, flag); err != nil {
			return nil, err
		}
		tstamp[0] = byte(header)
		secs := uint32(t.Unix())
		if secs > sarflags.MaxUint32 {
			return nil, errors.New("posix32:Seconds exceed 32 bits")
		}
		binary.BigEndian.PutUint32(tstamp[1:5], uint32(secs))
		copy(tstamp[5:], "")
	case "posix64":
		if header, err = sarflags.SetT(header, flag); err != nil {
			return nil, err
		}
		tstamp[0] = byte(header)
		secs := t.Unix()
		binary.BigEndian.PutUint64(tstamp[1:9], uint64(secs))
		copy(tstamp[9:], "")
	case "posix32_32":
		if header, err = sarflags.SetT(header, flag); err != nil {
			return nil, err
		}
		tstamp[0] = byte(header)
		nsecs := t.UnixNano()
		secs := nsecs / 1e9
		rem := nsecs - secs*1e9
		if secs > sarflags.MaxUint32 {
			return nil, errors.New("posix32_32:Seconds exceed 32 bits")
		}
		if rem > sarflags.MaxUint32 {
			return nil, errors.New("posix32_32:Remainder exceed 32 bits")
		}
		binary.BigEndian.PutUint32(tstamp[1:5], uint32(secs))
		binary.BigEndian.PutUint32(tstamp[5:9], uint32(rem))
		copy(tstamp[9:], "")
	case "posix64_32":
		if header, err = sarflags.SetT(header, flag); err != nil {
			return nil, err
		}
		tstamp[0] = byte(header)
		nsecs := t.UnixNano()
		secs := nsecs / 1e9
		rem := uint32(nsecs - secs*1e9)
		if rem > sarflags.MaxUint32 {
			return nil, errors.New("posix64_32:Remainder exceed 32 bits")
		}
		binary.BigEndian.PutUint64(tstamp[1:9], uint64(secs))
		binary.BigEndian.PutUint32(tstamp[9:13], uint32(rem))
		copy(tstamp[13:], "")
	case "epoch2000_32":
		if header, err = sarflags.SetT(header, flag); err != nil {
			return nil, err
		}
		tstamp[0] = byte(header)
		epoch2000, _ := time.Parse(time.RFC3339, "2000-01-01T00:00:00Z")
		secs := t.Unix()
		secs -= epoch2000.Unix()
		if secs < 0 || secs > sarflags.MaxUint32 {
			return nil, errors.New("epoch2000_32:Seconds out of bounds")
		}
		binary.BigEndian.PutUint32(tstamp[1:5], uint32(secs))
		copy(tstamp[5:], "")
	default: // Dont know this timestamp type
		if header, err = sarflags.SetT(header, flag); err != nil {
			return nil, err
		}
	}
	return tstamp, nil
}

// TimeStampNow -- Create a timestamp of the current time
func TimeStampNow(flag string) (ts []byte, err error) {
	if ts, err = TimeStampMake(flag, time.Now()); err != nil {
		e := "TimeStampNow invalid flag:" + flag
		err = errors.New(e)
	}
	return ts, err
}

// TimeStampGet -- Get  timestamp from buffer
func TimeStampGet(tstamp []byte) (Timestamp, error) {
	var t Timestamp

	t.flags = tstamp[0]
	switch sarflags.GetTStr(t.flags) {
	case "localinterp":
		copy(t.local, tstamp[1:])
		t.secs = 0
		t.nsecs = 0
	case "posix32":
		t.secs = uint64(binary.BigEndian.Uint32(tstamp[1:5]))
		t.nsecs = 0
	case "posix64":
		t.secs = uint64(binary.BigEndian.Uint64(tstamp[1:9]))
		t.nsecs = 0
	case "posix32_32":
		t.secs = uint64(binary.BigEndian.Uint32(tstamp[1:5]))
		t.nsecs = uint64(binary.BigEndian.Uint32(tstamp[5:9]))
	case "posix64_32":
		t.secs = uint64(binary.BigEndian.Uint64(tstamp[1:9]))
		t.nsecs = uint64(binary.BigEndian.Uint32(tstamp[9:13]))
	case "epoch2000_32":
		t.secs = uint64(binary.BigEndian.Uint32(tstamp[1:5]))
		t.nsecs = 0
	default:
		t.secs = 0
		t.nsecs = 0
		copy(t.local, tstamp[1:])
		return t, errors.New("timestamp.Get: Invalid Timestamp")
	}
	copy(t.local, tstamp[1:])
	return t, nil
}

// TimeStampPrint - Print out the UTC
func TimeStampPrint(t Timestamp) string {
	switch sarflags.GetTStr(t.flags) {
	case "posix32", "posix64":
		ti := time.Unix(int64(t.secs), int64(t.nsecs))
		ti = ti.UTC()
		return ti.Format("Mon Jan _2 15:05:05 2006 UTC")
	case "posix32_32", "posix64_32":
		ti := time.Unix(int64(t.secs), int64(t.nsecs))
		ti = ti.UTC()
		return ti.Format("Mon Jan _2 15:05:05.000 2006 UTC")
	case "epoch2000_32":
		epoch2000, _ := time.Parse(time.RFC3339, "2000-01-01T00:00:00Z")
		t.secs += uint64(epoch2000.Unix())
		ti := time.Unix(int64(t.secs), int64(t.nsecs))
		ti = ti.UTC()
		return ti.Format("Mon Jan _2 15:05:05 2006 UTC")
	default:
		return "LOCALINTERP"
	}
}
