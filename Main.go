package main

import "fmt"

// Length in bits of the saratoga header flag
const flagsize uint32 = 32

// Index for length and msb values in flagfield map
const fieldlen = 0
const fieldmsb = 1

var flagfield = map[string][2]uint32{
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

func main() {

	// Three flag types in Saratoga

	// var SarDFlag uint16 // Directory 16 bit Entry Flag
	// var SarTFlag uint8  // Time 8 bit Flag
	fmt.Println("Handle Saratoga Headers")

	// x = 0x23000000

	var x uint32
	var sarflag uint32 = 0x0

	var val uint32 = 0x01
	fmt.Println("Setting Version 1")
	sarflag = setfield(sarflag, "version", val)
	x = getfield(sarflag, "version")
	fmt.Printf("Sarflag=%08x Version=%08x\n", sarflag, x)

	val = 0x02
	fmt.Println("Setting Frametype 2")
	sarflag = setfield(sarflag, "frametype", val)
	x = getfield(sarflag, "frametype")
	fmt.Printf("Sarflag =%08x Version=%08x\n", sarflag, x)

}

// How many bits to shift across to get the flag
func shift(bits uint32, msb uint32) uint32 {
	return (32 - bits - msb)
}

// How many bits long is the flag
func mask(bits uint32) uint32 {
	return ((1 << bits) - 1)
}

// Given a current flag and bitfield name return the integer value of the bitfield
func getfield(curflag uint32, fieldname string) uint32 {
	len := flagfield[fieldname][fieldlen]
	msb := flagfield[fieldname][fieldmsb]
	var shiftbits = flagsize - len - msb
	var maskbits uint32 = (1 << len) - 1
	var setbits = maskbits << shiftbits
	return (curflag & setbits) >> shiftbits
}

func setfield(curflag uint32, fieldname string, newval uint32) uint32 {
	len := flagfield[fieldname][fieldlen]
	msb := flagfield[fieldname][fieldmsb]
	var shiftbits = flagsize - len - msb
	var maskbits uint32 = (1 << len) - 1
	var setbits = maskbits << shiftbits
	return ((newval)&(^setbits) | (curflag << shiftbits))
}

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

// BEACON FRAME FLAGS - Frame type 00000 = 0x20 in Ver 1
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
// |0|0|1|-> Version 1 - f_version
// | | | |0|0|0|0|0|-> Beacon Frame - f_frametype
// | | | | | | | | |X|X|-> Descriptor - f_descriptor
// | | | | | | | | | | |0|-> Undefined used to be Bundles
// | | | | | | | | | | | |X|-> Streaming - f_stream
// | | | | | | | | | | | | |X|X| | |-> Tx Willing - f_txwilling
// | | | | | | | | | | | | | | |X|X|-> Rx Willing - f_rxwilling
// | | | | | | | | | | | | | | | | |X|-> UDP Lite - f_udptype
// | | | | | | | | | | | | | | | | | |X|-> Freespace Advertise - f_freespace
// | | | | | | | | | | | | | | | | | | |X|X|-> Freespace Descriptor - f_freespaced
// | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1

// *******************************************************************
// REQUEST FRAME FLAGS - Frame tye 00001 = 0x21 in Ver 1
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
// |0|0|1|-> Version 1 - f_version
// | | | |0|0|0|0|1|-> Request Frame - f_frametype
// | | | | | | | | |X|X|-> Descriptor - f_descriptor
// | | | | | | | | | | |0|-> Undefined used to be Bundles
// | | | | | | | | | | | |X|-> Streams - f_stream
// | | | | | | | | | | | | | | | | |X|-> UDP Lite - f_udptype
// | | | | | | | | | | | | | | | | | | | | | | | | |X|X|X|X|X|X|X|X|-> f_requesttype
// | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1

// *******************************************************************
// METADATA FRAME FLAGS - Frame type 00010 = 0x22 in Ver 1
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
// |0|0|1|-> Version 1 - f_version
// | | | |0|0|0|1|0|-> Metadata Frame - f_frametype
// | | | | | | | | |X|X|-> Descriptor - f_descriptor
// | | | | | | | | | | |X|X|-> Type of Transfer - f_transfer
// | | | | | | | | | | | | |X|-> Transfer in Progress - f_progress
// | | | | | | | | | | | | | |X|-> Reliability - f_udptype
// | | | | | | | | | | | | | | | | | | | | | | | | |X|X|X|X|-> Checksum Length - f_csumlen
// | | | | | | | | | | | | | | | | | | | | | | | | | | | | |X|X|X|X|-> Checksum Type - f_csumtype
// | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1

// *******************************************************************
// DATA FRAME FLAGS - Frame type 00011 = 0x23 in Ver 1
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
// |0|0|1|-> Version 1 - f_version
// | | | |0|0|0|1|1|-> Data Frame - f_frametype
// | | | | | | | | |X|X|-> Descriptor - f_descriptor
// | | | | | | | | | | |X|X|-> Type of Transfer - f_transfer
// | | | | | | | | | | | | |X|-> Timestamps - f_reqtstamp
// | | | | | | | | | | | | | | | |X|-> Request Status - f_reqstatus
// | | | | | | | | | | | | | | | | |X|-> End of Data - f_eod
// | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1

// *******************************************************************
// STATUS FRAME FLAGS - Frame Type 00100 = 0x24 in Ver 1
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
// |0|0|1|-> Version 1 - f_version
// | | | |0|0|1|0|0|-> Status Frame - f_frametype
// | | | | | | | | |X|X|-> Descriptor - f_descriptor
// | | | | | | | | | | | | |X|-> Timestamp - f_reqtstamp
// | | | | | | | | | | | | | |X|->Metadata Received - f_metadatarecvd
// | | | | | | | | | | | | | | |X|-> All Holes - f_allholes
// | | | | | | | | | | | | | | | |X|-> Holes Requested or Sent - f_reqholes
// | | | | | | | | | | | | | | | | | | | | | | | | |X|X|X|X|X|X|X|X|-> Error Code - f_errcode
// | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1

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

// *******************************************************************
