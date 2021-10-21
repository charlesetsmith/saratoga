// Frame Handling Interface

package frames

import (
	"net"
)

// import "github.com/charlesetsmith/saratoga/holes"

// Frame - Handler for different frames
// 	beacon, data, metadata, request, status
type Frame interface {
	Encode() ([]byte, error)      // Encode from frame struct into []bytes
	Decode([]byte) error          // Decode from []bytes into frame struct (beacon, request, data, metadata, status)
	Print() string                // Print out contents of some type of frame
	ShortPrint() string           // Quick summary print out of some type of frame
	UDPWrite(*net.UDPConn) string // "success" is OK any other string is sent back to callier on channel
	New(string, interface{}) error
	Make(uint32, interface{}) error

	/*
		New() error
		Make() error

		NewData(flags string, session uint32, offset uint64, payload []byte) error
		NewRequest(flags string, session uint32, fname string, auth []byte) error
		NewMetaData(flags string, session uint32, fname string) error
		NewStatus(flags string, session uint32, progress uint64, inrespto uint64, holes holes.Holes) error
		NewBeacon(flags string) error

		MakeData(header uint32, session uint32, offset uint64, payload []byte) error
		MakeRequest(header uint32, session uint32, fname string, auth []byte) error
		MakeMetadata(header uint32, session uint32, fname string) error
		MakeStatus(header uint32, session uint32, progress uint64, inrespto uint64, holes holes.Holes) error
		MakeBeacon(header uint32, eid string, freespace uint64) error
	*/
}

// Decode a frame into its structure via Frame interface
func Decode(f Frame, buf []byte) error {
	return f.Decode(buf)
}

// Encode a frame into its structure via Frame interface
func Encode(f Frame) ([]byte, error) {
	return f.Encode()
}

// Print a frame into its structure via Frame interface
func Print(f Frame) string {
	return f.Print()
}

// ShortPrint a frame into its structure via Frame interface
func ShortPrint(f Frame) string {
	return f.ShortPrint()
}

// Encode and Send a Frame to UDP Connection
func UDPWrite(f Frame, conn *net.UDPConn) string {
	var wframe []byte
	var err error
	if wframe, err = Encode(f); err != nil {
		return "badpacket"
	}
	if _, err = conn.Write(wframe); err != nil { // And send it
		return "cantsend"
	}
	return "success"
}

func New(f Frame, header string, info interface{}) error {
	return f.New(header, info)
}

func Make(f Frame, header uint32, info interface{}) error {
	return f.Make(header, info)
}
