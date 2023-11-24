package cmds

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/jroimartin/gocui"

	"github.com/charlesetsmith/saratoga/beacon"
	"github.com/charlesetsmith/saratoga/frames"
	"github.com/charlesetsmith/saratoga/sarflags"
	"github.com/charlesetsmith/saratoga/sarwin"
	"github.com/charlesetsmith/saratoga/transfer"
)

// prhelp -- return command help string
func prhelp(cf string, c *sarflags.Cliflags) string {
	for key, val := range sarflags.Commands {
		if key == cf {
			return key + ":" + val.Help
		}
	}
	return "Invalid Command"
}

// prusage -- return command usage string
func prusage(cf string, c *sarflags.Cliflags) string {
	for key, val := range sarflags.Commands {
		if key == cf {
			return "usage:" + val.Usage
		}
	}
	return "Invalid Command"
}

// removeIndex -- Remove an entry in a slice of strings by index #
func removeIndex(s []string, index int) []string {
	ret := make([]string, 0)
	ret = append(ret, s[:index]...)
	return append(ret, s[index+1:]...)
}

// removeValue -- Remove all entries in slice of strings matching val
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

/* ********************************************************************************* */

// All of the different command line input handlers
// Send count beacons to host
func sendbeacons(g *gocui.Gui, flags string, count uint, interval uint, host string, port int) {
	// We have a hostname maybe with multiple addresses
	var addrs []string
	var err error
	var txb beacon.Beacon // The assembled beacon to transmit
	b := &txb

	errflag := make(chan string, 1) // The return channel holding the saratoga errflag

	if addrs, err = net.LookupHost(host); err != nil {
		sarwin.MsgPrintln(g, "red_black", "Cannot resolve hostname:", err)
		return
	}
	// Loop thru the address(s) for the host and send beacons to them
	for _, addr := range addrs {
		sarwin.MsgPrintln(g, "cyan_black", "Sending beacon to ", addr)
		binfo := beacon.Binfo{Freespace: 0, Eid: ""}
		if err := frames.New(b, flags, &binfo); err == nil {
			go txb.Send(addr, port, count, interval, errflag)
			errcode := <-errflag
			if errcode != "success" {
				sarwin.ErrPrintln(g, "red_black", "Error:", errcode,
					"Unable to send beacon to ", addr)
			} else {
				sarwin.PacketPrintln(g, "cyan_black", "Tx ", b.ShortPrint())
			}
		} else {
			sarwin.ErrPrintln(g, "red_black", "cannot create beacon in txb.New:", err.Error())
		}
	}
}

/* ********************************************************************************* */

// All of the different command line input handlers
// Beacon CLI Info
type Beaconcmd struct {
	flags    string   // Header Flags set for beacons
	count    uint     // How many beacons to send 0|1 == 1
	interval uint     // interval in seconds between beacons 0|1 == 1
	v4mcast  bool     // Sending beacons to V4 Multicast
	v6mcast  bool     // Sending beacons to V6 Multicast
	host     []string // Send unicast beacon to List of hosts
}

var clibeacon Beaconcmd

// cmdBeacon - Beacon commands
func cmdBeacon(g *gocui.Gui, args []string, c *sarflags.Cliflags) {

	// var bmu sync.Mutex // Protects beacon.Beacon structure (EID)
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()
	clibeacon.flags = sarflags.Setglobal("beacon", c) // Initialise Global Beacon flags
	clibeacon.interval = c.Timeout.Binterval          // Set up the correct interval

	switch len(args) {
	// Show current Cbeacon flags and lists - beacon
	case 1:
		if clibeacon.count != 0 {
			sarwin.MsgPrintln(g, "yellow_black", clibeacon.count, "Beacons to be sent every %d secs",
				clibeacon.interval)
		} else {
			sarwin.MsgPrintln(g, "yellow_black", "Single Beacon to be sent")
		}
		if clibeacon.v4mcast {
			sarwin.MsgPrintln(g, "yellow_black", "Sending IPv4 multicast beacons")
		}
		if clibeacon.v6mcast {
			sarwin.MsgPrintln(g, "yellow_black", "Sending IPv6 multicast beacons")
		}
		if len(clibeacon.host) > 0 {
			sarwin.MsgPrintln(g, "cyan_black", "Sending beacons to:")
			for _, i := range clibeacon.host {
				sarwin.MsgPrintln(g, "cyan_black", "\t", i)
			}
		}
		if !clibeacon.v4mcast && !clibeacon.v6mcast &&
			len(clibeacon.host) == 0 {
			sarwin.MsgPrintln(g, "yellow_black", "No beacons currently being sent")
		}
		return
	case 2:
		switch args[1] {
		case "?": // usage
			sarwin.MsgPrintln(g, "green_black", prusage("beacon", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("beacon", c))
			return
		case "off": // remove and disable all beacons
			clibeacon.flags = sarflags.Setglobal("beacon", c)
			clibeacon.count = 0
			clibeacon.interval = c.Timeout.Binterval
			clibeacon.host = nil
			sarwin.MsgPrintln(g, "green_black", "Beacons Disabled")
			return
		case "v4": // V4 Multicast
			sarwin.MsgPrintln(g, "cyan_black", "Sending beacon to IPv4 Multicast")
			clibeacon.flags = sarflags.Setglobal("beacon", c)
			clibeacon.v4mcast = true
			clibeacon.count = 1
			// Start up the beacon client sending count IPv4 beacons
			go sendbeacons(g, clibeacon.flags, clibeacon.count, clibeacon.interval, c.V4Multicast, c.Port)
			return
		case "v6": // V6 Multicast
			sarwin.MsgPrintln(g, "cyan_black", "Sending beacon to IPv6 Multicast")
			clibeacon.flags = sarflags.Setglobal("beacon", c)
			clibeacon.v6mcast = true
			clibeacon.count = 1
			// Start up the beacon client sending count IPv6 beacons
			go sendbeacons(g, clibeacon.flags, clibeacon.count, clibeacon.interval, c.V6Multicast, c.Port)
			return
		default: // beacon <count> or beacon <ipaddr>
			if n, err := strconv.ParseUint(args[1], 10, 32); err == nil {
				// We have a number so it is a timer
				clibeacon.count = uint(n)
				sarwin.MsgPrintln(g, "green_black", "Beacons timer set to ", clibeacon.count, " seconds")
			} else {
				sarwin.MsgPrintln(g, "cyan_black", "Sending ", clibeacon.count, " beacons to ", args[1])
				go sendbeacons(g, clibeacon.flags, clibeacon.count, clibeacon.interval, args[1], c.Port)
			}
			return
		}
	}

	// beacon off <ipaddr> ...
	if args[1] == "off" && len(args) > 2 { // turn off following addresses
		sarwin.MsgPrintf(g, "green_black", "%s ", "Beacons turned off to")
		for i := 2; i < len(args); i++ { // Remove Address'es from lists
			if net.ParseIP(args[i]) != nil { // Do We have a valid IP Address
				clibeacon.host = removeValue(clibeacon.host, args[i])
				sarwin.MsgPrintf(g, "green_black", "%s ", args[i])
				if i == len(args)-1 {
					sarwin.MsgPrintln(g, "green_black", "")
				}
			} else {
				sarwin.MsgPrintln(g, "red_black", "Invalid IP Address:", args[i])
				sarwin.CmdPrintln(g, "red_black", prusage("beacon", c))
			}
		}
		return
	}

	// beacon <count> <ipaddr> ...
	var addrstart = 1
	u32, err := strconv.ParseUint(args[1], 10, 32)
	if err == nil { // We have a number so it is a timer
		clibeacon.count = uint(u32)
		sarwin.MsgPrintln(g, "green_black", "Beacon counter set to ", clibeacon.count)
		addrstart = 2
	}
	// beacon [count] <ipaddr> ...
	sarwin.MsgPrintf(g, "cyan_black", "Sending %d beacons to:",
		clibeacon.count)
	for i := addrstart; i < len(args); i++ { // Add Address'es to lists
		sarwin.MsgPrintf(g, "cyan_black", "%s ", args[i])
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
	sarwin.MsgPrintln(g, "green_black", "")
}

func cmdCancel(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarwin.MsgPrintln(g, "green_black", args)
}

func cmdChecksum(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		sarwin.MsgPrintln(g, "green_black", "Checksum ", c.Global["csumtype"])
		return
	case 2:
		switch args[1] {
		case "?": // usage
			sarwin.MsgPrintln(g, "green_black", prusage("checksum", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("checksum", c))
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
			sarwin.CmdPrintln(g, "green_red", prusage("checksum", c))
		}
		return
	}
	sarwin.CmdPrintln(g, "green_red", prusage("checksum", c))
}

// cmdDescriptor -- set descriptor size 16,32,64,128 bits
func cmdDescriptor(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		sarwin.MsgPrintln(g, "green_black", "Descriptor ", c.Global["descriptor"])
		return
	case 2:
		switch args[1] {
		case "?": // usage
			sarwin.MsgPrintln(g, "green_black", prusage("descriptor", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("descriptor", c))
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
			sarwin.MsgPrintln(g, "red_black", "128 bit descriptors not supported on this platform")
		case "d16":
			if sarflags.MaxUint > sarflags.MaxUint16 {
				c.Global["descriptor"] = "d16"
			} else {
				sarwin.MsgPrintln(g, "red_black", "16 bit descriptors not supported on this platform")
			}
		case "d32":
			if sarflags.MaxUint > sarflags.MaxUint32 {
				c.Global["descriptor"] = "d32"
			} else {
				sarwin.MsgPrintln(g, "red_black", "32 bit descriptors not supported on this platform")
			}
		case "d64":
			if sarflags.MaxUint <= sarflags.MaxUint64 {
				c.Global["descriptor"] = "d64"
			} else {
				sarwin.MsgPrintln(g, "red_black", "64 bit descriptors are not supported on this platform")
				sarwin.MsgPrintln(g, "red_black", "MaxUint=", sarflags.MaxUint,
					" <= MaxUint64=", sarflags.MaxUint64)
			}
		case "d128":
			sarwin.MsgPrintln(g, "red_black", "128 bit descriptors not supported on this platform")
		default:
			sarwin.MsgPrintln(g, "red_black", "usage:", prusage("descriptor", c))
		}
		sarwin.MsgPrintln(g, "green_black", "Descriptor size is ", c.Global["descriptor"])
		return
	}
	sarwin.MsgPrintln(g, "red_black", "usage:", prusage("descriptor", c))
}

// Cexit = Exit level to quit from saratoga
var Cexit = -1

// cmdExit -- Quit saratoga
func cmdExit(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	switch len(args) {
	case 1: // exit 0
		Cexit = 0
		sarwin.MsgPrintln(g, "green_black", "Good Bye!")
		return
	case 2:
		switch args[1] {
		case "?": // Usage
			sarwin.MsgPrintln(g, "green_black", prusage("exit", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("exit", c))
		case "0": // exit 0
			Cexit = 0
			sarwin.MsgPrintln(g, "green_black", "Good Bye!")
		case "1": // exit 1
			Cexit = 1
			sarwin.MsgPrintln(g, "green_black", "Good Bye!")
		default: // Help
			sarwin.MsgPrintln(g, "red_black", prusage("exit", c))
		}
	default:
		sarwin.MsgPrintln(g, "red_black", prusage("exit", c))
	}
}

// MORE WORK TO DO HERE!!!!! USE TRANSFERS LIST
// cmdFiiles -- show currently open files
func cmdFiles(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	var flist []string

	switch len(args) {
	case 1:
		if len(flist) == 0 {
			sarwin.MsgPrintln(g, "green_black", "No currently open files")
			return
		}
		for _, i := range flist {
			sarwin.MsgPrintln(g, "green_black", i)
		}
		return
	case 2:
		if args[1] == "?" { // usage
			sarwin.MsgPrintln(g, "green_black", prusage("files", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("files", c))
			return
		}
	}
	sarwin.MsgPrintln(g, "red_black", prusage("files", c))
}

func cmdFreespace(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		if c.Global["freespace"] == "yes" {
			sarwin.MsgPrintln(g, "green_black", "Free space is advertised")
		} else {
			sarwin.MsgPrintln(g, "green_black", "Free space is not advertised")
		}
		return
	case 2:
		switch args[1] {
		case "?": // usage
			sarwin.MsgPrintln(g, "green_black", prusage("freespace", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("freespace", c))
			return
		case "yes":
			sarwin.MsgPrintln(g, "green_black", "freespace is advertised")
			c.Global["freespace"] = "yes"
			return
		case "no":
			sarwin.MsgPrintln(g, "green_black", "freespace is not advertised")
			c.Global["freespace"] = "no"
			return
		}
	}
	sarwin.MsgPrintln(g, "red_black", "usage:", prusage("freespace", c))
}

// Initiator _get_
func cmdGet(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	switch len(args) {
	case 1:
		transfer.Info(g, "get")
		return
	case 2:
		if args[1] == "?" {
			sarwin.MsgPrintln(g, "green_black", prusage("get", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("get", c))
			return
		}
	case 3:
		// var t transfer.CTransfer
		if _, err := transfer.NewInitiator(g, "get", args[1], args[2], c); err != nil {
			return
		}
		return
	}
	sarwin.MsgPrintln(g, "red_black", prusage("get", c))
}

// Initiator _getdir_
func cmdGetdir(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	switch len(args) {
	case 1:
		transfer.Info(g, "getdir")
		return
	case 2:
		if args[1] == "?" {
			sarwin.MsgPrintln(g, "green_black", prusage("getdir", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("getdir", c))
			return
		}
	case 3:
		if _, err := transfer.NewInitiator(g, "getdir", args[1], args[2], c); err != nil {
			sarwin.MsgPrintln(g, "green_black", prusage("getdir", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("getdir", c))
		}
		return
	}
	sarwin.MsgPrintln(g, "red_black", prusage("getdir", c))
}

// Initiator _get_ then _delete_
func cmdGetrm(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	switch len(args) {
	case 1:
		transfer.Info(g, "getrm")
		return
	case 2:
		if args[1] == "?" {
			sarwin.MsgPrintln(g, "green_black", prusage("getrm", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("getrm", c))
			return
		}
	case 3:
		if _, err := transfer.NewInitiator(g, "getrm", args[1], args[2], c); err != nil {
			sarwin.MsgPrintln(g, "green_black", prusage("getrm", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("getrm", c))
			return
		}
		return
	}
	sarwin.MsgPrintln(g, "red_black", prusage("getrm", c))
}

func cmdHelp(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
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
		sarwin.MsgPrintln(g, "magenta_black", sbuf)
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
			sarwin.MsgPrintln(g, "magenta_black", sbuf)
			return
		}
	}
	for key, val := range sarflags.Commands {
		if key == "help" {
			sarwin.MsgPrintln(g, "red_black", fmt.Sprintf("%s - %s",
				key,
				val.Help))
		}
	}
}

func cmdInterval(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		if c.Timeout.Binterval == 0 {
			sarwin.MsgPrintln(g, "yellow_black", "Single Beacon Interation")
		} else {
			sarwin.MsgPrintln(g, "yellow_black", "Beacons sent every ",
				c.Timeout.Binterval, " seconds")
		}
		return
	case 2:
		switch args[1] {
		case "?":
			sarwin.MsgPrintln(g, "green_black", prusage("interval", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("interval", c))
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
		sarwin.MsgPrintln(g, "red_black", prusage("interval", c))
	}
	sarwin.MsgPrintln(g, "red_black", prusage("interval", c))
}

func cmdHistory(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	switch len(args) {
	case 1:
		sarwin.MsgPrintln(g, "green_black", "History not implemented yet")
		return
	case 2:
		switch args[1] {
		case "?":
			sarwin.MsgPrintln(g, "green_black", prusage("history", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("history", c))
			return
		default:
			sarwin.MsgPrintln(g, "green_black", "History not implemented yet")
			return
		}
	}
	sarwin.MsgPrintln(g, "red_black", prusage("history", c))
}

func cmdHome(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	switch len(args) {
	case 1:
		sarwin.MsgPrintln(g, "green_black", "Home not implemented yet")
		return
	case 2:
		switch args[1] {
		case "?":
			sarwin.MsgPrintln(g, "green_black", prusage("home", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("home", c))
			return
		}
	}
	sarwin.MsgPrintln(g, "red_black", prusage("home", c))
}

func cmdLs(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	if len(args) != 0 {
		sarwin.MsgPrintln(g, "red_bblack", prusage("ls", c))
		return
	}
	switch args[1] {
	case "?":
		sarwin.MsgPrintln(g, "green_black", prusage("ls", c))
		sarwin.MsgPrintln(g, "green_black", prhelp("ls", c))
		return
	}
	sarwin.MsgPrintln(g, "green_black", "ls not implemented yet")
}

// Display all of the peer information learned frm beacons
func cmdPeers(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	if len(args) != 1 {
		sarwin.MsgPrintln(g, "red_bblack", prusage("peers", c))
		return
	}

	if len(beacon.Peers) == 0 {
		sarwin.MsgPrintln(g, "magenta_black", "No Peers")
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
	sarwin.MsgPrintln(g, "magenta_black", sbuf)
}

// Initiator _put_
// send a file to a destination
func cmdPut(g *gocui.Gui, args []string, c *sarflags.Cliflags) {

	switch len(args) {
	case 1:
		transfer.Info(g, "put")
		return
	case 2:
		if args[1] == "?" {
			sarwin.MsgPrintln(g, "green_black", prusage("put", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("put", c))
			return
		}
	case 3:
		if t, err := transfer.NewInitiator(g, "put", args[1], args[2], c); err == nil && t != nil {
			errflag := make(chan error, 1) // The return channel holding the saratoga errflag
			go t.Do(g, errflag)            // Actually do the transfer
			errcode := <-errflag
			if errcode != "success" {
				sarwin.ErrPrintln(g, "red_black", "Error:", errcode,
					" Unable to send file:", t.Print())
				if derr := t.Remove(); derr != nil {
					sarwin.MsgPrintln(g, "red_black", "Unable to remove transfer:", t.Print())
				}
			}
			sarwin.MsgPrintln(g, "green_black", "put completed closing channel")
			close(errflag)
		} else {
			sarwin.MsgPrintln(g, "red_black", "Cannot add transfer:", err.Error())
		}
		return
	}
	sarwin.MsgPrintln(g, "red_black", prusage("put", c))
}

// Initiator _put_
// blind send a file to a destination not expecting return _status_ from Responder
func cmdPutblind(g *gocui.Gui, args []string, c *sarflags.Cliflags) {

	errflag := make(chan string, 1) // The return channel holding the saratoga errflag

	switch len(args) {
	case 1:
		transfer.Info(g, "putrm")
		return
	case 2:
		if args[1] == "?" {
			sarwin.MsgPrintln(g, "green_black", prusage("putblind", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("putblind", c))
			return
		}
	case 3:
		// We send the Metadata and do not bother with request/status exchange
		if t, err := transfer.NewInitiator(g, "putblind", args[1], args[2], c); err == nil && t != nil {
			go t.Do(g, errflag)
			errcode := <-errflag
			if errcode != "success" {
				sarwin.ErrPrintln(g, "red_black", "Error:", errcode,
					"Unable to send file:", t.Print())
			}
		} else {
			sarwin.ErrPrintln(g, "red_black", "Cannot create Transfer:", error.Error(err))
		}
		return
	}
	sarwin.MsgPrintln(g, "red_black", prusage("putblind", c))
}

// Initiator _put_
// send a file file to a remote destination then remove it from the origin
func cmdPutrm(g *gocui.Gui, args []string, c *sarflags.Cliflags) {

	errflag := make(chan string, 1) // The return channel holding the saratoga errflag

	switch len(args) {
	case 1:
		transfer.Info(g, "putrm")
		return
	case 2:
		if args[1] == "?" {
			sarwin.MsgPrintln(g, "green_black", prusage("putrm", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("putrm", c))
			return
		}
	case 3:
		// var t *transfer.Transfer
		if t, err := transfer.NewInitiator(g, "putrm", args[1], args[2], c); err == nil && t != nil {
			go transfer.Do(g, errflag)
			errcode := <-errflag
			if errcode != "success" {
				sarwin.ErrPrintln(g, "red_black", "Error:", errcode,
					" Unable to send file:", t.Print())
			} else {
				sarwin.MsgPrintln(g, "red_black",
					"Put and now removing (NOT) (ADD MORE CODE  HERE!!!!) file:", t.Print())
			}
		}
		return
	}
	sarwin.MsgPrintln(g, "red_black", prusage("putrm", c))
}

func cmdReqtstamp(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		if c.Global["reqtstamp"] == "yes" {
			sarwin.MsgPrintln(g, "green_black", "Time stamps requested")
		} else {
			sarwin.MsgPrintln(g, "green_black", "Time stamps not requested")
		}
		return
	case 2:
		switch args[1] {
		case "?": // usage
			sarwin.MsgPrintln(g, "green_black", prusage("reqtstamp", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("reqtstamp", c))
			return
		case "yes":
			c.Global["reqtstamp"] = "yes"
			return
		case "no":
			c.Global["reqtstamp"] = "no"
			return
		}
	}
	sarwin.MsgPrintln(g, "red_black", "usage:", prusage("reqtstamp", c))
}

// Initiator _delete_
// remove a file from a remote destination
func cmdRm(g *gocui.Gui, args []string, c *sarflags.Cliflags) {

	switch len(args) {
	case 1:
		transfer.Info(g, "rm")
		return
	case 2:
		if args[1] == "?" {
			sarwin.MsgPrintln(g, "green_black", prusage("rm", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("rm", c))
			return
		}
	case 3:
		if _, err := transfer.NewInitiator(g, "rm", args[1], args[2], c); err != nil {
			sarwin.MsgPrintln(g, "green_black", prusage("rm", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("rm", c))
			return
		}
	}
	sarwin.MsgPrintln(g, "red_black", prusage("rm", c))
}

// Initiator _getdir_, _delete_ ...
// remove a directory from a remote destination
func cmdRmdir(g *gocui.Gui, args []string, c *sarflags.Cliflags) {

	switch len(args) {
	case 1:
		transfer.Info(g, "rmdir")
		return
	case 2:
		if args[1] == "?" {
			sarwin.MsgPrintln(g, "green_black", prusage("rmdir", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("rmdir", c))
			return
		}
	case 3:
		if _, err := transfer.NewInitiator(g, "rmdir", args[1], args[2], c); err != nil {
			sarwin.MsgPrintln(g, "green_black", prusage("rmdir", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("rmdir", c))
			return
		}
	}
	sarwin.MsgPrintln(g, "red_black", prusage("rmdir", c))
}

func cmdRmtran(g *gocui.Gui, args []string, c *sarflags.Cliflags) {

	switch len(args) {
	case 1:
		sarwin.MsgPrintln(g, "green_black", prusage("rmtran", c))
		sarwin.MsgPrintln(g, "green_black", prhelp("rmtran", c))
		return
	case 2:
		if args[1] == "?" {
			sarwin.MsgPrintln(g, "green_black", prusage("rmtran", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("rmtran", c))
			return
		}
	case 4:
		ttype := args[1]
		addr := args[2]
		fname := args[3]
		if t := transfer.Match(addr, fname); t != nil {
			if err := t.Remove(); err != nil {
				sarwin.MsgPrintln(g, "red_black", err.Error())
			}
		} else {
			sarwin.MsgPrintln(g, "red_black", "No such transfer:", ttype, " ", addr, " ", fname)
		}
		return
	}
	sarwin.MsgPrintln(g, "red_black", prusage("rmtran", c))
}

// Are we willing to transmit files
func cmdRxwilling(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		sarwin.MsgPrintln(g, "green_black", "Receive Files:", c.Global["rxwilling"])
		return
	case 2:
		switch args[1] {
		case "?":
			sarwin.MsgPrintln(g, "green_black", prusage("rxwilling", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("rxwilling", c))
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
	sarwin.MsgPrintln(g, "red_black", prusage("rxwilling", c))
}

// Initiator _put_ not expecting _status_
// source is a named pipe not a file
func cmdStream(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		if c.Global["stream"] == "yes" {
			sarwin.MsgPrintln(g, "green_black", "Can stream")
		} else {
			sarwin.MsgPrintln(g, "green_black", "Cannot stream")
		}
		return
	case 2:
		switch args[1] {
		case "?":
			sarwin.MsgPrintln(g, "green_black", prusage("stream", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("stream", c))
			return
		case "yes":
			c.Global["stream"] = "yes"
			return
		case "no":
			c.Global["stream"] = "no"
			return
		}
	}
	sarwin.MsgPrintln(g, "red_black", prusage("stream", c))
}

// Timeout - set timeouts for responses to request/status/transfer in seconds
func cmdTimeout(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		if c.Timeout.Metadata == 0 {
			sarwin.MsgPrintln(g, "green_black", "metadata:No Timeout")
		} else {
			sarwin.MsgPrintln(g, "green_black", "metadata:", c.Timeout.Metadata, " sec")
		}
		if c.Timeout.Request == 0 {
			sarwin.MsgPrintln(g, "green_black", "request:No Timeout")
		} else {
			sarwin.MsgPrintln(g, "green_black", "request:", c.Timeout.Request, " sec")
		}
		if c.Timeout.Status == 0 {
			sarwin.MsgPrintln(g, "green_black", "status:No Timeout")
		} else {
			sarwin.MsgPrintln(g, "green_black", "status:", c.Timeout.Status, " sec")
		}
		if c.Timeout.Datacounter == 0 {
			c.Timeout.Datacounter = 100
			sarwin.MsgPrintln(g, "green_black", "Data Counter every 100 frames")
		} else {
			sarwin.MsgPrintln(g, "green_black", "Datacnt:", c.Timeout.Datacounter, " frames")
		}
		if c.Timeout.Transfer == 0 {
			sarwin.MsgPrintln(g, "green_black", "transfer:No Timeout")
		} else {
			sarwin.MsgPrintln(g, "green_black", "transfer:", c.Timeout.Transfer, " sec")
		}
		return
	case 2:
		switch args[1] {
		case "?":
			sarwin.MsgPrintln(g, "green_black", prusage("timeout", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("timeout", c))
		case "request":
			if c.Timeout.Request == 0 {
				sarwin.MsgPrintln(g, "green_black", "request:No Timeout")
			} else {
				sarwin.MsgPrintln(g, "green_black", "request:", c.Timeout.Request, " sec")
			}
		case "metadata":
			if c.Timeout.Request == 0 {
				sarwin.MsgPrintln(g, "green_black", "metadata:No Timeout")
			} else {
				sarwin.MsgPrintln(g, "green_black", "metadata:", c.Timeout.Metadata, " sec")
			}
		case "status":
			if c.Timeout.Status == 0 {
				sarwin.MsgPrintln(g, "green_black", "status:No Timeout")
			} else {
				sarwin.MsgPrintln(g, "green_black", "status:", c.Timeout.Status, " sec")
			}
		case "datacounter":
			if c.Timeout.Datacounter == 0 {
				c.Timeout.Datacounter = 100
				sarwin.MsgPrintln(g, "green_black", "Datacounter:Never")
			} else {
				sarwin.MsgPrintln(g, "green_black", "Datacounter:", c.Timeout.Datacounter, " frames")
			}
		case "transfer":
			if c.Timeout.Transfer == 0 {
				sarwin.MsgPrintln(g, "green_black", "transfer:No Timeout")
			} else {
				sarwin.MsgPrintln(g, "green_black", "transfer:", c.Timeout.Transfer, " sec")
			}
		default:
			sarwin.MsgPrintln(g, "red_black", prusage("stream", c))
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
			case "datacnt":
				if n == 0 {
					n = 100
				}
				c.Timeout.Datacounter = n
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
			case "datacounter":
				c.Timeout.Datacounter = 100
			case "transfer":
				c.Timeout.Transfer = 60
			}
			return
		}
	}
	sarwin.MsgPrintln(g, "red_black", prusage("timeout", c))
}

// set the timestamp type we are using
func cmdTimestamp(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		sarwin.MsgPrintln(g, "green_black", "Timestamps  are",
			c.Global["reqtstamp"], " and ", c.Timestamp)
		return
	case 2:
		switch args[1] {
		case "?":
			sarwin.MsgPrintln(g, "green_black", prusage("timestamp", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("timestamp", c))
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
			sarwin.MsgPrintln(g, "red_black", prusage("timestamp", c))
		}
		return
	}
	sarwin.MsgPrintln(g, "red_black", prusage("timestamp", c))
}

// set the timezone we use for logs local or utc
func cmdTimezone(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		sarwin.MsgPrintln(g, "green_black", "Timezone:", c.Timezone)
		return
	case 2:
		switch args[1] {
		case "?":
			sarwin.MsgPrintln(g, "green_black", prusage("timezone", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("timezone", c))
		case "local":
			c.Timezone = "local"
		case "utc":
			c.Timezone = "utc"
		default:
			sarwin.MsgPrintln(g, "red_black", prusage("timezone", c))
		}
		return
	}
	sarwin.MsgPrintln(g, "red_black", prusage("timezone", c))
}

// show current transfers in progress & % completed
func cmdTran(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	switch len(args) {
	case 1:
		transfer.Info(g, "")
		return
	case 2:
		switch args[1] {
		case "?":
			sarwin.MsgPrintf(g, "green_black", "%s\n  %s\n",
				prusage("tran", c), prhelp("tran", c))
		default:
			for _, tt := range transfer.Ttypes {
				if args[1] == tt {
					transfer.Info(g, args[1])
					return
				}
			}
			sarwin.MsgPrintln(g, "green_black", prusage("tran", c))
		}
		return
	}
	sarwin.MsgPrintln(g, "green_black", prusage("tran", c))
}

// we are willing to transmit files
func cmdTxwilling(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		sarwin.MsgPrintln(g, "green_black", "Transmit Files:", c.Global["txwilling"])
		return
	case 2:
		switch args[1] {
		case "?":
			sarwin.MsgPrintln(g, "green_black", prusage("txwilling", c))
			sarwin.MsgPrintln(g, "green_black", prhelp("txwilling", c))
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
	sarwin.MsgPrintln(g, "red_black", prusage("txwilling", c))
}

// Show all commands usage
func cmdUsage(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
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
	sarwin.MsgPrintln(g, "magenta_black", sbuf)
}

// *********************************************************************************************************

type cmdfunc func(*gocui.Gui, []string, *sarflags.Cliflags)

// Commands and function pointers to handle them
var cmdhandler = map[string]cmdfunc{
	"?":          cmdHelp,
	"beacon":     cmdBeacon,
	"cancel":     cmdCancel,
	"checksum":   cmdChecksum,
	"descriptor": cmdDescriptor,
	"exit":       cmdExit,
	"files":      cmdFiles,
	"freespace":  cmdFreespace,
	"get":        cmdGet,    // _get_
	"getdir":     cmdGetdir, // _getdir_
	"getrm":      cmdGetrm,  // _get_,_delete_
	"help":       cmdHelp,
	"history":    cmdHistory,
	"home":       cmdHome,
	"interval":   cmdInterval,
	"ls":         cmdLs,
	"peers":      cmdPeers,
	"put":        cmdPut,      // _put_
	"putblind":   cmdPutblind, // _put_ (no _status_)
	"putrm":      cmdPutrm,    // _put_ (delete local)
	"quit":       cmdExit,
	"reqtstamp":  cmdReqtstamp,
	"rm":         cmdRm, // _delete_
	"rmtran":     cmdRmtran,
	"rmdir":      cmdRmdir, // _getdir_, _delete_ ...
	"rxwilling":  cmdRxwilling,
	"stream":     cmdStream,
	"timeout":    cmdTimeout,
	"timestamp":  cmdTimestamp,
	"timezone":   cmdTimezone,
	"tran":       cmdTran,
	"txwilling":  cmdTxwilling,
	"usage":      cmdUsage,
}

// Lookup the command and execute it
func Lookup(g *gocui.Gui, c *sarflags.Cliflags, name string) bool {
	if name == "" { // Handle just return
		return true
	}
	// Get rid of leading and trailing whitespace
	s := strings.TrimSpace(name)
	vals := strings.Fields(s)
	for key := range sarflags.Commands {
		if key == name {
			fn, ok := cmdhandler[name]
			if ok {
				fn(g, vals, c)
				return true
			}
			return false
		}
	}
	return false
}

// Docmd -- Execute the command entered
func Docmd(g *gocui.Gui, s string, c *sarflags.Cliflags) {
	if s == "" { // Handle just return
		return
	}
	if !Lookup(g, c, s) {
		sarwin.ErrPrintf(g, "red_black", "Invalid command:", s)
	}
}
