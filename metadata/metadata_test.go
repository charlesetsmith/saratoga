package metadata

import (
	"fmt"
	"testing"

	"github.com/charlesetsmith/saratoga/sarflags"
)

func TestMetadata(t *testing.T) {
	// var err error
	// var Cmdptr *sarflags.Cliflags
	// The Command line interface commands, help & usage to be read from saratoga.json
	// Cmdptr = new(sarflags.Cliflags)

	// The Command line interface commands, help & usage to be read from saratoga.json
	// Cmdptr := new(sarflags.Cliflags)
	conf := new(sarflags.Cliflags)

	// Read in JSON config file and parse it into the Config structure.
	if err := conf.ReadConfig("../saratoga/saratoga.json"); err != nil {
		emsg := "Cannot open or parse saratoga.json Readconf error: " + err.Error()
		t.Fatal(emsg)
		return
	}
	for i := range conf.Global {
		fmt.Println(i, "=", conf.Global[i])
	}
	fmt.Println("Timeouts:", conf.Timeout)

	// fmt.Println("Global Settings: ", Cmdptr.Global)
	var met Minfo
	// Load up the Status Structure
	met.Session = 1234
	met.Fname = "go.mod"

	var m MetaData
	mptr := &m
	f := "descriptor=d32,transfer=file,progress=inprogress,csumtype=md5,reliability=udponly"
	// Create a new Status Frame
	if err := mptr.New(f, &met); err != nil {
		t.Fatal(err)
	}
	t.Log(mptr.Print())
}
