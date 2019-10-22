package client

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"screen"
	"strconv"

	"github.com/charlesetsmith/saratoga/src/sarflags"
)

// V6McastBeacon - Start up a Multicast beacon to a group
func V6McastBeacon(addr string, port int, timer uint, errflag chan uint32) {
	screen.Fprintln(screen.Msg, "blue_black", "Sending Multicast beacons to ", addr, port)
	p := make([]byte, 2048)
	udpad := "[" + addr + "]:" + strconv.Itoa(port)
	errflag <- uint32(sarflags.Value("errcode", "success"))
}

// Beacon - Start up a beacon to a server
func Beacon(addr string, port int, timer uint, errflag chan uint32) {
	screen.Fprintln(screen.Msg, "blue_black", "Sending beacons to:", addr)
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
