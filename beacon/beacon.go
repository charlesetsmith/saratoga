// Beacon Frame Handler

package beacon

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"reflect"
	"strings"
	"sync"
	"syscall"

	"github.com/charlesetsmith/saratoga/sarflags"
	"github.com/charlesetsmith/saratoga/timestamp"
)

// Beacon -- Holds Beacon frame information
type Beacon struct {
	Header    uint32
	Freespace uint64 // Set in b.New
	Eid       string // This changes depending upon Host IP Address. Set in "b.Send"
}
type Binfo struct {
	Freespace uint64
	Eid       string
}

type Packet struct {
	Addr net.UDPAddr
	Info Beacon
}

// No pointers returned, as we are sending/receiving these to/on channels
func (b *Beacon) Val(addr *net.UDPAddr) Packet {
	return Packet{Addr: *addr,
		Info: Beacon{Header: b.Header, Freespace: b.Freespace, Eid: b.Eid}}
}

// New - Construct a beacon - Fill in the Beacon struct
func (b *Beacon) New(flags string, info interface{}) error {
	var err error

	// Always present in a Beacon
	if b.Header, err = sarflags.Set(b.Header, "version", "v1"); err != nil {
		return err
	}
	// And yes we are a Beacon Frame
	if b.Header, err = sarflags.Set(b.Header, "frametype", "beacon"); err != nil {
		return err
	}

	flags = strings.Replace(flags, " ", "", -1) // Get rid of extra spaces in flags

	// Grab the flags and set the frame header
	flag := strings.Split(flags, ",") // The name=val of the flag
	for fl := range flag {
		f := strings.Split(flag[fl], "=") // f[0]=name f[1]=val

		switch f[0] {
		case "descriptor", "stream", "txwilling", "rxwilling", "udplite", "freespace":
			if b.Header, err = sarflags.Set(b.Header, f[0], f[1]); err != nil {
				return err
			}
		case "freespaced":
			// Make sure freespace has been turned on
			if b.Header, err = sarflags.Set(b.Header, "freespace", "yes"); err != nil {
				return err
			}
			switch f[1] {
			case "d16", "d32", "d64": //, "d128":
				if b.Header, err = sarflags.Set(b.Header, f[0], f[1]); err != nil {
					return err
				}

			default:
				es := "Beacon.New: Invalid freespaced " + f[1]
				return errors.New(es)
			}
		default:
			e := "Beacon.New: Invalid Flag " + f[0] + "=" + f[1]
			return errors.New(e)
		}
	}
	e := reflect.ValueOf(info).Elem()
	// Set the Eid to what is passed in from binfo (normally "")
	b.Eid = e.FieldByName("Eid").String()

	if sarflags.GetStr(b.Header, "freespace") == "yes" {
		// Assign the values from the interface Dinfo structure
		b.Freespace = e.FieldByName("Freespace").Uint()

		// Ignore if freespaced is set, just set it to the correct size
		var fs syscall.Statfs_t
		var sardir string

		// Where we put/get files from/to
		if sardir, err = os.Getwd(); err != nil {
			b.Header, _ = sarflags.Set(b.Header, "freespace", "no")
			b.Freespace = 0
			return err
		}

		// Freespace is number of Kilobytes (1024 bytes) left on disk
		if b.Freespace == 0 { // It has not been set in info so set it here via Statfs
			if err := syscall.Statfs(sardir, &fs); err != nil {
				b.Header, _ = sarflags.Set(b.Header, "freespace", "no")
				b.Freespace = 0
				return nil
			}
			b.Freespace = (uint64(fs.Bsize) * fs.Bavail) / 1024
		}
		if b.Freespace < sarflags.MaxUint16 {
			b.Header, _ = sarflags.Set(b.Header, "freespaced", "d16")
			return nil
		}
		if b.Freespace < sarflags.MaxUint32 {
			b.Header, _ = sarflags.Set(b.Header, "freespaced", "d32")
			return nil
		}
		if b.Freespace < sarflags.MaxUint64 {
			b.Header, _ = sarflags.Set(b.Header, "freespaced", "d64")
			return nil
		}
		return errors.New("beacon.New: d128 not supported invalid beacon frame")
	}
	if sarflags.GetStr(b.Header, "freespace") == "no" {
		b.Freespace = 0
	}

	return nil
}

// Make - Construct a beacon with a given header - return byte slice of frame
// func (b *Beacon) Make(header uint32, eid string, freespace uint64) error {
func (b *Beacon) Make(header uint32, info interface{}) error {
	var err error

	// Always present in a Beacon
	if header, err = sarflags.Set(header, "version", "v1"); err != nil {
		return err
	}
	// And yes we are a Beacon Frame
	if header, err = sarflags.Set(header, "frametype", "beacon"); err != nil {
		return err
	}
	b.Header = header

	e := reflect.ValueOf(info).Elem()
	// Set the Eid to what is passed in from binfo (normally "")
	b.Eid = e.FieldByName("Eid").String()

	if sarflags.GetStr(b.Header, "freespace") == "yes" {
		// Assign the values from the interface Dinfo structure
		b.Freespace = e.FieldByName("Freespace").Uint()

		// Ignore if freespaced is set, just set it to the correct size
		var fs syscall.Statfs_t
		var sardir string

		// Where we put/get files from/to
		if sardir, err = os.Getwd(); err != nil {
			b.Header, _ = sarflags.Set(b.Header, "freespace", "no")
			b.Freespace = 0
			return err
		}

		// Freespace is number of Kilobytes (1024 bytes) left on disk
		if b.Freespace == 0 { // It has not been set in info so set it here via Statfs
			if err := syscall.Statfs(sardir, &fs); err != nil {
				b.Header, _ = sarflags.Set(b.Header, "freespace", "no")
				b.Freespace = 0
				return nil
			}
			b.Freespace = (uint64(fs.Bsize) * fs.Bavail) / 1024
		}
		if b.Freespace < sarflags.MaxUint16 {
			b.Header, _ = sarflags.Set(b.Header, "freespaced", "d16")
			return nil
		}
		if b.Freespace < sarflags.MaxUint32 {
			b.Header, _ = sarflags.Set(b.Header, "freespaced", "d32")
			return nil
		}
		if b.Freespace < sarflags.MaxUint64 {
			b.Header, _ = sarflags.Set(b.Header, "freespaced", "d64")
			return nil
		}
		return errors.New("beacon.New: d128 not supported invalid beacon frame")
	}
	if sarflags.GetStr(b.Header, "freespace") == "no" {
		b.Freespace = 0
	}
	return nil
}

// Put -- Encode the Saratoga Beacon into a Frame buffer
func (b Beacon) Encode() ([]byte, error) {

	var frame []byte

	framelen := 4 + len(b.Eid)
	if sarflags.GetStr(b.Header, "freespace") == "yes" {
		switch sarflags.GetStr(b.Header, "freespaced") {
		case "d16":
			framelen += 2
		case "d32":
			framelen += 4
		case "d64":
			framelen += 8
		case "d128":
			return nil, errors.New("d128 not supported invalid beacon frame")
			// framelen += 16
		default:
			return nil, errors.New("invalid beacon frame")
		}
	}

	frame = make([]byte, framelen)

	pos := 4
	binary.BigEndian.PutUint32(frame[:pos], b.Header)
	if sarflags.GetStr(b.Header, "freespace") == "yes" {
		switch sarflags.GetStr(b.Header, "freespaced") {
		case "d16":
			binary.BigEndian.PutUint16(frame[pos:6], uint16(b.Freespace))
			pos += 2
		case "d32":
			binary.BigEndian.PutUint32(frame[pos:8], uint32(b.Freespace))
			pos += 4
		case "d64":
			binary.BigEndian.PutUint64(frame[pos:12], uint64(b.Freespace))
			pos += 8
		case "d128": // d128 we use d64 for the moment untill we have 128 bit uints!
			return nil, errors.New("d128 not supported invalid beacon frame")
			// binary.BigEndian.PutUint64(frame[pos:20], uint64(b.Freespace))
		default:
			return nil, errors.New("invalid beacon frame")
		}
	}
	copy(frame[pos:], []byte(b.Eid))
	return frame, nil
}

// Get -- Decode Beacon byte slice frame into Beacon struct
func (b *Beacon) Decode(frame []byte) error {

	b.Header = binary.BigEndian.Uint32(frame[:4])
	if sarflags.GetStr(b.Header, "freespace") == "yes" {
		switch sarflags.GetStr(b.Header, "freespaced") {
		case "d16":
			b.Freespace = uint64(binary.BigEndian.Uint16(frame[4:6]))
			b.Eid = string(frame[6:])
		case "d32":
			b.Freespace = uint64(binary.BigEndian.Uint32(frame[4:8]))
			b.Eid = string(frame[8:])
		case "d64":
			b.Freespace = binary.BigEndian.Uint64(frame[4:12])
			b.Eid = string(frame[12:])
		case "d128":
			return errors.New("d128 not supported invalid beacon frame")
			// b.Freespace = binary.BigEndian.Uint64(frame[4:20])
			// b.Eid = string(frame[20:])
		default:
			b.Freespace = 0
			b.Eid = string(frame[4:])
			return errors.New("invalid beacon frame")
		}
		return nil
	}
	// No freespace to be reported
	b.Freespace = 0
	b.Eid = string(frame[4:])
	return nil
}

// Print - Print out details of Beacon struct
func (b *Beacon) Print() string {
	sflag := fmt.Sprintf("Beacon: 0x%x\n", b.Header)
	bflags := sarflags.Values("beacon")
	for _, f := range bflags {
		n := sarflags.GetStr(b.Header, f)
		sflag += fmt.Sprintf("  %s:%s\n", f, n)
	}
	if sarflags.GetStr(b.Header, "freespace") == "yes" {
		sflag += fmt.Sprintf("  free:%dkB\n", b.Freespace)
	}
	sflag += fmt.Sprintf("  EID:%s", b.Eid)
	return sflag
}

// ShortPrint - Quick printout of Beacon struct
func (b *Beacon) ShortPrint() string {
	sflag := fmt.Sprintf("Beacon: 0x%x\n", b.Header)
	if sarflags.GetStr(b.Header, "freespace") == "yes" {
		sflag += fmt.Sprintf("  free:%dkB\n", b.Freespace)
	}
	sflag += fmt.Sprintf("  EID:%s", b.Eid)
	return sflag
}

/* Send a data to an address on the connection */
func (b Beacon) Send(conn *net.UDPConn) error {
	var err error
	var buf []byte
	var wlen int

	//	txb := b
	//	eid := fmt.Sprintf("%s-%d", conn.LocalAddr().String(), os.Getpid())
	//	txb.Eid = eid

	if buf, err = b.Encode(); err != nil {
		return err
	}
	if wlen, err = conn.Write(buf); err != nil {
		return err
	}
	if wlen != len(buf) {
		return fmt.Errorf("Beacon sent (%d) to %s != frame size (%d)",
			wlen, conn.RemoteAddr().String(), len(buf))
	}
	return nil
}

// Peer - beacon peer
type Peer struct {
	Addr      string              // The Peer IP Address. is format net.UDPAddr.IP.String()
	Freespace uint64              // 0 if freespace not advertised
	Eid       string              // Exactly who sent this and from what PID
	Maxdesc   string              // The maximum descriptor size of the peer
	Created   timestamp.Timestamp // When was this Peer created
	Updated   timestamp.Timestamp // When was this Peer last updated
}

var pmu sync.Mutex // Protect Peers
// Peers - Slices of unique Peer information learned from beacons
var Peers []Peer

// NewPeer - Add/Change peer info from received beacon
func (b Beacon) NewPeer(from *net.UDPAddr) bool {
	if from == nil {
		return false
	}
	// Scan through existing Peers and change if the peer exists
	for p := range Peers {
		if Peers[p].Addr == from.IP.String() { // Source Ip Address matches existing Peer
			pmu.Lock()
			defer pmu.Unlock()
			// Has anything changed since the last beacon for this peer ?
			if Peers[p].Freespace != b.Freespace ||
				Peers[p].Eid != b.Eid ||
				Peers[p].Maxdesc != sarflags.GetStr(b.Header, "descriptor") {
				Peers[p].Freespace = b.Freespace
				Peers[p].Eid = b.Eid
				Peers[p].Maxdesc = sarflags.GetStr(b.Header, "descriptor")
				Peers[p].Updated.Now("posix32_32") // Last updated now
				return true
			}
			return false
		}
	}
	// We have a new Peer - add it
	// var newp Peer
	newp := new(Peer)
	newp.Addr = from.IP.String()
	newp.Freespace = b.Freespace
	newp.Eid = b.Eid
	newp.Maxdesc = sarflags.GetStr(b.Header, "descriptor")
	newp.Created.Now("posix32_32")
	newp.Updated = newp.Created
	pmu.Lock()
	defer pmu.Unlock()
	Peers = append(Peers, *newp)
	return true
}
