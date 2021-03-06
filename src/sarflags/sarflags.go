// Saratoga Flag Handlers

package sarflags

import (
	"errors"
	"log"
	"strings"
	"sync"
)

// Saratoga Sflag Header Field Format - 32 bit unsigned integer (uint32)

//             111111 11112222 22222233
//  01234567 89012345 67890123 45678901
// +--------+--------+--------+--------+
// |XXXYYYYY|ZZ      |        |        |
// +--------+--------+--------+--------+
//
// XXXYYYYY -> Version (001) and Frame Type (5 bits)
//         ZZ -> Descriptor Size = uint16, uint32 , uint64 or 128 bit
// Note: Descriptor of 128 bit is not implemented in this release.

// * BEACON FRAME FLAGS
// *  0                   1                   2                   3
// *  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// * | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
// * |0|0|1|-> Version 1 - f_version
// * | | | |0|0|0|0|0|-> Beacon Frame - f_frametype
// * | | | | | | | | |X|X|-> Descriptor - f_descriptor
// * | | | | | | | | | | |0|-> Undefined used to be Bundles
// * | | | | | | | | | | | |X|-> Streaming - f_stream
// * | | | | | | | | | | | | |X|X| | |-> Tx Willing - f_txwilling
// * | | | | | | | | | | | | | | |X|X|-> Rx Willing - f_rxwilling
// * | | | | | | | | | | | | | | | | |X|-> UDP Lite - f_udptype
// * | | | | | | | | | | | | | | | | | |X|-> Freespace Advertise - f_freespace
// * | | | | | | | | | | | | | | | | | | |X|X|-> Freespace Descriptor - f_freespaced
// * | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
// *  0                   1                   2                   3
// *  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// *******************************************************************

// * REQUEST FRAME FLAGS
// *  0                   1                   2                   3
// *  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// * | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
// * |0|0|1|-> Version 1 - f_version
// * | | | |0|0|0|0|1|-> Request Frame - f_frametype
// * | | | | | | | | |X|X|-> Descriptor - f_descriptor
// * | | | | | | | | | | |0|-> Undefined used to be Bundles
// * | | | | | | | | | | | |X|-> Streams - f_stream
// * | | | | | | | | | | | | | | | | |X|-> UDP Lite - f_udptype
// * | | | | | | | | | | | | | | | | | | | | | | | | |X|X|X|X|X|X|X|X|-> f_requesttype
// * | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
// *  0                   1                   2                   3
// *  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// *
// *******************************************************************

// * METADATA FRAME FLAGS
// *  0                   1                   2                   3
// *  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// * | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
// * |0|0|1|-> Version 1 - f_version
// * | | | |0|0|0|1|0|-> Metadata Frame - f_frametype
// * | | | | | | | | |X|X|-> Descriptor - f_descriptor
// * | | | | | | | | | | |X|X|-> Type of Transfer - f_transfer
// * | | | | | | | | | | | | |X|-> Transfer in Progress - f_progress
// * | | | | | | | | | | | | | |X|-> Reliability - f_udptype
// * | | | | | | | | | | | | | | | | | | | | | | | | |X|X|X|X|-> Checksum Length - f_csumlen
// * | | | | | | | | | | | | | | | | | | | | | | | | | | | | |X|X|X|X|-> Checksum Type - f_csumtype
// * | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
// *  0                   1                   2                   3
// *  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// *
// *******************************************************************

// * DATA FRAME FLAGS
// *  0                   1                   2                   3
// *  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// * | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
// * |0|0|1|-> Version 1 - f_version
// * | | | |0|0|0|1|1|-> Data Frame - f_frametype
// * | | | | | | | | |X|X|-> Descriptor - f_descriptor
// * | | | | | | | | | | |X|X|-> Type of Transfer - f_transfer
// * | | | | | | | | | | | | |X|-> Timestamps - f_reqtstamp
// * | | | | | | | | | | | | | | | |X|-> Request Status - f_reqstatus
// * | | | | | | | | | | | | | | | | |X|-> End of Data - f_eod
// * | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
// *  0                   1                   2                   3
// *  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// *
// *******************************************************************

// * STATUS FRAME FLAGS
// *  0                   1                   2                   3
// *  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// * | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
// * |0|0|1|-> Version 1 - f_version
// * | | | |0|0|1|0|0|-> Status Frame - f_frametype
// * | | | | | | | | |X|X|-> Descriptor - f_descriptor
// * | | | | | | | | | | | | |X|-> Timestamp - f_reqtstamp
// * | | | | | | | | | | | | | |X|->Metadata Received - f_metadatarecvd
// * | | | | | | | | | | | | | | |X|-> All Holes - f_allholes
// * | | | | | | | | | | | | | | | |X|-> Holes Requested or Sent - f_reqholes
// * | | | | | | | | | | | | | | | | | | | | | | | | |X|X|X|X|X|X|X|X|-> Error Code - f_errcode
// * | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
// *  0                   1                   2                   3
// *  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// *
// *******************************************************************

// MaxUint - Maximum unsigned int on this platform
const MaxUint = uint64(^uint(0)) // What is the biggest unsigned integer supported on platform

// MaxUint16 -- Biggest unsigned 16 bit integer
const MaxUint16 = uint64(65535)

// MaxUint32 -- Biggest unsigned 32 bit integer
const MaxUint32 = uint64(4294967295)

// MaxUint64 -- Biggest unsigned 64 bit integer
// It should be this but...
// const MaxUint64 = uint64(18446744073709551615)
// const MaxUint64 = uint64(0xFFFFFFFFFFFFFFFF)

// MaxUint64 - Biggest 64 bit unsigned integer
// It needs to be this as handling file i/o and slices requires an "int"
// which is a signed 64 bit number in go so lets "pretend"
const MaxUint64 = uint64(0x7FFFFFFFFFFFFFFF)

// Length in bits of the saratoga header flag
const flagsize uint32 = 32

// MaxBuff -- Maximum read []byte buffer, set to Jumbo to be sure
const MaxBuff = uint64(9000)

// MTU -- Maximum write []byte buffer, set to interface MTU
var MTU int

// Index for length and msb values in flagfield map
const fieldlen = 0
const fieldmsb = 1

// Map of Saratoga Flags in Frame Header
// First 32 bits of every frame have a combination of these flags
// The 0 element (fieldlen) value in the uint32[2] is the length in bits of the flag
// The 1 element (fieldmsb) value in the uint32[2] is the bit offset from the front.
// This is all in network byte order
var flagbits = map[string][2]uint32{
	"version":       [2]uint32{3, 0},
	"frametype":     [2]uint32{5, 3},
	"descriptor":    [2]uint32{2, 8},
	"stream":        [2]uint32{1, 11},
	"transfer":      [2]uint32{2, 10},
	"reqtstamp":     [2]uint32{1, 12},
	"progress":      [2]uint32{1, 12},
	"txwilling":     [2]uint32{2, 12},
	"udptype":       [2]uint32{1, 13},
	"metadatarecvd": [2]uint32{1, 13},
	"allholes":      [2]uint32{1, 14},
	"reqtype":       [2]uint32{8, 24},
	"rxwilling":     [2]uint32{2, 14},
	"reqholes":      [2]uint32{1, 15},
	"fileordir":     [2]uint32{1, 15},
	"reqstatus":     [2]uint32{1, 15},
	"udplite":       [2]uint32{1, 16},
	"eod":           [2]uint32{1, 16},
	"freespace":     [2]uint32{1, 17},
	"freespaced":    [2]uint32{2, 18},
	"csumlen":       [2]uint32{4, 24},
	"csumtype":      [2]uint32{4, 28},
	"errcode":       [2]uint32{8, 24},
}

// Map of which flags are applicable to which frame types
var flagframe = map[string][]string{
	"version":       []string{"beacon", "request", "metadata", "data", "status"},
	"frametype":     []string{"beacon", "request", "metadata", "data", "status"},
	"descriptor":    []string{"beacon", "request", "metadata", "data", "status"},
	"stream":        []string{"beacon", "request"},
	"transfer":      []string{"metadata", "data"},
	"reqtstamp":     []string{"data", "status"},
	"progress":      []string{"metadata"},
	"txwilling":     []string{"beacon"},
	"udptype":       []string{"metadata"},
	"metadatarecvd": []string{"status"},
	"allholes":      []string{"status"},
	"reqtype":       []string{"request"},
	"rxwilling":     []string{"beacon"},
	"reqholes":      []string{"status"},
	"fileordir":     []string{"request"},
	"reqstatus":     []string{"data"},
	"udplite":       []string{"beacon", "request"},
	"eod":           []string{"data"},
	"freespace":     []string{"beacon"},
	"freespaced":    []string{"beacon"},
	"csumlen":       []string{"metadata"},
	"csumtype":      []string{"metadata"},
	"errcode":       []string{"status"},
}

/*
 Frame types with applicable flags:
	"beacon": "version", "frametype", "descriptor", "stream", "txwilling", "rxwilling", "udplite", "freespace", "freespaced"
	"request": "version", "frametype", "descriptor", "stream", "reqtype", "fileordir", "udplite"
	"metadata": "version", "frametype", "descriptor, "transfer", "progress", "udptype", "csumlen", "csumtype"
	"data": "version", "frametype", "descriptor", "transfer", "reqtstamp", "reqstatus", "eod"
	"status": "version", "frametype", "descriptor", "reqtstamp", "metadatarecvd", "allholes", "reqholes", "errcode"
*/

type flaginfo struct {
	name string
	val  uint32
}

var flagvals = map[string][]flaginfo{
	"version": []flaginfo{ // "beacon", "request", "metadata", "data", "status"
		flaginfo{name: "v0", val: 0},
		flaginfo{name: "v1", val: 1},
	},
	"frametype": []flaginfo{ // "beacon", "request", "metadata", "data", "status"
		flaginfo{name: "beacon", val: 0},
		flaginfo{name: "request", val: 1},
		flaginfo{name: "metadata", val: 2},
		flaginfo{name: "data", val: 3},
		flaginfo{name: "status", val: 4},
	},
	"descriptor": []flaginfo{ // "beacon", "request", "metadata", "data", "status"
		flaginfo{name: "d16", val: 0},
		flaginfo{name: "d32", val: 1},
		flaginfo{name: "d64", val: 2},
		// flaginfo{name: "d128", val: 3}, INVALID AT THIS TIME WAIT FOR 128 bit int's
	},
	"stream": []flaginfo{ // "beacon", "request"
		flaginfo{name: "no", val: 0},
		flaginfo{name: "yes", val: 1},
	},
	"transfer": []flaginfo{ // "metadata", "data"
		flaginfo{name: "file", val: 0},
		flaginfo{name: "directory", val: 1},
		flaginfo{name: "bundle", val: 2},
		flaginfo{name: "stream", val: 3},
	},
	"reqtstamp": []flaginfo{ // "data", "status"
		flaginfo{name: "no", val: 0},
		flaginfo{name: "yes", val: 1},
	},
	"progress": []flaginfo{ // "metadata"
		flaginfo{name: "inprogress", val: 0},
		flaginfo{name: "terminated", val: 1},
	},
	"txwilling": []flaginfo{ // "beacon"
		flaginfo{name: "no", val: 0},
		flaginfo{name: "invalid", val: 1},
		flaginfo{name: "capable", val: 2},
		flaginfo{name: "yes", val: 3},
	},
	"udptype": []flaginfo{ // "metadata"
		flaginfo{name: "udponly", val: 0},
		flaginfo{name: "udplite", val: 1},
	},
	"metadatarecvd": []flaginfo{ // "status
		flaginfo{name: "yes", val: 0},
		flaginfo{name: "no", val: 1},
	},
	"allholes": []flaginfo{ // "status"
		flaginfo{name: "yes", val: 0},
		flaginfo{name: "no", val: 1},
	},
	"reqtype": []flaginfo{ // "request"
		flaginfo{name: "noaction", val: 0},
		flaginfo{name: "get", val: 1},
		flaginfo{name: "put", val: 2},
		flaginfo{name: "getdelete", val: 3},
		flaginfo{name: "putdelete", val: 4},
		flaginfo{name: "delete", val: 5},
		flaginfo{name: "getdir", val: 6},
	},
	"rxwilling": []flaginfo{ // "beacon"
		flaginfo{name: "no", val: 0},
		//		flaginfo{name: "invalid", val: 1},
		flaginfo{name: "capable", val: 2},
		flaginfo{name: "yes", val: 3},
	},
	"reqholes": []flaginfo{ // "status"
		flaginfo{name: "requested", val: 0},
		flaginfo{name: "voluntarily", val: 1},
	},
	"fileordir": []flaginfo{ // "request"
		flaginfo{name: "file", val: 0},
		flaginfo{name: "directory", val: 1},
	},
	"reqstatus": []flaginfo{ // "data"
		flaginfo{name: "no", val: 0},
		flaginfo{name: "yes", val: 1},
	},
	"udplite": []flaginfo{ // "beacon", "request"
		flaginfo{name: "no", val: 0},
		flaginfo{name: "yes", val: 1},
	},
	"eod": []flaginfo{ // "data"
		flaginfo{name: "no", val: 0},
		flaginfo{name: "yes", val: 1},
	},
	"freespace": []flaginfo{ // "beacon"
		flaginfo{name: "no", val: 0},
		flaginfo{name: "yes", val: 1},
	},
	"freespaced": []flaginfo{ // "beacon"
		flaginfo{name: "d16", val: 0},
		flaginfo{name: "d32", val: 1},
		flaginfo{name: "d64", val: 2},
		// flaginfo{name: "d128", val: 3}, INVALID AT THIS TIME WIAT FOR 128 bit ints
	},
	"csumlen": []flaginfo{ // "metadata"
		flaginfo{name: "none", val: 0},
		flaginfo{name: "crc32", val: 1},
		// flaginfo{name: "invalid2", val: 2},
		// flaginfo{name: "invalid3", val: 3},
		flaginfo{name: "md5", val: 4},
		flaginfo{name: "sha1", val: 5},
	},
	"csumtype": []flaginfo{ // "metadata"
		flaginfo{name: "none", val: 0},
		flaginfo{name: "crc32", val: 1},
		flaginfo{name: "md5", val: 2},
		flaginfo{name: "sha1", val: 3},
	},
	"errcode": []flaginfo{ // "status"
		flaginfo{name: "success", val: 0x0},     // Process the status and continue
		flaginfo{name: "unspecified", val: 0x1}, // All others immediately kill the transfer
		flaginfo{name: "cantsend", val: 0x2},
		flaginfo{name: "cantreceive", val: 0x3},
		flaginfo{name: "filenotfound", val: 0x4},
		flaginfo{name: "accessdenied", val: 0x5},
		flaginfo{name: "unknownid", val: 0x6},
		flaginfo{name: "didnotdelete", val: 0x7},
		flaginfo{name: "filetobig", val: 0x8},
		flaginfo{name: "badoffset", val: 0x9},
		flaginfo{name: "badpacket", val: 0xA},
		flaginfo{name: "badrequest", val: 0xB},
		flaginfo{name: "internaltimeout", val: 0xC},
		flaginfo{name: "baddataflag", val: 0xD},
		flaginfo{name: "rxnotinterested", val: 0xE},
		flaginfo{name: "fileinuse", val: 0xF},
		flaginfo{name: "metadatarequired", val: 0x10},
		flaginfo{name: "badstatus", val: 0x11},
		flaginfo{name: "rxtimeout", val: 0x12},
	},
}

// Timeouts - Global Timeouts and counters
type Timeouts struct {
	Metadata  int  // Secs to wait
	Request   int  // Secs to wait
	Status    int  // Secs to wait
	Transfer  int  // Secs to wait before cancelling transfer when nothing recieved
	Binterval uint // Secs between sending beacon frames
}

// GTimeout - timeouts for responses 0 means no timeout
var GTimeout = Timeouts{}

// Cliflags - CLI Input flags
type Cliflags struct {
	Global    map[string]string // Global header flags set for frames
	Timestamp string            // What timestamp to use
	Timeout   Timeouts          // Various timeouts
	Datacnt   int               // # data frames to send before a request flag is set
	Timezone  string            // Timezone for logs utc or local time
}

// Cli - The CLI Flags that are entered in cli.go
var Cli = Cliflags{}

// Climu - Protect CLI input flags
var Climu sync.Mutex

// Valid - Check for valid flag and value
func Valid(field string, info string) bool {
	for f := range flagvals {
		if field == f {
			for _, fi := range flagvals[field] {
				if fi.name == info {
					return true
				}
			}
		}
	}
	return false
}

// Values - Return slice of flags applicable to frame type (field)
func Values(field string) []string {
	var ret []string
	for f := range flagframe {
		// fmt.Println("Flagframe:", f)
		for _, fi := range flagframe[f] {
			if fi == field {
				ret = append(ret, f)
			}
		}
	}
	// fmt.Println(ret)
	return ret
}

// Value - Return the integer value of the flag or -1 if not valid
func Value(field string, info string) int {
	for f := range flagvals {
		if field == f {
			for _, fi := range flagvals[field] {
				if fi.name == info {
					return int(fi.val)
				}
			}
		}
	}
	return -1
}

// Get - Given a current flag and bitfield name return the integer value of the bitfield
func Get(curflag uint32, field string) uint32 {
	if _, ok := flagbits[field]; !ok {
		log.Fatalln("Get lookup fail Invalid Flag", field)
	}

	var len, msb, shiftbits, maskbits, setbits uint32

	len = flagbits[field][fieldlen]
	msb = flagbits[field][fieldmsb]
	shiftbits = flagsize - len - msb
	maskbits = (1 << len) - 1
	setbits = maskbits << shiftbits
	return (curflag & setbits) >> shiftbits
}

// GetStr - Given a current flag and bitfield name return the string name of the bitfield set in curflag
func GetStr(curflag uint32, field string) string {

	val := Get(curflag, field)
	for _, fi := range flagvals[field] {
		// fmt.Printf("GetStr Curflag %0x Looking for %x val in %x=%s\n", curflag, val, fi.val, fi.name)
		if fi.val == val {
			return fi.name
		}
	}
	log.Fatalln("GetStr fail Invalid field", field, "in Flag", curflag)
	return ""
}

// Set - Given a current header and bitfield name with a new value return the revised header
// If invalid return the current flag and error
func Set(curflag uint32, field string, flagname string) (uint32, error) {
	if _, ok := flagbits[field]; !ok {
		log.Fatalln("Set lookup fail Invalid Flag", field)
		e := "invalid Flag: " + field
		return curflag, errors.New(e)
	}

	var newval uint32
	var found = false
	// Get the value of the flag
	for _, fi := range flagvals[field] {
		// log.Println("Flags for field ", field, fi.name, fi.val)
		if fi.name == flagname {
			newval = fi.val
			found = true
			break
		}
	}
	if !found {
		log.Fatalln("Set lookup fail Invalid flagname", flagname, "in Flag", field)
	}

	var len, msb, shiftbits, maskbits, setbits, result uint32

	len = flagbits[field][fieldlen]
	msb = flagbits[field][fieldmsb]
	shiftbits = flagsize - len - msb
	maskbits = (1 << len) - 1
	setbits = maskbits << shiftbits
	// log.Printf("Shiftbits=%d Maskbits=%b Setbits=%b\n", shiftbits, maskbits, setbits)
	result = ((curflag) & (^setbits))
	result |= (newval << shiftbits)
	// log.Printf("Result=%032b\n", result)
	return result, nil
}

// SetFlags - Set all flags in flag map flags["field"] = "value"
func SetFlags(curflag uint32, flags map[string]string) uint32 {
	for f := range flags {
		curflag, _ = Set(curflag, f, flags[f])
	}
	return curflag
}

// Test - true if the flag is set in curflag
func Test(curflag uint32, field, string, flagname string) bool {
	v, _ := Set(0, field, flagname)
	return Get(curflag, field) == Get(v, field)
}

// Name - return the name of the flag for field in curflag
func Name(curflag uint32, field string) string {

	x := Get(curflag, field)
	for _, fi := range flagvals[field] {
		// log.Println("Flags for field ", field, fi.name, fi.val)
		if fi.val == x {
			return fi.name
		}
	}
	log.Fatalln("Name out of range")
	return ""
}

// Fields - return a slice of flag fields that are used by frametype
func Fields(frametype string) []string {
	var s []string
	for k := range flagframe {
		for _, fi := range flagframe[k] {
			if fi == frametype {
				// fmt.Println(k)
				s = append(s, k)
			}
		}
	}
	return s
}

// Good - Is this a valid flagname
func Good(field string) bool {
	for k := range flagframe {
		if k == field {
			return true
		}
	}
	return false
}

// Setglobal - Set the global flags applicable for the particular frame type
// Dont set final descriptor here - Work it out in the transfer as it depends on file size
func Setglobal(frametype string) string {

	fs := ""
	for _, f := range Fields(frametype) {
		for g := range Cli.Global {
			if g == f {
				fs += f + "=" + Cli.Global[f] + ","
			}
		}
	}
	return strings.TrimRight(fs, ",")
}

// *******************************************************************

// Saratoga Dflag Header Field Format - 16 bit unsigned integer (uint16)
//  0          1
//  01234567 89012345
// +--------+--------+
// |1     XX|YY0     |
// +--------+--------+

// XX = d_properties
// YY = d_descriptor

// DIRECTORY ENTRY FLAGS
//  0                   1
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5
// | | | | | | | | | | | | | | | | |
// |1|-> Bit 0 is always set
// | | | | | | |X|X|-> Dirent Properties - d_properties
// | | | | | | | | |X|X|-> Dirent Descriptor - d_descriptor
// | | | | | | | | | | |0|-> Dirent Reserved
// | | | | | | | | | | | | | |X| | |-> Dirent d_reliability
// | | | | | | | | | | | | | | | | |
//  0                   1
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5

// Length in bits of the directory entry flag size
const dflagsize uint16 = 16

// Map of dir Flags
// The 0 element (fieldlen) value in the uint16[2] is the length in bits of the flag
// The 1 element (fieldmsb) value in the uint16[2] is the bit offset from the front.
// This is all in network byte order
var dflagbits = map[string][2]uint16{
	"sod":        [2]uint16{1, 0},
	"property":   [2]uint16{2, 6},
	"descriptor": [2]uint16{2, 8},
	//	"reserved":    [2]uint16{1, 10},
	"reliability": [2]uint16{1, 13},
}

type dflaginfo struct {
	name string
	val  uint16
}

var dflagvals = map[string][]dflaginfo{
	"sod": []dflaginfo{
		dflaginfo{name: "sod", val: 1},
	},
	"property": []dflaginfo{
		dflaginfo{name: "normalfile", val: 0},
		dflaginfo{name: "normaldirectory", val: 1},
		dflaginfo{name: "specialfile", val: 2},
		dflaginfo{name: "specialdirectory", val: 3},
	},
	"descriptor": []dflaginfo{
		dflaginfo{name: "d16", val: 0},
		dflaginfo{name: "d32", val: 1},
		dflaginfo{name: "d64", val: 2},
		// dflaginfo{name: "d128", val: 3}, INVALID AS OF THIS TIME WAIT FOR 128 bit int's
	},
	//	"reserved": []dflaginfo{
	//		dflaginfo{name: "reserved", val: 0},
	//	},
	"reliability": []dflaginfo{
		dflaginfo{name: "yes", val: 0},
		dflaginfo{name: "no", val: 1},
	},
}

// GetD - Given a current flag and bitfield name return the integer value of the bitfield
func GetD(curflag uint16, field string) uint16 {
	if _, ok := dflagbits[field]; !ok {
		log.Fatal("GetD lookup fail Invalid DFlag", field)
	}

	var len, msb, shiftbits, maskbits, setbits uint16

	len = dflagbits[field][fieldlen]
	msb = dflagbits[field][fieldmsb]
	shiftbits = dflagsize - len - msb
	maskbits = (1 << len) - 1
	setbits = maskbits << shiftbits
	return (curflag & setbits) >> shiftbits
}

// GetDStr - Given a current flag and bitfield name return the string name of the bitfield set in curflag
func GetDStr(curflag uint16, field string) string {
	if _, ok := dflagbits[field]; !ok {
		log.Fatalln("Get lookup fail Invalid DFlag", field)
	}
	var len, msb, shiftbits, maskbits, setbits, val uint16

	len = dflagbits[field][fieldlen]
	msb = dflagbits[field][fieldmsb]
	shiftbits = dflagsize - len - msb
	maskbits = (1 << len) - 1
	setbits = maskbits << shiftbits
	val = (curflag & setbits) >> shiftbits
	for _, fi := range dflagvals[field] {
		if fi.val == val {
			return fi.name
		}
	}
	log.Fatalln("GetDStr fail Invalid field", field, "in DFlag:", curflag)
	return ""
}

// SetD - Given a current header and bitfield name with a new value return the revised header
func SetD(curflag uint16, field string, flagname string) (uint16, error) {
	if _, ok := dflagbits[field]; !ok {
		log.Fatalln("Invalid DFlag SetD lookup fail", field)
		e := "invalid DFlag: " + field
		return curflag, errors.New(e)
	}

	var newval uint16
	var found = false
	// Get the value of the flag
	for _, fi := range dflagvals[field] {
		// log.Println("DFlags for field ", field, fi.name, fi.val)
		if fi.name == flagname {
			newval = fi.val
			found = true
			break
		}
	}
	if !found {
		log.Fatalln("SetD lookup fail Invalid flagname", flagname, "in DFlag", field)
		e := "invalid DFlag: " + field
		return curflag, errors.New(e)
	}

	var len, msb, shiftbits, maskbits, setbits, result uint16

	len = dflagbits[field][fieldlen]
	msb = dflagbits[field][fieldmsb]
	shiftbits = dflagsize - len - msb
	maskbits = (1 << len) - 1
	setbits = maskbits << shiftbits
	// log.Printf("Shiftbits=%d Maskbits=%b Setbits=%b\n", shiftbits, maskbits, setbits)
	result = ((curflag) & (^setbits))
	result |= (newval << shiftbits)
	// log.Printf("Result=%016b\n", result)
	return result, nil
}

// FrameD - return a slice of flag names matching field
func FrameD(field string) []string {
	var s []string
	for _, fi := range dflagvals[field] {
		s = append(s, fi.name)
	}
	return s
}

// FlagD - return a slice of flag names that are used by Dirent
func FlagD() []string {
	var s []string
	for fi := range dflagbits {
		s = append(s, fi)
	}
	return s
}

// TestD - true if the flag is set in curflag
func TestD(curdflag uint16, field string, flagname string) bool {
	v, _ := SetD(0, field, flagname)
	return GetD(curdflag, field) == GetD(v, field)
}

// NameD - return the name of the flag for field in curdflag
func NameD(curdflag uint16, field string) string {

	x := GetD(curdflag, field)
	for _, fi := range dflagvals[field] {
		// log.Println("Flags for field ", field, fi.name, fi.val)
		if fi.val == x {
			return fi.name
		}
	}
	log.Fatalln("NameD out of range")
	return ""
}

// GoodD -- Is this a valid Descriptor Flag
func GoodD(field string) bool {
	for f := range dflagvals {
		if f == field {
			return true
		}
	}
	return false
}

// *******************************************************************

// Saratoga Tflag Header Field Format - 8 bit unsigned integer (tflag_t)

//  01234567
// +--------+
// |     XXX|
// +--------+

// TIMESTAMP FLAGS
//  0 1 2 3 4 5 6 7
// | | | | | | | | |
// | | | | | |X|X|X|-> Timestamp Type - t_timestamp
// | | | | | | | | |
//  0 1 2 3 4 5 6 7

// Length in bits of the timestamp flag size
const tflagsize uint8 = 8

// Map of timestsmp Flags
// The 0 element (fieldlen) value in the uint8[2] is the length in bits of the flag
// The 1 element (fieldmsb) value in the uint8[2] is the bit offset from the front.
// This is all in network byte order
var tflagbits = map[string][2]uint8{
	"timestamp": [2]uint8{8, 0},
}

type tflaginfo struct {
	name string
	val  uint8
}

var tflagvals = map[string][]tflaginfo{
	"timestamp": []tflaginfo{
		tflaginfo{name: "localinterp", val: 0},
		tflaginfo{name: "posix32", val: 1},
		tflaginfo{name: "posix64", val: 2},
		tflaginfo{name: "posix32_32", val: 3},
		tflaginfo{name: "posix64_32", val: 4},
		tflaginfo{name: "epoch2000_32", val: 5},
	},
}

// GetT - Given a current flag and bitfield name return the integer value of the bitfield
func GetT(curflag uint8) uint8 {
	if _, ok := tflagbits["timestamp"]; !ok {
		log.Fatalln("GetT lookup fail Invalid TFlag A", curflag)
	}
	var len, msb, shiftbits, maskbits, setbits uint8

	len = tflagbits["timestamp"][fieldlen]
	msb = tflagbits["timestamp"][fieldmsb]
	shiftbits = tflagsize - len - msb
	maskbits = (1 << len) - 1
	setbits = maskbits << shiftbits
	return (curflag & setbits) >> shiftbits
}

// GetTStr - Given a current flag and bitfield name return the string name of the bitfield set in curflag
func GetTStr(curflag uint8) string {
	if _, ok := tflagbits["timestamp"]; !ok {
		log.Fatalln("Get lookup fail Invalid TFlag:", curflag)
	}

	var len, msb, shiftbits, maskbits, setbits, val uint8

	len = tflagbits["timestamp"][fieldlen]
	msb = tflagbits["timestamp"][fieldmsb]
	shiftbits = tflagsize - len - msb
	maskbits = (1 << len) - 1
	setbits = maskbits << shiftbits
	val = (curflag & setbits) >> shiftbits
	for _, fi := range tflagvals["timestamp"] {
		if fi.val == val {
			return fi.name
		}
	}

	log.Fatalln("GetTStr fail Invalid Tflag:", curflag)
	return ""
}

// SetT Given a current header and bitfield name with a new value return the revised header
func SetT(curflag uint8, flagname string) (uint8, error) {
	if _, ok := tflagbits["timestamp"]; !ok {
		log.Fatalln("SetT lookup fail Invalid TFlag:", flagname)
		e := "Invalid TFlag: " + flagname
		return curflag, errors.New(e)
	}

	var newval uint8
	var found = false
	// Get the value of the flag
	for _, fi := range tflagvals["timestamp"] {
		// log.Println("TFlags for field ", field, fi.name, fi.val)
		if fi.name == flagname {
			newval = fi.val
			found = true
			break
		}
	}
	if !found {
		log.Fatalln("SetT lookup fail Invalid flagname", flagname, "in TFlag")
		e := "invalid TFlag: " + flagname
		return curflag, errors.New(e)
	}

	var len = tflagbits["timestamp"][fieldlen]
	var msb = tflagbits["timestamp"][fieldmsb]
	var shiftbits = tflagsize - len - msb
	var maskbits uint8 = (1 << len) - 1
	var setbits = maskbits << shiftbits
	// log.Printf("Shiftbits=%d Maskbits=%b Setbits=%b\n", shiftbits, maskbits, setbits)
	var result = ((curflag) & (^setbits))
	result |= (newval << shiftbits)
	// log.Printf("Result=%08b\n", result)
	return result, nil
}

// TestT - true if the flag is set in curflag
func TestT(curflag uint8, flagname string) bool {
	v, _ := SetT(curflag, flagname)
	return GetT(curflag) == GetT(v)
}

// NameT - return the name of the flag for field in curtflag
func NameT(curflag uint8) string {
	x := GetT(curflag)
	for _, fi := range tflagvals["timestamp"] {
		// log.Println("Flags for field ", field, fi.name, fi.val)
		if fi.val == x {
			return fi.name
		}
	}
	log.Fatalln("NameT out of range")
	return ""
}

// FrameT - return a slice of flag names that are used by Timeinfo
func FrameT() []string {
	var s []string
	for _, fi := range tflagvals["timestamp"] {
		s = append(s, fi.name)
	}
	return s
}

// GoodT - Is this a valid time flag
func GoodT(field string) bool {
	for t := range tflagvals {
		if field == t {
			return true
		}
	}
	return false
}

// *******************************************************************
