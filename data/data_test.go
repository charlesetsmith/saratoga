package data

import (
	"fmt"
	"testing"

	"github.com/charlesetsmith/saratoga/sarflags"
)

func TestData(t *testing.T) {
	var err error
	var Cmdptr *sarflags.Cliflags
	// The Command line interface commands, help & usage to be read from saratoga.json
	Cmdptr = new(sarflags.Cliflags)

	// The Command line interface commands, help & usage to be read from saratoga.json
	// Cmdptr := new(sarflags.Cliflags)

	// Read in JSON config file and parse it into the Config structure.
	if Cmdptr, err = sarflags.ReadConfig("../saratoga/saratoga.json"); err != nil {
		fmt.Println("Cannot open saratoga config file we have a Readconf error ", "saratoga.json", " ", err)
		return
	}

	fmt.Println(Cmdptr.Global)
	var dat Dinfo

	var d Data
	dptr := &d

	dat.Session = 1234
	dat.Offset = 100
	dat.Payload = nil

	if err := dptr.New("descriptor=d64,reqstatus=no,eod=no,reqtstamp=no", &dat); err != nil {
		t.Fatal(err)
	}
	t.Log(dptr.Print())
}
