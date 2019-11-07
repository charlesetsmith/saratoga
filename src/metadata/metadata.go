package metadata

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"

	"dirent"
	"frames"
	"sarflags"
)

// MetaData -- Holds Data frame information
type MetaData struct {
	header   uint32
	session  uint32
	checksum []byte
	dir      dirent.DirEnt
}

// New - Construct a Metadata structure
// Flags is of format "flagname1=flagval1,flagname2=flagval2...
func (m *MetaData) New(flags string, session uint32, fname string) error {

	var err error

	flags = strings.Replace(flags, " ", "", -1) // Get rid of extra spaces in flags

	if m.header, err = sarflags.Set(m.header, "frametype", "metadata"); err != nil {
		return err
	}

	var direntflags string // Particular Flags for directory entry
	var csumtype string    // Checksum type

	// Grab the flags and set the frame header
	flag := strings.Split(flags, ",") // The name=val of the flag
	for fl := range flag {
		f := strings.Split(flag[fl], "=") // f[0]=name f[1]=val
		switch f[0] {
		case "descriptor":
			if m.header, err = sarflags.Set(m.header, f[0], f[1]); err != nil {
				return err
			}
			direntflags += f[0] + "=" + f[1] + ","
		case "progress", "udptype", "transfer":
			if m.header, err = sarflags.Set(m.header, f[0], f[1]); err != nil {
				return err
			}
		case "csumtype":
			if m.header, err = sarflags.Set(m.header, f[0], f[1]); err != nil {
				return err
			}
			// Set the correct csum length
			if m.header, err = sarflags.Set(m.header, "csumlen", f[1]); err != nil {
				return err
			}
			csumtype = f[1]
		case "reliability": // Directory Entry Flags
			direntflags += f[0] + "=" + f[1] + ","
		default:
			e := "Invalid Flag " + f[0] + " for MetaData Frame"
			return errors.New(e)
		}
	}

	// Stat the file to see what it is
	var file, dir, stream bool
	fi, err := os.Lstat(fname)
	if err != nil {
		return err
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
		return errors.New(e)
	}
	direntflags = strings.TrimSuffix(direntflags, ",") // Get rid of trailing comma

	switch sarflags.GetStr(m.header, "transfer") {
	case "stream":
		if !stream {
			e := "Stream specified but " + fname + " is not a named pipe"
			return errors.New(e)
		}
		// You can't get a checksum from a stream
		if sarflags.GetStr(m.header, "csumtype") != "none" {
			return errors.New("Cannot have checksum with stream transfers")
		}
	case "bundle":
		return errors.New("Bundle Transfers not supported")
	case "file":
		if !file {
			e := fname + " is not a file"
			return errors.New(e)
		}
	case "directory":
		if !dir {
			e := fname + " is not a directory"
			return errors.New(e)
		}
	default:
		return errors.New("Invalid Transfer type")
	}

	m.session = session
	// Checksum calculation
	if !stream { // Make sure we dont try and calc a checksum of a named pipe (it will wait forever)
		var checksum []byte

		if checksum, err = frames.Checksum(csumtype, fname); err != nil {
			return err
		}
		csumlen := len(checksum)
		if csumlen > 0 {
			m.checksum = make([]byte, len(checksum))
			copy(m.checksum, checksum)
		}
	} else {
		m.checksum = nil
	}

	// Directory Entry
	if err = m.dir.New(direntflags, fname); err != nil {
		return err
	}
	return nil
}

// Put -- Encode the Saratoga Metadata buffer
func (m MetaData) Put() ([]byte, error) {

	// Create the frame slice
	framelen := 4 + 4 // Header + Session
	framelen += len(m.checksum)
	de, _ := m.dir.Put()
	framelen += len(de)

	if framelen > sarflags.MaxFrameSize {
		return nil, errors.New("MetaData - Maximum Frame Size Exceeded")
	}
	frame := make([]byte, framelen)

	// Populate it
	binary.BigEndian.PutUint32(frame[:4], m.header)
	binary.BigEndian.PutUint32(frame[4:8], m.session)
	pos := 8

	// Checksum
	csumlen := len(m.checksum)
	if csumlen > 0 {
		copy(frame[pos:pos+csumlen], m.checksum)
		pos += csumlen
	}

	copy(frame[pos:], de)
	return frame, nil
}

// Get -- Decode Data byte slice frame into Data struct
func (m *MetaData) Get(frame []byte) (err error) {

	if len(frame) < 8 {
		return errors.New("MetaDataGet - Frame too short")
	}
	m.header = binary.BigEndian.Uint32(frame[:4])
	m.session = binary.BigEndian.Uint32(frame[4:8])
	pos := 8

	// Checksum
	csuml := int(sarflags.Get(m.header, "csumlen")) * 4
	m.checksum = make([]byte, csuml)
	copy(m.checksum, frame[pos:pos+csuml])
	pos += csuml
	// Directory Entry
	if err = m.dir.Get(frame[pos:]); err != nil {
		return err
	}
	return nil
}

// Print - Print out details of MetaData struct
func (m MetaData) Print() string {
	sflag := fmt.Sprintf("Metadata: 0x%x\n", m.header)
	dflags := sarflags.Values("metadata")
	// fmt.Println("dflags=", dflags)
	for f := range dflags {
		n := sarflags.GetStr(m.header, dflags[f])
		sflag += fmt.Sprintf("  %s:%s\n", dflags[f], n)
	}
	sflag += fmt.Sprintf("  session:%d\n", m.session)
	if cs := sarflags.GetStr(m.header, "csumtype"); cs != "none" {
		sflag += fmt.Sprintf("  Checksum [%s]:%x\n", cs, m.checksum)
	}
	sflag += fmt.Sprintf("%s", m.dir.Print())
	return sflag
}
