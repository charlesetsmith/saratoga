package request

import (
	"testing"

	"github.com/charlesetsmith/saratoga/sarflags"
)

func TestRequest(t *testing.T) {
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
	var req Rinfo
	// Load up the Status Structure
	req.Session = 1234
	req.Fname = "go.mod"

	var r Request
	rptr := &r
	f := "descriptor=d32,udplite=no,fileordir=file,reqtype=put,stream=no"
	// Create a new Status Frame
	if err := rptr.New(f, &req); err != nil {
		t.Fatal(err)
	}
	t.Log(rptr.Print())
}
