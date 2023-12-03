package timestamp

import (
	"testing"

	"github.com/charlesetsmith/saratoga/sarflags"
)

func TestTimestamp(t *testing.T) {

	conf := new(sarflags.Cliflags)
	// Read in JSON config file and parse it into the Config structure.
	if err := conf.ReadConfig("../saratoga/saratoga.json"); err != nil {
		emsg := "Cannot open or parse saratoga.json Readconf error: " + err.Error()
		t.Fatal(emsg)
		return
	}

	ts := new(Timestamp)
	if err := ts.Now("posix32_32"); err != nil {
		t.Fatal(err)
	}
	t.Log(ts.Print())
}
