package dirent

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sarflags"
	"strings"

	"github.com/charlesetsmith/saratoga/src/sys"
	"github.com/charlesetsmith/saratoga/src/timestamp"
)

// DirEnt -- Directory Entry
type DirEnt struct {
	header uint16
	size   uint64
	mtime  timestamp.Timestamp
	ctime  timestamp.Timestamp
	path   string
}

// StatFile -- Get file information - size, mtime, ctime
// The mtime & ctime values are y2k epoch formats
func (d *DirEnt) StatFile(name string) (err error) {

	// Stat the file/directory name
	fi, err := os.Stat(name)
	if err != nil {
		return err
	}

	d.size = uint64(fi.Size())

	/*
		// Mtime
		if err := d.mtime.New("epoch2000_32", fi.ModTime()); err != nil {
			return err
		}
		// Ctime
		// Special handling to get a ctime
		// THIS IS PLATFORM DEPENDENT!!!!!
		stat := fi.Sys().(*syscall.Stat_t)
		// atime = time.Unix(int64(stat.Atim.Sec), int64(stat.Atim.Nsec))
		ctime := time.Unix(int64(stat.Ctimespec.Sec), int64(stat.Ctimespec.Nsec))
		if err := d.ctime.New("epoch2000_32", ctime); err != nil {
			return err
		}
	*/

	var ft sys.FileTime
	ft.NewTime(fi)

	// Mtime
	if err := d.mtime.New("epoch2000_32", ft.Mtime); err != nil {
		return err
	}
	// Ctime
	if err := d.ctime.New("epoch2000_32", ft.Ctime); err != nil {
		return err
	}
	return nil

	/*
		// Ctime
		// Special handling to get a ctime
		// THIS IS PLATFORM DEPENDENT!!!!!
		stat := fi.Sys().(*syscall.Stat_t)
		// atime = time.Unix(int64(stat.Atim.Sec), int64(stat.Atim.Nsec))
		ctime := time.Unix(int64(stat.Ctimespec.Sec), int64(stat.Ctimespec.Nsec))
		if err := d.ctime.New("epoch2000_32", ctime); err != nil {
			return err
		}
	*/
}

// New - Construct a directory entry return byte slice of dirent
// Flags is of format "flagname1=flagval1,flagname2=flagval2...
// It looks up the local file systems path to get mtime & ctime
func (d *DirEnt) New(flags string, path string) error {

	var err error

	flags = strings.Replace(flags, " ", "", -1) // Get rid of extra spaces in flags

	if d.header, err = sarflags.SetD(d.header, "sod", "sod"); err != nil {
		return err
	}

	// Grab the flags and set the frame header
	flag := strings.Split(flags, ",") // The name=val of the flag
	for fl := range flag {
		f := strings.Split(flag[fl], "=") // f[0]=name f[1]=val
		switch f[0] {
		case "descriptor", "property", "reliability":
			if d.header, err = sarflags.SetD(d.header, f[0], f[1]); err != nil {
				return err
			}
		default:
			e := "Invalid Flag " + f[0] + " for Directory Entry"
			return errors.New(e)
		}
	}

	if len(path) > 1024 {
		return errors.New("Path name exceeds 1024 bytes")
	}

	prop := sarflags.GetDStr(d.header, "property")
	switch prop {
	case "normalfile", "normaldirecory":
		// Careful the mtime & ctime values are seconds elapsed since Y2K epoch NOT Unix time!!!!
		// We get th d.size, d.mtime & d.ctime from the stat
		if err := d.StatFile(path); err != nil {
			return err
		}
	case "specialfile": // Named Pipe so is a stream, lets set directroy entries to now
		// We set size to 0, d.ctime & d.mtime to now
		d.size = 0
		if err := d.mtime.Now("epoch2000_32"); err != nil {
			return err
		}
		d.ctime = d.mtime
	default:
		e := "Invalid Property:" + prop
		return errors.New(e)
	}

	d.path = path
	return nil
}

// Put - Encode the Saratoga directory entry
func (d DirEnt) Put() ([]byte, error) {

	// Create the dirent slice
	framelen := 2 + 4 + 4 // Header + Mtime + Ctime

	var dsize int

	// We need to work out the File size in order to set the actual descriptor size needed
	// We will return an error if it can't fit in what has been set here
	switch sarflags.GetDStr(d.header, "descriptor") { // Offset
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
	framelen += len(d.path) + 1 // Add in the required NULL at the end of path
	if len(d.path) > 1024 {
		return nil, errors.New("Path name exceeds 1024 bytes")
	}

	frame := make([]byte, framelen)

	binary.BigEndian.PutUint16(frame[:2], d.header)

	pos := 2
	switch dsize {
	case 2:
		if d.size > sarflags.MaxUint16 {
			e := fmt.Sprintf("Descriptor d16 too small for file size %d", d.size)
			return nil, errors.New(e)
		}
		binary.BigEndian.PutUint16(frame[pos:pos+dsize], uint16(d.size))
	case 4:
		if d.size > sarflags.MaxUint32 {
			e := fmt.Sprintf("Descriptor d32 too small for file size %d", d.size)
			return nil, errors.New(e)
		}
		binary.BigEndian.PutUint32(frame[pos:pos+dsize], uint32(d.size))
	case 8:
		if d.size > sarflags.MaxUint32 { // This should never happen!!!!
			e := fmt.Sprintf("Descriptor d64 too small for file size %d", d.size)
			return nil, errors.New(e)
		}
		binary.BigEndian.PutUint64(frame[pos:pos+dsize], uint64(d.size))
	default:
		e := fmt.Sprintf("Malformed Directory Entry Invalid descriptor size %d", d.size)
		return nil, errors.New(e)
	}
	pos += dsize

	binary.BigEndian.PutUint32(frame[pos:pos+4], uint32(d.mtime.Secs()))
	pos += 4
	binary.BigEndian.PutUint32(frame[pos:pos+4], uint32(d.ctime.Secs()))
	pos += 4
	endpos := pos + len(d.path)
	copy(frame[pos:endpos], d.path)
	copy(frame[endpos:], "\x00") // Add in the Null at the end
	return frame, nil
}

// Get -- Decode Directory Entry byte slice frame into DirEnt struct
func (d *DirEnt) Get(frame []byte) error {

	if len(frame) < 12 {
		return errors.New("DirEntGet - Entry too short")
	}
	d.header = binary.BigEndian.Uint16(frame[:2])

	pos := 2
	switch sarflags.GetDStr(d.header, "descriptor") {
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
		return errors.New("Invalid MetaData Frame")
	}
	// fmt.Printf("Header:%d Size:%d\n", d.header, d.size)

	var ts []byte
	ts = make([]byte, 5)
	ts[0], _ = sarflags.SetT(0, "epoch2000_32")

	copy(ts[1:], frame[pos:pos+4])
	if terr := d.mtime.Get(ts); terr != nil {
		return errors.New("Invalid Mtime")
	}
	pos += 4
	// fmt.Printf("Mtime:%s ", d.mtime.Print())

	copy(ts[1:], frame[pos:pos+4])
	if terr := d.ctime.Get(ts); terr != nil {
		return errors.New("Invalid Ctime")
	}
	pos += 4
	// fmt.Printf("Ctime:%s\n", d.ctime.Print())

	d.path = ""
	for i := range frame[pos:] { // Path is null terminated string
		if frame[pos+i] == '\x00' { // Hit null
			break
		}
		d.path += string(frame[pos+i])
	}
	// fmt.Printf("Path:<%s>\n", d.path)
	return nil
}

// Print - Print out details of Beacon struct
func (d DirEnt) Print() string {
	sflag := fmt.Sprintf("  Directory Entry: 0x%x\n", d.header)
	dflags := sarflags.FlagD()
	// sarflags.FrameD("dirent")
	for _, f := range dflags {
		n := sarflags.GetDStr(d.header, f)
		sflag += fmt.Sprintf("    %s:%s\n", f, n)
	}
	sflag += fmt.Sprintf("    Path:%s\n", d.path)
	sflag += fmt.Sprintf("    Size:%d\n", d.size)
	sflag += fmt.Sprintf("    Mtime:%s\n", d.mtime.Print())
	sflag += fmt.Sprintf("    Ctime:%s", d.ctime.Print())
	return sflag
}
