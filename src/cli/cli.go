package cli

import (
	"errors"
	"net"
	"os"
	"screen"
	"strconv"
	"strings"

	"sarflags"

	"client"
)

// MOVE THESE TO NETWORKING SECTION WHEN WE HAVE ONE!!!!

// SaratogaPort - IANA allocated Saratoga UDP & TCP Port #'s
const SaratogaPort = 7542

// IPv4Multicast -- IANA allocated Saratoga IPV4 all-hosts Multicast Address
const IPv4Multicast = "224.0.0.108"

// IPv6Multicast -- IANA allocated Saratoga IPV6 link-local Multicast Address
const IPv6Multicast = "FF02:0:0:0:0:0:0:6c"

// Remove an entry in a slice of strings by index #
func removeIndex(s []string, index int) []string {
	ret := make([]string, 0)
	ret = append(ret, s[:index]...)
	return append(ret, s[index+1:]...)
}

// Remove all entries in slice of strings matching val
func removeValue(s []string, val string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == val {
			s = removeIndex(s, i)
			s = removeValue(s, val) // Call me again to remove dupes
		}
	}
	return s
}

// Only append to a string slice if it is unique
func appendunique(slice []string, i string) []string {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}

// CurLine -- Current line number in buffer
var CurLine int

// All of the different command line input handlers

// Beacon CLI Info
type cmdBeacon struct {
	v4flag bool     // Send to all on the v4 multicast
	v6flag bool     // Send to all on the v6 multicast
	timer  uint     // How often to send beacon (secs) 0 = dont send beacons
	v4addr []string // List of v4 addresses to send beacons to
	v6addr []string // List of v6 addresses to send beacons to
}

// clibeacon - Beacon commands
var clibeacon = cmdBeacon{}

func beacon(args []string) {

	// Show current Cbeacon flags and lists
	if len(args) == 1 {
		if clibeacon.timer != 0 {
			screen.Fprintln(screen.Msg, "green_black", "Beacons set to be sent every", clibeacon.timer, "seconds")
		} else {
			screen.Fprintln(screen.Msg, "green_black", "Beacon timer not set")
		}
		if clibeacon.v4flag == true {
			screen.Fprintln(screen.Msg, "green_black", "Sending multicast beacons to IPv4")
		}
		if clibeacon.v6flag == true {
			screen.Fprintln(screen.Msg, "green_black", "Sending multicast beacons to IPv6")
		}
		if len(clibeacon.v4addr) > 0 {
			screen.Fprintln(screen.Msg, "green_black", "Sending Unicast IPv4 beacons to:")
			for _, i := range clibeacon.v4addr {
				screen.Fprintln(screen.Msg, "green_black", "\t", i)
			}
		}
		if len(clibeacon.v6addr) > 0 {
			screen.Fprintln(screen.Msg, "green_black", "Sending Unicast IPv6 Beacons to:")
			for _, i := range clibeacon.v6addr {
				screen.Fprintln(screen.Msg, "green_black", "\t", i)
			}
		}
		if clibeacon.v4flag == false && clibeacon.v6flag == false &&
			len(clibeacon.v4addr) == 0 && len(clibeacon.v6addr) == 0 {
			screen.Fprintln(screen.Msg, "green_black", "No beacons currently being sent")
		}
		return
	}
	if len(args) == 2 {
		switch args[1] {
		case "?": // usage
			screen.Fprintln(screen.Msg, "green_black", cmd["beacon"][0])
			screen.Fprintln(screen.Msg, "green_black", cmd["beacon"][1])
			return
		case "off": // remove and disable all beacons
			clibeacon.timer = 0
			clibeacon.v4flag = false
			clibeacon.v6flag = false
			clibeacon.v4addr = nil
			clibeacon.v6addr = nil
			screen.Fprintln(screen.Msg, "green_black", "Beacons Disabled")
			return
		case "v4": // V4 Multicast
			errflag := make(chan uint32)
			if !clibeacon.v4flag {
				clibeacon.v4flag = true
				screen.Fprintln(screen.Msg, "green_black", "Sending beacons to IPv4 Multicast")
				if clibeacon.timer > 0 {
					// Start up the beacon client sending IPv4 beacons every timer secs
					go client.Beacon(IPv4Multicast, SaratogaPort, clibeacon.timer, errflag)
				}
				f := <-errflag
				if sarflags.GetStr(errflag, "errcode") != "success" {
					screen.Fprintln(screen.Msg, "red_black", "Bad Beacon")
				}
				return
			}
			screen.Fprintln(screen.Msg, "green_red", "Beacon IPv4 Multicast already being sent")
			return
		case "v6": // V6 Multicast
			if !clibeacon.v6flag {
				clibeacon.v6flag = true
				screen.Fprintln(screen.Msg, "green_black", "Sending beacons to IPv6 Multicast")
				if clibeacon.timer > 0 {
					// Start up the beacon client sending IPv6 beacons every timer secs
					go client.Beacon(IPv6Multicast, SaratogaPort, clibeacon.timer, errflag)
				}
				return
			}
			screen.Fprintln(screen.Msg, "green_red", "Beacon IPv6 Multicast already being sent")
			return

		default: // beacon <timer> or beacon <ipaddr>
			u32, err := strconv.ParseUint(args[1], 10, 32)
			if err == nil { // We have a number so it is a timer
				clibeacon.timer = uint(u32)
				screen.Fprintln(screen.Msg, "green_black", "Beacon timer", clibeacon.timer, "secs")
				return
			}
			if net.ParseIP(args[1]) != nil {
				if strings.Contains(args[1], ".") == true { // IPv4

					clibeacon.v4addr = appendunique(clibeacon.v4addr, args[1])
					screen.Fprintln(screen.Msg, "green_black", "Sending beacons to", clibeacon.v4addr)
					return
				}
				if strings.Contains(args[1], ":") == true { // IPv6
					clibeacon.v6addr = appendunique(clibeacon.v6addr, args[1])
					screen.Fprintln(screen.Msg, "green_black", "Sending beacons to", clibeacon.v6addr)
					return
				}
			}
		}
		screen.Fprintln(screen.Cmd, "red_black", cmd["beacon"][0])
		return
	}

	// beacon off <ipaddr> ...
	if args[1] == "off" && len(args) > 2 { // turn off following addresses
		screen.Fprintf(screen.Msg, "green_black", "%s ", "Beacons turned off to")
		for i := 2; i < len(args); i++ { // Remove Address'es from lists
			if net.ParseIP(args[i]) != nil { // Do We have a valid IP Address
				if strings.Contains(args[i], ".") == true { // IPv4
					clibeacon.v4addr = removeValue(clibeacon.v4addr, args[i])
				}
				if strings.Contains(args[i], ":") == true { // IPv6
					clibeacon.v6addr = removeValue(clibeacon.v6addr, args[i])
				}
				screen.Fprintf(screen.Msg, "green_black", "%s ", args[i])
				if i == len(args)-1 {
					screen.Fprintln(screen.Msg, "green_black", "")
				}
			} else {
				screen.Fprintln(screen.Msg, "red_black", "Invalid IP Address:", args[i])
				screen.Fprintln(screen.Cmd, "red_black", cmd["beacon"][0])
			}
		}
		return
	}
	// beacon <ipaddr> ...
	screen.Fprintf(screen.Msg, "green_black", "Sending beacons to:")
	for i := 1; i < len(args); i++ { // Add Address'es to lists
		if net.ParseIP(args[i]) != nil { // We have a valid IP Address
			if strings.Contains(args[i], ".") == true { // IPv4
				clibeacon.v4addr = appendunique(clibeacon.v4addr, args[i])
			}
			if strings.Contains(args[i], ":") == true { // IPv6
				clibeacon.v6addr = appendunique(clibeacon.v6addr, args[i])
			}
			screen.Fprintf(screen.Msg, "green_black", " %s", args[i])
			if i == len(args)-1 {
				screen.Fprintln(screen.Msg, "green_black", "")
			}
		} else {
			screen.Fprintln(screen.Cmd, "red_black", cmd["beacon"][0])
		}
	}
}

func cancel(args []string) {
	screen.Fprintln(screen.Msg, "green_black", args)
}

func checksum(args []string) {
	if len(args) == 1 {
		screen.Fprintln(screen.Msg, "green_black", "Checksum", sarflags.Global.Checksum)
		return
	}
	if len(args) == 2 && args[1] == "?" { // usage
		screen.Fprintln(screen.Msg, "green_black", cmd["checksum"][0])
		screen.Fprintln(screen.Msg, "green_black", cmd["checksum"][1])
		return
	}
	if len(args) == 2 {
		switch args[1] {
		case "off", "none":
			sarflags.Global.Checksum = "none"
		case "crc32":
			sarflags.Global.Checksum = "crc32"
		case "md5":
			sarflags.Global.Checksum = "md5"
		case "sha1":
			sarflags.Global.Checksum = "sha1"
		default:
			screen.Fprintln(screen.Cmd, "green_red", cmd["checksum"][0])
		}
		return
	}
	screen.Fprintln(screen.Cmd, "green_red", cmd["checksum"][0])
}

// Cdebug - Debug level 0 is off
var Cdebug = 0

func debug(args []string) {
	if len(args) == 1 {
		screen.Fprintln(screen.Msg, "green_black", "Debug level", Cdebug)
		return
	}
	if len(args) == 2 && args[1] == "?" { // usage
		screen.Fprintln(screen.Msg, "green_black", cmd["debug"][0])
		screen.Fprintln(screen.Msg, "green_black", cmd["debug"][1])
		return
	}
	if len(args) == 2 {
		switch args[1] {
		case "off":
			Cdebug = 0
		case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
			Cdebug, _ = strconv.Atoi(args[1])
		default:
			screen.Fprintln(screen.Msg, "green_red", cmd["debug"][0])
		}
		return
	}
	screen.Fprintln(screen.Msg, "green_red", cmd["debug"][0])
}

func descriptor(args []string) {
	if len(args) == 1 {
		screen.Fprintln(screen.Msg, "green_black", "Descriptor", sarflags.Global.Descriptor)
		return
	}
	if len(args) == 2 && args[1] == "?" { // usage
		screen.Fprintln(screen.Msg, "green_black", cmd["descriptor"][0])
		screen.Fprintln(screen.Msg, "green_black", cmd["descriptor"][1])
		return
	}
	if len(args) == 2 {
		switch args[1] {
		case "auto":
			if sarflags.MaxUint <= sarflags.MaxUint16 {
				sarflags.Global.Descriptor = "d16"
			}
			if sarflags.MaxUint <= sarflags.MaxUint32 {
				sarflags.Global.Descriptor = "d32"
			}
			if sarflags.MaxUint <= sarflags.MaxUint64 {
				sarflags.Global.Descriptor = "d64"
			}
		case "64":
			if sarflags.MaxUint <= sarflags.MaxUint64 {
				sarflags.Global.Descriptor = "d64"
			} else {
				screen.Fprintln(screen.Msg, "red_black", "64 bit descriptors not supported on this platform")
			}
		case "16":
			if sarflags.MaxUint <= sarflags.MaxUint16 {
				sarflags.Global.Descriptor = "d16"
			} else {
				screen.Fprintln(screen.Msg, "red_black", "16 bit descriptors not supported on this platform")
			}
		case "32":
			if sarflags.MaxUint <= sarflags.MaxUint32 {
				sarflags.Global.Descriptor = "d32"
			} else {
				screen.Fprintln(screen.Msg, "red_black", "16 bit descriptors not supported on this platform")
			}
		case "128":
			screen.Fprintln(screen.Msg, "red_black", "128 bit descriptors not supported on this platform")
		default:
			screen.Fprintln(screen.Msg, "green_red", cmd["descriptor"][0])
		}
		screen.Fprintln(screen.Msg, "green_red", "Descriptor size set to", sarflags.Global.Descriptor)
		return
	}
	screen.Fprintln(screen.Msg, "green_red", cmd["descriptor"][0])
}

func eid(args []string) {
	if len(args) == 1 {
		screen.Fprintln(screen.Msg, "green_black", "EID", sarflags.Global.Eid)
		return
	}
	if len(args) == 2 && args[1] == "?" { // usage
		screen.Fprintln(screen.Msg, "green_black", cmd["eid"][0])
		screen.Fprintln(screen.Msg, "green_black", cmd["eid"][1])
		return
	}
	if len(args) == 2 {
		if args[1] == "off" { // Default is the PID
			sarflags.Global.Eid = os.Getpid()
			return
		}
		if n, err := strconv.Atoi(args[1]); err == nil && n >= 0 {
			sarflags.Global.Eid = n
			return
		}
	}
	screen.Fprintln(screen.Msg, "red_black", cmd["eid"][0])
}

// Cexit = Exit level to quit from saratoga
var Cexit = -1

// Quit saratoga
func exit(args []string) {
	if len(args) > 2 { // usage
		screen.Fprintln(screen.Msg, "red_black", cmd["exit"][0])
		return
	}
	if len(args) == 1 { // exit 0
		Cexit = 0
		screen.Fprintln(screen.Msg, "green_black", "Good Bye!")
		return
	}
	if len(args) == 2 {
		switch args[1] {
		case "?": // Usage
			screen.Fprintln(screen.Msg, "green_black", cmd["exit"][0])
			screen.Fprintln(screen.Msg, "green_black", cmd["exit"][1])
		case "0": // exit 0
			Cexit = 0
			screen.Fprintln(screen.Msg, "green_black", "Good Bye!")
		case "1": // exit 1
			Cexit = 1
			screen.Fprintln(screen.Msg, "green_black", "Good Bye!")
		default: // Help
			screen.Fprintln(screen.Msg, "red_black", cmd["exit"][0])
		}
		return
	}
	screen.Fprintln(screen.Msg, "red_black", cmd["exit"][0])
}

// Cfiles - Currently open file list
var Cfiles []string

func files(args []string) {
	if len(args) == 1 {
		if len(Cfiles) == 0 {
			screen.Fprintln(screen.Msg, "green_black", "No currently open files")
			return
		}
		for _, i := range Cfiles {
			screen.Fprintln(screen.Msg, "green_black", i)
		}
		return
	}
	if len(args) == 2 && args[1] == "?" { // usage
		screen.Fprintln(screen.Msg, "green_black", cmd["files"][0])
		screen.Fprintln(screen.Msg, "green_black", cmd["files"][1])
		return
	}
	screen.Fprintln(screen.Msg, "red_black", cmd["files"][0])
}

func freespace(args []string) {
	if len(args) == 1 {
		if sarflags.Global.Freespace == "yes" {
			screen.Fprintln(screen.Msg, "green_black", "Free space advertised")
		} else {
			screen.Fprintln(screen.Msg, "green_black", "Free space not advertised")
		}
		return
	}
	if len(args) == 2 && args[1] == "?" { // usage
		screen.Fprintln(screen.Msg, "green_black", cmd["freespace"][0])
		screen.Fprintln(screen.Msg, "green_black", cmd["freespace"][1])
		return
	}
	if len(args) == 2 {
		if args[1] == "on" {
			sarflags.Global.Freespace = "yes"
			return
		}
		if args[1] == "off" {
			sarflags.Global.Freespace = "no"
			return
		}
	}
	screen.Fprintln(screen.Msg, "red_black", cmd["freespace"][0])
}

type cmdGet struct {
	rmflag   bool   // Remove remote file after successful completion
	peer     string // Host we are getting file from
	filename string // File name to get from remote host
}

// Cget - Get file list used in get and getrm
var Cget = []cmdGet{}

func get(args []string) {
	if len(args) == 1 {
		if len(Cget) == 0 {
			screen.Fprintln(screen.Msg, "green_black", "No current get transactions")
		} else {
			for _, i := range Cget {
				screen.Fprintln(screen.Msg, "green_black", i.peer, i.filename, i.rmflag)
			}
		}
	}
	if len(args) == 2 && args[1] == "?" {
		screen.Fprintln(screen.Msg, "green_black", cmd["get"][0])
		screen.Fprintln(screen.Msg, "green_black", cmd["get"][1])
		return
	}
	if len(args) == 3 {
		if addr := net.ParseIP(args[1]); addr != nil { // We have a valid IP Address
			for _, i := range Cget { // Don't add duplicates
				if args[1] == i.peer && args[2] == i.filename {
					return
				}
			}
			Cget = append(Cget, cmdGet{rmflag: false, peer: args[1], filename: args[2]})
			return
		}
	}
	screen.Fprintln(screen.Msg, "red_black", cmd["get"][0])
}

func getrm(args []string) {
	if len(args) == 1 {
		if len(Cget) == 0 {
			screen.Fprintln(screen.Msg, "green_black", "No current get transactions")
		} else {
			for _, i := range Cget {
				screen.Fprintln(screen.Msg, "green_black", i.peer, i.filename, i.rmflag)
			}
		}
	}
	if len(args) == 2 && args[1] == "?" {
		screen.Fprintln(screen.Msg, "green_black", cmd["getrm"][0])
		screen.Fprintln(screen.Msg, "green_black", cmd["getrm"][1])
		return
	}
	if len(args) == 3 {
		if addr := net.ParseIP(args[1]); addr != nil { // We have a valid IP Address
			for _, i := range Cget { // Don't add duplicates
				if args[1] == i.peer && args[2] == i.filename {
					return
				}
			}
			Cget = append(Cget, cmdGet{rmflag: true, peer: args[1], filename: args[2]})
			return
		}
	}
	screen.Fprintln(screen.Msg, "red_black", cmd["get"][0])
}

func help(args []string) {
	for key, val := range cmd {
		screen.Fprintln(screen.Msg, "magenta_black", key, "-", val[1])
	}
}

func history(args []string) {
	screen.Fprintln(screen.Msg, "green_black", args)
}

func home(args []string) {
	screen.Fprintln(screen.Msg, "green_black", args)
}

func ls(args []string) {
	screen.Fprintln(screen.Msg, "green_black", args)
}

// Cprompt - Command line prompt
var Cprompt = "saratoga"

func prompt(args []string) {
	if len(args) == 1 {
		screen.Fprintln(screen.Msg, "green_black", "Current prompt is", Cprompt)
		return
	}
	if len(args) == 2 && args[1] == "?" { // usage
		screen.Fprintln(screen.Msg, "green_black", cmd["prompt"][0])
		screen.Fprintln(screen.Msg, "green_black", cmd["prompt"][1])
		return
	}
	if len(args) == 2 {
		Cprompt = args[1]
		return
	}
}

func peers(args []string) {
	screen.Fprintln(screen.Msg, "green_black", args)
}

type cmdPut struct {
	rmflag   bool   // Remove local file after successful comletion
	peer     string // Host we are putting file to
	filename string // Local file name to  put to remote host
}

// Cput - Put file command
var Cput = []cmdPut{}

func put(args []string) {
	if len(args) == 1 {
		if len(Cput) == 0 {
			screen.Fprintln(screen.Msg, "green_black", "No current put transactions")
		} else {
			for _, i := range Cput {
				screen.Fprintln(screen.Msg, "green_black", i.peer, i.filename, i.rmflag)
			}
		}
	}
	if len(args) == 2 && args[1] == "?" {
		screen.Fprintln(screen.Msg, "green_black", cmd["put"][0])
		screen.Fprintln(screen.Msg, "green_black", cmd["put"][1])
		return
	}
	if len(args) == 3 {
		if addr := net.ParseIP(args[1]); addr != nil { // We have a valid IP Address
			for _, i := range Cput { // Don't add duplicates
				if args[1] == i.peer && args[2] == i.filename {
					return
				}
			}
			Cput = append(Cput, cmdPut{rmflag: false, peer: args[1], filename: args[2]})
			return
		}
	}
	screen.Fprintln(screen.Msg, "red_black", cmd["put"][0])
}

func putrm(args []string) {
	if len(args) == 1 {
		if len(Cput) == 0 {
			screen.Fprintln(screen.Msg, "green_black", "No current put transactions")
		} else {
			for _, i := range Cput {
				screen.Fprintln(screen.Msg, "green_black", i.peer, i.filename, i.rmflag)
			}
		}
	}
	if len(args) == 2 && args[1] == "?" {
		screen.Fprintln(screen.Msg, "green_black", cmd["putrm"][0])
		screen.Fprintln(screen.Msg, "green_black", cmd["putrm"][1])
		return
	}
	if len(args) == 3 {
		if addr := net.ParseIP(args[1]); addr != nil { // We have a valid IP Address
			for _, i := range Cput { // Don't add duplicates
				if args[1] == i.peer && args[2] == i.filename {
					return
				}
			}
			Cput = append(Cput, cmdPut{rmflag: true, peer: args[1], filename: args[2]})
			return
		}
	}
	screen.Fprintln(screen.Msg, "red_black", cmd["putrm"][0])
}

func quit(args []string) {
	exit(args)
}

type cmdRm struct {
	dirflag  bool   // Is a directory or not
	peer     string // Host we remove file from
	filename string // File or directory name to remove
}

// Crm - Remove a file or directory command
var Crm = []cmdRm{}

func rm(args []string) {
	if len(args) == 1 {
		if len(Crm) == 0 {
			screen.Fprintln(screen.Msg, "green_black", "No current rm transactions")
		} else {
			for _, i := range Crm {
				screen.Fprintln(screen.Msg, "green_black", i.peer, i.filename, i.dirflag)
			}
		}
	}
	if len(args) == 2 && args[1] == "?" {
		screen.Fprintln(screen.Msg, "green_black", cmd["rm"][0])
		screen.Fprintln(screen.Msg, "green_black", cmd["rm"][1])
		return
	}
	if len(args) == 3 {
		if addr := net.ParseIP(args[1]); addr != nil { // We have a valid IP Address
			for _, i := range Crm { // Don't add duplicates
				if args[1] == i.peer && args[2] == i.filename {
					return
				}
			}
			Crm = append(Crm, cmdRm{dirflag: false, peer: args[1], filename: args[2]})
			return
		}
	}
	screen.Fprintln(screen.Msg, "red_black", cmd["rm"][0])
}

func rmdir(args []string) {
	if len(args) == 1 {
		if len(Crm) == 0 {
			screen.Fprintln(screen.Msg, "green_black", "No current rm transactions")
		} else {
			for _, i := range Crm {
				screen.Fprintln(screen.Msg, "green_black", i.peer, i.filename, i.dirflag)
			}
		}
	}
	if len(args) == 2 && args[1] == "?" {
		screen.Fprintln(screen.Msg, "green_black", cmd["rmdir"][0])
		screen.Fprintln(screen.Msg, "green_black", cmd["rmdir"][1])
		return
	}
	if len(args) == 3 {
		if addr := net.ParseIP(args[1]); addr != nil { // We have a valid IP Address
			for _, i := range Crm { // Don't add duplicates
				if args[1] == i.peer && args[2] == i.filename {
					return
				}
			}
			Crm = append(Crm, cmdRm{dirflag: true, peer: args[1], filename: args[2]})
			return
		}
	}
	screen.Fprintln(screen.Msg, "red_black", cmd["rmdir"][0])
}

func rxwilling(args []string) {
	if len(args) == 1 {
		screen.Fprintln(screen.Msg, "green_black", "Receive Files", sarflags.Global.Rxwilling)
		return
	}
	if len(args) == 2 && args[1] == "?" {
		screen.Fprintln(screen.Msg, "green_black", cmd["rxwilling"][0])
		screen.Fprintln(screen.Msg, "green_black", cmd["rxwilling"][1])
		return
	}
	if len(args) == 2 {
		if args[1] == "on" {
			sarflags.Global.Rxwilling = "yes"
		}
		if args[1] == "off" {
			sarflags.Global.Rxwilling = "no"
		}
		if args[1] == "capable" {
			sarflags.Global.Rxwilling = "capable"
		}
		return
	}
	screen.Fprintln(screen.Msg, "red_black", cmd["rxwilling"][0])
}

// Cstream - Can the transfer be a stream (ie not a file)
var Cstream = "off"

func stream(args []string) {
	if len(args) == 1 {
		if sarflags.Global.Stream == "yes" {
			screen.Fprintln(screen.Msg, "green_black", "Can stream")
		} else {
			screen.Fprintln(screen.Msg, "green_black", "Cannot stream")
		}
		return
	}
	if len(args) == 2 && args[1] == "?" {
		screen.Fprintln(screen.Msg, "green_black", cmd["stream"][0])
		screen.Fprintln(screen.Msg, "green_black", cmd["stream"][1])
		return
	}
	if len(args) == 2 {
		if args[1] == "yes" {
			sarflags.Global.Stream = "yes"
		}
		if args[1] == "no" {
			sarflags.Global.Stream = "no"
		}
		return
	}
	screen.Fprintln(screen.Msg, "red_black", cmd["stream"][0])
}

type cmdTimeout struct {
	beacon   int
	request  int
	status   int
	transfer int
}

// Ctimeout - timeouts for responses 0 means no timeout
var Ctimeout = cmdTimeout{}

func timeout(args []string) {
	if len(args) == 1 {
		if Ctimeout.beacon == 0 {
			screen.Fprintln(screen.Msg, "green_black", "beacon: No timeout")
		} else {
			screen.Fprintln(screen.Msg, "green_black", "beacon:", Ctimeout.beacon, "seconds")
		}
		if Ctimeout.request == 0 {
			screen.Fprintln(screen.Msg, "green_black", "request: No timeout")
		} else {
			screen.Fprintln(screen.Msg, "green_black", "request:", Ctimeout.request, "seconds")
		}
		if Ctimeout.status == 0 {
			screen.Fprintln(screen.Msg, "green_black", "status: No timeout")
		} else {
			screen.Fprintln(screen.Msg, "green_black", "status:", Ctimeout.status, "seconds")
		}
		if Ctimeout.transfer == 0 {
			screen.Fprintln(screen.Msg, "green_black", "transfer: No timeout")
		} else {
			screen.Fprintln(screen.Msg, "green_black", "transfer:", Ctimeout.transfer, "seconds")
		}
		return
	}
	if len(args) == 2 {
		switch args[1] {
		case "?":
			screen.Fprintln(screen.Msg, "green_black", cmd["stream"][0])
			screen.Fprintln(screen.Msg, "green_black", cmd["stream"][1])
		case "beacon":
			if Ctimeout.beacon == 0 {
				screen.Fprintln(screen.Msg, "green_black", "beacon: No timeout")
			} else {
				screen.Fprintln(screen.Msg, "green_black", "beacon:", Ctimeout.beacon, "seconds")
			}
		case "request":
			if Ctimeout.request == 0 {
				screen.Fprintln(screen.Msg, "green_black", "request: No timeout")
			} else {
				screen.Fprintln(screen.Msg, "green_black", "request:", Ctimeout.request, "seconds")
			}
		case "status":
			if Ctimeout.status == 0 {
				screen.Fprintln(screen.Msg, "green_black", "status: No timeout")
			} else {
				screen.Fprintln(screen.Msg, "green_black", "status:", Ctimeout.status, "seconds")
			}
		case "transfer":
			if Ctimeout.transfer == 0 {
				screen.Fprintln(screen.Msg, "green_black", "transfer: No timeout")
			} else {
				screen.Fprintln(screen.Msg, "green_black", "transfer:", Ctimeout.transfer, "seconds")
			}
		default:
			screen.Fprintln(screen.Msg, "red_black", cmd["stream"][0])
		}
		return
	}
	if len(args) == 3 {
		if n, err := strconv.Atoi(args[2]); err == nil && n >= 0 {
			switch args[1] {
			case "beacon":
				Ctimeout.beacon = n
			case "request":
				Ctimeout.request = n
			case "status":
				Ctimeout.status = n
			case "transfer":
				Ctimeout.transfer = n
			}
			return
		}
		if args[2] == "off" {
			switch args[1] {
			case "beacon":
				Ctimeout.beacon = 0
			case "request":
				Ctimeout.request = 0
			case "status":
				Ctimeout.status = 0
			case "transfer":
				Ctimeout.transfer = 0
			}
			return
		}
	}
	screen.Fprintln(screen.Msg, "red_black", cmd["timeout"][0])
}

// Ctimestamp - What timestamp type are we using
var Ctimestamp = "off"

func timestamp(args []string) {
	if len(args) == 1 {
		screen.Fprintln(screen.Msg, "green_black", "Timestamps type is", Ctimestamp)
		return
	}
	if len(args) == 2 {
		switch args[1] {
		case "?":
			screen.Fprintln(screen.Msg, "green_black", cmd["timestamp"][0])
			screen.Fprintln(screen.Msg, "green_black", cmd["timestamp"][1])
		case "off":
			Ctimestamp = "off"
		case "32":
			Ctimestamp = "32"
		case "32_32":
			Ctimestamp = "32_32"
		case "64_32":
			Ctimestamp = "64_32"
		case "32_y2k":
			Ctimestamp = "32_y2k"
		default:
			screen.Fprintln(screen.Msg, "red_black", cmd["timestamp"][0])
		}
		return
	}
	screen.Fprintln(screen.Msg, "red_black", cmd["timestamp"][0])
}

// Ctimezone - What timezone to use for log - local or utc
var Ctimezone = "local"

func timezone(args []string) {
	if len(args) == 1 {
		screen.Fprintln(screen.Msg, "green_black", "Timezone is", Ctimezone)
		return
	}
	if len(args) == 2 {
		switch args[1] {
		case "?":
			screen.Fprintln(screen.Msg, "green_black", cmd["timezone"][0])
			screen.Fprintln(screen.Msg, "green_black", cmd["timezone"][1])
		case "local":
			Ctimezone = "local"
		case "utc":
			Ctimezone = "utc"
		default:
			screen.Fprintln(screen.Msg, "red_black", cmd["timezone"][0])
		}
		return
	}
	screen.Fprintln(screen.Msg, "red_black", cmd["timezone"][0])
}

func transfers(args []string) {
	if len(args) == 1 {
		if len(Cget) > 0 {
			screen.Fprintln(screen.Msg, "green_black", "Get Transfers in progress:")
			for _, i := range Cget {
				screen.Fprintln(screen.Msg, "green_black", "\t", i.peer, i.filename, i.rmflag)
			}
		}
		if len(Cput) > 0 {
			screen.Fprintln(screen.Msg, "green_black", "Put Transfers in progress:")
			for _, i := range Cput {
				screen.Fprintln(screen.Msg, "green_black", i.peer, i.filename, i.rmflag)
			}
		}
	}
	if len(args) == 2 {
		switch args[1] {
		case "?":
			screen.Fprintln(screen.Msg, "green_black", cmd["transfers"][0])
			screen.Fprintln(screen.Msg, "green_black", cmd["transfers"][1])
		case "get":
			if len(Cget) > 0 {
				screen.Fprintln(screen.Msg, "green_black", "Get Transfers in progress:")
				for _, i := range Cget {
					screen.Fprintln(screen.Msg, "green_black", "\t", i.peer, i.filename, i.rmflag)
				}
			} else {
				screen.Fprintln(screen.Msg, "green_black", "No current get transfers in progress")
			}
		case "put":
			if len(Cput) > 0 {
				screen.Fprintln(screen.Msg, "green_black", "Put Transfers in progress:")
				for _, i := range Cput {
					screen.Fprintln(screen.Msg, "green_black", "\t", i.peer, i.filename, i.rmflag)
				}
			} else {
				screen.Fprintln(screen.Msg, "green_black", "No current put transfers in progress")
			}
		default:
			screen.Fprintln(screen.Msg, "green_black", cmd["transfers"][0])
		}
		return
	}
	screen.Fprintln(screen.Msg, "green_black", cmd["transfers"][0])
}

func txwilling(args []string) {
	if len(args) == 1 {
		screen.Fprintln(screen.Msg, "green_black", "Transmit Files", sarflags.Global.Txwilling)
		return
	}
	if len(args) == 2 && args[1] == "?" {
		screen.Fprintln(screen.Msg, "green_black", cmd["txwilling"][0])
		screen.Fprintln(screen.Msg, "green_black", cmd["txwilling"][1])
		return
	}
	if len(args) == 2 {
		if args[1] == "on" {
			sarflags.Global.Txwilling = "on"
		}
		if args[1] == "off" {
			sarflags.Global.Txwilling = "off"
		}
		if args[1] == "capable" {
			sarflags.Global.Txwilling = "capable"
		}
		return
	}
	screen.Fprintln(screen.Msg, "red_black", cmd["txwilling"][0])
}

// Show all commands usage
func usage(args []string) {
	for _, val := range cmd {
		screen.Fprintln(screen.Msg, "cyan_black", val[0])
	}
}

type cmdfunc func([]string)

// Commands and function pointers to handle them
var cmdhandler = map[string]cmdfunc{
	"?":          help,
	"beacon":     beacon,
	"cancel":     cancel,
	"checksum":   checksum,
	"debug":      debug,
	"descriptor": descriptor,
	"eid":        eid,
	"exit":       exit,
	"files":      files,
	"freespace":  freespace,
	"get":        get,
	"getrm":      getrm,
	"help":       help,
	"history":    history,
	"home":       home,
	"ls":         ls,
	"peers":      peers,
	"prompt":     prompt,
	"put":        put,
	"putrm":      putrm,
	"quit":       exit,
	"rm":         rm,
	"rmdir":      rmdir,
	"rxwilling":  rxwilling,
	"stream":     stream,
	"timeout":    timeout,
	"timestamp":  timestamp,
	"timezone":   timezone,
	"transfers":  transfers,
	"txwilling":  txwilling,
	"usage":      usage,
}

// Command line interface usage and help
// Yes I would of loved to have these in the same map as above
// but could not work out how ... YET ...
var cmd = map[string][2]string{
	"?": [2]string{
		"?",
		"show valid commands. cmd ? shows the individual commands usage",
	},
	"beacon": [2]string{
		"beacon [off] [v4|v6|<ip>...] [secs]",
		"send a beacon every n secs",
	},
	"cancel": [2]string{
		"cancel <transfer>",
		"cancel a current transfer in progress",
	},
	"checksum": [2]string{
		"checksum [off|none|crc32|md5|sha1]",
		"set checksums required and type",
	},
	"debug": [2]string{
		"debug [off|0..9]",
		"set debug level 0..9",
	},
	"descriptor": [2]string{
		"descriptor [auto|16|32|64|128",
		"advertise & set default descriptor size",
	},
	"eid": [2]string{
		"eid [off|<eid>]",
		"manually set the eid #",
	},
	"exit": [2]string{
		"exit [0|1]",
		"exit saratoga",
	},
	"files": [2]string{
		"files",
		"list local files currently open and mode",
	},
	"freespace": [2]string{
		"freespace [yes|no]",
		"advertise freespace or show amount left",
	},
	"get": [2]string{
		"get [<peer> <filename>]",
		"get a file from a peer",
	},
	"getrm": [2]string{
		"getrm [<peer> <filename>",
		"get a file from a peer and remove it from peer when successful",
	},
	"help": [2]string{
		"help",
		"show commands",
	},
	"history": [2]string{
		"history",
		"show command history",
	},
	"home": [2]string{
		"home <dirname>",
		"set home directory for transfers",
	},
	"ls": [2]string{
		"ls [<peer> [<dirname>>]]",
		"show local or a peers directory contents",
	},
	"peers": [2]string{
		"peers",
		"list current peers found",
	},
	"prompt": [2]string{
		"prompt [<prompt>]",
		"set or show current prompt",
	},
	"put": [2]string{
		"put <peer> <filename>",
		"send a file to a peer",
	},
	"putrm": [2]string{
		"putrm <peer> <filename>",
		"send a file to a peer and then remove it from peer when successful",
	},
	"quit": [2]string{
		"quit [0|1]",
		"exit saratoga",
	},
	"rm": [2]string{
		"rm <peer> <filename>",
		"remove a file from a peer",
	},
	"rmdir": [2]string{
		"rmdir <peer> <dirname>",
		"remove a directory from a peer",
	},
	"rxwilling": [2]string{
		"rxwilling [on|off|capable",
		"current receive status or turn receive on/off/capable",
	},
	"stream": [2]string{
		"stream [yes|no]",
		"current stream status or can/cannot handle stream",
	},
	"timeout": [2]string{
		"timeout [request|transfer|status|beacon] <secs|off>",
		"timeout seconds for beacons, requests, transfers and status",
	},
	"timestamp": [2]string{
		"timestamp [off|32|64|32_32|64_32|32_y2k",
		"timestamp type to send",
	},
	"timezone": [2]string{
		"timezone [utc|local]",
		"show current or set to use local or universal time",
	},
	"transfers": [2]string{
		"transfers [get|put]",
		"list current active transfers",
	},
	"txwilling": [2]string{
		"txwilling [on|off|]",
		"show current transfer capability or set on/off/capable",
	},
	"usage": [2]string{
		"usage",
		"show usage of commands",
	},
}

// Docmd -- Execute the command entered
func Docmd(s string) error {
	if s == "" { // Handle just return
		return nil
	}

	vals := strings.Fields(s)
	// Look for the command and do it
	for c := range cmd {
		if c == vals[0] {
			fn, ok := cmdhandler[c]
			if ok {
				fn(vals)
				return nil
			}
		}
	}
	screen.Fprintln(screen.Msg, "bright_red_black", "Invalid command:", vals[0])
	return errors.New("Invalid command")
}
