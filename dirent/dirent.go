// Handling Directory Entry Structure for Saratoga

package dirent

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/charlesetsmith/saratoga/fileio"
	"github.com/charlesetsmith/saratoga/sarflags"
	"github.com/charlesetsmith/saratoga/sarsys"
	"github.com/charlesetsmith/saratoga/timestamp"
)

// DirEnt -- Directory Entry
type DirEnt struct {
	Header uint16
	Size   uint64
	Mtime  timestamp.Timestamp
	Ctime  timestamp.Timestamp
	Path   string
}

// Copy Directory Entry information
func (from *DirEnt) Copy() (to *DirEnt) {
	to = new(DirEnt)
	to.Header = from.Header
	to.Size = from.Size
	to.Mtime = from.Mtime
	to.Ctime = from.Ctime
	to.Path = from.Path
	return to
}

func (d *DirEnt) Value() DirEnt {
	return DirEnt{Header: d.Header, Size: d.Size, Mtime: d.Mtime, Ctime: d.Ctime, Path: d.Path}
}

// StatFile -- Get file information - size, mtime, ctime
// The mtime & ctime values are y2k epoch formats
func (d *DirEnt) Statfile(name string) error {

	var fi *fileio.FileMetaData
	var err error

	// Stat the file/directory name
	if fi, err = fileio.FileMeta(name); err != nil {
		// fi, err := os.Stat(name)
		return err
	}

	d.Size = uint64(fi.Size)

	var ft sarsys.FileTime
	ft.NewTime(fi.Info)

	// Mtime
	if err = d.Mtime.New("epoch2000_32", ft.Mtime); err != nil {
		return err
	}
	// Ctime
	if err = d.Ctime.New("epoch2000_32", ft.Ctime); err != nil {
		return err
	}
	return nil
}

// New - Construct a directory entry return byte slice of dirent
// Flags is of format "flagname1=flagval1,flagname2=flagval2...
// property - normalfile, normaldirectory, specialfile, specialdirectory
// descriptor - d16, d32, d64, d128
// It looks up the local file systems path to get mtime & ctime
func (d *DirEnt) New(flags string, path string) error {

	var err error

	flags = strings.Replace(flags, " ", "", -1) // Get rid of extra spaces in flags

	if d.Header, err = sarflags.SetD(d.Header, "sod", "sod"); err != nil {
		return err
	}

	// Grab the flags and set the frame header
	flag := strings.Split(flags, ",") // The name=val of the flag
	for fl := range flag {
		f := strings.Split(flag[fl], "=") // f[0]=name f[1]=val
		switch f[0] {
		case "descriptor", "property":
			if d.Header, err = sarflags.SetD(d.Header, f[0], f[1]); err != nil {
				return err
			}
		default:
			e := "Invalid Flag " + f[0] + " for Directory Entry"
			return errors.New(e)
		}
	}

	if len(path) > 1024 {
		return errors.New("path name exceeds 1024 bytes")
	}

	prop := sarflags.GetDStr(d.Header, "property")
	switch prop {
	case "normalfile", "normaldirectory":
		// Careful the mtime & ctime values are seconds elapsed since Y2K epoch NOT Unix time!!!!
		// We get the d.size, d.mtime & d.ctime from the stat
		if err = d.Statfile(path); err != nil {
			return err
		}
	case "specialfile", "specialdirectory": // Named Pipe so is a stream, lets set directroy entries to now
		// We set size to 0, d.ctime & d.mtime to now
		d.Size = 0
		if err := d.Mtime.Now("epoch2000_32"); err != nil {
			return err
		}
		d.Ctime = d.Mtime
	default:
		e := "Invalid Property:" + prop
		return errors.New(e)
	}

	d.Path = path
	return nil
}

// Put - Encode the Saratoga directory entry
func (d *DirEnt) Encode() ([]byte, error) {

	// Create the dirent slice
	framelen := 2 + 4 + 4 // Header + Mtime + Ctime

	var dsize int

	// We need to work out the File size in order to set the actual descriptor size needed
	// We will return an error if it can't fit in what has been set here
	switch sarflags.GetDStr(d.Header, "descriptor") { // Offset
	case "d16":
		dsize = 2
	case "d32":
		dsize = 4
	case "d64":
		dsize = 8
	case "d128":
		dsize = 16
	default:
		return nil, errors.New("invalid descriptor in MetaData frame")
	}
	framelen += dsize
	framelen += len(d.Path) + 1 // Add in the required NULL at the end of path
	if len(d.Path) > 1024 {
		return nil, errors.New("path name exceeds 1024 bytes")
	}

	frame := make([]byte, framelen)

	binary.BigEndian.PutUint16(frame[:2], d.Header)

	pos := 2
	switch dsize {
	case 2:
		if d.Size > sarflags.MaxUint16 {
			e := fmt.Sprintf("Descriptor d16 too small for file size %d", d.Size)
			return nil, errors.New(e)
		}
		binary.BigEndian.PutUint16(frame[pos:pos+dsize], uint16(d.Size))
	case 4:
		if d.Size > sarflags.MaxUint32 {
			e := fmt.Sprintf("Descriptor d32 too small for file size %d", d.Size)
			return nil, errors.New(e)
		}
		binary.BigEndian.PutUint32(frame[pos:pos+dsize], uint32(d.Size))
	case 8:
		if d.Size > sarflags.MaxUint32 { // This should never happen!!!!
			e := fmt.Sprintf("Descriptor d64 too small for file size %d", d.Size)
			return nil, errors.New(e)
		}
		binary.BigEndian.PutUint64(frame[pos:pos+dsize], uint64(d.Size))
	default:
		e := fmt.Sprintf("Malformed Directory Entry Invalid descriptor size %d", d.Size)
		return nil, errors.New(e)
	}
	pos += dsize

	binary.BigEndian.PutUint32(frame[pos:pos+4], uint32(d.Mtime.Secs()))
	pos += 4
	binary.BigEndian.PutUint32(frame[pos:pos+4], uint32(d.Ctime.Secs()))
	pos += 4
	endpos := pos + len(d.Path)
	copy(frame[pos:endpos], d.Path)
	copy(frame[endpos:], "\x00") // Add in the Null at the end
	return frame, nil
}

// Get -- Decode Directory Entry byte slice frame into DirEnt struct
func (d *DirEnt) Decode(frame []byte) error {

	if len(frame) < 12 {
		return errors.New("DirEntGet - Entry too short")
	}
	d.Header = binary.BigEndian.Uint16(frame[:2])

	pos := 2
	switch sarflags.GetDStr(d.Header, "descriptor") {
	case "d16":
		dsize := 2
		d.Size = uint64(binary.BigEndian.Uint16(frame[pos : pos+dsize]))
		pos += dsize
	case "d32":
		dsize := 4
		d.Size = uint64(binary.BigEndian.Uint32(frame[pos : pos+dsize]))
		pos += dsize
	case "d64":
		dsize := 8
		d.Size = uint64(binary.BigEndian.Uint64(frame[pos : pos+dsize]))
		pos += dsize
	default:
		return errors.New("invalid MetaData Frame")
	}
	// fmt.Printf("Header:%d Size:%d\n", d.header, d.size)

	ts := make([]byte, 5)
	ts[0], _ = sarflags.SetT(0, "epoch2000_32")

	copy(ts[1:], frame[pos:pos+4])
	if terr := d.Mtime.Get(ts); terr != nil {
		return errors.New("invalid Mtime")
	}
	pos += 4
	// fmt.Printf("Mtime:%s ", d.mtime.Print())

	copy(ts[1:], frame[pos:pos+4])
	if terr := d.Ctime.Get(ts); terr != nil {
		return errors.New("invalid Ctime")
	}
	pos += 4
	// fmt.Printf("Ctime:%s\n", d.ctime.Print())

	d.Path = ""
	for i := range frame[pos:] { // Path is null terminated string
		if frame[pos+i] == '\x00' { // Hit null
			break
		}
		d.Path += string(frame[pos+i])
	}
	// fmt.Printf("Path:<%s>\n", d.path)
	return nil
}

// Print - Print out details of Beacon struct
func (d DirEnt) Print() string {
	sflag := fmt.Sprintf("  Directory Entry: 0x%x\n", d.Header)
	dflags := sarflags.FlagD()
	// sarflags.FrameD("dirent")
	for _, f := range dflags {
		n := sarflags.GetDStr(d.Header, f)
		sflag += fmt.Sprintf("    %s:%s\n", f, n)
	}
	sflag += fmt.Sprintf("    Path:%s\n", d.Path)
	sflag += fmt.Sprintf("    Size:%d\n", d.Size)
	sflag += fmt.Sprintf("    Mtime:%s\n", d.Mtime.Print())
	sflag += fmt.Sprintf("    Ctime:%s", d.Ctime.Print())
	return sflag
}

func (d DirEnt) ShortPrint() string {
	sflag := fmt.Sprintf("  Dir Entry: 0x%x\n", d.Header)
	sflag += fmt.Sprintf("    Path:%s\n", d.Path)
	sflag += fmt.Sprintf("    Size:%d\n", d.Size)
	sflag += fmt.Sprintf("    Mtime:%s\n", d.Mtime.Print())
	sflag += fmt.Sprintf("    Ctime:%s", d.Ctime.Print())
	return sflag
}
