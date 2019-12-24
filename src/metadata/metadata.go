package metadata

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"dirent"
	"frames"
	"sarflags"

	"github.com/charlesetsmith/saratoga/src/screen"
	"github.com/jroimartin/gocui"
)

// MetaData -- Holds Data frame information
type MetaData struct {
	Header   uint32
	Session  uint32
	Checksum []byte
	Dir      dirent.DirEnt
}

// New - Construct a Metadata structure
// Flags is of format "flagname1=flagval1,flagname2=flagval2...
func (m *MetaData) New(flags string, session uint32, fname string) error {

	var err error

	flags = strings.Replace(flags, " ", "", -1) // Get rid of extra spaces in flags

	if m.Header, err = sarflags.Set(m.Header, "version", "v1"); err != nil {
		return err
	}
	if m.Header, err = sarflags.Set(m.Header, "frametype", "metadata"); err != nil {
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
			if m.Header, err = sarflags.Set(m.Header, f[0], f[1]); err != nil {
				return err
			}
			direntflags += f[0] + "=" + f[1] + ","
		case "progress", "udptype", "transfer":
			if m.Header, err = sarflags.Set(m.Header, f[0], f[1]); err != nil {
				return err
			}
		case "csumtype":
			if m.Header, err = sarflags.Set(m.Header, f[0], f[1]); err != nil {
				return err
			}
			// Set the correct csum length
			if m.Header, err = sarflags.Set(m.Header, "csumlen", f[1]); err != nil {
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

	switch sarflags.GetStr(m.Header, "transfer") {
	case "stream":
		if !stream {
			e := "Stream specified but " + fname + " is not a named pipe"
			return errors.New(e)
		}
		// You can't get a checksum from a stream
		if sarflags.GetStr(m.Header, "csumtype") != "none" {
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

	m.Session = session
	// Checksum calculation
	if !stream { // Make sure we dont try and calc a checksum of a named pipe (it will wait forever)
		var checksum []byte

		if checksum, err = frames.Checksum(csumtype, fname); err != nil {
			return err
		}
		csumlen := len(checksum)
		if csumlen > 0 {
			m.Checksum = make([]byte, len(checksum))
			copy(m.Checksum, checksum)
		}
	} else {
		m.Checksum = nil
	}

	// Directory Entry
	if err = m.Dir.New(direntflags, fname); err != nil {
		return err
	}
	return nil
}

// Put -- Encode the Saratoga Metadata buffer
func (m MetaData) Put() ([]byte, error) {

	// Create the frame slice
	framelen := 4 + 4 // Header + Session
	framelen += len(m.Checksum)
	de, _ := m.Dir.Put()
	framelen += len(de)

	frame := make([]byte, framelen)

	// Populate it
	binary.BigEndian.PutUint32(frame[:4], m.Header)
	binary.BigEndian.PutUint32(frame[4:8], m.Session)
	pos := 8

	// Checksum
	csumlen := len(m.Checksum)
	if csumlen > 0 {
		copy(frame[pos:pos+csumlen], m.Checksum)
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
	m.Header = binary.BigEndian.Uint32(frame[:4])
	m.Session = binary.BigEndian.Uint32(frame[4:8])
	pos := 8

	// Checksum
	csuml := int(sarflags.Get(m.Header, "csumlen")) * 4
	m.Checksum = make([]byte, csuml)
	copy(m.Checksum, frame[pos:pos+csuml])
	pos += csuml
	// Directory Entry
	if err = m.Dir.Get(frame[pos:]); err != nil {
		return err
	}
	return nil
}

// Print - Print out details of MetaData struct
func (m MetaData) Print() string {
	sflag := fmt.Sprintf("Metadata: 0x%x\n", m.Header)
	dflags := sarflags.Values("metadata")
	// fmt.Println("dflags=", dflags)
	for f := range dflags {
		n := sarflags.GetStr(m.Header, dflags[f])
		sflag += fmt.Sprintf("  %s:%s\n", dflags[f], n)
	}
	sflag += fmt.Sprintf("  session:%d\n", m.Session)
	if cs := sarflags.GetStr(m.Header, "csumtype"); cs != "none" {
		sflag += fmt.Sprintf("  Checksum [%s]:%x\n", cs, m.Checksum)
	}
	sflag += fmt.Sprintf("%s", m.Dir.Print())
	return sflag
}

// Handler - We have some incoming metadata for a session. Add the metadata to the session
func (m *MetaData) Handler(g *gocui.Gui, from *net.UDPAddr, session uint32) string {
	screen.Fprintln(g, "msg", "yellow_black", m.Print())
	// Return an errcode string
	return "success"
}
