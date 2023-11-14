// Saratoga Flag Handlers

package sarflags

import (
	"encoding/json"
	"errors"
	"fmt"

	// "io/ioutil"
	"log"
	"math"
	"os"
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
// | | | | | | |Y|Y|-> Dirent Properties - d_properties
// | | | | | | | | |X|X|-> Dirent Descriptor - d_descriptor
// | | | | | | | | | | |0|-> Dirent Reserved
// | | | | | | | | | | | | | |X| | |-> Dirent d_reliability
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

// The Maximum sizes of various types of integers
// They are all cast to a uint64 so we are consistant in our math's
var MaxUint uint64 = math.MaxUint
var MaxInt uint64 = math.MaxInt
var MaxUint16 uint64 = math.MaxUint16
var MaxInt16 uint64 = math.MaxInt16
var MaxUint32 uint64 = math.MaxUint32
var MaxInt32 uint64 = math.MaxInt32
var MaxUint64 uint64 = math.MaxUint64
var MaxInt64 uint64 = math.MaxInt64

// The Maximum descriptor size d16, d32, d64, d128
var MaxDescriptor string

// Length in bits of the saratoga header flag
const flagsize uint32 = 32

// MaxBuff -- Maximum read []byte buffer, set to Jumbo to be sure
const MaxBuff = uint64(9000)

// Timeouts - Global Timeouts and counters
type Timeouts struct {
	Metadata  int  `json:"metadata"`  // Secs to wait for a metadatarecvd before a resend
	Request   int  `json:"request"`   // Secs to wait before a resend
	Status    int  `json:"status"`    // Secs to wait before request status again
	Transfer  int  `json:"transfer"`  // Secs to wait before cancelling transfer when nothing recieved
	Binterval uint `json:"binterval"` // Secs between sending beacon frames
}

// GTimeout - timeouts for responses 0 means no timeout
var GTimeout = Timeouts{}

// Cmds - JSON Config for command usage & help
type Cmdtype struct {
	Usage string `json:"usage"`
	Help  string `json:"help"`
}

// Global map of commands
var Commands map[string]Cmdtype

// Flag Information
type Flagtype struct {
	Frametypes []string          // What Frametypes are applicable to this Flag
	Len        uint32            // Bit Length of the flag within the header
	Msb        uint32            // Most significant bit within the header
	Options    map[string]uint32 // What are the Options for the Flag
}

// Global map of flag decode info
// These DO NOT Change
var Flags map[string]Flagtype

// DirentFlag Information
type DirentFlagtype struct {
	Len     uint16            // Bit Length of the flag within the header
	Msb     uint16            // Mosost significant bit within the header
	Options map[string]uint16 // What are the Options for the DateFlag
}

// Global map of directory decode info
// These DO NOT Change
var DirentFlags map[string]DirentFlagtype

// TimeStamp Information
type TimeStamptype struct {
	Len     uint8
	Msb     uint8
	Options map[string]uint8
}

// Global var of time decode info
// These DO NOT Change
var TimeStamps TimeStamptype

// Global map of what flags are applicable to which frame types
// These DO NOT Change
var Frameflags map[string][]string

// Config - JSON Config Default Global Settings & Commands
type config struct {
	V4multicast string   `json:"v4multicast"` // IPv4 Muluticast address
	V6multicast string   `json:"v6multicast"` // IPv6 Multicast address
	Port        int      `json:"port"`        // Deefault Saratoga Port to listen and send on
	Descriptor  string   `json:"descriptor"`  // Default Descriptor: d16,d32,d64
	Csumtype    string   `json:"csumtype"`    // Default Checksum type: none
	Freespace   string   `json:"freespace"`   // Is freespace tp be advertised: yes,no
	Txwilling   string   `json:"txwilling"`   // Can files/streams be sent: yes,no
	Rxwilling   string   `json:"rxwilling"`   // Can files/streams be received: yes,no
	Stream      string   `json:"stream"`      // Can files/streams be transmitted: yes,no
	Reqtstamp   string   `json:"reqtstamp"`   // Request timestamps: yes,no
	Reqstatus   string   `json:"reqstatus"`   // Request status frame to be sent/received: yes,no
	Udplite     string   `json:"udplite"`     // Is UDP Lite supported: yes,no
	Timestamp   string   `json:"timestamp"`   // What is the default timestamp format: anything for local,posix32,posix32_323,posix64,posix64_32,epoch2000_32,
	Timezone    string   `json:"timezone"`    // What timezone is to be used in timestamps: utc
	Sardir      string   `json:"sardir"`      // What is the default directory for saratoga files
	Prompt      string   `json:"prompt"`      // Command line prompt: saratoga
	Ppad        int      `json:"ppad"`        // Padding length in prompt for []:
	Timeout     Timeouts // Various Timers
	Datacounter int      `json:"datacounter"` // How many data frames received before a status is requested
}

// Climu - Protect CLI input flags
var Climu sync.Mutex

// Cliflags - CLI Input flags
// These values can change via cli interface for the user
type Cliflags struct {
	Global      map[string]string // Global header flags set for frames
	V4Multicast string
	V6Multicast string
	Port        int
	Timestamp   string   // What timestamp to use
	Timeout     Timeouts // Various timeouts
	Datacnt     int      // # data frames to send before a request flag is set
	Timezone    string   // Timezone for logs utc or local time
	Prompt      string   // Prompt
	Ppad        int      // Length of Padding around Prompt []: = 3
	Sardir      string   // Saratoga working directory
}

// MTU -- Maximum write []byte buffer, set to interface MTU
var mtu int

// Set the MTU
func MtuSet(m int) {
	mtu = m
}

// Getet the MTU
func Mtu() int {
	return mtu
}

// Calculate the maximum descriptor size on this platform
// d16, d32, d64, d128
func getmaxdesc() (desc string, err error) {
	err = nil
	switch MaxUint {
	case MaxUint16:
		desc = "d16"
	case MaxUint32:
		desc = "d32"
	case MaxUint64:
		desc = "d64"
	default: // yes we never get here till we have 128 bit integers
		desc = "d128"
		err = errors.New("descriptor d128 not supported")
	}
	return desc, err
}

// Read  in the JSON Config data
func ReadConfig(fname string) (*Cliflags, error) {
	var err error

	// var cmu sync.Mutex
	Flags = make(map[string]Flagtype)             // Setup the Flags global map
	Frameflags = make(map[string][]string)        // Setup Frameflags global map
	DirentFlags = make(map[string]DirentFlagtype) // Setup Direntflags global map
	Commands = make(map[string]Cmdtype)           // Setup Commands global map

	// Find the maximum descriptor on this platform
	if MaxDescriptor, err = getmaxdesc(); err != nil {
		return nil, err
	}

	var confdata []byte
	if confdata, err = os.ReadFile(fname); err != nil {
		fmt.Println("Cannot open the saratoga config file", os.Args[1], ":", err)
		return nil, err
	}

	var sarconfdata map[string]interface{}
	if err = json.Unmarshal([]byte(confdata), &sarconfdata); err != nil {
		fmt.Println("Cannot Unmarshal json from saratoga config file", os.Args[1], ":", err)
		return nil, err
	}
	// Lock them up while we are changing the values
	Climu.Lock()
	defer Climu.Unlock()
	// Now decode all of those variables, arrays & maps in the json into the config struct's
	var conf config
	for key, value := range sarconfdata {
		// fmt.Println(key, "=", value)
		switch key {
		case "v4multicast":
			conf.V4multicast = value.(string)
		case "v6multicast":
			conf.V6multicast = value.(string)
		case "udpport":
			conf.Port = int(value.(float64))
		case "descriptor":
			conf.Descriptor = value.(string)
		case "csumtype":
			conf.Csumtype = value.(string)
		case "freespace":
			conf.Freespace = value.(string)
		case "txwilling":
			conf.Txwilling = value.(string)
		case "rxwilling":
			conf.Rxwilling = value.(string)
		case "stream":
			conf.Stream = value.(string)
		case "reqtstamp":
			conf.Reqtstamp = value.(string)
		case "reqstatus":
			conf.Reqstatus = value.(string)
		case "udplite":
			conf.Udplite = value.(string)
		case "timestamp":
			conf.Timestamp = value.(string)
		case "timezone":
			conf.Timezone = value.(string)
		case "sardir":
			conf.Sardir = value.(string)
		case "prompt":
			conf.Prompt = value.(string)
		case "ppad":
			conf.Ppad = int(value.(float64))
		case "timeout": // This is a map in json so copy it to the Timeout structure vars
			// fmt.Println(key, "=", value)
			timers := value.(map[string]interface{})
			for keyt, valuet := range timers {
				// fmt.Println("  keyt=", keyt, "= valuet=", valuet)
				switch keyt {
				case "metadata":
					conf.Timeout.Metadata = int(valuet.(float64))
				case "request":
					conf.Timeout.Request = int(valuet.(float64))
				case "status":
					conf.Timeout.Status = int(valuet.(float64))
				case "transfer":
					conf.Timeout.Transfer = int(valuet.(float64))
				case "binterval":
					conf.Timeout.Transfer = int(valuet.(float64))
				}
			}
		case "datacounter": // Defaul number of data frames before a status is requested
			conf.Datacounter = int(value.(float64))
		case "commands": //This is a map in json so copy it to the Commands array
			cmds := value.(map[string]interface{})
			for cmd, value := range cmds {
				var c Cmdtype
				info := value.(map[string]interface{})
				for keyx, valx := range info {
					switch keyx {
					case "help":
						c.Help = valx.(string)
					case "usage":
						c.Usage = valx.(string)
					}
				}
				Commands[cmd] = c
			}
			// for _, v := range conf.Commands {
			// fmt.Println(v.Cmd, " | ", v.Usage, " | ", v.Help)
			// }
		case "frameflags": // Map of what Flags are applicable to what Frame types
			frameflags := value.(map[string]interface{})
			for key, val := range frameflags {
				// fmt.Println("THE KEY IS:", key, "THE VAL IS ", val)
				info := val.([]interface{})
				for _, v := range info {
					Frameflags[key] = append(Frameflags[key], v.(string))
				}
			}
			//for k, r := range conf.Frameflags {
			//	fmt.Println(k, "=", r)
			//}
		case "flags": // The flags within the header
			flags := value.(map[string]interface{})
			// Flags := make(map[string][]Flagtype)
			// conf.Flaginfo = make(map[string][]Flagtype)
			for key, val := range flags {
				var tmp Flagtype
				// fmt.Println("FLAG=", key, "INFO=", val)
				info := val.(map[string]interface{})
				for ikey, ival := range info {
					switch ikey {
					case "frametypes":
						ftinfo := ival.([]interface{})
						for _, v := range ftinfo {
							tmp.Frametypes = append(tmp.Frametypes, v.(string))
						}
						// fmt.Println("Frametypes for", key, "=", tmp.Frametypes)
					case "len":
						tmp.Len = uint32(ival.(float64))
					case "msb":
						tmp.Msb = uint32(ival.(float64))
					case "options":
						oinfo := ival.(map[string]interface{})
						tmp.Options = make(map[string]uint32)
						// fmt.Println("Options for", key, "=", oinfo)
						for okey, oval := range oinfo {
							tmp.Options[okey] = uint32(oval.(float64))
						}
						// fmt.Println("Options for", key, tmp.Options)
					}
				}
				Flags[key] = tmp
				// conf.Flaginfo[key] = append(conf.Flaginfo[key], tmp)
			}
			// for f, v := range Flags {
			// fmt.Println("FLAG=", f, v)
			//	fmt.Println(f, "=", v.Options)
			// }
		case "direntflags": // Now this is the HARD ONE!!!!!
			direntflags := value.(map[string]interface{})
			for dkey, val := range direntflags {
				var tmp DirentFlagtype
				// fmt.Println("FLAG=", key, "INFO=", val)
				dainfo := val.(map[string]interface{})
				for dikey, info := range dainfo {
					switch dikey {
					case "len":
						tmp.Len = uint16(info.(float64))
					case "msb":
						tmp.Msb = uint16(info.(float64))
					case "options":
						oinfo := info.(map[string]interface{})
						tmp.Options = make(map[string]uint16)
						// fmt.Println("DateFlag Options for", key, "=", oinfo)
						for okey, oval := range oinfo {
							tmp.Options[okey] = uint16(oval.(float64))
						}
					}
				}
				// fmt.Println("dateflag for", key, "=", tmp)
				DirentFlags[dkey] = tmp
			}
		case "timestamps":
			timestamps := value.(interface{})
			info := timestamps.(map[string]interface{})
			for ikey, ival := range info {
				switch ikey {
				case "len":
					TimeStamps.Len = uint8(ival.(float64))
				case "msb":
					TimeStamps.Msb = uint8(ival.(float64))
				case "options":
					oinfo := ival.(map[string]interface{})
					TimeStamps.Options = make(map[string]uint8)
					for okey, oval := range oinfo {
						TimeStamps.Options[okey] = uint8(oval.(float64))
					}
				}
			}
		}

	}

	c := new(Cliflags)
	// Give default values to flags from saratoga JSON config
	c.Global = make(map[string]string)
	c.Global["csumtype"] = conf.Csumtype
	c.Global["freespace"] = conf.Freespace
	c.Global["txwilling"] = conf.Txwilling
	c.Global["rxwilling"] = conf.Rxwilling
	c.Global["stream"] = conf.Stream
	c.Global["reqtstamp"] = conf.Reqtstamp
	c.Global["reqstatus"] = conf.Reqstatus
	c.Global["udplite"] = conf.Udplite
	c.Global["descriptor"] = conf.Descriptor
	c.V4Multicast = conf.V4multicast
	c.V6Multicast = conf.V6multicast
	c.Port = conf.Port
	c.Timestamp = conf.Timestamp                 // Default timestamp type to use
	c.Timeout.Metadata = conf.Timeout.Metadata   // Seconds
	c.Timeout.Request = conf.Timeout.Request     // Seconds
	c.Timeout.Status = conf.Timeout.Status       // Seconds
	c.Timeout.Transfer = conf.Timeout.Transfer   // Seconds
	c.Timeout.Binterval = conf.Timeout.Binterval // Seconds between beacons
	c.Datacnt = conf.Datacounter                 // # Data frames between request for status
	c.Timezone = conf.Timezone                   // TImezone to use for logs
	c.Prompt = conf.Prompt                       // Prompt Prefix in cmd
	c.Ppad = conf.Ppad                           // For []: in prompt = 3

	// Get the default directory for sarotaga transfers from environment
	// We default to what is in the environment variable otherwise what is in saratoga.json
	var sardir string
	if sardir = os.Getenv("SARDIR"); sardir == "" {
		sardir = conf.Sardir // If no env variable set then set it to conf file value
	}
	c.Sardir = sardir

	for f, v := range c.Global {
		if !Valid(f, c.Global[f]) {
			ps := "Invalid Flag:" + f + "=" + v
			panic(ps)
		}
	}
	return c, nil
}

// Valid - Check for valid flag and value
func Valid(flag string, option string) bool {
	if Good(flag) {
		_, ok := Flags[flag].Options[option]
		return ok
	}
	return false
}

// Add - CHeck for and Add a flag to a current list of flags
func AddFlag(curflag string, flag string, option string) string {
	var newflags string
	if Valid(flag, option) {
		if curflag != "" {
			newflags = curflag + "," + flag + "=" + option
		} else {
			newflags = flag + "=" + option
		}
		return newflags
	}
	log.Fatalln("Invalid Flag:", flag, "or option:", option)
	return curflag
}

// Look for and return value of a particular flag in flags
// e.g flags:descriptor=d32,timestamp=off flag:timestamp return:off
func FlagValue(flags, flag string) string {
	flags = strings.Replace(flags, " ", "", -1) // Get rid of extra spaces in flags
	// Grab the flags and set the frame header
	flagslice := strings.Split(flags, ",") // The name=val of the flag
	for fl := range flagslice {
		f := strings.Split(flagslice[fl], "=") // f[0]=name f[1]=val
		if f[0] == flag {
			return f[1]
		}
	}
	return ""
}

// Replace an existing flag or add a new flag
func ReplaceFlag(flags, value, option string) string {
	var newflags string
	var replaced bool

	for _, flag := range strings.Split(flags, ",") {
		if strings.Split(flag, "=")[0] == value {
			replaced = true
			// We are replacing this current flags value with a new one
			newflags = AddFlag(newflags, value, option)
		} else {
			// Just add the current flag to the list
			newflags = AddFlag(newflags, strings.Split(flag, "=")[0], strings.Split(flag, "=")[1])
		}
	}
	if !replaced { // We are adding this flag as it does not currently exist
		newflags = AddFlag(newflags, value, option)
	}
	return newflags
}

// CopyCliflags - copy from source to desination the Clieflags structure
func (s *Cliflags) CopyCliflags() (*Cliflags, error) {
	d := new(Cliflags)
	d.V4Multicast = s.V4Multicast
	d.V6Multicast = s.V4Multicast
	d.Port = s.Port
	d.Timestamp = s.Timestamp
	d.Datacnt = s.Datacnt
	d.Timezone = s.Timezone
	d.Prompt = s.Prompt
	d.Ppad = s.Ppad
	d.Sardir = s.Sardir
	// Copy the various Timeouts
	d.Timeout.Binterval = s.Timeout.Binterval
	d.Timeout.Metadata = s.Timeout.Metadata
	d.Timeout.Request = s.Timeout.Request
	d.Timeout.Status = s.Timeout.Status
	d.Timeout.Transfer = s.Timeout.Transfer
	// Copy the Global flag defaults
	if len(s.Global) == 0 {
		return nil, errors.New("no global flags defined in copyflags")
	}
	d.Global = make(map[string]string)
	for g := range s.Global {
		d.Global[g] = s.Global[g]
	}
	return d, nil
}

// Values - Return slice of flags applicable to frame type (field)
func Values(ftype string) []string {
	return Frameflags[ftype]
}

// Value - Return the integer value of the flag option
func Value(flag string, option string) int {
	opt, ok := Flags[flag].Options[option]
	if !ok {
		return -1
	}
	return int(opt)
}

// Get - Given a current flag and bitfield name return the integer value of the bitfield
func Get(curflag uint32, field string) uint32 {
	fl := Flags[field]
	shiftbits := uint32(flagsize - fl.Len - fl.Msb)
	maskbits := uint32((1 << fl.Len) - 1)
	setbits := uint32(maskbits << shiftbits)
	return uint32((curflag & setbits) >> shiftbits)
}

// GetStr - Given a current flag and bitfield name return the string name of the bitfield set in curflag
func GetStr(curflag uint32, field string) string {
	ff := Get(curflag, field)
	for k, f := range Flags[field].Options {
		// fmt.Printf("GetStr Curflag %0x Looking for %x val in %x=%s\n", curflag, val, f, k)
		if ff == f {
			return k
		}
	}
	log.Fatalln("GetStr fail Invalid field", field, "in Flag", curflag)
	return ""
}

// Set - Given a current header and bitfield name with a new value return the revised header
// If invalid return the current flag and error
func Set(curflag uint32, field string, flagname string) (uint32, error) {
	fl, ok := Flags[field]
	if !ok {
		log.Fatalln("Set lookup fail Invalid Flag", field)
		e := "invalid Flag: " + field
		return curflag, errors.New(e)
	}

	if !Good(field) {
		e := "Set - Invalid Field:" + field + ":"
		log.Fatalln(e)
		return curflag, errors.New(e)
	}
	// Get the value of the flag
	newval, ok := Flags[field].Options[flagname]
	if !ok {
		e := "Set lookup fail Invalid flagname" + flagname + "in DFlag" + field
		log.Fatalln(e)
		return curflag, errors.New(e)
	}

	shiftbits := uint32(flagsize - fl.Len - fl.Msb)
	maskbits := uint32((1 << fl.Len) - 1)
	setbits := uint32(maskbits << shiftbits)
	// log.Printf("Shiftbits=%d Maskbits=%b Setbits=%b\n", shiftbits, maskbits, setbits)
	result := uint32(((curflag) & (^setbits)))
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
	fl := Flags[field]
	x := Get(curflag, field)
	for k, f := range fl.Options {
		// log.Println("Flags for field ", field, fi.name, fi.val)
		if f == x {
			return k
		}
	}
	log.Fatalln("Name out of range")
	return ""
}

// Fields - return a slice of flag fields that are used by frametype
func Fields(frametype string) []string {
	return Values(frametype)
}

// Good - Is this a valid flagname
func Good(field string) bool {
	_, ok := Flags[field]
	return ok
}

// Setglobal - Set the global flags applicable for the particular frame type
// Dont set final descriptor here - Work it out in the transfer as it depends on file size
func Setglobal(frametype string, c *Cliflags) string {
	var cmu sync.Mutex

	cmu.Lock()
	fs := ""
	for _, f := range Fields(frametype) {
		for g := range c.Global {
			if g == f {
				fs += f + "=" + c.Global[f] + ","
			}
		}
	}
	cmu.Unlock()
	return strings.TrimRight(fs, ",")
}

// Length in bits of the directory entry flag size
const dflagsize uint16 = 16

// GetD - Given a current flag and bitfield name return the integer value of the bitfield
func GetD(curflag uint16, field string) uint16 {

	var shiftbits, maskbits, setbits uint16

	shiftbits = dflagsize - DirentFlags[field].Len - DirentFlags[field].Msb
	maskbits = (1 << DirentFlags[field].Len) - 1
	setbits = maskbits << shiftbits
	return (curflag & setbits) >> shiftbits
}

// GetDStr - Given a current flag and bitfield name return the string name of the bitfield set in curflag
func GetDStr(curflag uint16, field string) string {
	var shiftbits, maskbits, setbits, val uint16

	shiftbits = dflagsize - DirentFlags[field].Len - DirentFlags[field].Msb
	maskbits = (1 << DirentFlags[field].Len) - 1
	setbits = maskbits << shiftbits
	val = (curflag & setbits) >> shiftbits
	for ki, fi := range DirentFlags[field].Options {
		if fi == val {
			return ki
		}
	}
	log.Fatalln("GetDStr fail Invalid field", field, "in DFlag:", curflag)
	return ""
}

// SetD - Given a current header and bitfield name with a new value return the revised header
func SetD(curflag uint16, field string, flagname string) (uint16, error) {

	if !GoodD(field) {
		e := "Invalid Date Field:" + field + ":"
		log.Fatalln(e)
		return curflag, errors.New(e)
	}
	// Get the value of the flag
	newval, ok := DirentFlags[field].Options[flagname]
	if !ok {
		e := "SetD lookup fail Invalid flagname" + flagname + "in DFlag" + field
		log.Fatalln(e)
		return curflag, errors.New(e)
	}

	shiftbits := uint16(dflagsize - DirentFlags[field].Len - DirentFlags[field].Msb)
	maskbits := uint16((1 << DirentFlags[field].Len) - 1)
	setbits := uint16(maskbits << shiftbits)
	// log.Printf("Shiftbits=%d Maskbits=%b Setbits=%b\n", shiftbits, maskbits, setbits)
	result := uint16(((curflag) & (^setbits)))
	result |= (newval << shiftbits)
	// log.Printf("Result=%016b\n", result)
	return result, nil
}

// FrameD - return a slice of flag names matching field
func FrameD(field string) []string {
	var s []string
	for k := range DirentFlags[field].Options {
		s = append(s, k)
	}
	return s
}

// FlagD - return a slice of flag names that are used by Dirent
func FlagD() []string {
	var s []string
	for k := range DirentFlags {
		s = append(s, k)
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
	for ki, fi := range DirentFlags[field].Options {
		// log.Println("Flags for field ", field, fi.name, fi.val)
		if fi == x {
			return ki
		}
	}
	log.Fatalln("NameD out of range")
	return ""
}

// GoodD -- Is this a valid Descriptor Flag
func GoodD(field string) bool {
	_, ok := DirentFlags[field]
	return ok
}

// Valid - Check for valid flag and value
func ValidD(flag string, option string) bool {
	if GoodD(flag) {
		_, ok := DirentFlags[flag].Options[option]
		return ok
	}
	return false
}

// Add - CHeck for and Add a flag to a current list of flags
func AddFlagD(curflag string, flag string, option string) string {
	var newflags string
	if ValidD(flag, option) {
		if curflag != "" {
			newflags = curflag + "," + flag + "=" + option
		} else {
			newflags = flag + "=" + option
		}
		return newflags
	}
	log.Fatalln("Invalid Directory Flag:", flag, "or option:", option)
	return curflag
}

// Replace an existing flag or add a new flag
func ReplaceFlagD(flags, value, option string) string {
	var newflags string
	var replaced bool

	for _, flag := range strings.Split(flags, ",") {
		if strings.Split(flag, "=")[0] == value {
			replaced = true
			// We are replacing this current flags value with a new one
			newflags = AddFlagD(newflags, value, option)
		} else {
			// Just add the current flag to the list
			newflags = AddFlagD(newflags, strings.Split(flag, "=")[0], strings.Split(flag, "=")[1])
		}
	}
	if !replaced { // We are adding this flag as it does not currently exist
		newflags = AddFlagD(newflags, value, option)
	}
	return newflags
}

// Length in bits of the timestamp flag size
const tflagsize uint8 = 8

// GetT - Given a current flag and bitfield name return the integer value of the bitfield
func GetT(curflag uint8) uint8 {
	var shiftbits, maskbits, setbits uint8

	shiftbits = tflagsize - TimeStamps.Len - TimeStamps.Msb
	maskbits = (1 << TimeStamps.Len) - 1
	setbits = maskbits << shiftbits
	return (curflag & setbits) >> shiftbits
}

// GetTStr - Given a current flag and bitfield name return the string name of the bitfield set in curflag
func GetTStr(curflag uint8) string {
	var shiftbits, maskbits, setbits, val uint8

	shiftbits = tflagsize - TimeStamps.Len - TimeStamps.Msb
	maskbits = (1 << TimeStamps.Len) - 1
	setbits = maskbits << shiftbits
	val = (curflag & setbits) >> shiftbits
	for ki, fi := range TimeStamps.Options {
		if fi == val {
			return ki
		}
	}
	log.Fatalln("GetTStr fail Invalid Tflag:", curflag)
	return ""
}

// SetT Given a current header and bitfield name with a new value return the revised header
func SetT(curflag uint8, flagname string) (uint8, error) {

	newval, found := TimeStamps.Options[flagname]
	if !found {
		log.Fatalln("SetT lookup fail Invalid flagname", flagname, "in TFlag")
		e := "invalid TFlag: " + flagname
		return curflag, errors.New(e)
	}

	shiftbits := uint8(tflagsize - TimeStamps.Len - TimeStamps.Msb)
	maskbits := uint8((1 << TimeStamps.Len) - 1)
	setbits := maskbits << shiftbits
	// log.Printf("Shiftbits=%d Maskbits=%b Setbits=%b\n", shiftbits, maskbits, setbits)
	result := ((curflag) & (^setbits))
	result |= (newval << shiftbits)
	// log.Printf("Result=%08b\n", result)
	return result, nil
}

// NameT - return the name of the flag for field in curtflag
func NameT(curtflag uint8) string {
	x := GetT(curtflag)
	for ki, fi := range TimeStamps.Options {
		// log.Println("Flags for field ", field, ki, val)
		if fi == x {
			return ki
		}
	}
	log.Fatalln("NameT out of range")
	return ""
}

// FrameT - return a slice of flag names that are used by Timeinfo
func FrameT() []string {
	var s []string
	for ki := range TimeStamps.Options {
		s = append(s, ki)
	}
	return s
}

// GoodT - Is this a valid time flag
func GoodT(field string) bool {
	_, ok := TimeStamps.Options[field]
	return ok
}

// *******************************************************************
