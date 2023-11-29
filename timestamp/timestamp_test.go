package timestamp

import (
	"fmt"
	"testing"

	"github.com/charlesetsmith/saratoga/sarflags"
)

func TestTimestamp(t *testing.T) {

	// Read in JSON config file and parse it into the Config structure.
	if _, err := sarflags.ReadConfig("../saratoga/saratoga.json"); err != nil {
		fmt.Println("Cannot open saratoga config file we have a Readconf error ", "saratoga.json", " ", err)
		return
	}

	ts := new(Timestamp)
	if err := ts.Now("posix32_32"); err != nil {
		t.Fatal(err)
	}
	t.Log(ts.Print())
}
