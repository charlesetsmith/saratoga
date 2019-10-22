package frames

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"

	"sarflags"
)

// MetaData -- Holds Data frame information
type MetaData struct {
	flags    uint32
	session  uint32
	checksum []byte
	dirent   DirEnt
}

// MetaDataMake - Construct a data frame - return byte slice of frame
// Flags is of format "flagname1=flagval1,flagname2=flagval2...
// The timestamp type to use is also in the flags as "timestamp=flagval"
func MetaDataMake(flags string, session uint32, fname string) ([]byte, error) {

	var header uint32
	var err error

	flags = strings.Replace(flags, " ", "", -1) // Get rid of extra spaces in flags

	if header, err = sarflags.Set(header, "frametype", "metadata"); err != nil {
		return nil, err
	}

	var direntflags string // Particular Flags for directory entry
	var csumtype string    // Checksum type

	// Grab the flags and set the frame header
	flag := strings.Split(flags, ",") // The name=val of the flag
	for fl := range flag {
		f := strings.Split(flag[fl], "=") // f[0]=name f[1]=val
		switch f[0] {
		case "descriptor":
			if header, err = sarflags.Set(header, f[0], f[1]); err != nil {
				return nil, err
			}
			direntflags += f[0] + "=" + f[1] + ","
		case "progress", "udptype", "transfer":
			if header, err = sarflags.Set(header, f[0], f[1]); err != nil {
				return nil, err
			}
		case "csumtype":
			if header, err = sarflags.Set(header, f[0], f[1]); err != nil {
				return nil, err
			}
			// Set the correct csum length
			if header, err = sarflags.Set(header, "csumlen", f[1]); err != nil {
				return nil, err
			}
			csumtype = f[1]
		case "reliability": // Directory Entry Flags
			direntflags += f[0] + "=" + f[1] + ","
		default:
			e := "Invalid Flag " + f[0] + " for MetaData Frame"
			return nil, errors.New(e)
		}
	}

	// Create the frame slice
	framelen := 4 + 4 // Header + Session

	// Stat the file to see what it is
	var file, dir, stream bool
	fi, err := os.Lstat(fname)
	if err != nil {
		return nil, err
	}
	switch mode := fi.Mode(); {
	case mode.IsRegular():
		file = true
		direntflags += "property=normalfile,"
	case mode.IsDir():
		dir = true
		direntflags += "property=normaldirectory,"
	case mode&os.ModeNamedPipe != 0:
		stream = true
		direntflags += "property=specialfile,"
	default:
		e := fmt.Sprintf("Unsupported file, directory or stream type %o for %s", mode, fname)
		return nil, errors.New(e)
	}
	direntflags = strings.TrimSuffix(direntflags, ",") // Get rid of trailing comma

	switch sarflags.GetStr(header, "transfer") {
	case "stream":
		if stream != true {
			e := "Stream specified but " + fname + " is not a named pipe"
			return nil, errors.New(e)
		}
		// You can't get a checksum from a stream
		if sarflags.GetStr(header, "csumtype") != "none" {
			return nil, errors.New("Cannot have checksum with stream transfers")
		}
	case "bundle":
		return nil, errors.New("Bundle Transfers not supported")
	case "file":
		if !file {
			e := fname + " is not a file"
			return nil, errors.New(e)
		}
	case "directory":
		if !dir {
			e := fname + " is not a directory"
			return nil, errors.New(e)
		}
	default:
		return nil, errors.New("Invalid Transfer type")
	}

	csuml := sarflags.Get(header, "csumlen")
	framelen += int(csuml) * 4 // Checksum length in uint32

	framelen += 2 + 4 + 4                        // Dflag + Mtime + Ctime
	des := sarflags.GetStr(header, "descriptor") // FileSize
	switch des {
	case "d16":
		framelen += 2
	case "d32":
		framelen += 4
	case "d64":
		framelen += 8
	default:
		return nil, errors.New("Invalid descriptor in MetaData frame")
	}
	framelen += len(fname) + 1 // For the NULL at end

	if framelen > sarflags.MaxFrameSize {
		return nil, errors.New("MetaData - Maximum Frame Size Exceeded")
	}
	frame := make([]byte, framelen)

	// Populate it
	binary.BigEndian.PutUint32(frame[:4], header)
	binary.BigEndian.PutUint32(frame[4:8], session)
	pos := 8

	// Checksum calculation
	var checksum []byte
	if !stream { // Make sure we dont try and calc a checksum of a named pipe (it will wait forever)
		if checksum, err = Checksum(csumtype, fname); err != nil {
			return nil, err
		}
	}
	csumlen := len(checksum)
	if csumlen > 0 {
		copy(frame[pos:pos+len(checksum)], checksum)
		pos += csumlen
	}

	// Directory Entry
	var de []byte
	if de, err = DirEntMake(direntflags, fname); err != nil {
		return frame, err
	}
	copy(frame[pos:], de)
	return frame, nil
}

// MetaDataGet -- Decode Data byte slice frame into Data struct
func MetaDataGet(frame []byte) (m MetaData, err error) {

	if len(frame) < 8 {
		return m, errors.New("MetaDataGet - Frame too short")
	}
	m.flags = binary.BigEndian.Uint32(frame[:4])
	m.session = binary.BigEndian.Uint32(frame[4:8])
	pos := 8

	// Checksum
	csuml := int(sarflags.Get(m.flags, "csumlen")) * 4
	m.checksum = make([]byte, csuml)
	copy(m.checksum, frame[pos:pos+csuml])
	pos += csuml
	// Directory Entry
	if m.dirent, err = DirEntGet(frame[pos:]); err != nil {
		return m, err
	}
	return m, nil
}

// MetaDataPrint - Print out details of MetaData struct
func MetaDataPrint(d MetaData) string {
	sflag := fmt.Sprintf("Metadata: 0x%x\n", d.flags)
	dflags := sarflags.Frame("metadata")
	// fmt.Println("dflags=", dflags)
	for f := range dflags {
		n := sarflags.GetStr(d.flags, dflags[f])
		sflag += fmt.Sprintf("  %s:%s\n", dflags[f], n)
	}
	sflag += fmt.Sprintf("  session:%d\n", d.session)
	if cs := sarflags.GetStr(d.flags, "csumtype"); cs != "none" {
		sflag += fmt.Sprintf("  Checksum [%s]:%x\n", cs, d.checksum)
	}
	sflag += fmt.Sprintf("%s", DirEntPrint(d.dirent))
	return sflag
}
