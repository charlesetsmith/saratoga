package frames

// Frame - Handler for different frames
// 	beacon, data, metadata, request, status
type Frame interface {
	Put() ([]byte, error)   // Put some type of frame
	Get(frame []byte) error // Get some type of frame
	Print() string          // Print out contents of some type of frame
	ShortPrint() string     // Quick summary print out of some type of frame
}
