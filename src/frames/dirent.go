package frames

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sarflags"
	"strings"
	"syscall"
	"time"
)

// DirEnt -- Directory Entry
type DirEnt struct {
	flags uint16
	size  uint64
	mtime Timestamp
	ctime Timestamp
	path  string
}

// StatFile -- Get file information
// The mtime & ctime values are y2k epoch formats
func StatFile(name string) (size uint64, e2kMtime Timestamp, e2kCtime Timestamp, err error) {
	fi, err := os.Stat(name)
	if err != nil {
		return size, e2kMtime, e2kCtime, err
	}

	size = uint64(fi.Size())
	mtime := fi.ModTime()

	// Special handling to get a ctime
	// THIS IS PLATFORM DEPENDANT!!!!!

	stat := fi.Sys().(*syscall.Stat_t)
	// atime = time.Unix(int64(stat.Atim.Sec), int64(stat.Atim.Nsec))
	ctime := time.Unix(int64(stat.Ctimespec.Sec), int64(stat.Ctimespec.Nsec))

	var tserr error
	var tbuf []byte

	// Mtime
	if tbuf, tserr = TimeStampMake("epoch2000_32", mtime); tserr != nil {
		return size, e2kMtime, e2kCtime, tserr
	}
	if e2kMtime, tserr = TimeStampGet(tbuf); tserr != nil {
		return size, e2kMtime, e2kCtime, tserr
	}

	// Ctime
	if tbuf, tserr = TimeStampMake("epoch2000_32", ctime); tserr != nil {
		return size, e2kMtime, e2kCtime, tserr
	}
	if e2kCtime, tserr = TimeStampGet(tbuf); tserr != nil {
		return size, e2kMtime, e2kCtime, tserr
	}
	return size, e2kMtime, e2kCtime, nil
}

// DirEntMake - Construct a directory entry return byte slice of dirent
// Flags is of format "flagname1=flagval1,flagname2=flagval2...
// It looks up the local file systems path to get mtime & ctime
func DirEntMake(flags string, path string) ([]byte, error) {

	var header uint16
	var err error

	flags = strings.Replace(flags, " ", "", -1) // Get rid of extra spaces in flags

	if header, err = sarflags.SetD(header, "sod", "sod"); err != nil {
		return nil, err
	}

	// Grab the flags and set the frame header
	flag := strings.Split(flags, ",") // The name=val of the flag
	for fl := range flag {
		f := strings.Split(flag[fl], "=") // f[0]=name f[1]=val
		switch f[0] {
		case "descriptor", "property", "reliability":
			if header, err = sarflags.SetD(header, f[0], f[1]); err != nil {
				return nil, err
			}
		default:
			e := "Invalid Flag " + f[0] + " for Directory Entry"
			return nil, errors.New(e)
		}
	}

	// Create the dirent slice
	framelen := 2 + 4 + 4 // Header + Mtime + Ctime

	var dsize int

	// We need to work out the File size in order to set the actual descriptor size needed
	// We will return an error if it can't fit in what has been set here
	switch sarflags.GetDStr(header, "descriptor") { // Offset
	case "d16":
		dsize = 2
	case "d32":
		dsize = 4
	case "d64":
		dsize = 8
	default:
		return nil, errors.New("Invalid descriptor in MetaData frame")
	}
	framelen += dsize
	framelen += len(path) + 1 // Add in the required NULL at the end of path
	if len(path) > 1024 {
		return nil, errors.New("Path name exceeds 1024 bytes")
	}

	var mtime, ctime Timestamp
	var size uint64

	prop := sarflags.GetDStr(header, "property")
	switch prop {
	case "normalfile", "normaldirecory":
		// Careful the mtime & ctime values are seconds elapsed since Y2K epoch NOT Unix time!!!!
		if size, mtime, ctime, err = StatFile(path); err != nil {
			return nil, err
		}
	case "specialfile": // Named Pipe so is a stream, lets set directroy entries to now
		size = 0
		b, _ := TimeStampNow("epoch2000_32")
		mtime, _ = TimeStampGet(b)
		ctime = mtime
	default:
		e := "Invalid Property:" + prop
		return nil, errors.New(e)
	}

	frame := make([]byte, framelen)

	binary.BigEndian.PutUint16(frame[:2], header)

	// fmt.Printf("Frmaelen:%d Header:%d Size:%d Mtime:%s Ctime:%s\n", framelen, header, size,
	//	TimeStampPrint(mtime), TimeStampPrint(ctime))

	pos := 2
	switch dsize {
	case 2:
		if size > sarflags.MaxUint16 {
			e := fmt.Sprintf("Descriptor d16 too small for file size %d", dsize)
			return nil, errors.New(e)
		}
		binary.BigEndian.PutUint16(frame[pos:pos+dsize], uint16(size))
		pos += dsize

	case 4:
		if size > sarflags.MaxUint32 {
			e := fmt.Sprintf("Descriptor d32 too small for file size %d", dsize)
			return nil, errors.New(e)
		}
		binary.BigEndian.PutUint32(frame[pos:pos+dsize], uint32(size))
		pos += dsize
	case 8:
		if size > sarflags.MaxUint32 { // This should never happen!!!!
			e := fmt.Sprintf("Descriptor d64 too small for file size %d", dsize)
			return nil, errors.New(e)
		}
		binary.BigEndian.PutUint64(frame[pos:pos+dsize], uint64(size))
		pos += dsize
	default:
		e := fmt.Sprintf("Malformed Directory Entry Invalid descriptor size %d", dsize)
		return nil, errors.New(e)
	}
	// fmt.Printf("Pos is: %d\n", pos)
	binary.BigEndian.PutUint32(frame[pos:pos+4], uint32(mtime.secs))
	pos += 4
	binary.BigEndian.PutUint32(frame[pos:pos+4], uint32(ctime.secs))
	pos += 4
	endpos := pos + len(path)
	copy(frame[pos:endpos], path)
	copy(frame[endpos:], "\x00") // Add in the Null at the end

	// fmt.Printf("End Pos is: %d\n", endpos)
	// fmt.Printf("Frame Len: %d Frame:%x\n", len(frame), frame)
	return frame, nil
}

// DirEntGet -- Decode Directory Entry byte slice frame into DirEnt struct
func DirEntGet(frame []byte) (DirEnt, error) {
	var d DirEnt

	if len(frame) < 12 {
		return d, errors.New("DirEntGet - Entry too short")
	}
	d.flags = binary.BigEndian.Uint16(frame[:2])

	pos := 2
	switch sarflags.GetDStr(d.flags, "descriptor") {
	case "d16":
		dsize := 2
		d.size = uint64(binary.BigEndian.Uint16(frame[pos : pos+dsize]))
		pos += dsize
	case "d32":
		dsize := 4
		d.size = uint64(binary.BigEndian.Uint32(frame[pos : pos+dsize]))
		pos += dsize
	case "d64":
		dsize := 8
		d.size = uint64(binary.BigEndian.Uint64(frame[pos : pos+dsize]))
		pos += dsize
	default:
		return d, errors.New("Invalid MetaData Frame")
	}
	// fmt.Printf("Header:%d Size:%d\n", d.flags, d.size)

	var terr error

	var ts []byte
	ts = make([]byte, 5)
	ts[0], _ = sarflags.SetT(0, "epoch2000_32")

	copy(ts[1:], frame[pos:pos+4])
	if d.mtime, terr = TimeStampGet(ts); terr != nil {
		return d, errors.New("Invalid Mtime")
	}
	pos += 4
	// fmt.Printf("Mtime:%s ", TimeStampPrint(d.mtime))

	copy(ts[1:], frame[pos:pos+4])
	if d.ctime, terr = TimeStampGet(ts); terr != nil {
		return d, errors.New("Invalid Ctime")
	}
	pos += 4
	// fmt.Printf("Ctime:%s\n", TimeStampPrint(d.ctime))

	for i := range frame[pos:] { // Path is null terminated string
		if frame[pos+i] == '\x00' { // Hit null
			break
		}
		d.path += string(frame[pos+i])
	}
	// fmt.Printf("Path:<%s>\n", d.path)
	return d, nil
}

// DirEntPrint - Print out details of Beacon struct
func DirEntPrint(d DirEnt) string {
	sflag := fmt.Sprintf("  Directory Entry: 0x%x\n", d.flags)
	dflags := sarflags.FlagD()
	// sarflags.FrameD("dirent")
	for f := range dflags {
		n := sarflags.GetDStr(d.flags, dflags[f])
		sflag += fmt.Sprintf("    %s:%s\n", dflags[f], n)
	}
	sflag += fmt.Sprintf("    Filename:%s\n", d.path)
	sflag += fmt.Sprintf("    Size:%d\n", d.size)
	sflag += fmt.Sprintf("    Mtime:%s\n", TimeStampPrint(d.mtime))
	sflag += fmt.Sprintf("    Ctime:%s", TimeStampPrint(d.ctime))
	return sflag
}
