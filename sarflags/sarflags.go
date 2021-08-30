// Saratoga Flag Handlers

package sarflags

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
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

var Commands map[string]Cmdtype

// Flag Information
type Flagtype struct {
	Frametypes []string          // What Frametypes are applicable to this Flag
	Len        uint32            // Bit Length of the flag within the header
	Msb        uint32            // Most significant bit within the header
	Options    map[string]uint32 // What are the Options for the Flag
}

// Global map of flag info
var Flags map[string]Flagtype

// DateFlag Information
type DateFlagtype struct {
	Len     uint16            // Bit Length of the flag within the header
	Msb     uint16            // Mosost significant bit within the header
	Options map[string]uint16 // What are the Options for the DateFlag
}

var DateFlags map[string]DateFlagtype

// TimeStamp Information
type TimeStamptype struct {
	Len     uint8
	Msb     uint8
	Options map[string]uint8
}

// Global map of what Timestamps are applicable
var TimeStamps TimeStamptype

// Global map of what flags are applicable to which frame types
var Frameflags map[string][]string

// Climu - Protect CLI input flags
var Climu sync.Mutex

// var conf config

// Config - JSON Config Default Global Settings & Commands
type config struct {
	Descriptor  string   `json:"descriptor"` // Default Descriptor: d16,d32,d64
	Csumtype    string   `json:"csumtype"`   // Default Checksum type: none
	Freespace   string   `json:"freespace"`  // Is freespace tp be advertised: yes,no
	Txwilling   string   `json:"txwilling"`  // Can files/streams be sent: yes,no
	Rxwilling   string   `json:"rxwilling"`  // Can files/streams be received: yes,no
	Stream      string   `json:"stream"`     // Can files/streams be transmitted: yes,no
	Reqtstamp   string   `json:"reqtstamp"`  // Request timestamps: yes,no
	Reqstatus   string   `json:"reqstatus"`  // Request status frame to be sent/received: yes,no
	Udplite     string   `json:"udplite"`    // Is UDP Lite supported: yes,no
	Timestamp   string   `json:"timestamp"`  // What is the default timestamp format: anything for local,posix32,posix32_323,posix64,posix64_32,epoch2000_32,
	Timezone    string   `json:"timezone"`   // What timezone is to be used in timestamps: utc
	Sardir      string   `json:"sardir"`     // What is the default directory for saratoga files
	Prompt      string   `json:"prompt"`     // Command line prompt: saratoga
	Ppad        int      `json:"ppad"`       // Padding length in prompt for []:
	Timeout     Timeouts // Various Timers
	Datacounter int      `json:"datacounter"` // How many data frames received before a status is requested
}

// Cliflags - CLI Input flags
type Cliflags struct {
	Global    map[string]string // Global header flags set for frames
	Timestamp string            // What timestamp to use
	Timeout   Timeouts          // Various timeouts
	Datacnt   int               // # data frames to send before a request flag is set
	Timezone  string            // Timezone for logs utc or local time
	Prompt    string            // Prompt
	Ppad      int               // Length of Padding around Prompt []: = 3
	Sardir    string            // Saratoga working directory
}

// Read  in the JSON Config data
func ReadConfig(fname string, c *Cliflags) error {
	var sarconfdata map[string]interface{}
	// var sartimeouts map[string]interface{}
	var confdata []byte
	var conf config
	var err error
	// var cmu sync.Mutex
	Flags = make(map[string]Flagtype)         // Setup the Flags global map
	Frameflags = make(map[string][]string)    // Setup Frameflags global map
	DateFlags = make(map[string]DateFlagtype) // Setup Dateflags global map
	Commands = make(map[string]Cmdtype)       // Setup Commands global map

	if confdata, err = ioutil.ReadFile(fname); err != nil {
		fmt.Println("Cannot open the saratoga config file", os.Args[1], ":", err)
		return err
	}

	if err = json.Unmarshal([]byte(confdata), &sarconfdata); err != nil {
		fmt.Println("Cannot Unmarshal json from saratoga config file", os.Args[1], ":", err)
		return err
	}
	// Now decode all of those variables, arrays & maps in the json into the struct's
	for key, value := range sarconfdata {
		// fmt.Println(key, "=", value)
		switch key {
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
		case "datacounter":
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
		case "frameflags":
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
		case "flags": // Now this is the HARD ONE!!!!!
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
			for f, v := range Flags {
				// fmt.Println("FLAG=", f, v)
				fmt.Println(f, "=", v.Options)
			}
		case "dateflags": // Now this is the HARD ONE!!!!!
			dateflags := value.(map[string]interface{})
			for dkey, val := range dateflags {
				var tmp DateFlagtype
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
				fmt.Println("dateflag for", key, "=", tmp)
				DateFlags[dkey] = tmp
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

	// panic("All DONE PRINTING")

	//for key, value := range sarconf {
	//	switch key: {
	//		case ""
	//	}
	// }
	// cmu.Lock()
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

	// cmu.Unlock()
	return nil
}

// Valid - Check for valid flag and value
func Valid(flag string, option string) bool {
	for k, v := range Flags { // Loop through all the flags looking for a match
		if k == flag {
			for k1, v1 := range v.Options { // Loop through the options for the flag
				if k1 == option { // Yep it is a valid option
					fmt.Println(k1, "=", v1)
					return true
				}
			}
			return false
		}
	}
	return false
}

// CopyCliflags - copy from source to desination the Clieflags structure
func CopyCliflags(d *Cliflags, s *Cliflags) error {
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
		return errors.New("no global flags defined in copyflags")
	}
	d.Global = make(map[string]string)
	for g := range s.Global {
		d.Global[g] = s.Global[g]
	}
	return nil
}

// Values - Return slice of flags applicable to frame type (field)
func Values(ftype string) []string {
	return Frameflags[ftype]
}

// Value - Return the integer value of the flag option
func Value(flag string, option string) int {
	for k, f := range Flags {
		if flag == k {
			for o := range f.Options {
				if option == o {
					return int(f.Options[o])
				}
			}
			return -1
		}
	}
	return -1
}

// Get - Given a current flag and bitfield name return the integer value of the bitfield
func Get(curflag uint32, field string) uint32 {
	var shiftbits, maskbits, setbits uint32

	fl := Flags[field]
	shiftbits = flagsize - fl.Len - fl.Msb
	maskbits = (1 << fl.Len) - 1
	setbits = maskbits << shiftbits
	return (curflag & setbits) >> shiftbits
}

// GetStr - Given a current flag and bitfield name return the string name of the bitfield set in curflag
func GetStr(curflag uint32, field string) string {
	val := Get(curflag, field)
	fl := Flags[field]
	for k, f := range fl.Options {
		// fmt.Printf("GetStr Curflag %0x Looking for %x val in %x=%s\n", curflag, val, fi.val, fi.name)
		if f == val {
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

	var newval uint32
	var found = false
	// Get the value of the flag
	for k, f := range fl.Options {
		// log.Println("Flags for field ", field, fi.name, fi.val)
		if k == flagname {
			newval = f
			found = true
			break
		}
	}
	if !found {
		log.Fatalln("Set lookup fail Invalid flagname", flagname, "in Flag", field)
	}

	var shiftbits, maskbits, setbits, result uint32

	shiftbits = flagsize - fl.Len - fl.Msb
	maskbits = (1 << fl.Len) - 1
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

// GetD - Given a current flag and bitfield name return the integer value of the bitfield
func GetD(curflag uint16, field string) uint16 {

	var len, msb, shiftbits, maskbits, setbits uint16

	len = DateFlags[field].Len
	msb = DateFlags[field].Msb
	shiftbits = dflagsize - len - msb
	maskbits = (1 << len) - 1
	setbits = maskbits << shiftbits
	return (curflag & setbits) >> shiftbits
}

// GetDStr - Given a current flag and bitfield name return the string name of the bitfield set in curflag
func GetDStr(curflag uint16, field string) string {
	var len, msb, shiftbits, maskbits, setbits, val uint16

	len = DateFlags[field].Len
	msb = DateFlags[field].Msb
	shiftbits = dflagsize - len - msb
	maskbits = (1 << len) - 1
	setbits = maskbits << shiftbits
	val = (curflag & setbits) >> shiftbits
	for ki, fi := range DateFlags[field].Options {
		if fi == val {
			return ki
		}
	}
	log.Fatalln("GetDStr fail Invalid field", field, "in DFlag:", curflag)
	return ""
}

// SetD - Given a current header and bitfield name with a new value return the revised header
func SetD(curflag uint16, field string, flagname string) (uint16, error) {
	var newval uint16
	var found = false
	// Get the value of the flag
	for ki, fi := range DateFlags[field].Options {
		// log.Println("DFlags for field ", field, fi.name, fi.val)
		if ki == flagname {
			newval = fi
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

	len = DateFlags[field].Len
	msb = DateFlags[field].Msb
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
	for k := range DateFlags[field].Options {
		s = append(s, k)
	}
	return s
}

// FlagD - return a slice of flag names that are used by Dirent
func FlagD() []string {
	var s []string
	for k := range DateFlags {
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
	for ki, fi := range DateFlags[field].Options {
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
	for f := range DateFlags {
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

// GetT - Given a current flag and bitfield name return the integer value of the bitfield
func GetT(curflag uint8) uint8 {
	var tlen, msb, shiftbits, maskbits, setbits uint8

	tlen = TimeStamps.Len
	msb = TimeStamps.Msb
	shiftbits = tflagsize - tlen - msb
	maskbits = (1 << tlen) - 1
	setbits = maskbits << shiftbits
	return (curflag & setbits) >> shiftbits
}

// GetTStr - Given a current flag and bitfield name return the string name of the bitfield set in curflag
func GetTStr(curflag uint8) string {
	var tlen, msb, shiftbits, maskbits, setbits, val uint8

	tlen = TimeStamps.Len
	msb = TimeStamps.Msb
	shiftbits = tflagsize - tlen - msb
	maskbits = (1 << tlen) - 1
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
	var newval uint8
	var found = false
	// Get the value of the flag
	for ki, fi := range TimeStamps.Options {
		// log.Println("TFlags for field ", field, fi.name, fi.val)
		if ki == flagname {
			newval = fi
			found = true
			break
		}
	}
	if !found {
		log.Fatalln("SetT lookup fail Invalid flagname", flagname, "in TFlag")
		e := "invalid TFlag: " + flagname
		return curflag, errors.New(e)
	}

	var tlen = TimeStamps.Len
	var msb = TimeStamps.Msb
	var shiftbits = tflagsize - tlen - msb
	var maskbits uint8 = (1 << tlen) - 1
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
	for ki, fi := range TimeStamps.Options {
		// log.Println("Flags for field ", field, fi.name, fi.val)
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
	for t := range TimeStamps.Options {
		if field == t {
			return true
		}
	}
	return false
}

// *******************************************************************
