package client

import (
	"net"
	"strconv"
	"strings"

	"beacon"
	"sarflags"
	"sarnet"
	"screen"
)

// V4McastBeacon - Start up a Multicast beacon to a group
func V4McastBeacon(timer uint, flags string, errflag chan uint32) {
	screen.Fprintln(screen.Msg, "blue_black", "Sending Multicast beacons to ",
		sarnet.IPv4Multicast, sarnet.Port())
	// p := make([]byte, 2048)
	// udpad := addr + ":" + strconv.Itoa(port)
	errflag <- uint32(sarflags.Value("errcode", "success"))
}

// V6McastBeacon - Start up a Multicast beacon to a group
func V6McastBeacon(timer uint, flags string, errflag chan uint32) {
	screen.Fprintln(screen.Msg, "blue_black", "Sending Multicast beacons to ",
		sarnet.IPv6Multicast, sarnet.Port())
	// p := make([]byte, 2048)
	// udpad := "[" + addr + "]:" + strconv.Itoa(port)
	errflag <- uint32(sarflags.Value("errcode", "success"))
}

// Beacon - Send a beacon to a servers address(s)
func Beacon(addr string, timer uint, flags string, errflag chan uint32) {

	// var flagstr string
	// var df uint64
	var eid string

	// Construct the appropriate Beacon header flags from the Globals set on cli
	if net.ParseIP(addr) != nil {
		pstr := strconv.Itoa(sarnet.Port())
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
	var b beacon.Beacon
	var newb beacon.Beacon

	if err = b.New(flags, eid, 1000); err != nil {
		ret, _ := sarflags.Set(0, "errcode", "badpacket")
		errflag <- ret
	}
	if frame, err = b.Put(); err != nil {
		ret, _ := sarflags.Set(0, "errcode", "badpacket")
		errflag <- ret
	}
	screen.Fprintln(screen.Msg, "yellow_black", "Sending Beacon to ", addr, ":", b.Print())

	if err = newb.Get(frame); err != nil {
		ret, _ := sarflags.Set(0, "errcode", "badpacket")
		errflag <- ret
	}
	screen.Fprintln(screen.Msg, "green_black", "Receiving Beacon from ", addr, ":", newb.Print())
	/*
		p := make([]byte, 2048)
		udpad := addr + ":" + strconv.Itoa(sarnet.Port())
		conn, err := net.Dial("udp", udpad)
		defer conn.Close()
		if err != nil {
			log.Fatalf("Cannot open UDP Socket to %s %v", udpad, err)
			return
		}
		fmt.Fprintf(conn, "Hi UDP Server, How are you doing?")
		_, err = bufio.NewReader(conn).Read(p)
		if err == nil {
			fmt.Printf("%s\n", p)
		} else {
			fmt.Printf("Some error %v\n", err)
		}
	*/
	errflag <- uint32(sarflags.Value("errcode", "success"))
}
