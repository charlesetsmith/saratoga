package client

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"screen"
	"strconv"
	"strings"

	"frames"
	"sarflags"
	"sys"
)

// V4McastBeacon - Start up a Multicast beacon to a group
func V4McastBeacon(addr string, port int, timer uint, header uint32 errflag chan uint32) {
	screen.Fprintln(screen.Msg, "blue_black", "Sending Multicast beacons to ", addr, port)
	// p := make([]byte, 2048)
	// udpad := addr + ":" + strconv.Itoa(port)
	errflag <- uint32(sarflags.Value("errcode", "success"))
}

// V6McastBeacon - Start up a Multicast beacon to a group
func V6McastBeacon(addr string, port int, timer uint, header uint32, errflag chan uint32) {
	screen.Fprintln(screen.Msg, "blue_black", "Sending Multicast beacons to ", addr, port)
	// p := make([]byte, 2048)
	// udpad := "[" + addr + "]:" + strconv.Itoa(port)
	errflag <- uint32(sarflags.Value("errcode", "success"))
}

// Beacon - Start up a beacon to a server
func Beacon(addr string, port int, timer uint, header uint32, errflag chan uint32) {

	var flagstr string
	var df uint64
	var eid string

	// Construct the appropriate Beacon header flags from the Globals set on cli
	if net.ParseIP(addr) != nil {
		pstr := strconv.Itoa(port)
		if strings.Contains(addr, ".") { // IPv4
			eid = addr + ":" + pstr
		} else if strings.Contains(addr, ":") { // IPv6
			eid = "[" + addr + "]:" + pstr
		} else {
			ret, _ := sarflags.Set(0, "errcode", "badpacket")
			errflag <- ret
			return
		}
	}

	// Assemble the beacon frame
	var frame []byte
	var err error

	if frame, err = frames.BeaconMake(header, eid, df); err != nil {
		ret, _ := sarflags.Set(0, "errcode", "badpacket")
		errflag <- ret
	}
	b, _ := frames.BeaconGet(frame)
	screen.Fprintln(screen.Msg, "magenta_black", "Sending Beacon to ", addr, ":", frames.BeaconPrint(b))

	p := make([]byte, 2048)
	udpad := addr + ":" + strconv.Itoa(port)
	conn, err := net.Dial("udp", udpad)
	defer conn.Close()
	if err != nil {
		log.Fatalf("Cannot open UDP Socket %s %v", udpad, err)
		return
	}
	fmt.Fprintf(conn, "Hi UDP Server, How are you doing?")
	_, err = bufio.NewReader(conn).Read(p)
	if err == nil {
		fmt.Printf("%s\n", p)
	} else {
		fmt.Printf("Some error %v\n", err)
	}
	errflag <- uint32(sarflags.Value("errcode", "success"))
}
