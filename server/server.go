// Server Transfer

package transfer

/*
// STransfer Server Transfer Info
type STransfer struct {
	Session  uint32              // Session + peer is the unique key
	Ttype    string              // STransfer type "get,getrm,put,putrm,blindput,rm"
	Tstamp   timestamp.Timestamp // Timestamp type used in transfer
	Peer     net.IP              // Remote Host
	Stflags  string              // Status Flags currently set WORK ON THIS!!!!!
	Filename string              // Remote File name to get/put
	Csumtype string              // What type of checksum are we using
	Havemeta bool                // Have we recieved a metadata yet
	Checksum []byte              // Checksum of the remote file to be get/put if requested
	Dir      dirent.DirEnt       // Directory entry info of the remote file to be get/put
	Fp       *os.File            // Local File to write to/read from
	Data     []byte              // Buffered data
	Dcount   int                 // Count of Data frames so we can schedule status
	Progress uint64              // Current Progress indicator
	Inrespto uint64              // In respose to indicator
	CurFills holes.Holes         // What has been received
}

// Strmu - Protect transfer
var Strmu sync.Mutex

// STransfers - Slice of Server transfers in progress
var STransfers = []STransfer{}

// Dcount - Data frmae counter
var Dcount int

// SMatch - Return a pointer to the STransfer if we find it, nil otherwise
func SMatch(ip string, session uint32) *STransfer {

	// Check that ip address is valid
	var addr net.IP
	if addr = net.ParseIP(ip); addr == nil { // We have a valid IP Address
		return nil
	}

	for _, i := range STransfers {
		if addr.Equal(i.Peer) && session == i.Session {
			return &i
		}
	}
	return nil
}

// SNew - Add a new transfer to the STransfers list upon receipt of a request
func SNew(g *gocui.Gui, ttype string, r request.Request, ip string, session uint32) error {

	var t STransfer
	if addr := net.ParseIP(ip); addr != nil { // We have a valid IP Address
		for _, i := range STransfers { // Don't add duplicates
			if addr.Equal(i.Peer) && session == i.Session {
				emsg := fmt.Sprintf("STransfer for session %d to %s is currently in progress, cannnot add transfer",
					session, i.Peer.String())
				sarwin.ErrPrintln(g, "red_black", emsg)
				return errors.New(emsg)
			}
		}

		// Lock it as we are going to add a new transfer slice
		Strmu.Lock()
		defer Strmu.Unlock()
		t.Ttype = ttype
		t.Session = session
		t.Peer = addr
		t.Havemeta = false
		t.Dcount = 0
		// t.filename = fname

		msg := fmt.Sprintf("Added %s Transfer to %s session %d",
			t.Ttype, t.Peer.String(), t.Session)
		STransfers = append(STransfers, t)
		sarwin.MsgPrintln(g, "green_black", msg)
		return nil
	}
	sarwin.ErrPrintln(g, "red_black", "CTransfer not added, invalid IP address ", ip)
	return errors.New(" invalid IP Address")
}

// SChange - Add metadata information to the STransfer in STransfers list upon receipt of a metadata
func (t *STransfer) SChange(g *gocui.Gui, m metadata.MetaData) error {
	// Lock it as we are going to add a new transfer slice
	Strmu.Lock()
	t.Csumtype = sarflags.GetStr(m.Header, "csumtype")
	t.Checksum = make([]byte, len(m.Checksum))
	copy(t.Checksum, m.Checksum)
	t.Dir = m.Dir
	// Create the file buffer for the transfer
	// AT THE MOMENT WE ARE HOLDING THE WHOLE FILE IN A MEMORY BUFFER!!!!
	// OF COURSE WE NEED TO SORT THIS OUT LATER
	if len(t.Data) == 0 { // Create the buffer only once
		t.Data = make([]byte, t.Dir.Size)
	}
	if len(t.Data) != (int)(m.Dir.Size) {
		emsg := fmt.Sprintf("Size of File Differs - Old=%d New=%d",
			len(t.Data), m.Dir.Size)
		return errors.New(emsg)
	}
	t.Havemeta = true
	sarwin.MsgPrintln(g, "yellow_black", "Added metadata to transfer and file buffer size ", len(t.Data))
	Strmu.Unlock()
	return nil
}
*/
