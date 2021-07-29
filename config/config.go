package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// Timeouts - JSON Config Default Global Timeout Settings
type Timeouts struct {
	Metadata int // If no Metadata is received after x seconds cancel transfer
	Request  int // If no request is received after x seconds cancel transfer
	Status   int // Send a status every x seconds
	Transfer int // If no data has been received after x seconds cancel transfer
}

// Commands - JSON Config for command usage & help
type Cmds struct {
	Cmd   string
	Usage string
	Help  string
}

// Config - JSON Config Default Global Settings
type Config struct {
	Descriptor  string   // Default Descriptor: d16,d32,d64
	Csumtype    string   // Default Checksum type: none
	Freespace   string   // Is freespace tp be advertised: yes,no
	Txwilling   string   // Can files/streams be sent: yes,no
	Rxwilling   string   // Can files/streams be received: yes,no
	Stream      string   // Can files/streams be transmitted: yes,no
	Reqtstamp   string   // Request timestamps: yes,no
	Reqstatus   string   // Request status frame to be sent/received: yes,no
	Udplite     string   // Is UDP Lite supported: yes,no
	Timestamp   string   // What is the default timestamp format: anything for local,posix32,posix32_323,posix64,posix64_32,epoch2000_32,
	Timezone    string   // What timezone is to be used in timestamps: utc
	Sardir      string   // What is the default directory for saratoga files
	Prompt      string   // Command line prompt: saratoga
	Ppad        int      // Padding length in prompt for []:
	Timeout     Timeouts // Various Timers
	Datacounter int      // How many data frames received before a status is requested
	Commands    []Cmds   // Command name, usage & help
}

// Holds json decoded data in the Config struct
var Conf Config

// Read  in the JSON Config data
func ReadConf(fname string) error {
	var confdata []byte
	var err error

	if confdata, err = ioutil.ReadFile(fname); err != nil {
		fmt.Println("Cannot open saratoga config file", os.Args[1], ":", err)
		return err
	}
	if err = json.Unmarshal(confdata, &Conf); err != nil {
		fmt.Println("Cannot read saratoga config file", os.Args[1], ":", err)
		return err
	}
	return nil
}
