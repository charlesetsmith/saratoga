package cli

import (
	"errors"
	"net"
	"strconv"
	"strings"

	"github.com/charlesetsmith/saratoga/src/transfer"

	"github.com/charlesetsmith/saratoga/src/beacon"
	"github.com/charlesetsmith/saratoga/src/sarflags"
	"github.com/charlesetsmith/saratoga/src/sarnet"
	"github.com/charlesetsmith/saratoga/src/screen"
	"github.com/jroimartin/gocui"
)

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

// Send count beacons to host
func sendbeacons(g *gocui.Gui, flags string, count uint, interval uint, host string) {
	// We have a hostname maybe with multiple addresses
	var addrs []string
	var err error
	var txb beacon.Beacon // The assembled beacon to transmit

	errflag := make(chan string, 1) // The return channel holding the saratoga errflag

	if addrs, err = net.LookupHost(host); err != nil {
		screen.Fprintln(g, "msg", "red_black", "Cannot resolve hostname: ", err)
		return
	}
	// Loop thru the address(s) for the host and send beacons to them
	for _, addr := range addrs {
		if err := txb.New(flags); err == nil {
			go txb.Send(g, addr, count, interval, errflag)
			errcode := <-errflag
			if errcode != "success" {
				screen.Fprintln(g, "msg", "red_black", "Error:", errcode,
					"Unable to send beacon to ", addr)
			}
			// screen.Fprintln(g, "msg", "green_black", "Sent: ", txb.Print())
		}
	}
	return
}

/* ********************************************************************************* */

// All of the different command line input handlers

// Beacon CLI Info
type cmdBeacon struct {
	flags    string   // Header Flags set for beacons
	count    uint     // How many beacons to send 0|1 == 1
	interval uint     // interval between beacons 0|1 == 1
	v4mcast  bool     // Sending to V4 Multicast
	v6mcast  bool     // Sending the V6 Multicast
	host     []string // Send unicast beacon to List of hosts
}

// clibeacon - Beacon commands
var clibeacon cmdBeacon

func handlebeacon(g *gocui.Gui, args []string) {

	// var bmu sync.Mutex // Protects beacon.Beacon structure (EID)

	clibeacon.flags = sarflags.Setglobal("beacon", "") // Initialise Global Beacon flags
	clibeacon.interval = Cinterval                     // Set up the correct interval

	// Show current Cbeacon flags and lists - beacon
	if len(args) == 1 {
		if clibeacon.count != 0 {
			screen.Fprintln(g, "msg", "green_black", clibeacon.count, "Beacons to be sent every %d secs",
				clibeacon.interval)
		} else {
			screen.Fprintln(g, "msg", "green_black", "Single Beacon to be sent")
		}
		if clibeacon.v4mcast == true {
			screen.Fprintln(g, "msg", "green_black", "Sending IPv4 multicast beacons")
		}
		if clibeacon.v6mcast == true {
			screen.Fprintln(g, "msg", "green_black", "Sending IPv6 multicast beacons")
		}
		if len(clibeacon.host) > 0 {
			screen.Fprintln(g, "msg", "green_black", "Sending beacons to:")
			for _, i := range clibeacon.host {
				screen.Fprintln(g, "msg", "green_black", "\t", i)
			}
		}
		if clibeacon.v4mcast == false && clibeacon.v6mcast == false &&
			len(clibeacon.host) == 0 {
			screen.Fprintln(g, "msg", "green_black", "No beacons currently being sent")
		}
		return
	}

	if len(args) == 2 {
		switch args[1] {
		case "?": // usage
			screen.Fprintln(g, "msg", "green_black", cmd["beacon"][0])
			screen.Fprintln(g, "msg", "green_black", cmd["beacon"][1])
			return
		case "off": // remove and disable all beacons
			clibeacon.flags = sarflags.Setglobal("beacon", "")
			clibeacon.count = 0
			clibeacon.interval = Cinterval
			clibeacon.host = nil
			screen.Fprintln(g, "msg", "green_black", "Beacons Disabled")
			return
		case "v4": // V4 Multicast
			screen.Fprintln(g, "msg", "green_black", "Sending beacons to IPv4 Multicast")
			clibeacon.flags = sarflags.Setglobal("beacon", "")
			clibeacon.v4mcast = true
			clibeacon.count = 1
			// Start up the beacon client sending count IPv4 beacons
			go sendbeacons(g, clibeacon.flags, clibeacon.count, clibeacon.interval, sarnet.IPv4Multicast)
			return
		case "v6": // V6 Multicast
			screen.Fprintln(g, "msg", "green_black", "Sending beacons to IPv6 Multicast")
			clibeacon.flags = sarflags.Setglobal("beacon", "")
			clibeacon.v6mcast = true
			clibeacon.count = 1
			// Start up the beacon client sending count IPv6 beacons
			go sendbeacons(g, clibeacon.flags, clibeacon.count, clibeacon.interval, sarnet.IPv6Multicast)
			return
		default: // beacon <count> or beacon <ipaddr>
			u32, err := strconv.ParseUint(args[1], 10, 32)
			if err == nil { // We have a number so it is a timer
				clibeacon.count = uint(u32)
				screen.Fprintln(g, "msg", "green_black", "Beacon count", clibeacon.count)
				return
			}
			go sendbeacons(g, clibeacon.flags, clibeacon.count, clibeacon.interval, args[1])
			return
		}
	}

	// beacon off <ipaddr> ...
	if args[1] == "off" && len(args) > 2 { // turn off following addresses
		screen.Fprintf(g, "msg", "green_black", "%s ", "Beacons turned off to")
		for i := 2; i < len(args); i++ { // Remove Address'es from lists
			if net.ParseIP(args[i]) != nil { // Do We have a valid IP Address
				clibeacon.host = removeValue(clibeacon.host, args[i])
				screen.Fprintf(g, "msg", "green_black", "%s ", args[i])
				if i == len(args)-1 {
					screen.Fprintln(g, "msg", "green_black", "")
				}
			} else {
				screen.Fprintln(g, "msg", "red_black", "Invalid IP Address:", args[i])
				screen.Fprintln(g, "cmd", "red_black", cmd["beacon"][0])
			}
		}
		return
	}

	// beacon <count> <ipaddr> ...
	var addrstart = 1
	u32, err := strconv.ParseUint(args[1], 10, 32)
	if err == nil { // We have a number so it is a timer
		clibeacon.count = uint(u32)
		screen.Fprintln(g, "msg", "green_black", "Beacon counter set to", clibeacon.count)
		addrstart = 2
	}
	// beacon [count] <ipaddr> ...
	screen.Fprintf(g, "msg", "green_black", "Sending %d beacons to: ",
		clibeacon.count)
	for i := addrstart; i < len(args); i++ { // Add Address'es to lists
		screen.Fprintf(g, "msg", "green_black", "%s ", args[i])
		switch args[i] {
		case "v4":
			go sendbeacons(g, clibeacon.flags, clibeacon.count,
				clibeacon.interval, sarnet.IPv4Multicast)
		case "v6":
			go sendbeacons(g, clibeacon.flags, clibeacon.count,
				clibeacon.interval, sarnet.IPv6Multicast)
		default:
			go sendbeacons(g, clibeacon.flags, clibeacon.count,
				clibeacon.interval, args[i])
		}
	}
	screen.Fprintln(g, "msg", "green_black", "")
}

// blind put/send a file to a destination
func putblind(g *gocui.Gui, args []string) {

	errflag := make(chan string, 1) // The return channel holding the saratoga errflag

	if len(args) == 1 {
		transfer.Info(g, "putblind")
		return
	}
	if len(args) == 2 && args[1] == "?" {
		screen.Fprintln(g, "msg", "green_black", cmd["blindput"][0])
		screen.Fprintln(g, "msg", "green_black", cmd["blindput"][1])
		return
	}
	if len(args) == 3 {
		var t transfer.Transfer
		// We send the Metadata and do not bother with request/status exchange
		if err := t.New(g, "putblind", args[1], args[2], true, "transfer=file"); err != nil {
			go t.Client(g, errflag)
			errcode := <-errflag
			if errcode != "success" {
				screen.Fprintln(g, "msg", "red_black", "Error:", errcode,
					"Unable to send file: ", t.Print())
			}
		}
		return
	}
	screen.Fprintln(g, "msg", "red_black", cmd["blindput"][0])
}

func cancel(g *gocui.Gui, args []string) {
	screen.Fprintln(g, "msg", "green_black", args)
}

func checksum(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		screen.Fprintln(g, "msg", "green_black", "Checksum", sarflags.Global["csumtype"])
		return
	}
	if len(args) == 2 && args[1] == "?" { // usage
		screen.Fprintln(g, "msg", "green_black", cmd["checksum"][0])
		screen.Fprintln(g, "msg", "green_black", cmd["checksum"][1])
		return
	}
	if len(args) == 2 {
		switch args[1] {
		case "off", "none":
			sarflags.Global["csumtype"] = "none"
		case "crc32":
			sarflags.Global["csumtype"] = "crc32"
		case "md5":
			sarflags.Global["csumtype"] = "md5"
		case "sha1":
			sarflags.Global["csumtype"] = "sha1"
		default:
			screen.Fprintln(g, "cmd", "green_red", cmd["checksum"][0])
		}
		return
	}
	screen.Fprintln(g, "cmd", "green_red", cmd["checksum"][0])
}

// Cdebug - Debug level 0 is off
var Cdebug = 0

func debug(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		screen.Fprintln(g, "msg", "green_black", "Debug level", Cdebug)
		return
	}
	if len(args) == 2 && args[1] == "?" { // usage
		screen.Fprintln(g, "msg", "green_black", cmd["debug"][0])
		screen.Fprintln(g, "msg", "green_black", cmd["debug"][1])
		return
	}
	if len(args) == 2 {
		switch args[1] {
		case "off":
			Cdebug = 0
		case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
			Cdebug, _ = strconv.Atoi(args[1])
		default:
			screen.Fprintln(g, "msg", "green_red", cmd["debug"][0])
		}
		return
	}
	screen.Fprintln(g, "msg", "green_red", cmd["debug"][0])
}

func descriptor(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		screen.Fprintln(g, "msg", "green_black", "Descriptor", sarflags.Global["descriptor"])
		return
	}
	if len(args) == 2 && args[1] == "?" { // usage
		screen.Fprintln(g, "msg", "green_black", cmd["descriptor"][0])
		screen.Fprintln(g, "msg", "green_black", cmd["descriptor"][1])
		return
	}
	if len(args) == 2 {
		switch args[1] {
		case "auto":
			if sarflags.MaxUint <= sarflags.MaxUint16 {
				sarflags.Global["descriptor"] = "d16"
				break
			}
			if sarflags.MaxUint <= sarflags.MaxUint32 {
				sarflags.Global["descriptor"] = "d32"
				break
			}
			if sarflags.MaxUint <= sarflags.MaxUint64 {
				sarflags.Global["descriptor"] = "d64"
				break
			}
			screen.Fprintln(g, "msg", "red_black", "128 bit descriptors not supported on this platform")
			break
		case "d16":
			if sarflags.MaxUint > sarflags.MaxUint16 {
				sarflags.Global["descriptor"] = "d16"
			} else {
				screen.Fprintln(g, "msg", "red_black", "16 bit descriptors not supported on this platform")
			}
		case "d32":
			if sarflags.MaxUint > sarflags.MaxUint32 {
				sarflags.Global["descriptor"] = "d32"
			} else {
				screen.Fprintln(g, "msg", "red_black", "32 bit descriptors not supported on this platform")
			}
		case "d64":
			if sarflags.MaxUint <= sarflags.MaxUint64 {
				sarflags.Global["descriptor"] = "d64"
			} else {
				screen.Fprintln(g, "msg", "red_black", "64 bit descriptors not supported on this platform")
			}
		case "d128":
			screen.Fprintln(g, "msg", "red_black", "128 bit descriptors not supported on this platform")
		default:
			screen.Fprintln(g, "msg", "red_black", "usage: ", cmd["descriptor"][0])
		}
		screen.Fprintln(g, "msg", "green_black", "Descriptor size is", sarflags.Global["descriptor"])
		return
	}
	screen.Fprintln(g, "msg", "red_black", "usage: ", cmd["descriptor"][0])
}

// Cexit = Exit level to quit from saratoga
var Cexit = -1

// Quit saratoga
func exit(g *gocui.Gui, args []string) {
	if len(args) > 2 { // usage
		screen.Fprintln(g, "msg", "red_black", cmd["exit"][0])
		return
	}
	if len(args) == 1 { // exit 0
		Cexit = 0
		screen.Fprintln(g, "msg", "green_black", "Good Bye!")
		return
	}
	if len(args) == 2 {
		switch args[1] {
		case "?": // Usage
			screen.Fprintln(g, "msg", "green_black", cmd["exit"][0])
			screen.Fprintln(g, "msg", "green_black", cmd["exit"][1])
		case "0": // exit 0
			Cexit = 0
			screen.Fprintln(g, "msg", "green_black", "Good Bye!")
		case "1": // exit 1
			Cexit = 1
			screen.Fprintln(g, "msg", "green_black", "Good Bye!")
		default: // Help
			screen.Fprintln(g, "msg", "red_black", cmd["exit"][0])
		}
		return
	}
	screen.Fprintln(g, "msg", "red_black", cmd["exit"][0])
}

// Cfiles - Currently open file list
var Cfiles []string

func files(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		if len(Cfiles) == 0 {
			screen.Fprintln(g, "msg", "green_black", "No currently open files")
			return
		}
		for _, i := range Cfiles {
			screen.Fprintln(g, "msg", "green_black", i)
		}
		return
	}
	if len(args) == 2 && args[1] == "?" { // usage
		screen.Fprintln(g, "msg", "green_black", cmd["files"][0])
		screen.Fprintln(g, "msg", "green_black", cmd["files"][1])
		return
	}
	screen.Fprintln(g, "msg", "red_black", cmd["files"][0])
}

func freespace(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		if sarflags.Global["freespace"] == "yes" {
			screen.Fprintln(g, "msg", "green_black", "Free space advertised")
		} else {
			screen.Fprintln(g, "msg", "green_black", "Free space not advertised")
		}
		return
	}
	if len(args) == 2 && args[1] == "?" { // usage
		screen.Fprintln(g, "msg", "green_black", cmd["freespace"][0])
		screen.Fprintln(g, "msg", "green_black", cmd["freespace"][1])
		return
	}
	if len(args) == 2 {
		if args[1] == "yes" {
			sarflags.Global["freespace"] = "yes"
			return
		}
		if args[1] == "no" {
			sarflags.Global["freespace"] = "no"
			return
		}
		// screen.Fprintln(g, "msg", "red_black", "usage: ", cmd["freespace][0]"])
	}
	screen.Fprintln(g, "msg", "red_black", "usage: ", cmd["freespace"][0])
}

func get(g *gocui.Gui, args []string) {

	if len(args) == 1 {
		transfer.Info(g, "get")
		return
	}
	if len(args) == 2 && args[1] == "?" {
		screen.Fprintln(g, "msg", "green_black", cmd["get"][0])
		screen.Fprintln(g, "msg", "green_black", cmd["get"][1])
		return
	}
	if len(args) == 3 {
		var t transfer.Transfer
		if err := t.New(g, "get", args[1], args[2], false, "reqtype=get,fileordir=file"); err != nil {

		}
		return
	}
	screen.Fprintln(g, "msg", "red_black", cmd["get"][0])
}

func getdir(g *gocui.Gui, args []string) {

	if len(args) == 1 {
		transfer.Info(g, "getdir")
		return
	}
	if len(args) == 2 && args[1] == "?" {
		screen.Fprintln(g, "msg", "green_black", cmd["getdir"][0])
		screen.Fprintln(g, "msg", "green_black", cmd["getdir"][1])
		return
	}
	if len(args) == 3 {
		var t transfer.Transfer
		if err := t.New(g, "getdir", args[1], args[2], false, "reqtype=getdir,fileordir=directory"); err != nil {

		}
		return
	}
	screen.Fprintln(g, "msg", "red_black", cmd["getdir"][0])
}

func getrm(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		transfer.Info(g, "getrm")
		return
	}
	if len(args) == 2 && args[1] == "?" {
		screen.Fprintln(g, "msg", "green_black", cmd["getrm"][0])
		screen.Fprintln(g, "msg", "green_black", cmd["getrm"][1])
		return
	}
	if len(args) == 3 {
		var t transfer.Transfer
		if err := t.New(g, "getrm", args[1], args[2], false, "reqtype=getdelete,fileordir=file"); err != nil {

		}
		return
	}
	screen.Fprintln(g, "msg", "red_black", cmd["getrm"][0])
}

func help(g *gocui.Gui, args []string) {
	for key, val := range cmd {
		screen.Fprintln(g, "msg", "magenta_black", key, "-", val[1])
	}
}

// Cinterval - seconds between beacon sends
var Cinterval uint

func interval(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		if Cinterval == 0 {
			screen.Fprintln(g, "msg", "green_black", "Single Beacon Interation")
		} else {
			screen.Fprintln(g, "msg", "green_black", "Beacons sent every", Cinterval, "seconds")
		}
		return
	}
	if len(args) == 2 {
		switch args[1] {
		case "?":
			screen.Fprintln(g, "msg", "green_black", cmd["interval"][0])
			screen.Fprintln(g, "msg", "green_black", cmd["interval"][1])
		case "off":
			Cinterval = 0

		default:
			if n, err := strconv.Atoi(args[1]); err == nil && n >= 0 {
				Cinterval = uint(n)
				return
			}
		}
		screen.Fprintln(g, "msg", "red_black", cmd["interval"][0])
	}

}

func history(g *gocui.Gui, args []string) {
	screen.Fprintln(g, "msg", "green_black", args)
}

func home(g *gocui.Gui, args []string) {
	screen.Fprintln(g, "msg", "green_black", args)
}

func ls(g *gocui.Gui, args []string) {
	screen.Fprintln(g, "msg", "green_black", args)
}

// Cprompt - Command line prompt
var Cprompt = "saratoga"

func prompt(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		screen.Fprintln(g, "msg", "green_black", "Current prompt is", Cprompt)
		return
	}
	if len(args) == 2 && args[1] == "?" { // usage
		screen.Fprintln(g, "msg", "green_black", cmd["prompt"][0])
		screen.Fprintln(g, "msg", "green_black", cmd["prompt"][1])
		return
	}
	if len(args) == 2 {
		Cprompt = args[1]
		return
	}
}

// Display all of the peer information learned frm beacons
func peers(g *gocui.Gui, args []string) {
	if len(beacon.Peers) == 0 {
		screen.Fprintln(g, "msg", "purple_black", "No Peers")
		return
	}
	screen.Fprintf(g, "msg", "green_black", "Address | Freespace | EID | Max Desc Size | Created | Modified\n")
	for p := range beacon.Peers {
		screen.Fprintf(g, "msg", "green_black", "%s | %dMB | %s | %s | %s | %s\n",
			beacon.Peers[p].Addr,
			beacon.Peers[p].Freespace/1024,
			beacon.Peers[p].Eid,
			beacon.Peers[p].Maxdesc,
			beacon.Peers[p].Created.Print(),
			beacon.Peers[p].Updated.Print())
	}
}

// put/send a file to a destination
func put(g *gocui.Gui, args []string) {

	errflag := make(chan string, 1) // The return channel holding the saratoga errflag

	if len(args) == 1 {
		transfer.Info(g, "put")
		return
	}
	if len(args) == 2 && args[1] == "?" {
		screen.Fprintln(g, "msg", "green_black", cmd["put"][0])
		screen.Fprintln(g, "msg", "green_black", cmd["put"][1])
		return
	}
	if len(args) == 3 {
		var t transfer.Transfer
		if err := t.New(g, "put", args[1], args[2], false, "reqtype=put,fileordir=file"); err != nil {
			go t.Client(g, errflag)
			errcode := <-errflag
			if errcode != "success" {
				screen.Fprintln(g, "msg", "red_black", "Error:", errcode,
					"Unable to send file: ", t.Print())
			}
		}
		return
	}
	screen.Fprintln(g, "msg", "red_black", cmd["put"][0])
}

// put/send a file file to a remote destination then remove it from the origin
func putrm(g *gocui.Gui, args []string) {

	errflag := make(chan string, 1) // The return channel holding the saratoga errflag

	if len(args) == 1 {
		transfer.Info(g, "putrm")
		return
	}
	if len(args) == 2 && args[1] == "?" {
		screen.Fprintln(g, "msg", "green_black", cmd["putrm"][0])
		screen.Fprintln(g, "msg", "green_black", cmd["putrm"][1])
		return
	}
	if len(args) == 3 {
		var t transfer.Transfer
		if err := t.New(g, "putrm", args[1], args[2], false, "reqtype=putdelete,fileordir=file"); err != nil {
			go t.Client(g, errflag)
			errcode := <-errflag
			if errcode != "success" {
				screen.Fprintln(g, "msg", "red_black", "Error:", errcode,
					"Unable to send file: ", t.Print())
			} else {
				// NOW REMOVE THE LOCAL FILE AS IT SUCCEEDED
			}
		}
		return
	}
	screen.Fprintln(g, "msg", "red_black", cmd["putrm"][0])
}

// remove a file from a remote destination
func rm(g *gocui.Gui, args []string) {

	if len(args) == 1 {
		transfer.Info(g, "rm")
		return
	}
	if len(args) == 2 && args[1] == "?" {
		screen.Fprintln(g, "msg", "green_black", cmd["rm"][0])
		screen.Fprintln(g, "msg", "green_black", cmd["rm"][1])
		return
	}
	if len(args) == 3 {
		var t transfer.Transfer
		if err := t.New(g, "rm", args[1], args[2], false, "reqtype=delete,fileordir=file"); err != nil {

		}
		return
	}
	screen.Fprintln(g, "msg", "red_black", cmd["rm"][0])
}

// remove a directory from a remote destination
func rmdir(g *gocui.Gui, args []string) {

	if len(args) == 1 {
		transfer.Info(g, "rmdir")
		return
	}
	if len(args) == 2 && args[1] == "?" {
		screen.Fprintln(g, "msg", "green_black", cmd["rmdir"][0])
		screen.Fprintln(g, "msg", "green_black", cmd["rmdir"][1])
		return
	}
	if len(args) == 3 {
		var t transfer.Transfer
		if err := t.New(g, "rmdir", args[1], args[2], false, "reqtype=delete,fileordir=directory"); err != nil {

		}
		return
	}
	screen.Fprintln(g, "msg", "red_black", cmd["rmdir"][0])
}

// Are we willing to transmit files
func rxwilling(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		screen.Fprintln(g, "msg", "green_black", "Receive Files", sarflags.Global["rxwilling"])
		return
	}
	if len(args) == 2 && args[1] == "?" {
		screen.Fprintln(g, "msg", "green_black", cmd["rxwilling"][0])
		screen.Fprintln(g, "msg", "green_black", cmd["rxwilling"][1])
		return
	}
	if len(args) == 2 {
		if args[1] == "on" {
			sarflags.Global["rxwilling"] = "yes"
		}
		if args[1] == "off" {
			sarflags.Global["rxwilling"] = "no"
		}
		if args[1] == "capable" {
			sarflags.Global["rxwilling"] = "capable"
		}
		return
	}
	screen.Fprintln(g, "msg", "red_black", cmd["rxwilling"][0])
}

// Cstream - Can the transfer be a stream (ie not a file)
var Cstream = "off"

// source is a named pipe not a file
func stream(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		if sarflags.Global["stream"] == "yes" {
			screen.Fprintln(g, "msg", "green_black", "Can stream")
		} else {
			screen.Fprintln(g, "msg", "green_black", "Cannot stream")
		}
		return
	}
	if len(args) == 2 && args[1] == "?" {
		screen.Fprintln(g, "msg", "green_black", cmd["stream"][0])
		screen.Fprintln(g, "msg", "green_black", cmd["stream"][1])
		return
	}
	if len(args) == 2 {
		if args[1] == "yes" {
			sarflags.Global["stream"] = "yes"
		}
		if args[1] == "no" {
			sarflags.Global["stream"] = "no"
		}
		return
	}
	screen.Fprintln(g, "msg", "red_black", cmd["stream"][0])
}

type cmdTimeout struct {
	request  int
	status   int
	transfer int
}

// Ctimeout - timeouts for responses 0 means no timeout
var Ctimeout = cmdTimeout{}

// set timeouts for responses to request/status/transfer in seconds
func timeout(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		if Ctimeout.request == 0 {
			screen.Fprintln(g, "msg", "green_black", "request: No timeout")
		} else {
			screen.Fprintln(g, "msg", "green_black", "request:", Ctimeout.request, "seconds")
		}
		if Ctimeout.status == 0 {
			screen.Fprintln(g, "msg", "green_black", "status: No timeout")
		} else {
			screen.Fprintln(g, "msg", "green_black", "status:", Ctimeout.status, "seconds")
		}
		if Ctimeout.transfer == 0 {
			screen.Fprintln(g, "msg", "green_black", "transfer: No timeout")
		} else {
			screen.Fprintln(g, "msg", "green_black", "transfer:", Ctimeout.transfer, "seconds")
		}
		return
	}
	if len(args) == 2 {
		switch args[1] {
		case "?":
			screen.Fprintln(g, "msg", "green_black", cmd["stream"][0])
			screen.Fprintln(g, "msg", "green_black", cmd["stream"][1])
		case "request":
			if Ctimeout.request == 0 {
				screen.Fprintln(g, "msg", "green_black", "request: No timeout")
			} else {
				screen.Fprintln(g, "msg", "green_black", "request:", Ctimeout.request, "seconds")
			}
		case "status":
			if Ctimeout.status == 0 {
				screen.Fprintln(g, "msg", "green_black", "status: No timeout")
			} else {
				screen.Fprintln(g, "msg", "green_black", "status:", Ctimeout.status, "seconds")
			}
		case "transfer":
			if Ctimeout.transfer == 0 {
				screen.Fprintln(g, "msg", "green_black", "transfer: No timeout")
			} else {
				screen.Fprintln(g, "msg", "green_black", "transfer:", Ctimeout.transfer, "seconds")
			}
		default:
			screen.Fprintln(g, "msg", "red_black", cmd["stream"][0])
		}
		return
	}
	if len(args) == 3 {
		if n, err := strconv.Atoi(args[2]); err == nil && n >= 0 {
			switch args[1] {
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
	screen.Fprintln(g, "msg", "red_black", cmd["timeout"][0])
}

// Ctimestamp - What timestamp type are we using
var Ctimestamp = "off"

// set the timestamp type we are using
func timestamp(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		screen.Fprintln(g, "msg", "green_black", "Timestamps type is", Ctimestamp)
		return
	}
	if len(args) == 2 {
		switch args[1] {
		case "?":
			screen.Fprintln(g, "msg", "green_black", cmd["timestamp"][0])
			screen.Fprintln(g, "msg", "green_black", cmd["timestamp"][1])
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
			screen.Fprintln(g, "msg", "red_black", cmd["timestamp"][0])
		}
		return
	}
	screen.Fprintln(g, "msg", "red_black", cmd["timestamp"][0])
}

// Ctimezone - What timezone to use for log - local or utc
var Ctimezone = "local"

// set the timezone we use for logs local or utc
func timezone(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		screen.Fprintln(g, "msg", "green_black", "Timezone is", Ctimezone)
		return
	}
	if len(args) == 2 {
		switch args[1] {
		case "?":
			screen.Fprintln(g, "msg", "green_black", cmd["timezone"][0])
			screen.Fprintln(g, "msg", "green_black", cmd["timezone"][1])
		case "local":
			Ctimezone = "local"
		case "utc":
			Ctimezone = "utc"
		default:
			screen.Fprintln(g, "msg", "red_black", cmd["timezone"][0])
		}
		return
	}
	screen.Fprintln(g, "msg", "red_black", cmd["timezone"][0])
}

// show current transfers in progress & % completed
func transfers(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		transfer.Info(g, "")
		return
	}
	if len(args) == 2 {
		switch args[1] {
		case "?":
			screen.Fprintln(g, "msg", "green_black", cmd["transfers"][0])
			screen.Fprintln(g, "msg", "green_black", cmd["transfers"][1])
		case "get", "getrm", "getdir", "put", "putblind", "putrm", "rm", "rmdir":
			transfer.Info(g, args[1])
		default:
			screen.Fprintln(g, "msg", "green_black", cmd["transfers"][0])
		}
		return
	}
	screen.Fprintln(g, "msg", "green_black", cmd["transfers"][0])
}

// we are willing to transmit files
func txwilling(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		screen.Fprintln(g, "msg", "green_black", "Transmit Files", sarflags.Global["txwilling"])
		return
	}
	if len(args) == 2 && args[1] == "?" {
		screen.Fprintln(g, "msg", "green_black", cmd["txwilling"][0])
		screen.Fprintln(g, "msg", "green_black", cmd["txwilling"][1])
		return
	}
	if len(args) == 2 {
		if args[1] == "on" {
			sarflags.Global["txwilling"] = "on"
		}
		if args[1] == "off" {
			sarflags.Global["txwilling"] = "off"
		}
		if args[1] == "capable" {
			sarflags.Global["txwilling"] = "capable"
		}
		return
	}
	screen.Fprintln(g, "msg", "red_black", cmd["txwilling"][0])
}

// Show all commands usage
func usage(g *gocui.Gui, args []string) {
	for _, val := range cmd {
		screen.Fprintln(g, "msg", "cyan_black", val[0])
	}
}

/* ************************************************************************** */

type cmdfunc func(*gocui.Gui, []string)

// Commands and function pointers to handle them
var cmdhandler = map[string]cmdfunc{
	"?":          help,
	"beacon":     handlebeacon,
	"cancel":     cancel,
	"checksum":   checksum,
	"debug":      debug,
	"descriptor": descriptor,
	"exit":       exit,
	"files":      files,
	"freespace":  freespace,
	"get":        get,
	"getdir":     getdir,
	"getrm":      getrm,
	"help":       help,
	"history":    history,
	"home":       home,
	"interval":   interval,
	"ls":         ls,
	"peers":      peers,
	"prompt":     prompt,
	"put":        put,
	"putblind":   putblind,
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
	"blindput": [2]string{
		"blindput <peer> <filename>",
		"send a file to a peer with no initial request/status exchange",
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
		"descriptor [auto|d16|d32|d64|d128",
		"advertise & set default descriptor size",
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
	"getdir": [2]string{
		"getdir [<peer> <dirname>]",
		"get a directlory listing from a peer",
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
	"interval": [2]string{
		"interval [seconds]",
		"set interval between beacons",
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
	// Timeout for a request is how long I wait after I send a request before I cancel it
	// Timout for transfer is how long I wait before I receive next frame in a transfer
	// Timeout for status is how long I wait between receiving a status frame
	"timeout": [2]string{
		"timeout [request|transfer|status] <secs|off>",
		"timeout in seconds for requests, transfers and status",
	},
	"timestamp": [2]string{
		"timestamp [off|32|64|32_32|64_32|32_y2k]",
		"timestamp type to send",
	},
	"timezone": [2]string{
		"timezone [utc|local]",
		"show current or set to use local or universal time",
	},
	"transfers": [2]string{
		"transfers [get|getrm|getdir|put|putblind|putrm|rm|rmdir]",
		"list current active transfers of specific type or all",
	},
	"txwilling": [2]string{
		"txwilling [on|off|capable]",
		"show current transfer capability or set on/off/capable",
	},
	"usage": [2]string{
		"usage",
		"show usage of commands",
	},
}

// Docmd -- Execute the command entered
func Docmd(g *gocui.Gui, s string) error {
	if s == "" { // Handle just return
		return nil
	}

	// Get rid of leading and trailing whitespace
	s = strings.TrimSpace(s)
	vals := strings.Fields(s)
	// Look for the command and do it
	for c := range cmd {
		if c == vals[0] {
			fn, ok := cmdhandler[c]
			if ok {
				fn(g, vals)
				return nil
			}
		}
	}
	screen.Fprintln(g, "msg", "red_black", "Invalid command:", vals[0])
	return errors.New("Invalid command")
}
