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

package main

import "fmt"

// Length in bits of the saratoga header flag
const flagsize uint32 = 32

// Index for length and msb values in flagfield map
const fieldlen = 0
const fieldmsb = 1

// Map of Saratoga Flags in Frame Header
// First 32 bits of every frame have a combination of these flags
// The 0 element (fieldlen) value in the uint32[2] is the length in bits of the flag
// The 1 element (fieldmsb) value in the uint32[2] is the bit offset from the front.
// This is all in network byte order
var flagfield = map[string][2]uint32{
	"version":       [2]uint32{3, 0},
	"frametype":     [2]uint32{5, 3},
	"descriptor":    [2]uint32{2, 8},
	"stream":        [2]uint32{1, 11},
	"transfer":      [2]uint32{2, 10},
	"reqtstamp":     [2]uint32{1, 12},
	"progress":      [2]uint32{1, 12},
	"txwilling":     [2]uint32{2, 12},
	"udpsupport":    [2]uint32{1, 13},
	"metadatarecvd": [2]uint32{1, 13},
	"allholes":      [2]uint32{1, 14},
	"requesttype":   [2]uint32{8, 24},
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
	"txwilling":     []string{"beacon", "request"},
	"udptype":       []string{"metadata"},
	"metadatarecvd": []string{"status"},
	"allholes":      []string{"status"},
	"requesttype":   []string{"request"},
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

type flaginfo struct {
	name string
	val  uint32
}

var flagvals = map[string][]flaginfo{
	"version": []flaginfo{
		flaginfo{name: "v0", val: 0},
		flaginfo{name: "v1", val: 1},
	},
	"frametype": []flaginfo{
		flaginfo{name: "beacon", val: 0},
		flaginfo{name: "request", val: 1},
		flaginfo{name: "metadata", val: 2},
		flaginfo{name: "data", val: 3},
		flaginfo{name: "status", val: 4},
	},
	"descriptor": []flaginfo{
		flaginfo{name: "d16", val: 0},
		flaginfo{name: "d32", val: 1},
		flaginfo{name: "d64", val: 3},
		flaginfo{name: "d128", val: 4},
	},
	"stream": []flaginfo{
		flaginfo{name: "no", val: 0},
		flaginfo{name: "yes", val: 1},
	},
	"transfer": []flaginfo{
		flaginfo{name: "file", val: 0},
		flaginfo{name: "directory", val: 1},
		flaginfo{name: "bundle", val: 2},
		flaginfo{name: "stream", val: 3},
	},
	"reqtstamp": []flaginfo{
		flaginfo{name: "no", val: 0},
		flaginfo{name: "yes", val: 1},
	},
	"progress": []flaginfo{
		flaginfo{name: "inprogress", val: 0},
		flaginfo{name: "terminated", val: 1},
	},
	"txwilling": []flaginfo{
		flaginfo{name: "no", val: 0},
		flaginfo{name: "invalid", val: 1},
		flaginfo{name: "capable", val: 2},
		flaginfo{name: "yes", val: 3},
	},
	"udptype": []flaginfo{
		flaginfo{name: "udponly", val: 0},
		flaginfo{name: "udplitecapable", val: 1},
	},
	"metadatarecvd": []flaginfo{
		flaginfo{name: "yes", val: 0},
		flaginfo{name: "no", val: 1},
	},
	"allholes": []flaginfo{
		flaginfo{name: "yes", val: 0},
		flaginfo{name: "no", val: 1},
	},
	"requesttype": []flaginfo{
		flaginfo{name: "noaction", val: 0},
		flaginfo{name: "get", val: 1},
		flaginfo{name: "put", val: 2},
		flaginfo{name: "getdelete", val: 3},
		flaginfo{name: "putdelete", val: 4},
		flaginfo{name: "delete", val: 5},
		flaginfo{name: "getdir", val: 6},
	},
	"rxwilling": []flaginfo{
		flaginfo{name: "no", val: 0},
		flaginfo{name: "invalid", val: 1},
		flaginfo{name: "capable", val: 2},
		flaginfo{name: "yes", val: 3},
	},
	"reqholes": []flaginfo{
		flaginfo{name: "requested", val: 0},
		flaginfo{name: "voluntarily", val: 1},
	},
	"fileordir": []flaginfo{
		flaginfo{name: "file", val: 0},
		flaginfo{name: "directory", val: 1},
	},
	"reqstatus": []flaginfo{
		flaginfo{name: "no", val: 0},
		flaginfo{name: "yes", val: 1},
	},
	"udplite": []flaginfo{
		flaginfo{name: "udpoly", val: 0},
		flaginfo{name: "udplitecapable", val: 1},
	},
	"eod": []flaginfo{
		flaginfo{name: "no", val: 0},
		flaginfo{name: "yes", val: 1},
	},
	"freespace": []flaginfo{
		flaginfo{name: "no", val: 0},
		flaginfo{name: "yes", val: 1},
	},
	"freespaced": []flaginfo{
		flaginfo{name: "d16", val: 0},
		flaginfo{name: "d32", val: 1},
		flaginfo{name: "d64", val: 2},
		flaginfo{name: "d128", val: 3},
	},
	"csumlen": []flaginfo{
		flaginfo{name: "none", val: 0},
		flaginfo{name: "crc32", val: 1},
		flaginfo{name: "invalid2", val: 2},
		flaginfo{name: "invalid3", val: 3},
		flaginfo{name: "md5", val: 4},
		flaginfo{name: "sha1", val: 5},
	},
	"csumtype": []flaginfo{
		flaginfo{name: "none", val: 0},
		flaginfo{name: "crc32", val: 1},
		flaginfo{name: "md5", val: 2},
		flaginfo{name: "sha1", val: 3},
	},
	"errcode": []flaginfo{
		flaginfo{name: "success", val: 0},
		flaginfo{name: "unspecified", val: 1},
		flaginfo{name: "nosend", val: 2},
		flaginfo{name: "noreceive", val: 3},
		flaginfo{name: "nofile", val: 4},
		flaginfo{name: "noaccess", val: 5},
		flaginfo{name: "noid", val: 6},
		flaginfo{name: "toobig", val: 7},
		flaginfo{name: "baddescriptor", val: 8},
		flaginfo{name: "badpacket", val: 9},
		flaginfo{name: "badflag", val: 10},
		flaginfo{name: "shutdown", val: 11},
		flaginfo{name: "pause", val: 12},
		flaginfo{name: "resume", val: 13},
		flaginfo{name: "inuse", val: 14},
		flaginfo{name: "nometadata", val: 15},
	},
}

// Given a current flag and bitfield name return the integer value of the bitfield
func getfield(curflag uint32, field string) uint32 {
	if _, ok := flagfield[field]; !ok {
		fmt.Println("Invalid field name:", field)
		panic("Flagfield lookup fail")
	}

	var len = flagfield[field][fieldlen]
	var msb = flagfield[field][fieldmsb]
	var shiftbits = flagsize - len - msb
	var maskbits uint32 = (1 << len) - 1
	var setbits = maskbits << shiftbits
	return (curflag & setbits) >> shiftbits
}

// Given a current header and bitfield name with a new value return the revised header
func setfield(curflag uint32, field string, flagname string) uint32 {
	if _, ok := flagfield[field]; !ok {
		fmt.Println("Invalid field name:", field)
		panic("Flagfield lookup fail")
	}

	var newval uint32 = 9
	for _, fi := range flagvals[field] {
		fmt.Println("Flags for field ", field, fi.name, fi.val)

		if fi.name == flagname {
			newval = fi.val
		}
	}

	var len = flagfield[field][fieldlen]
	var msb = flagfield[field][fieldmsb]
	var shiftbits = flagsize - len - msb
	var maskbits uint32 = (1 << len) - 1
	var setbits = maskbits << shiftbits
	// fmt.Printf("Shiftbits=%d Maskbits=%b Setbits=%b\n", shiftbits, maskbits, setbits)
	var result = ((curflag) & (^setbits))
	// var newval = flagvals[field].flaginfo[name].val
	result |= (newval << shiftbits)
	// fmt.Printf("Result=%032b\n", result)
	return result
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
// | | | | | | | | | | |0|-> Dirent File
// | | | | | | | | | | | | | | | | |
//  0                   1
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5

// Length in bits of the directory entry flag size
const dflagsize uint16 = 16

// Map of dir Flags
// The 0 element (fieldlen) value in the uint16[2] is the length in bits of the flag
// The 1 element (fieldmsb) value in the uint16[2] is the bit offset from the front.
// This is all in network byte order
var dflagfield = map[string][2]uint16{
	"sof":        [2]uint16{1, 0},
	"properties": [2]uint16{2, 6},
	"descriptor": [2]uint16{2, 8},
	"file":       [2]uint16{1, 10},
}

// Given a current flag and bitfield name return the integer value of the bitfield
func dgetfield(curflag uint16, fieldname string) uint16 {
	if _, ok := dflagfield[fieldname]; !ok {
		fmt.Println("Invalid Dflagfield name:", fieldname)
		panic("Dflagfield lookup fail")
	}

	len := dflagfield[fieldname][fieldlen]
	msb := dflagfield[fieldname][fieldmsb]
	var shiftbits = dflagsize - len - msb
	var maskbits uint16 = (1 << len) - 1
	var setbits = maskbits << shiftbits
	return (curflag & setbits) >> shiftbits
}

// Given a current header and bitfield name with a new value return the revised header
func dsetfield(curflag uint16, fieldname string, newval uint16) uint16 {
	if _, ok := dflagfield[fieldname]; !ok {
		fmt.Println("Invalid Dflagfield name:", fieldname)
		panic("Dflagfield lookup fail")
	}
	len := dflagfield[fieldname][fieldlen]
	msb := dflagfield[fieldname][fieldmsb]
	var shiftbits = dflagsize - len - msb
	var maskbits uint16 = (1 << len) - 1
	var setbits = maskbits << shiftbits
	// fmt.Printf("DShiftbits=%d DMaskbits=%b DSetbits=%b\n", shiftbits, maskbits, setbits)
	var result = ((curflag) & (^setbits))
	result |= (newval << shiftbits)
	// fmt.Printf("DResult=%016b\n", result)
	return result
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
var tflagfield = map[string][2]uint8{
	"timestamp": [2]uint8{3, 5},
}

// Given a current flag and bitfield name return the integer value of the bitfield
func tgetfield(curflag uint8, fieldname string) uint8 {
	if _, ok := tflagfield[fieldname]; !ok {
		fmt.Println("Invalid Tflagfield name:", fieldname)
		panic("Tflagfield lookup fail")
	}

	len := tflagfield[fieldname][fieldlen]
	msb := tflagfield[fieldname][fieldmsb]
	var shiftbits = tflagsize - len - msb
	var maskbits uint8 = (1 << len) - 1
	var setbits = maskbits << shiftbits
	return (curflag & setbits) >> shiftbits
}

// Given a current header and bitfield name with a new value return the revised header
func tsetfield(curflag uint8, fieldname string, newval uint8) uint8 {
	if _, ok := tflagfield[fieldname]; !ok {
		fmt.Println("Invalid Tflagfield name:", fieldname)
		panic("Tflagfield lookup fail")
	}
	len := tflagfield[fieldname][fieldlen]
	msb := tflagfield[fieldname][fieldmsb]
	var shiftbits = tflagsize - len - msb
	var maskbits uint8 = (1 << len) - 1
	var setbits = maskbits << shiftbits
	// fmt.Printf("TShiftbits=%d TMaskbits=%b TSetbits=%b\n", shiftbits, maskbits, setbits)
	var result = ((curflag) & (^setbits))
	result |= (newval << shiftbits)
	// fmt.Printf("DResult=%08b\n", result)
	return result
}

// *******************************************************************

func main() {

	// Three flag types in Saratoga

	// var SarDFlag uint16 // Directory 16 bit Entry Flag
	// var SarTFlag uint8  // Time 8 bit Flag
	fmt.Println("Handle Saratoga Headers")

	var x uint32
	var sarflag uint32 = 0x0

	fmt.Println("Setting Version 1")
	sarflag = setfield(sarflag, "version", "v1")
	x = getfield(sarflag, "version")
	fmt.Printf("Sarflag=%0b Version=%0b\n", sarflag, x)

	sarflag = setfield(sarflag, "frametype", "beacon")
	x = getfield(sarflag, "frametype")
	fmt.Printf("Sarflag =%032b Version=%0b\n", sarflag, x)

	sarflag = setfield(sarflag, "descriptor", "d32")
	x = getfield(sarflag, "descriptor")
	fmt.Printf("Sarflag =%032b Descriptor=%0b\n", sarflag, x)

	var y uint16
	var dflag uint16 = 0x0

	var dval uint16 = 0x01
	fmt.Println("Setting sof to 1")
	dflag = dsetfield(dflag, "sof", dval)
	y = dgetfield(dflag, "sof")
	fmt.Printf("dflag =%016b sof=%0b\n", dflag, y)

	dval = 0x03
	fmt.Println("Setting properties to 3")
	dflag = dsetfield(dflag, "properties", dval)
	y = dgetfield(dflag, "properties")
	fmt.Printf("dflag =%016b properties=%0b\n", dflag, y)

	dval = 0x02
	fmt.Println("Setting descriptor to 2")
	dflag = dsetfield(dflag, "descriptor", dval)
	y = dgetfield(dflag, "descriptor")
	fmt.Printf("dflag =%016b descriptor=%0b\n", dflag, y)

	dval = 0x01
	fmt.Println("Setting file to 1")
	dflag = dsetfield(dflag, "file", dval)
	y = dgetfield(dflag, "file")
	fmt.Printf("dflag =%016b file=%0b\n", dflag, y)

}
