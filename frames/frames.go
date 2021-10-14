// Frame Handling Interface

package frames

import "github.com/charlesetsmith/saratoga/holes"

// Frame - Handler for different frames
// 	beacon, data, metadata, request, status
type Frame interface {
	Put() ([]byte, error)   // Put some type of frame
	Get(frame []byte) error // Get some type of frame
	Print() string          // Print out contents of some type of frame
	ShortPrint() string     // Quick summary print out of some type of frame
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
}
