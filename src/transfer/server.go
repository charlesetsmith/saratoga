package transfer

// STransfer Information
type STransfer struct {
	direction string // "client|server"
	ttype     string // CTransfer type "get,getrm,put,putrm,blindput,rm"
	tstamp    string // TImestamp type used in transfer
	session   uint32 // Session ID - This is the unique key
	peer      net.IP // Remote Host
	filename  string // File name to get from remote host
	fp        *os.File
	frames    [][]byte // Frame queue
	holes     []Hole   // Holes
}

var strmu sync.Mutex

// CTransfers - Get list used in get,getrm,getdir,put,putrm & delete
var STransfers = []STransfer{}

