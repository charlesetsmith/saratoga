package status

import (
	"fmt"
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
		fmt.Println("Cannot open saratoga config file we have a Readconf error ", "saratoga.json", " ", err)
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
	flags := "descriptor=d32,stream=no,reqtstamp=yes,metadatarecvd=no,allholes=no,reqholes=requested,errcode=success"
	// Create a new Status Frame
	if err := sptr.New(flags, &sta); err != nil {
		t.Fatal(err)
	}
	// fmt.Println("Data Frame: ", dptr.Print())
	t.Log(sptr.Print())
}
