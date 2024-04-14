package fileio

import (
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/charlesetsmith/saratoga/sarflags"
)

func TestFileio(t *testing.T) {

	var err error
	// The Command line interface commands, help & usage to be read from saratoga.json
	// Cmdptr := new(sarflags.Cliflags)
	conf := new(sarflags.Cliflags)
	// Read in JSON config file and parse it into the Config structure.
	if err = conf.ReadConfig("../saratoga/saratoga.json"); err != nil {
		emsg := "Cannot open or parse saratoga.json Readconf error: " + err.Error()
		t.Fatal(emsg)
		return
	}

	var fp *os.File
	fname := conf.Sardir + "/testfile.temp"

	// Remove the file
	if err = FileRm(fname); err != nil {
		t.Log("Cannot remove file: ", fname)
	}

	// Create the file
	if fp, err = FileOpen(fname, "get"); err != nil || fp == nil {
		t.Fatal(err)
	}
	defer fp.Close()
	t.Log("Created file for writing: ", fname)

	// Create a slice three bytes bigger than buffersize for the test
	var b []byte
	var c byte
	for j := 0; j < 4; j++ {
		switch j { // Yes I know this is CRAP!
		case 0:
			c = 'A'
		case 1:
			c = 'B'
		case 2:
			c = 'C'
		case 3:
			c = 'D'
		}
		for i := 0; i < conf.Buffersize; i++ {
			b = append(b, c)
		}
	}
	b = append(b, '0')
	b = append(b, '1')
	b = append(b, '2')

	var filelength int = 0

	var framenumb int
	var n int
	blen := (uint64)(len(b))
	totframes := (int)(blen) / (int)(conf.Buffersize)
	for framenumb = 0; framenumb <= totframes; framenumb++ {
		pos := (uint64)(framenumb) * (uint64)(conf.Buffersize)
		if pos < blen-1024 {
			if n, err = FileWrite(fp, pos, b[pos:pos+(uint64)(conf.Buffersize)]); n != conf.Buffersize || err != nil {
				t.Fatal(err)
			}
		} else {
			remainder := (blen - pos) % (uint64)(conf.Buffersize)
			if n, err = FileWrite(fp, pos, b[pos:pos+(uint64)(remainder)]); (uint64)(n) != remainder || err != nil {
				t.Fatal(err)
			}
		}
		s := fmt.Sprintf("Write framenumb: %d pos: %d len: %d", framenumb, pos, n)
		filelength += n
		t.Log(s)
	}
	t.Log("Write total file length: ", filelength)
	if err = FileClose(fp); err != nil {
		t.Fatal(err)
	}
	// Now calculate its checksum
	var csum []byte

	if csum, err = Checksum("crc32", fname); err != nil {
		t.Fatal(err)
	}
	t.Log("CRC32 Checksum: ", fmt.Sprint(csum))
	if csum, err = Checksum("md5", fname); err != nil {
		t.Fatal(err)
	}
	t.Log("MD5 Checksum: ", fmt.Sprint(csum))
	if csum, err = Checksum("sha1", fname); err != nil {
		t.Fatal(err)
	}
	t.Log("SHA1 Checksum: ", fmt.Sprint(csum))

	// Now lets read in a file to a buffer
	var pos uint64 = 0
	var pfp *os.File
	if pfp, err = FileOpen(fname, "put"); err != nil || pfp == nil {
		t.Fatal(err)
	}
	var rb []byte
	for {
		var b []byte
		if b, err = FileRead(pfp, pos, conf.Buffersize); len(b) < conf.Buffersize {
			if err != nil {
				t.Fatal(err.Error())
			}
			rb = slices.Concat(rb, b)
			t.Log("read buffer len: ", len(b))
			break
		}
		t.Log("read buffer len: ", len(b))
		pos += (uint64)(conf.Buffersize)
		rb = slices.Concat(rb, b)
	}
	t.Log("Read Total File Length: ", len(rb))
	if err = FileClose(pfp); err != nil {
		t.Fatal(err)
	}

	// Now calculate its checksum
	var cb []byte

	if cb, err = Checksum("crc32", fname); err != nil {
		t.Fatal(err)
	}
	s := string(cb)
	t.Log(s)
}
