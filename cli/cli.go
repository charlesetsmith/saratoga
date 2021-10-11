//Command Linein Interface

package cli

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/charlesetsmith/saratoga/beacon"
	"github.com/charlesetsmith/saratoga/sarflags"
	"github.com/charlesetsmith/saratoga/sarscreen"
	"github.com/charlesetsmith/saratoga/transfer"
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
/* THIS IS NOT USED YET
func appendunique(slice []string, i string) []string {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}
*/

// Send count beacons to host
func sendbeacons(g *gocui.Gui, flags string, count uint, interval uint, host string, port int) {
	// We have a hostname maybe with multiple addresses
	var addrs []string
	var err error
	var txb beacon.Beacon // The assembled beacon to transmit

	errflag := make(chan string, 1) // The return channel holding the saratoga errflag

	if addrs, err = net.LookupHost(host); err != nil {
		sarscreen.MsgPrintln(g, "red_black", "Cannot resolve hostname: ", err)
		return
	}
	// Loop thru the address(s) for the host and send beacons to them
	for _, addr := range addrs {
		if err := txb.New(flags); err == nil {
			go txb.Send(g, addr, port, count, interval, errflag)
			errcode := <-errflag
			if errcode != "success" {
				sarscreen.MsgPrintln(g, "red_black", "Error:", errcode,
					"Unable to send beacon to ", addr)
			}
			sarscreen.MsgPrintln(g, "green_black", "Sending beacon to", addr)
		} else {
			sarscreen.MsgPrintln(g, "red_black", "cannot create beacon in txb.New", err.Error())
		}
	}
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

func cmdbeacon(g *gocui.Gui, args []string, c *sarflags.Cliflags) {

	// var bmu sync.Mutex // Protects beacon.Beacon structure (EID)
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()
	clibeacon.flags = sarflags.Setglobal("beacon", c) // Initialise Global Beacon flags
	clibeacon.interval = c.Timeout.Binterval          // Set up the correct interval

	switch len(args) {
	// Show current Cbeacon flags and lists - beacon
	case 1:
		if clibeacon.count != 0 {
			sarscreen.MsgPrintln(g, "green_black", clibeacon.count, "Beacons to be sent every %d secs",
				clibeacon.interval)
		} else {
			sarscreen.MsgPrintln(g, "green_black", "Single Beacon to be sent")
		}
		if clibeacon.v4mcast {
			sarscreen.MsgPrintln(g, "green_black", "Sending IPv4 multicast beacons")
		}
		if clibeacon.v6mcast {
			sarscreen.MsgPrintln(g, "green_black", "Sending IPv6 multicast beacons")
		}
		if len(clibeacon.host) > 0 {
			sarscreen.MsgPrintln(g, "green_black", "Sending beacons to:")
			for _, i := range clibeacon.host {
				sarscreen.MsgPrintln(g, "green_black", "\t", i)
			}
		}
		if !clibeacon.v4mcast && !clibeacon.v6mcast &&
			len(clibeacon.host) == 0 {
			sarscreen.MsgPrintln(g, "green_black", "No beacons currently being sent")
		}
		return
	case 2:
		switch args[1] {
		case "?": // usage
			sarscreen.MsgPrintln(g, "green_black", prusage("beacon", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("beacon", c))
			return
		case "off": // remove and disable all beacons
			clibeacon.flags = sarflags.Setglobal("beacon", c)
			clibeacon.count = 0
			clibeacon.interval = c.Timeout.Binterval
			clibeacon.host = nil
			sarscreen.MsgPrintln(g, "green_black", "Beacons Disabled")
			return
		case "v4": // V4 Multicast
			sarscreen.MsgPrintln(g, "green_black", "Sending beacon to IPv4 Multicast")
			clibeacon.flags = sarflags.Setglobal("beacon", c)
			clibeacon.v4mcast = true
			clibeacon.count = 1
			// Start up the beacon client sending count IPv4 beacons
			go sendbeacons(g, clibeacon.flags, clibeacon.count, clibeacon.interval, c.V4Multicast, c.Port)
			return
		case "v6": // V6 Multicast
			sarscreen.MsgPrintln(g, "green_black", "Sending beacon to IPv6 Multicast")
			clibeacon.flags = sarflags.Setglobal("beacon", c)
			clibeacon.v6mcast = true
			clibeacon.count = 1
			// Start up the beacon client sending count IPv6 beacons
			go sendbeacons(g, clibeacon.flags, clibeacon.count, clibeacon.interval, c.V4Multicast, c.Port)
			return
		default: // beacon <count> or beacon <ipaddr>
			u32, err := strconv.ParseUint(args[1], 10, 32)
			if err == nil { // We have a number so it is a timer
				clibeacon.count = uint(u32)
				sarscreen.MsgPrintln(g, "green_black", "Beacons timer set to", clibeacon.count, "seconds")
			} else {
				sarscreen.MsgPrintln(g, "green_black", "Sending", clibeacon.count, "beacons to", args[1])
				go sendbeacons(g, clibeacon.flags, clibeacon.count, clibeacon.interval, args[1], c.Port)
			}
			return
		}
	}

	// beacon off <ipaddr> ...
	if args[1] == "off" && len(args) > 2 { // turn off following addresses
		sarscreen.MsgPrintf(g, "green_black", "%s ", "Beacons turned off to")
		for i := 2; i < len(args); i++ { // Remove Address'es from lists
			if net.ParseIP(args[i]) != nil { // Do We have a valid IP Address
				clibeacon.host = removeValue(clibeacon.host, args[i])
				sarscreen.MsgPrintf(g, "green_black", "%s ", args[i])
				if i == len(args)-1 {
					sarscreen.MsgPrintln(g, "green_black", "")
				}
			} else {
				sarscreen.MsgPrintln(g, "red_black", "Invalid IP Address:", args[i])
				sarscreen.CmdPrintln(g, "red_black", prusage("beacon", c))
			}
		}
		return
	}

	// beacon <count> <ipaddr> ...
	var addrstart = 1
	u32, err := strconv.ParseUint(args[1], 10, 32)
	if err == nil { // We have a number so it is a timer
		clibeacon.count = uint(u32)
		sarscreen.MsgPrintln(g, "green_black", "Beacon counter set to", clibeacon.count)
		addrstart = 2
	}
	// beacon [count] <ipaddr> ...
	sarscreen.MsgPrintf(g, "green_black", "Sending %d beacons to: ",
		clibeacon.count)
	for i := addrstart; i < len(args); i++ { // Add Address'es to lists
		sarscreen.MsgPrintf(g, "green_black", "%s ", args[i])
		switch args[i] {
		case "v4":
			go sendbeacons(g, clibeacon.flags, clibeacon.count,
				clibeacon.interval, c.V4Multicast, c.Port)
		case "v6":
			go sendbeacons(g, clibeacon.flags, clibeacon.count,
				clibeacon.interval, c.V6Multicast, c.Port)
		default:
			go sendbeacons(g, clibeacon.flags, clibeacon.count,
				clibeacon.interval, args[i], c.Port)
		}
	}
	sarscreen.MsgPrintln(g, "green_black", "")
}

func cancel(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarscreen.MsgPrintln(g, "green_black", args)
}

func checksum(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		sarscreen.MsgPrintln(g, "green_black", "Checksum", c.Global["csumtype"])
		return
	case 2:
		switch args[1] {
		case "?": // usage
			sarscreen.MsgPrintln(g, "green_black", prusage("checksum", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("checksum", c))
			return
		case "off", "none":
			c.Global["csumtype"] = "none"
		case "crc32":
			c.Global["csumtype"] = "crc32"
		case "md5":
			c.Global["csumtype"] = "md5"
		case "sha1":
			c.Global["csumtype"] = "sha1"
		default:
			sarscreen.CmdPrintln(g, "green_red", prusage("checksum", c))
		}
		return
	}
	sarscreen.CmdPrintln(g, "green_red", prusage("checksum", c))
}

func descriptor(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		sarscreen.MsgPrintln(g, "green_black", "Descriptor", c.Global["descriptor"])
		return
	case 2:
		switch args[1] {
		case "?": // usage
			sarscreen.MsgPrintln(g, "green_black", prusage("descriptor", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("descriptor", c))
			return
		case "auto":
			if sarflags.MaxUint <= sarflags.MaxUint16 {
				c.Global["descriptor"] = "d16"
				break
			}
			if sarflags.MaxUint <= sarflags.MaxUint32 {
				c.Global["descriptor"] = "d32"
				break
			}
			if sarflags.MaxUint <= sarflags.MaxUint64 {
				c.Global["descriptor"] = "d64"
				break
			}
			sarscreen.MsgPrintln(g, "red_black", "128 bit descriptors not supported on this platform")
		case "d16":
			if sarflags.MaxUint > sarflags.MaxUint16 {
				c.Global["descriptor"] = "d16"
			} else {
				sarscreen.MsgPrintln(g, "red_black", "16 bit descriptors not supported on this platform")
			}
		case "d32":
			if sarflags.MaxUint > sarflags.MaxUint32 {
				c.Global["descriptor"] = "d32"
			} else {
				sarscreen.MsgPrintln(g, "red_black", "32 bit descriptors not supported on this platform")
			}
		case "d64":
			if sarflags.MaxUint <= sarflags.MaxUint64 {
				c.Global["descriptor"] = "d64"
			} else {
				sarscreen.MsgPrintln(g, "red_black", "64 bit descriptors are not supported on this platform")
				sarscreen.MsgPrintln(g, "red_black", "MaxUint=", sarflags.MaxUint,
					"<= MaxUint64=", sarflags.MaxUint64)
			}
		case "d128":
			sarscreen.MsgPrintln(g, "red_black", "128 bit descriptors not supported on this platform")
		default:
			sarscreen.MsgPrintln(g, "red_black", "usage: ", prusage("descriptor", c))
		}
		sarscreen.MsgPrintln(g, "green_black", "Descriptor size is", c.Global["descriptor"])
		return
	}
	sarscreen.MsgPrintln(g, "red_black", "usage: ", prusage("descriptor", c))
}

// Cexit = Exit level to quit from saratoga
var Cexit = -1

// Quit saratoga
func exit(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	switch len(args) {
	case 1: // exit 0
		Cexit = 0
		sarscreen.MsgPrintln(g, "green_black", "Good Bye!")
		return
	case 2:
		switch args[1] {
		case "?": // Usage
			sarscreen.MsgPrintln(g, "green_black", prusage("exit", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("exit", c))
		case "0": // exit 0
			Cexit = 0
			sarscreen.MsgPrintln(g, "green_black", "Good Bye!")
		case "1": // exit 1
			Cexit = 1
			sarscreen.MsgPrintln(g, "green_black", "Good Bye!")
		default: // Help
			sarscreen.MsgPrintln(g, "red_black", prusage("exit", c))
		}
	default:
		sarscreen.MsgPrintln(g, "red_black", prusage("exit", c))
	}
}

// MORE WORK TO DO HERE!!!!! USE TRANSFERS LIST
func files(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	var flist []string

	switch len(args) {
	case 1:
		if len(flist) == 0 {
			sarscreen.MsgPrintln(g, "green_black", "No currently open files")
			return
		}
		for _, i := range flist {
			sarscreen.MsgPrintln(g, "green_black", i)
		}
		return
	case 2:
		if args[1] == "?" { // usage
			sarscreen.MsgPrintln(g, "green_black", prusage("files", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("files", c))
			return
		}
	}
	sarscreen.MsgPrintln(g, "red_black", prusage("files", c))
}

func freespace(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		if c.Global["freespace"] == "yes" {
			sarscreen.MsgPrintln(g, "green_black", "Free space is advertised")
		} else {
			sarscreen.MsgPrintln(g, "green_black", "Free space is not advertised")
		}
		return
	case 2:
		switch args[1] {
		case "?": // usage
			sarscreen.MsgPrintln(g, "green_black", prusage("freespace", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("freespace", c))
			return
		case "yes":
			sarscreen.MsgPrintln(g, "green_black", "freespace is advertised")
			c.Global["freespace"] = "yes"
			return
		case "no":
			sarscreen.MsgPrintln(g, "green_black", "freespace is not advertised")
			c.Global["freespace"] = "no"
			return
		}
	}
	sarscreen.MsgPrintln(g, "red_black", "usage: ", prusage("freespace", c))
}

func get(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	switch len(args) {
	case 1:
		transfer.Info(g, "get")
		return
	case 2:
		if args[1] == "?" {
			sarscreen.MsgPrintln(g, "green_black", prusage("get", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("get", c))
			return
		}
	case 3:
		var t transfer.CTransfer
		if err := t.CNew(g, "get", args[1], args[2], c); err != nil {
			return
		}
		return
	}
	sarscreen.MsgPrintln(g, "red_black", prusage("get", c))
}

func getdir(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	switch len(args) {
	case 1:
		transfer.Info(g, "getdir")
		return
	case 2:
		if args[1] == "?" {
			sarscreen.MsgPrintln(g, "green_black", prusage("getdir", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("getdir", c))
			return
		}
	case 3:
		var t transfer.CTransfer
		if err := t.CNew(g, "getdir", args[1], args[2], c); err != nil {
			sarscreen.MsgPrintln(g, "green_black", prusage("getdir", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("getdir", c))
		}
		return
	}
	sarscreen.MsgPrintln(g, "red_black", prusage("getdir", c))
}

func getrm(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	switch len(args) {
	case 1:
		transfer.Info(g, "getrm")
		return
	case 2:
		if args[1] == "?" {
			sarscreen.MsgPrintln(g, "green_black", prusage("getrm", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("getrm", c))
			return
		}
	case 3:
		var t transfer.CTransfer
		if err := t.CNew(g, "getrm", args[1], args[2], c); err != nil {
			sarscreen.MsgPrintln(g, "green_black", prusage("getrm", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("getrm", c))
			return
		}
		return
	}
	sarscreen.MsgPrintln(g, "red_black", prusage("getrm", c))
}

func help(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	switch len(args) {
	case 1:
		var sslice sort.StringSlice
		for key, val := range sarflags.Commands {
			sslice = append(sslice, fmt.Sprintf("%s - %s",
				key,
				val.Help))
		}
		sort.Sort(sslice)
		var sbuf string
		for key := 0; key < len(sslice); key++ {
			sbuf += fmt.Sprintf("%s\n", sslice[key])
		}
		sarscreen.MsgPrintln(g, "magenta_black", sbuf)
		return
	case 2:
		if args[1] == "?" {
			var sslice sort.StringSlice
			for key, val := range sarflags.Commands {
				sslice = append(sslice, fmt.Sprintf("%s - %s\n  %s",
					key,
					val.Help,
					val.Usage))
			}
			sort.Sort(sslice)
			var sbuf string
			for key := 0; key < len(sslice); key++ {
				sbuf += fmt.Sprintf("%s\n", sslice[key])
			}
			sarscreen.MsgPrintln(g, "magenta_black", sbuf)
			return
		}
	}
	for key, val := range sarflags.Commands {
		if key == "help" {
			sarscreen.MsgPrintln(g, "red_black", fmt.Sprintf("%s - %s",
				key,
				val.Help))
		}
	}
}

func interval(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		if c.Timeout.Binterval == 0 {
			sarscreen.MsgPrintln(g, "green_black", "Single Beacon Interation")
		} else {
			sarscreen.MsgPrintln(g, "green_black", "Beacons sent every",
				c.Timeout.Binterval, "seconds")
		}
		return
	case 2:
		switch args[1] {
		case "?":
			sarscreen.MsgPrintln(g, "green_black", prusage("interval", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("interval", c))
			return
		case "off":
			c.Timeout.Binterval = 0
			return
		default:
			if n, err := strconv.Atoi(args[1]); err == nil && n >= 0 {
				c.Timeout.Binterval = uint(n)
				return
			}
		}
		sarscreen.MsgPrintln(g, "red_black", prusage("interval", c))
	}
	sarscreen.MsgPrintln(g, "red_black", prusage("interval", c))
}

func history(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	switch len(args) {
	case 1:
		sarscreen.MsgPrintln(g, "green_black", "History not implemented yet")
		return
	case 2:
		switch args[1] {
		case "?":
			sarscreen.MsgPrintln(g, "green_black", prusage("history", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("history", c))
			return
		default:
			sarscreen.MsgPrintln(g, "green_black", "History not implemented yet")
			return
		}
	}
	sarscreen.MsgPrintln(g, "red_black", prusage("history", c))
}

func home(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	switch len(args) {
	case 1:
		sarscreen.MsgPrintln(g, "green_black", "Home not implemented yet")
		return
	case 2:
		switch args[1] {
		case "?":
			sarscreen.MsgPrintln(g, "green_black", prusage("home", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("home", c))
			return
		}
	}
	sarscreen.MsgPrintln(g, "red_black", prusage("home", c))
}

func ls(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	if len(args) != 0 {
		sarscreen.MsgPrintln(g, "red_bblack", prusage("ls", c))
		return
	}
	switch args[1] {
	case "?":
		sarscreen.MsgPrintln(g, "green_black", prusage("ls", c))
		sarscreen.MsgPrintln(g, "green_black", prhelp("ls", c))
		return
	}
	sarscreen.MsgPrintln(g, "green_black", "ls not implemented yet")
}

// Display all of the peer information learned frm beacons
func peers(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	if len(args) != 1 {
		sarscreen.MsgPrintln(g, "red_bblack", prusage("peers", c))
		return
	}

	if len(beacon.Peers) == 0 {
		sarscreen.MsgPrintln(g, "magenta_black", "No Peers")
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
		sslice = append(sslice, pinfo)
	}
	sort.Sort(sslice)

	sbuf := sborder
	sbuf += fmt.Sprintf(sfmt, "IP", "GB", "EID", "Des", "Date Created", "Date Modified")
	sbuf += sborder
	for key := 0; key < len(sslice); key++ {
		sbuf += sslice[key]
	}
	sbuf += sborder
	sarscreen.MsgPrintln(g, "magenta_black", sbuf)
}

// Cprompt - Command line prompt
var Cprompt = "saratoga" // If not set in saratoga.json set it to saratoga

// put/send a file to a destination
func put(g *gocui.Gui, args []string, c *sarflags.Cliflags) {

	switch len(args) {
	case 1:
		transfer.Info(g, "put")
		return
	case 2:
		if args[1] == "?" {
			sarscreen.MsgPrintln(g, "green_black", prusage("put", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("put", c))
			return
		}
	case 3:
		t := new(transfer.CTransfer)

		if err := t.CNew(g, "put", args[1], args[2], c); err == nil {
			errflag := make(chan string, 1)     // The return channel holding the saratoga errflag
			go transfer.Doclient(t, g, errflag) // Actually do the transfer
			errcode := <-errflag
			if errcode != "success" {
				sarscreen.MsgPrintln(g, "red_black", "Error:", errcode,
					"Unable to send file: ", t.Print())
				if derr := t.Remove(); derr != nil {
					sarscreen.MsgPrintln(g, "red_black", "Unable to remove transfer: ", t.Print())
				}
			}
			sarscreen.MsgPrintln(g, "green_black", "put completed closing channel")
			close(errflag)
		} else {
			sarscreen.MsgPrintln(g, "red_black", "Cannot add transfer: ", err.Error())
		}
		return
	}
	sarscreen.MsgPrintln(g, "red_black", prusage("put", c))
}

// blind put/send a file to a destination
func putblind(g *gocui.Gui, args []string, c *sarflags.Cliflags) {

	errflag := make(chan string, 1) // The return channel holding the saratoga errflag

	switch len(args) {
	case 1:
		transfer.Info(g, "putblind")
		return
	case 2:
		if args[1] == "?" {
			sarscreen.MsgPrintln(g, "green_black", prusage("putblind", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("putblind", c))
			return
		}
	case 3:
		t := new(transfer.CTransfer)
		// We send the Metadata and do not bother with request/status exchange
		if err := t.CNew(g, "putblind", args[1], args[2], c); err != nil {
			go transfer.Doclient(t, g, errflag)
			errcode := <-errflag
			if errcode != "success" {
				sarscreen.MsgPrintln(g, "red_black", "Error:", errcode,
					"Unable to send file: ", t.Print())
			}
		}
		return
	}
	sarscreen.MsgPrintln(g, "red_black", prusage("putblind", c))
}

// put/send a file file to a remote destination then remove it from the origin
func putrm(g *gocui.Gui, args []string, c *sarflags.Cliflags) {

	errflag := make(chan string, 1) // The return channel holding the saratoga errflag

	switch len(args) {
	case 1:
		transfer.Info(g, "putrm")
		return
	case 2:
		if args[1] == "?" {
			sarscreen.MsgPrintln(g, "green_black", prusage("putrm", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("putrm", c))
			return
		}
	case 3:
		t := new(transfer.CTransfer)
		if err := t.CNew(g, "putrm", args[1], args[2], c); err != nil {
			go transfer.Doclient(t, g, errflag)
			errcode := <-errflag
			if errcode != "success" {
				sarscreen.MsgPrintln(g, "red_black", "Error:", errcode,
					"Unable to send file: ", t.Print())
			} else {
				sarscreen.MsgPrintln(g, "red_black",
					"Put and now removing (NOT) file: ", t.Print())
			}
		}
		return
	}
	sarscreen.MsgPrintln(g, "red_black", prusage("putrm", c))
}

func reqtstamp(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		if c.Global["reqtstamp"] == "yes" {
			sarscreen.MsgPrintln(g, "green_black", "Time stamps requested")
		} else {
			sarscreen.MsgPrintln(g, "green_black", "Time stamps not requested")
		}
		return
	case 2:
		switch args[1] {
		case "?": // usage
			sarscreen.MsgPrintln(g, "green_black", prusage("reqtstamp", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("reqtstamp", c))
			return
		case "yes":
			c.Global["reqtstamp"] = "yes"
			return
		case "no":
			c.Global["reqtstamp"] = "no"
			return
		}
		// screen.Fprintln(g,  "red_black", "usage: ", cmd["reqtstamp][0]"])
	}
	sarscreen.MsgPrintln(g, "red_black", "usage: ", prusage("reqtstamp", c))
}

// remove a file from a remote destination
func rm(g *gocui.Gui, args []string, c *sarflags.Cliflags) {

	switch len(args) {
	case 1:
		transfer.Info(g, "rm")
		return
	case 2:
		if args[1] == "?" {
			sarscreen.MsgPrintln(g, "green_black", prusage("rm", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("rm", c))
			return
		}
	case 3:
		var t transfer.CTransfer
		if err := t.CNew(g, "rm", args[1], args[2], c); err != nil {
			sarscreen.MsgPrintln(g, "green_black", prusage("rm", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("rm", c))
			return
		}
	}
	sarscreen.MsgPrintln(g, "red_black", prusage("rm", c))
}

// remove a directory from a remote destination
func rmdir(g *gocui.Gui, args []string, c *sarflags.Cliflags) {

	switch len(args) {
	case 1:
		transfer.Info(g, "rmdir")
		return
	case 2:
		if args[1] == "?" {
			sarscreen.MsgPrintln(g, "green_black", prusage("rmdir", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("rmdir", c))
			return
		}
	case 3:
		var t transfer.CTransfer
		if err := t.CNew(g, "rmdir", args[1], args[2], c); err != nil {
			sarscreen.MsgPrintln(g, "green_black", prusage("rmdir", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("rmdir", c))
			return
		}
	}
	sarscreen.MsgPrintln(g, "red_black", prusage("rmdir", c))
}

func rmtran(g *gocui.Gui, args []string, c *sarflags.Cliflags) {

	switch len(args) {
	case 1:
		sarscreen.MsgPrintln(g, "green_black", prusage("rmtran", c))
		sarscreen.MsgPrintln(g, "green_black", prhelp("rmtran", c))
		return
	case 2:
		if args[1] == "?" {
			sarscreen.MsgPrintln(g, "green_black", prusage("rmtran", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("rmtran", c))
			return
		}
	case 4:
		ttype := args[1]
		addr := args[2]
		fname := args[3]
		if t := transfer.CMatch(ttype, addr, fname); t != nil {
			if err := t.Remove(); err != nil {
				sarscreen.MsgPrintln(g, "red_black", err.Error())
			}
		} else {
			sarscreen.MsgPrintln(g, "red_black", "No such transfer: ", ttype, addr, fname)
		}
		return
	}
	sarscreen.MsgPrintln(g, "red_black", prusage("rmtran", c))
}

// Are we willing to transmit files
func rxwilling(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		sarscreen.MsgPrintln(g, "green_black", "Receive Files", c.Global["rxwilling"])
		return
	case 2:
		switch args[1] {
		case "?":
			sarscreen.MsgPrintln(g, "green_black", prusage("rxwilling", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("rxwilling", c))
			return
		case "on":
			c.Global["rxwilling"] = "yes"
			return
		case "off":
			c.Global["rxwilling"] = "no"
			return
		case "capable":
			c.Global["rxwilling"] = "capable"
			return
		}
	}
	sarscreen.MsgPrintln(g, "red_black", prusage("rxwilling", c))
}

// source is a named pipe not a file
func stream(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		if c.Global["stream"] == "yes" {
			sarscreen.MsgPrintln(g, "green_black", "Can stream")
		} else {
			sarscreen.MsgPrintln(g, "green_black", "Cannot stream")
		}
		return
	case 2:
		switch args[1] {
		case "?":
			sarscreen.MsgPrintln(g, "green_black", prusage("stream", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("stream", c))
			return
		case "yes":
			c.Global["stream"] = "yes"
			return
		case "no":
			c.Global["stream"] = "no"
			return
		}
	}
	sarscreen.MsgPrintln(g, "red_black", prusage("stream", c))
}

// Timeout - set timeouts for responses to request/status/transfer in seconds
func timeout(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		if c.Timeout.Metadata == 0 {
			sarscreen.MsgPrintln(g, "green_black", "metadata: No Timeout")
		} else {
			sarscreen.MsgPrintln(g, "green_black", "metadata:", c.Timeout.Metadata, "seconds")
		}
		if c.Timeout.Request == 0 {
			sarscreen.MsgPrintln(g, "green_black", "request: No Timeout")
		} else {
			sarscreen.MsgPrintln(g, "green_black", "request:", c.Timeout.Request, "seconds")
		}
		if c.Timeout.Status == 0 {
			sarscreen.MsgPrintln(g, "green_black", "status: No Timeout")
		} else {
			sarscreen.MsgPrintln(g, "green_black", "status:", c.Timeout.Status, "seconds")
		}
		if c.Datacnt == 0 {
			c.Datacnt = 100
			sarscreen.MsgPrintln(g, "green_black", "Datacnt every 100 frames")
		} else {
			sarscreen.MsgPrintln(g, "green_black", "Datacnt:", c.Datacnt, "frames")
		}
		if c.Timeout.Transfer == 0 {
			sarscreen.MsgPrintln(g, "green_black", "transfer: No Timeout")
		} else {
			sarscreen.MsgPrintln(g, "green_black", "transfer:", c.Timeout.Transfer, "seconds")
		}
		return
	case 2:
		switch args[1] {
		case "?":
			sarscreen.MsgPrintln(g, "green_black", prusage("timeout", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("timeout", c))
		case "request":
			if c.Timeout.Request == 0 {
				sarscreen.MsgPrintln(g, "green_black", "request: No Timeout")
			} else {
				sarscreen.MsgPrintln(g, "green_black", "request:", c.Timeout.Request, "seconds")
			}
		case "metadata":
			if c.Timeout.Request == 0 {
				sarscreen.MsgPrintln(g, "green_black", "metadata: No Timeout")
			} else {
				sarscreen.MsgPrintln(g, "green_black", "metadata:", c.Timeout.Metadata, "seconds")
			}
		case "status":
			if c.Timeout.Status == 0 {
				sarscreen.MsgPrintln(g, "green_black", "status: No Timeout")
			} else {
				sarscreen.MsgPrintln(g, "green_black", "status:", c.Timeout.Status, "seconds")
			}
		case "Datacnt":
			if c.Datacnt == 0 {
				c.Datacnt = 100
				sarscreen.MsgPrintln(g, "green_black", "Datacnt: Never")
			} else {
				sarscreen.MsgPrintln(g, "green_black", "Datacnt:", c.Datacnt, "frames")
			}
		case "transfer":
			if c.Timeout.Transfer == 0 {
				sarscreen.MsgPrintln(g, "green_black", "transfer: No Timeout")
			} else {
				sarscreen.MsgPrintln(g, "green_black", "transfer:", c.Timeout.Transfer, "seconds")
			}
		default:
			sarscreen.MsgPrintln(g, "red_black", prusage("stream", c))
		}
		return
	case 3:
		if n, err := strconv.Atoi(args[2]); err == nil && n >= 0 {
			switch args[1] {
			case "metadata":
				c.Timeout.Metadata = n
			case "request":
				c.Timeout.Request = n
			case "status":
				c.Timeout.Status = n
			case "Datacnt":
				if n == 0 {
					n = 100
				}
				c.Datacnt = n
			case "transfer":
				c.Timeout.Transfer = n
			}
			return
		}
		if args[2] == "off" {
			switch args[1] {
			case "metadata":
				c.Timeout.Metadata = 60
			case "request":
				c.Timeout.Request = 60
			case "status":
				c.Timeout.Status = 60
			case "Datacnt":
				c.Datacnt = 100
			case "transfer":
				c.Timeout.Transfer = 60
			}
			return
		}
	}
	sarscreen.MsgPrintln(g, "red_black", prusage("timeout", c))
}

// set the timestamp type we are using
func timestamp(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		sarscreen.MsgPrintln(g, "green_black", "Timestamps are",
			c.Global["reqtstamp"], "and", c.Timestamp)
		return
	case 2:
		switch args[1] {
		case "?":
			sarscreen.MsgPrintln(g, "green_black", prusage("timestamp", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("timestamp", c))
		case "off":
			c.Global["reqtstamp"] = "no"
			// Don't change the TGlobal from what it was
		case "32":
			c.Global["reqtstamp"] = "yes"
			c.Timestamp = "posix32"
		case "32_32":
			c.Global["reqtstamp"] = "yes"
			c.Timestamp = "posix32_32"
		case "64":
			c.Global["reqtstamp"] = "yes"
			c.Timestamp = "posix64"
		case "64_32":
			c.Global["reqtstamp"] = "yes"
			c.Timestamp = "posix64_32"
		case "32_y2k":
			c.Global["reqtstamp"] = "yes"
			c.Timestamp = "epoch2000_32"
		case "local":
			c.Global["reqtstamp"] = "yes"
			c.Timestamp = "localinterp"
		default:
			sarscreen.MsgPrintln(g, "red_black", prusage("timestamp", c))
		}
		return
	}
	sarscreen.MsgPrintln(g, "red_black", prusage("timestamp", c))
}

// set the timezone we use for logs local or utc
func timezone(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		sarscreen.MsgPrintln(g, "green_black", "Timezone is", c.Timezone)
		return
	case 2:
		switch args[1] {
		case "?":
			sarscreen.MsgPrintln(g, "green_black", prusage("timezone", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("timezone", c))
		case "local":
			c.Timezone = "local"
		case "utc":
			c.Timezone = "utc"
		default:
			sarscreen.MsgPrintln(g, "red_black", prusage("timezone", c))
		}
		return
	}
	sarscreen.MsgPrintln(g, "red_black", prusage("timezone", c))
}

// show current transfers in progress & % completed
func tran(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	switch len(args) {
	case 1:
		transfer.Info(g, "")
		return
	case 2:
		switch args[1] {
		case "?":
			sarscreen.MsgPrintf(g, "green_black", "%s\n  %s\n",
				prusage("tran", c), prhelp("tran", c))
		default:
			for _, tt := range transfer.Ttypes {
				if args[1] == tt {
					transfer.Info(g, args[1])
					return
				}
			}
			sarscreen.MsgPrintln(g, "green_black", prusage("tran", c))
		}
		return
	}
	sarscreen.MsgPrintln(g, "green_black", prusage("tran", c))
}

// we are willing to transmit files
func txwilling(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		sarscreen.MsgPrintln(g, "green_black", "Transmit Files", c.Global["txwilling"])
		return
	case 2:
		switch args[1] {
		case "?":
			sarscreen.MsgPrintln(g, "green_black", prusage("txwilling", c))
			sarscreen.MsgPrintln(g, "green_black", prhelp("txwilling", c))
			return
		case "on":
			c.Global["txwilling"] = "on"
			return
		case "off":
			c.Global["txwilling"] = "off"
			return
		case "capable":
			c.Global["txwilling"] = "capable"
			return
		}
	}
	sarscreen.MsgPrintln(g, "red_black", prusage("txwilling", c))
}

func prhelp(cf string, c *sarflags.Cliflags) string {
	for key, val := range sarflags.Commands {
		if key == cf {
			return key + ": " + val.Help
		}
	}
	return "Invalid Command"
}

func prusage(cf string, c *sarflags.Cliflags) string {
	for key, val := range sarflags.Commands {
		if key == cf {
			return "usage: " + val.Usage
		}
	}
	return "Invalid Command"
}

// Show all commands usage
func usage(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	var sslice sort.StringSlice

	for key, val := range sarflags.Commands {
		sslice = append(sslice, fmt.Sprintf("%s - %s",
			key,
			val.Usage))
	}

	sort.Sort(sslice)
	var sbuf string
	for key := 0; key < len(sslice); key++ {
		sbuf += fmt.Sprintf("%s\n", sslice[key])
	}
	sarscreen.MsgPrintln(g, "magenta_black", sbuf)
}

/* ************************************************************************** */

type cmdfunc func(*gocui.Gui, []string, *sarflags.Cliflags)

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

// Docmd -- Execute the command entered
func Docmd(g *gocui.Gui, s string, c *sarflags.Cliflags) {
	if s == "" { // Handle just return
		return
	}
	// Get rid of leading and trailing whitespace
	s = strings.TrimSpace(s)
	vals := strings.Fields(s)
	// Lookup the command and execute it
	for key := range sarflags.Commands {
		if key == vals[0] {
			fn, ok := cmdhandler[vals[0]]
			if ok {
				fn(g, vals, c)
				return
			}
			sarscreen.MsgPrintln(g, "bwhite_red", "Cannot execute:", vals[0])
			return
		}
	}
	sarscreen.MsgPrintln(g, "bwhite_red", "Invalid command:", vals[0])
}
