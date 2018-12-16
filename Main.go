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

// Fversion - Version Number - BEACON,REQUEST,METADATA,DATA,STATUS
type Fversion uint32

const (
	Fversion1 Fversion = iota
	Fversion2
)

// Fframetype - Saratoga Frame Type - BEACON,REQUEST,METADATA,DATA,STATUS
type Fframetype uint32

const (
	Fbeacon Fframetype = iota
	Frequest
	Fmetadata
	Fdata
	Fstatus
)

// Fdescriptor - Descriptor Size - BEACON,REQUEST,METADATA,DATA,STATUS
type Fdescriptor uint32

const (
	Fdescriptor16 Fdescriptor = iota
	Fdescriptor32
	Fdescriptor64
	Fdescriptor128
)

// Fstreams - Are streams supported - BEACON,REQUEST
type Fstreams uint32

const (
	Fstreamsno = iota
	Fstreamsyes
)

// Ftransfer - Transfer Type - METADATA, DATA
type Ftransfer uint32

const (
	Ftransferfile Ftransfer = iota
	Ftransferdir
	Ftransferbundle
	Ftransferstream
)

// Freqtstamp - Timestamp/nonce - DATA,STATUS
type Freqtstamp uint32

const (
	Ftimestampno Freqtstamp = iota
	Ftimestampyes
)

// Fprogress - Transfer progress - METADATA
type Fprogress uint32

const (
	Finprogress Fprogress = iota
	Fterminated
)

// Ftxwilling - Transmitter willingness - BEACON,REQUEST
type Ftxwilling uint32

const (
	Ftxwillingno Ftxwilling = iota
	Ftxwillinginvalid
	Ftxwillingcapable
	Ftxwillingyes
)

// Fudplite - UDP & UDP Lite Supported - METADATA
type Fudptype uint32

const (
	Fudptypeudponly Fudplite = iota
	Fudptypeudplitecapable
)

// Fmetadatarecvd - Has metadata been received yet - STATUS
type Fmetadatarecvd uint32

const (
	Fmetadatarecvdyes Fmetadatarecvd = iota
	Fmetadatarecvdno
)

// Fallholes - Are all holes in this packet - STATUS
type Fallholes uint32

const (
	Fallholesyes Fallholes = iota
	Fallholesno
)

// Freqesttype - What transaction request is this - REQUEST
type Frequesttype uint32

const (
	Frequestnoaction Frequesttype = iota
	Frequestget
	Frequestput
	Frequestgetdelete
	Frequestputdelete
	Frequestdelete
	Frequestgetdir
)

// Frxwilling - Is our receiver ready - BEACON
type Frxwilling uint32

const (
	Frxwillingno Frxwilling = iota
	Frxwillinginvalid
	Frxwillingcapable
	Frxwillingyes
)

// Freqholes - Were holes requested or sent voluntarily - STATUS
type Freqholes uint32

const (
	Fholesrequested Freqholes = iota
	Fholessentvoluntarily
)

// Ffileordir - Are we requesting a file or directory - REQUEST
type Ffileordir uint32

const (
	Ffileordirfile Ffileordir = iota
	Ffileordirdirectory
)

// Freqstatus - Do we wish to request for a status update - DATA
type Freqstatus uint32

const (
	Freqstatusno Freqstatus = iota
	Freqstatusyes
)

// Fudplite - UDP and/or UDP Lite Supported - BEACON,REQUEST
type Fudplite uint32

const (
	Fudpliteudponly Fudplite = iota
	Fudpliteudplitecapable
)

// Feod - Last frame in the data transfer eof or eos - DATA
type Feod uint32

const (
	Feodno Feod = iota
	Feodyes
)

// Ffreespace - Do we Advertise Free Space - BEACON
type Ffreespace uint32

const (
	Ffreespaceno Ffreespace = iota
	Ffreespaceyes
)

// Ffreespaced Saratoga Header - Advertise Descriptor Size of Free Space - BEACON
type Ffreespaced uint32

const (
	Ffreespaced16 Ffreespaced = iota
	Ffreespaced32
	Ffreespaced64
	Ffreespaced128
)

// Fcsumlen Saratoga Header - Checksum Length - METADATA
type Fcsumlen uint32

const (
	Fcsumlennone Fcsumlen = iota
	Fcsumlencrc32
	Fcsumleninvalid2
	Fcsumleninvalid3
	Fcsumlenmd5
	Fcsumlensha1
)

// Fcsumtype Saratoga Header - Checksum Type - METADATA
type Fcsumtype uint32

const (
	Fcsumnone Fcsumtype = iota
	Fcsumcrc32
	Fcsummd5
	Fcsumsha1
)

// Ferrcode Saratoga Header - Status & Error codes - STATUS
type Ferrcode uint32

const (
	Ferrsuccess Ferrcode = iota
	Ferrunspec
	Ferrnosend
	Ferrnoreceive
	Ferrnofile
	Ferrnoaccess
	Ferrnoid
	Ferrnodelete
	Ferrtoobig
	Ferrbaddesc
	Ferrbadpacket
	Ferrbadflag
	Ferrshutdown
	Ferrpause
	Ferrresume
	Ferrinuse
	Ferrnometadata
)

// Given a current flag and bitfield name return the integer value of the bitfield
func getfield(curflag uint32, fieldname string) uint32 {
	if _, ok := flagfield[fieldname]; !ok {
		fmt.Println("Invalid Flagfield name:", fieldname)
		panic("Flagfield lookup fail")
	}

	var len = flagfield[fieldname][fieldlen]
	var msb = flagfield[fieldname][fieldmsb]
	var shiftbits = flagsize - len - msb
	var maskbits uint32 = (1 << len) - 1
	var setbits = maskbits << shiftbits
	return (curflag & setbits) >> shiftbits
}

// Given a current header and bitfield name with a new value return the revised header
func setfield(curflag uint32, fieldname string, newval uint32) uint32 {
	if _, ok := flagfield[fieldname]; !ok {
		fmt.Println("Invalid Flagfield name:", fieldname)
		panic("Flagfield lookup fail")
	}
	var len = flagfield[fieldname][fieldlen]
	var msb = flagfield[fieldname][fieldmsb]
	var shiftbits = flagsize - len - msb
	var maskbits uint32 = (1 << len) - 1
	var setbits = maskbits << shiftbits
	// fmt.Printf("Shiftbits=%d Maskbits=%b Setbits=%b\n", shiftbits, maskbits, setbits)
	var result = ((curflag) & (^setbits))
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

	var val uint32 = 0x01
	fmt.Println("Setting Version 1")
	sarflag = setfield(sarflag, "version", val)
	x = getfield(sarflag, "version")
	fmt.Printf("Sarflag=%0b Version=%0b\n", sarflag, x)

	val = 0x03
	fmt.Println("Setting Frametype 3")
	sarflag = setfield(sarflag, "frametype", val)
	x = getfield(sarflag, "frametype")
	fmt.Printf("Sarflag =%032b Version=%0b\n", sarflag, x)

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
