// Frame Handling Interface

package frames

import (
	"net"
)

type FrameVal interface {
	Val() Frame
	Print() string
}

// Frame - Handler for different frames
//
//	beacon, data, metadata, request, status
type Frame interface {
	Encode() ([]byte, error)               // Encode from frame struct into []bytes
	Decode([]byte) error                   // Decode from []bytes into frame struct (beacon, request, data, metadata, status)
	Print() string                         // Print out contents of some type of frame
	ShortPrint() string                    // Quick summary print out of some type of frame
	New(string, interface{}) error         // Create New Frame with flags & info via interface
	Make(uint32, interface{}) error        // Make New Frame with header & info via interface
	Send(*net.UDPConn, *net.UDPAddr) error // Send a Frame to the remote peer
	Val(*net.UDPAddr) Frame                // Frame field information (beacon,data,request,metadata,status)
}

func Val(f Frame, addr *net.UDPAddr) Frame {
	return f.Val(addr)
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

func New(f Frame, header string, info interface{}) error {
	return f.New(header, info)
}

func Make(f Frame, header uint32, info interface{}) error {
	return f.Make(header, info)
}

func Send(f Frame, conn *net.UDPConn, a *net.UDPAddr) error {
	return f.Send(conn, a)
}
