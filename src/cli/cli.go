package cli

import (
	"errors"
	"fmt"
	"net"
	"sort"
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

func cmdbeacon(g *gocui.Gui, args []string) {

	// var bmu sync.Mutex // Protects beacon.Beacon structure (EID)

	clibeacon.flags = sarflags.Setglobal("beacon")      // Initialise Global Beacon flags
	clibeacon.interval = sarflags.Cli.Timeout.Binterval // Set up the correct interval

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
			clibeacon.flags = sarflags.Setglobal("beacon")
			clibeacon.count = 0
			clibeacon.interval = sarflags.Cli.Timeout.Binterval
			clibeacon.host = nil
			screen.Fprintln(g, "msg", "green_black", "Beacons Disabled")
			return
		case "v4": // V4 Multicast
			screen.Fprintln(g, "msg", "green_black", "Sending beacons to IPv4 Multicast")
			clibeacon.flags = sarflags.Setglobal("beacon")
			clibeacon.v4mcast = true
			clibeacon.count = 1
			// Start up the beacon client sending count IPv4 beacons
			go sendbeacons(g, clibeacon.flags, clibeacon.count, clibeacon.interval, sarnet.IPv4Multicast)
			return
		case "v6": // V6 Multicast
			screen.Fprintln(g, "msg", "green_black", "Sending beacons to IPv6 Multicast")
			clibeacon.flags = sarflags.Setglobal("beacon")
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

func cancel(g *gocui.Gui, args []string) {
	screen.Fprintln(g, "msg", "green_black", args)
}

func checksum(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		screen.Fprintln(g, "msg", "green_black", "Checksum", sarflags.Cli.Global["csumtype"])
		return
	}
	if len(args) == 2 && args[1] == "?" { // usage
		screen.Fprintln(g, "msg", "green_black", cmd["checksum"][0])
		screen.Fprintln(g, "msg", "green_black", cmd["checksum"][1])
		return
	}
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()
	if len(args) == 2 {
		switch args[1] {
		case "off", "none":
			sarflags.Cli.Global["csumtype"] = "none"
		case "crc32":
			sarflags.Cli.Global["csumtype"] = "crc32"
		case "md5":
			sarflags.Cli.Global["csumtype"] = "md5"
		case "sha1":
			sarflags.Cli.Global["csumtype"] = "sha1"
		default:
			screen.Fprintln(g, "cmd", "green_red", cmd["checksum"][0])
		}
		return
	}
	screen.Fprintln(g, "cmd", "green_red", cmd["checksum"][0])
}

func descriptor(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		screen.Fprintln(g, "msg", "green_black", "Descriptor", sarflags.Cli.Global["descriptor"])
		return
	}
	if len(args) == 2 && args[1] == "?" { // usage
		screen.Fprintln(g, "msg", "green_black", cmd["descriptor"][0])
		screen.Fprintln(g, "msg", "green_black", cmd["descriptor"][1])
		return
	}
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()
	if len(args) == 2 {
		switch args[1] {
		case "auto":
			if sarflags.MaxUint <= sarflags.MaxUint16 {
				sarflags.Cli.Global["descriptor"] = "d16"
				break
			}
			if sarflags.MaxUint <= sarflags.MaxUint32 {
				sarflags.Cli.Global["descriptor"] = "d32"
				break
			}
			if sarflags.MaxUint <= sarflags.MaxUint64 {
				sarflags.Cli.Global["descriptor"] = "d64"
				break
			}
			screen.Fprintln(g, "msg", "red_black", "128 bit descriptors not supported on this platform")
			break
		case "d16":
			if sarflags.MaxUint > sarflags.MaxUint16 {
				sarflags.Cli.Global["descriptor"] = "d16"
			} else {
				screen.Fprintln(g, "msg", "red_black", "16 bit descriptors not supported on this platform")
			}
		case "d32":
			if sarflags.MaxUint > sarflags.MaxUint32 {
				sarflags.Cli.Global["descriptor"] = "d32"
			} else {
				screen.Fprintln(g, "msg", "red_black", "32 bit descriptors not supported on this platform")
			}
		case "d64":
			if sarflags.MaxUint <= sarflags.MaxUint64 {
				sarflags.Cli.Global["descriptor"] = "d64"
			} else {
				screen.Fprintln(g, "msg", "red_black", "64 bit descriptors not supported on this platform")
			}
		case "d128":
			screen.Fprintln(g, "msg", "red_black", "128 bit descriptors not supported on this platform")
		default:
			screen.Fprintln(g, "msg", "red_black", "usage: ", cmd["descriptor"][0])
		}
		screen.Fprintln(g, "msg", "green_black", "Descriptor size is", sarflags.Cli.Global["descriptor"])
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

// MORE WORK TO DO HERE!!!!! USE TRANSFERS LIST
func files(g *gocui.Gui, args []string) {
	var flist []string

	if len(args) == 1 {
		if len(flist) == 0 {
			screen.Fprintln(g, "msg", "green_black", "No currently open files")
			return
		}
		for _, i := range flist {
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
		if sarflags.Cli.Global["freespace"] == "yes" {
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
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()
	if len(args) == 2 {
		if args[1] == "yes" {
			sarflags.Cli.Global["freespace"] = "yes"
			return
		}
		if args[1] == "no" {
			sarflags.Cli.Global["freespace"] = "no"
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
		if err := t.New(g, "get", args[1], args[2]); err != nil {

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
		if err := t.New(g, "getdir", args[1], args[2]); err != nil {

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
		if err := t.New(g, "getrm", args[1], args[2]); err != nil {

		}
		return
	}
	screen.Fprintln(g, "msg", "red_black", cmd["getrm"][0])
}

func help(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		var sslice sort.StringSlice
		for key, val := range cmd {
			sslice = append(sslice, fmt.Sprintf("%s - %s", key, val[1]))
		}
		sort.Sort(sslice)
		var sbuf string
		for key := 0; key < len(sslice); key++ {
			sbuf += fmt.Sprintf("%s\n", sslice[key])
		}
		screen.Fprintln(g, "msg", "magenta_black", sbuf)
		return
	}
	if len(args) == 2 {
		if args[1] == "?" {
			var sslice sort.StringSlice
			for key, val := range cmd {
				sslice = append(sslice, fmt.Sprintf("%s - %s\n  %s", key, val[0], val[1]))
			}
			sort.Sort(sslice)
			var sbuf string
			for key := 0; key < len(sslice); key++ {
				sbuf += fmt.Sprintf("%s\n", sslice[key])
			}
			screen.Fprintln(g, "msg", "magenta_black", sbuf)
			return
		}
	}
	screen.Fprintln(g, "msg", "red_black", cmd["help"][0])
}

func interval(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		if sarflags.Cli.Timeout.Binterval == 0 {
			screen.Fprintln(g, "msg", "green_black", "Single Beacon Interation")
		} else {
			screen.Fprintln(g, "msg", "green_black", "Beacons sent every",
				sarflags.Cli.Timeout.Binterval, "seconds")
		}
		return
	}
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()
	if len(args) == 2 {
		switch args[1] {
		case "?":
			screen.Fprintln(g, "msg", "green_black", cmd["interval"][0])
			screen.Fprintln(g, "msg", "green_black", cmd["interval"][1])
		case "off":
			sarflags.Cli.Timeout.Binterval = 0

		default:
			if n, err := strconv.Atoi(args[1]); err == nil && n >= 0 {
				sarflags.Cli.Timeout.Binterval = uint(n)
				return
			}
		}
		screen.Fprintln(g, "msg", "red_black", cmd["interval"][0])
	}

}

func history(g *gocui.Gui, args []string) {
	if len(args) == 2 {
		switch args[1] {
		case "?":
			screen.Fprintln(g, "msg", "green_black", cmd["history"][0])
			screen.Fprintln(g, "msg", "green_black", cmd["history"][1])
		default:
			screen.Fprintln(g, "msg", "green_black", "History not implemented yet")
		}
	}
	screen.Fprintln(g, "msg", "green_black", "History not implemented yet")
}

func home(g *gocui.Gui, args []string) {
	if len(args) == 2 {
		switch args[1] {
		case "?":
			screen.Fprintln(g, "msg", "green_black", cmd["home"][0])
			screen.Fprintln(g, "msg", "green_black", cmd["home"][1])
		default:
			screen.Fprintln(g, "msg", "green_black", "Home not implemented yet")
		}
	}
	screen.Fprintln(g, "msg", "green_black", "Home not implemented yet")
}

func ls(g *gocui.Gui, args []string) {
	if len(args) == 2 {
		switch args[1] {
		case "?":
			screen.Fprintln(g, "msg", "green_black", cmd["ls"][0])
			screen.Fprintln(g, "msg", "green_black", cmd["ls"][1])
		default:
			screen.Fprintln(g, "msg", "green_black", "ls not implemented yet")
		}
	}
	screen.Fprintln(g, "msg", "green_black", "ls not implemented yet")
}

// Display all of the peer information learned frm beacons
func peers(g *gocui.Gui, args []string) {
	if len(beacon.Peers) == 0 {
		screen.Fprintln(g, "msg", "purple_black", "No Peers")
		return
	}
	// Table format
	// Work out the max length of each field
	var addrlen, eidlen, dcrelen, dmodlen int
	for p := range beacon.Peers {
		if len(beacon.Peers[p].Addr) > addrlen {
			addrlen = len(beacon.Peers[p].Addr)
		}
		if len(beacon.Peers[p].Eid) > eidlen {
			eidlen = len(beacon.Peers[p].Eid)
		}
		if len(beacon.Peers[p].Created.Print()) > dcrelen {
			dcrelen = len(beacon.Peers[p].Created.Print())
		}
		if len(beacon.Peers[p].Created.Print()) > dmodlen {
			dmodlen = len(beacon.Peers[p].Updated.Print())
		}
	}
	if eidlen < 3 {
		eidlen = 3
	}

	sfmt := fmt.Sprintf("|%%%ds|%%6s|%%%ds|%%3s|%%%ds|%%%ds|\n",
		addrlen, eidlen, dcrelen, dmodlen)
	sborder := fmt.Sprintf(sfmt,
		strings.Repeat("-", addrlen),
		strings.Repeat("-", 6),
		strings.Repeat("-", eidlen),
		strings.Repeat("-", 3),
		strings.Repeat("-", dcrelen),
		strings.Repeat("-", dmodlen))

	var sslice sort.StringSlice
	for key := range beacon.Peers {
		pinfo := fmt.Sprintf(sfmt, beacon.Peers[key].Addr,
			strconv.Itoa(int(beacon.Peers[key].Freespace/1024/1024)),
			beacon.Peers[key].Eid,
			beacon.Peers[key].Maxdesc,
			beacon.Peers[key].Created.Print(),
			beacon.Peers[key].Updated.Print())
		sslice = append(sslice, fmt.Sprintf("%s", pinfo))
	}
	sort.Sort(sslice)

	sbuf := sborder
	sbuf += fmt.Sprintf(sfmt, "IP", "GB", "EID", "Des", "Date Created", "Date Modified")
	sbuf += sborder
	for key := 0; key < len(sslice); key++ {
		sbuf += fmt.Sprintf("%s", sslice[key])
	}
	sbuf += sborder
	screen.Fprintln(g, "msg", "magenta_black", sbuf)
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

// put/send a file to a destination
func put(g *gocui.Gui, args []string) {

	// errflag := make(chan string, 1) // The return channel holding the saratoga errflag

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
		t := new(transfer.Transfer)
		if err := t.New(g, "put", args[1], args[2]); err == nil {
			errflag := make(chan string, 1) // The return channel holding the saratoga errflag
			defer close(errflag)
			go transfer.Doclient(t, g, errflag) // Actually do the transfer
			errcode := <-errflag
			if errcode != "success" {
				screen.Fprintln(g, "msg", "red_black", "Error:", errcode,
					"Unable to send file: ", t.Print())
				if derr := t.Remove(); derr != nil {
					screen.Fprintln(g, "msg", "red_black", "Unable to remove transfer: ", t.Print())
				}
			}
		} else {
			screen.Fprintln(g, "msg", "red_black", "Cannot add transfer: ", err.Error())
		}
		return
	}
	screen.Fprintln(g, "msg", "red_black", cmd["put"][0])
}

// blind put/send a file to a destination
func putblind(g *gocui.Gui, args []string) {

	errflag := make(chan string, 1) // The return channel holding the saratoga errflag

	if len(args) == 1 {
		transfer.Info(g, "putblind")
		return
	}
	if len(args) == 2 && args[1] == "?" {
		screen.Fprintln(g, "msg", "green_black", cmd["putblind"][0])
		screen.Fprintln(g, "msg", "green_black", cmd["putblind"][1])
		return
	}
	if len(args) == 3 {
		t := new(transfer.Transfer)
		// We send the Metadata and do not bother with request/status exchange
		if err := t.New(g, "putblind", args[1], args[2]); err != nil {
			go transfer.Doclient(t, g, errflag)
			errcode := <-errflag
			if errcode != "success" {
				screen.Fprintln(g, "msg", "red_black", "Error:", errcode,
					"Unable to send file: ", t.Print())
			}
		}
		return
	}
	screen.Fprintln(g, "msg", "red_black", cmd["putblind"][0])
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
		t := new(transfer.Transfer)
		if err := t.New(g, "putrm", args[1], args[2]); err != nil {
			go transfer.Doclient(t, g, errflag)
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

func reqtstamp(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		if sarflags.Cli.Global["reqtstamp"] == "yes" {
			screen.Fprintln(g, "msg", "green_black", "Time stamps requested")
		} else {
			screen.Fprintln(g, "msg", "green_black", "Time stamps not requested")
		}
		return
	}
	if len(args) == 2 && args[1] == "?" { // usage
		screen.Fprintln(g, "msg", "green_black", cmd["reqtstamp"][0])
		screen.Fprintln(g, "msg", "green_black", cmd["reqtstamp"][1])
		return
	}
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()
	if len(args) == 2 {
		if args[1] == "yes" {
			sarflags.Cli.Global["reqtstamp"] = "yes"
			return
		}
		if args[1] == "no" {
			sarflags.Cli.Global["reqtstamp"] = "no"
			return
		}
		// screen.Fprintln(g, "msg", "red_black", "usage: ", cmd["reqtstamp][0]"])
	}
	screen.Fprintln(g, "msg", "red_black", "usage: ", cmd["reqtstamp"][0])
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
		if err := t.New(g, "rm", args[1], args[2]); err != nil {

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
		if err := t.New(g, "rmdir", args[1], args[2]); err != nil {

		}
		return
	}
	screen.Fprintln(g, "msg", "red_black", cmd["rmdir"][0])
}

func rmtran(g *gocui.Gui, args []string) {

	if len(args) == 1 || (len(args) == 2 && args[1] == "?") || len(args) != 4 {
		screen.Fprintln(g, "msg", "green_black", cmd["rmtran"][0])
		screen.Fprintln(g, "msg", "green_black", cmd["rmtran"][1])
		return
	}
	ttype := args[1]
	addr := args[2]
	fname := args[3]
	if t := transfer.Match(ttype, addr, fname); t != nil {
		if err := t.Remove(); err != nil {
			screen.Fprintln(g, "msg", "red_black", err.Error())
		}
	} else {
		screen.Fprintln(g, "msg", "red_black", "No such transfer: ", ttype, addr, fname)
	}
	return
}

// Are we willing to transmit files
func rxwilling(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		screen.Fprintln(g, "msg", "green_black", "Receive Files", sarflags.Cli.Global["rxwilling"])
		return
	}
	if len(args) == 2 && args[1] == "?" {
		screen.Fprintln(g, "msg", "green_black", cmd["rxwilling"][0])
		screen.Fprintln(g, "msg", "green_black", cmd["rxwilling"][1])
		return
	}
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()
	if len(args) == 2 {
		if args[1] == "on" {
			sarflags.Cli.Global["rxwilling"] = "yes"
		}
		if args[1] == "off" {
			sarflags.Cli.Global["rxwilling"] = "no"
		}
		if args[1] == "capable" {
			sarflags.Cli.Global["rxwilling"] = "capable"
		}
		return
	}
	screen.Fprintln(g, "msg", "red_black", cmd["rxwilling"][0])
}

// source is a named pipe not a file
func stream(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		if sarflags.Cli.Global["stream"] == "yes" {
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
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()
	if len(args) == 2 {
		if args[1] == "yes" {
			sarflags.Cli.Global["stream"] = "yes"
		}
		if args[1] == "no" {
			sarflags.Cli.Global["stream"] = "no"
		}
		return
	}
	screen.Fprintln(g, "msg", "red_black", cmd["stream"][0])
}

// Timeout - set timeouts for responses to request/status/transfer in seconds
func timeout(g *gocui.Gui, args []string) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()
	if len(args) == 1 {
		if sarflags.Cli.Timeout.Metadata == 0 {
			screen.Fprintln(g, "msg", "green_black", "metadata: No Timeout")
		} else {
			screen.Fprintln(g, "msg", "green_black", "metadata:", sarflags.Cli.Timeout.Metadata, "seconds")
		}
		if sarflags.Cli.Timeout.Request == 0 {
			screen.Fprintln(g, "msg", "green_black", "request: No Timeout")
		} else {
			screen.Fprintln(g, "msg", "green_black", "request:", sarflags.Cli.Timeout.Request, "seconds")
		}
		if sarflags.Cli.Timeout.Status == 0 {
			screen.Fprintln(g, "msg", "green_black", "status: No Timeout")
		} else {
			screen.Fprintln(g, "msg", "green_black", "status:", sarflags.Cli.Timeout.Status, "seconds")
		}
		if sarflags.Cli.Datacnt == 0 {
			sarflags.Cli.Datacnt = 100
			screen.Fprintln(g, "msg", "green_black", "Datacnt every 100 frames")
		} else {
			screen.Fprintln(g, "msg", "green_black", "Datacnt:", sarflags.Cli.Datacnt, "frames")
		}
		if sarflags.Cli.Timeout.Transfer == 0 {
			screen.Fprintln(g, "msg", "green_black", "transfer: No Timeout")
		} else {
			screen.Fprintln(g, "msg", "green_black", "transfer:", sarflags.Cli.Timeout.Transfer, "seconds")
		}
		return
	}
	if len(args) == 2 {
		switch args[1] {
		case "?":
			screen.Fprintln(g, "msg", "green_black", cmd["Timeout"][0])
			screen.Fprintln(g, "msg", "green_black", cmd["Timeout"][1])
		case "request":
			if sarflags.Cli.Timeout.Request == 0 {
				screen.Fprintln(g, "msg", "green_black", "request: No Timeout")
			} else {
				screen.Fprintln(g, "msg", "green_black", "request:", sarflags.Cli.Timeout.Request, "seconds")
			}
		case "metadata":
			if sarflags.Cli.Timeout.Request == 0 {
				screen.Fprintln(g, "msg", "green_black", "metadata: No Timeout")
			} else {
				screen.Fprintln(g, "msg", "green_black", "metadata:", sarflags.Cli.Timeout.Metadata, "seconds")
			}
		case "status":
			if sarflags.Cli.Timeout.Status == 0 {
				screen.Fprintln(g, "msg", "green_black", "status: No Timeout")
			} else {
				screen.Fprintln(g, "msg", "green_black", "status:", sarflags.Cli.Timeout.Status, "seconds")
			}
		case "Datacnt":
			if sarflags.Cli.Datacnt == 0 {
				sarflags.Cli.Datacnt = 100
				screen.Fprintln(g, "msg", "green_black", "Datacnt: Never")
			} else {
				screen.Fprintln(g, "msg", "green_black", "Datacnt:", sarflags.Cli.Datacnt, "frames")
			}
		case "transfer":
			if sarflags.Cli.Timeout.Transfer == 0 {
				screen.Fprintln(g, "msg", "green_black", "transfer: No Timeout")
			} else {
				screen.Fprintln(g, "msg", "green_black", "transfer:", sarflags.Cli.Timeout.Transfer, "seconds")
			}
		default:
			screen.Fprintln(g, "msg", "red_black", cmd["stream"][0])
		}
		return
	}
	if len(args) == 3 {
		if n, err := strconv.Atoi(args[2]); err == nil && n >= 0 {
			switch args[1] {
			case "metadata":
				sarflags.Cli.Timeout.Metadata = n
			case "request":
				sarflags.Cli.Timeout.Request = n
			case "status":
				sarflags.Cli.Timeout.Status = n
			case "Datacnt":
				if n == 0 {
					n = 100
				}
				sarflags.Cli.Datacnt = n
			case "transfer":
				sarflags.Cli.Timeout.Transfer = n
			}
			return
		}
		if args[2] == "off" {
			switch args[1] {
			case "metadata":
				sarflags.Cli.Timeout.Metadata = 60
			case "request":
				sarflags.Cli.Timeout.Request = 60
			case "status":
				sarflags.Cli.Timeout.Status = 60
			case "Datacnt":
				sarflags.Cli.Datacnt = 100
			case "transfer":
				sarflags.Cli.Timeout.Transfer = 60
			}
			return
		}
	}
	screen.Fprintln(g, "msg", "red_black", cmd["Timeout"][0])
}

// set the timestamp type we are using
func timestamp(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		screen.Fprintln(g, "msg", "green_black", "Timestamps are",
			sarflags.Cli.Global["reqtstamp"], "and", sarflags.Cli.Timestamp)
		return
	}
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()
	if len(args) == 2 {
		switch args[1] {
		case "?":
			screen.Fprintln(g, "msg", "green_black", cmd["timestamp"][0])
			screen.Fprintln(g, "msg", "green_black", cmd["timestamp"][1])
		case "off":
			sarflags.Cli.Global["reqtstamp"] = "off"
			// Don't change the TGlobal from what it was
		case "32":
			sarflags.Cli.Global["reqtstamp"] = "on"
			sarflags.Cli.Timestamp = "posix32"
		case "32_32":
			sarflags.Cli.Global["reqtstamp"] = "on"
			sarflags.Cli.Timestamp = "posix32_32"
		case "64":
			sarflags.Cli.Global["reqtstamp"] = "on"
			sarflags.Cli.Timestamp = "posix64"
		case "64_32":
			sarflags.Cli.Global["reqtstamp"] = "on"
			sarflags.Cli.Timestamp = "posix64_32"
		case "32_y2k":
			sarflags.Cli.Global["reqtstamp"] = "on"
			sarflags.Cli.Timestamp = "epoch2000_32"
		case "local":
			sarflags.Cli.Global["reqtstamp"] = "on"
			sarflags.Cli.Timestamp = "localinterp"
		default:
			screen.Fprintln(g, "msg", "red_black", cmd["timestamp"][0])
		}
		return
	}
	screen.Fprintln(g, "msg", "red_black", cmd["timestamp"][0])
}

// set the timezone we use for logs local or utc
func timezone(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		screen.Fprintln(g, "msg", "green_black", "Timezone is", sarflags.Cli.Timezone)
		return
	}
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()
	if len(args) == 2 {
		switch args[1] {
		case "?":
			screen.Fprintln(g, "msg", "green_black", cmd["timezone"][0])
			screen.Fprintln(g, "msg", "green_black", cmd["timezone"][1])
		case "local":
			sarflags.Cli.Timezone = "local"
		case "utc":
			sarflags.Cli.Timezone = "utc"
		default:
			screen.Fprintln(g, "msg", "red_black", cmd["timezone"][0])
		}
		return
	}
	screen.Fprintln(g, "msg", "red_black", cmd["timezone"][0])
}

// show current transfers in progress & % completed
func tran(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		transfer.Info(g, "")
		return
	}
	if len(args) == 2 {
		switch args[1] {
		case "?":
			screen.Fprintf(g, "msg", "green_black", "%s\n  %s\n",
				cmd["tran"][0], cmd["tran"][1])
		default:
			for _, tt := range transfer.Ttypes {
				if args[1] == tt {
					transfer.Info(g, args[1])
					return
				}
			}
			screen.Fprintln(g, "msg", "green_black", cmd["tran"][0])
		}
		return
	}
	screen.Fprintln(g, "msg", "green_black", cmd["tran"][0])
}

// we are willing to transmit files
func txwilling(g *gocui.Gui, args []string) {
	if len(args) == 1 {
		screen.Fprintln(g, "msg", "green_black", "Transmit Files", sarflags.Cli.Global["txwilling"])
		return
	}
	if len(args) == 2 && args[1] == "?" {
		screen.Fprintln(g, "msg", "green_black", cmd["txwilling"][0])
		screen.Fprintln(g, "msg", "green_black", cmd["txwilling"][1])
		return
	}
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()
	if len(args) == 2 {
		if args[1] == "on" {
			sarflags.Cli.Global["txwilling"] = "on"
		}
		if args[1] == "off" {
			sarflags.Cli.Global["txwilling"] = "off"
		}
		if args[1] == "capable" {
			sarflags.Cli.Global["txwilling"] = "capable"
		}
		return
	}
	screen.Fprintln(g, "msg", "red_black", cmd["txwilling"][0])
}

// Show all commands usage
func usage(g *gocui.Gui, args []string) {
	var sslice sort.StringSlice

	for key, val := range cmd {
		sslice = append(sslice, fmt.Sprintf("%s - %s", key, val[0]))
	}
	sort.Sort(sslice)
	var sbuf string
	for key := 0; key < len(sslice); key++ {
		sbuf += fmt.Sprintf("%s\n", sslice[key])
	}
	screen.Fprintln(g, "msg", "magenta_black", sbuf)
	return
}

/* ************************************************************************** */

type cmdfunc func(*gocui.Gui, []string)

// Commands and function pointers to handle them
var cmdhandler = map[string]cmdfunc{
	"?":          help,
	"beacon":     cmdbeacon,
	"cancel":     cancel,
	"checksum":   checksum,
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
	"reqtstamp":  reqtstamp,
	"rm":         rm,
	"rmtran":     rmtran,
	"rmdir":      rmdir,
	"rxwilling":  rxwilling,
	"stream":     stream,
	"timeout":    timeout,
	"timestamp":  timestamp,
	"timezone":   timezone,
	"tran":       tran,
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
	"putblind": [2]string{
		"putblind <peer> <filename>",
		"send a file to a peer with no initial request/status exchange",
	},
	"putrm": [2]string{
		"putrm <peer> <filename>",
		"send a file to a peer and then remove it from peer when successful",
	},
	"quit": [2]string{
		"quit [0|1]",
		"exit saratoga",
	},
	"reqtstamp": [2]string{
		"reqtstamp [off|on]",
		"request timestamps",
	},
	"rm": [2]string{
		"rm <peer> <filename>",
		"remove a file from a peer",
	},
	"rmdir": [2]string{
		"rmdir <peer> <dirname>",
		"remove a directory from a peer",
	},
	"rmtran": [2]string{
		"rmtran <ttype> <peer> <filename>",
		"remove a current transfer",
	},
	"rxwilling": [2]string{
		"rxwilling [on|off|capable]",
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
		"timeout [metadata|request|transfer|status] <secs|off>",
		"timeout in seconds for metadata, request frames, status receipts & transfer completion",
	},
	"timestamp": [2]string{
		"timestamp [off|32|64|32_32|64_32|32_y2k]",
		"timestamp type to send",
	},
	"timezone": [2]string{
		"timezone [utc|local]",
		"show current or set to use local or universal time",
	},
	"tran": [2]string{
		"tran [get|getrm|getdir|put|putblind|putrm|rm|rmdir]",
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
