package status

import (
	"testing"

	"github.com/charlesetsmith/saratoga/sarflags"
)

func TestStatus(t *testing.T) {
	// var err error
	// var Cmdptr *sarflags.Cliflags
	// The Command line interface commands, help & usage to be read from saratoga.json
	// Cmdptr = new(sarflags.Cliflags)

	// The Command line interface commands, help & usage to be read from saratoga.json
	// Cmdptr := new(sarflags.Cliflags)

	// Read in JSON config file and parse it into the Config structure.
	if _, err := sarflags.ReadConfig("../saratoga/saratoga.json"); err != nil {
		emsg := "Cannot open or parse saratoga.json Readconf error: " + err.Error()
		t.Fatal(emsg)
		return
	}

	// fmt.Println("Global Settings: ", Cmdptr.Global)
	var sta Sinfo
	// Load up the Status Structure
	sta.Session = 1234
	sta.Progress = 56789
	sta.Inrespto = 3456

	var s Status
	sptr := &s

	f := "descriptor=d32,stream=no,reqtstamp=yes,metadatarecvd=no,allholes=no,reqholes=requested,errcode=success,posix64_32"
	// Create a new Status Frame
	if err := s.New(f, &sta); err != nil {
		t.Fatal(err)
	}
	t.Log(sptr.Print())
}
