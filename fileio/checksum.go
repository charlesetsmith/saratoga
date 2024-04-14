// Checksum Handling of Files

package fileio

import (
	"crypto/md5"
	"crypto/sha1"
	"errors"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"os"

	"github.com/charlesetsmith/saratoga/sarflags"
)

// Polynomials used for CRC32
const (
	// IEEE is by far and away the most common CRC-32 polynomial.
	// Used by ethernet (IEEE 802.3), v.42, fddi, gzip, zip, png, ...
	IEEE = 0xedb88320

	// Castagnoli's polynomial, used in iSCSI.
	// Has better error detection characteristics than IEEE.
	// https://dx.doi.org/10.1109/26.231911
	Castagnoli = 0x82f63b78

	// Koopman's polynomial.
	// Also has better error detection characteristics than IEEE.
	// https://dx.doi.org/10.1109/DSN.2002.1028931
	Koopman = 0xeb31d82e
)

// Checksum -- Calculate the checksum of the file
func Checksum(csumtype string, fname string) ([]byte, error) {

	var hash hash.Hash
	var fp *os.File
	var err error
	var csum []byte

	// Open the the file located at the given path and check for errors "put" means for reading in
	// this circumstance
	if fp, err = FileOpen(fname, "put"); err != nil {
		return nil, err
	}
	//Tell the function to close the file when the function returns
	defer fp.Close()

	switch csumtype {
	case "none":
		return csum, nil
	case "crc32":
		//Create the table with the given polynomial
		tablePolynomial := crc32.MakeTable(IEEE)
		//Open a new hash interface to write the file to
		hash = crc32.New(tablePolynomial)
	case "md5":
		hash = md5.New()
	case "sha1":
		hash = sha1.New()
	default:
		e := "Checksum " + csumtype + " not supported"
		return csum, errors.New(e)
	}
	// input := strings.NewReader(fname)

	if _, err = io.Copy(hash, fp); err != nil {
		return csum, err
	}
	csum = hash.Sum(nil)

	var bsize int

	if bsize = sarflags.Value("csumlen", csumtype); bsize < 0 {
		return nil, fmt.Errorf("Checksum Length of %d not supported", bsize)
	}
	if bsize*4 != len(csum) {
		return nil, fmt.Errorf("Checksum Length of %d != %d", bsize*4, len(csum))
	}
	return csum, nil

}
