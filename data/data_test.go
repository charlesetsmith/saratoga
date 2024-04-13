package data

import (
	"testing"

	"github.com/charlesetsmith/saratoga/sarflags"
)

func TestData(t *testing.T) {

	// The Command line interface commands, help & usage to be read from saratoga.json
	// Cmdptr := new(sarflags.Cliflags)
	conf := new(sarflags.Cliflags)
	// Read in JSON config file and parse it into the Config structure.
	if err := conf.ReadConfig("../saratoga/saratoga.json"); err != nil {
		emsg := "Cannot open or parse saratoga.json Readconf error: " + err.Error()
		t.Fatal(emsg)
		return
	}
	// fmt.Println("Global Settings: ", Cmdptr.Global)
	var dat Dinfo
	// Load up the Data Structure
	dat.Session = 1234
	dat.Offset = 56789
	var i uint8
	for i = 0; i < 255; i++ {
		dat.Payload = append(dat.Payload, i)
	}

	var err error
	var d Data
	var buff []byte
	// var buff []byte
	dptr := &d

	// Create a new Data Frame
	f := "descriptor=d32,reqstatus=yes,eod=no,reqtstamp=yes"
	if err = dptr.New(f, &dat); err != nil {
		t.Fatal(err)
	}

	if buff, err = dptr.Encode(); err != nil {
		t.Fatal(err.Error())
	}
	t.Log("New:" + dptr.Print())
	if err = dptr.Decode(buff); err != nil {
		t.Fatal(err.Error())
	}
	t.Log("Decode:" + dptr.Print())

	// fmt.Println("Data Frame: ", dptr.Print())
}
