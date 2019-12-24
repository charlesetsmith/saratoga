package sarnet

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"syscall"

	"github.com/charlesetsmith/saratoga/src/beacon"
	"github.com/charlesetsmith/saratoga/src/data"
	"github.com/charlesetsmith/saratoga/src/metadata"
	"github.com/charlesetsmith/saratoga/src/request"
	"github.com/charlesetsmith/saratoga/src/sarflags"
	"github.com/charlesetsmith/saratoga/src/status"
)

// SaratogaPort - IANA allocated Saratoga UDP & TCP Port #'s
const saratogaport = 7542

// IPv4Multicast -- IANA allocated Saratoga IPV4 all-hosts Multicast Address
const IPv4Multicast = "224.0.0.108"

// IPv6Multicast -- IANA allocated Saratoga IPV6 link-local Multicast Address
// const IPv6Multicast = "FF02:0:0:0:0:0:0:6c"
const IPv6Multicast = "FF02::6c"

// MaxFrameSize -- Maximum Saratoga Frame Size
// Move this to Network Section & Calculate it
const MaxFrameSize = 1500 - 60 // After MTU & IPv6 Header

// Port -- Lookup saratoga official port number in /etc/services
func Port() int {

	var port int
	var err error

	if port, err = net.LookupPort("udp", "saratoga"); err != nil {
		port = saratogaport // Set to default if not in /etc/services
	}
	return port
}

// UDPinfo - Return string of IP Address and Port #
func UDPinfo(addr *net.UDPAddr) string {
	if addr.IP.To4() == nil { // IPv6 Address
		a := "[" + addr.IP.String() + "]:" + strconv.Itoa(addr.Port)
		return a
	}
	// IPv4 Address
	a := addr.IP.String() + ":" + strconv.Itoa(addr.Port)
	return a
}

// OutboundIP - Get preferred outbound ip of this host
// typ is "IPv4" or "IPv6"
func OutboundIP(typ string) net.IP {

	host, _ := os.Hostname()
	addrs, _ := net.LookupIP(host)
	for _, addr := range addrs {
		if ipv4 := addr.To4(); typ == "IPv4" && ipv4 != nil {
			return ipv4
		}
		if ipv6 := addr.To16(); typ == "IPv6" && ipv6 != nil {
			return ipv6
		}
	}
	log.Fatal("getoutboundIP: type must be IPv4 or IPv6")
	return nil
}

// Rxbeacon We have an inbound beacon frame so, send a beacon back
// to the client giving it our info
func Rxbeacon(from *net.UDPAddr, b *beacon.Beacon) string {
	// fmt.Println(b.Print())
	// We add / alter the peer information
	if b.NewPeer(from) == true {
		fmt.Printf("PEERS ARE:\n")
		for p := range beacon.Peers {
			fmt.Printf("%s %dMb %s %s %s\n",
				beacon.Peers[p].Addr,
				beacon.Peers[p].Freespace/1024,
				beacon.Peers[p].Eid,
				beacon.Peers[p].Created.Print(),
				beacon.Peers[p].Updated.Print())
		}
	} else {
		fmt.Printf("NO CHANGES TO PEERS\n")
	}
	return "success"
}

// Rxrequest - We have a request starting up a new session or updating the session info
// Create a new, or change an existing sessions information
func Rxrequest(from *net.UDPAddr, session uint32, r *request.Request) string {
	fmt.Println(r.Print())
	switch sarflags.GetStr(r.Header, "reqtype") {
	case "noaction":
		fmt.Println("Request Noaction from ", UDPinfo(from), " session ", session)
	case "get":
		fmt.Println("Request Get from ", UDPinfo(from), " session ", session)
	case "put":
		fmt.Println("Request Put from ", UDPinfo(from), " session ", session)
	case "getdelete":
		fmt.Println("Request GetDelete from ", UDPinfo(from), " session ", session)
	case "putdelete":
		fmt.Println("Request PutDelete from ", UDPinfo(from), " session ", session)
	case "delete":
		fmt.Println("Request Delete from ", UDPinfo(from), " session ", session)
	case "getdir":
		fmt.Println("Request GetDir from ", UDPinfo(from), " session ", session)
	default:
		fmt.Println("Invalid Request from ", UDPinfo(from), " session ", session)
		return "badrequest"
	}
	return "success"
	// Return an errcode string
}

// Rxdata - We have some incoming data for a session. Add the data to the session
// For this implementation of saratoga if no session exists then just dump the data
func Rxdata(from *net.UDPAddr, session uint32, d *data.Data) string {
	fmt.Println(d.Print())
	// Return an errcode string
	return "success"
}

// Rxmetadata - We have some incoming metadata for a session. Add the metadata to the session
func Rxmetadata(from *net.UDPAddr, session uint32, m *metadata.MetaData) string {
	fmt.Println(m.Print())
	// Return an errcode string
	return "success"
}

// Rxstatus - We have a status update for a session. Process the holes or close the session
func Rxstatus(from *net.UDPAddr, session uint32, s *status.Status) string {
	fmt.Println(s.Print())
	// Return an errcode string
	return "success"
}

// Listen -- IPv4 & IPv6 for an incoming frames
func Listen(conn *net.UDPConn, quit chan struct{}) {

	buf := make([]byte, MaxFrameSize+100) // Just in case...
	framelen := 0
	err := error(nil)
	remoteAddr := new(net.UDPAddr)
next:
	for err == nil { // Loop forever grabbing frames
		// Read into buf
		framelen, remoteAddr, err = conn.ReadFromUDP(buf)
		if err != nil {
			break
		}

		// Very basic frame checks before we get into what it is
		if framelen < 8 {
			// Saratoga packet too small
			var se status.Status
			// We don't know the session # so use 0
			_ = se.New("errcode=badpacket", 0, 0, 0, nil)
			fmt.Println("Rx Saratoga Frame too short from ",
				UDPinfo(remoteAddr))
			goto next
		}
		if framelen > MaxFrameSize {
			// Saratoga packet too long
			var se status.Status
			// We can't even know the session # so use 0
			_ = se.New("errcode=badpacket", 0, 0, 0, nil)
			fmt.Println("Rx Saratoga Frame too long from ",
				UDPinfo(remoteAddr))
			goto next
		}

		// OK so we might have a valid frame so copy it to to the frame byte slice
		frame := make([]byte, framelen)
		copy(frame, buf[:framelen])
		// fmt.Println("We have a frame of length:", framelen)
		// Grab the Saratoga Header
		header := binary.BigEndian.Uint32(frame[:4])
		if sarflags.GetStr(header, "version") != "v1" { // Make sure we are Version 1
			if header, err = sarflags.Set(0, "errno", "badpacket"); err != nil {
				// Bad Packet send back a Status to the client
				var se status.Status
				_ = se.New("errcode=badpacket", 0, 0, 0, nil)
				fmt.Println("Not Saratoga Version 1 Frame from ",
					UDPinfo(remoteAddr))
			}
			goto next
		}

		// Process the frame
		switch sarflags.GetStr(header, "frametype") {
		case "beacon":
			var rxb beacon.Beacon
			if rxerr := rxb.Get(frame); rxerr != nil {
				// We just drop bad beacons
				fmt.Println("Bad Beacon:", rxerr, " from ",
					UDPinfo(remoteAddr))
				goto next
			}
			// Handle the beacon
			if errcode := rxbeacon(remoteAddr, &rxb); errcode != "success" {
				fmt.Println("Bad Beacon:", errcode, " from ",
					UDPinfo(remoteAddr))
				goto next
			}

		case "request":
			var r request.Request
			if rxerr := r.Get(frame); rxerr != nil {
				session := binary.BigEndian.Uint32(frame[4:8])
				var se status.Status
				// Send back a Status to the client
				_ = se.New("errcode=badpacket", session, 0, 0, nil)
				fmt.Println("Bad Request:", rxerr, " from ",
					UDPinfo(remoteAddr), " session ", session)
				goto next
			} // Handle the request
			session := binary.BigEndian.Uint32(frame[4:8])
			if errcode := rxrequest(remoteAddr, session, &r); errcode != "success" {

			}
		case "data":
			var d data.Data
			if rxerr := d.Get(frame); rxerr != nil {
				session := binary.BigEndian.Uint32(frame[4:8])
				var se status.Status
				// Bad Packet send back a Status to the client
				_ = se.New("errcode=badpacket", session, 0, 0, nil)
				fmt.Println("Bad Data:", rxerr, " from ",
					UDPinfo(remoteAddr), " session ", session)
				goto next
			} // Handle the data
			session := binary.BigEndian.Uint32(frame[4:8])
			if errcode := rxdata(remoteAddr, session, &d); errcode != "success" {

			}
		case "metadata":
			var m metadata.MetaData
			if rxerr := m.Get(frame); rxerr != nil {
				session := binary.BigEndian.Uint32(frame[4:8])
				var se status.Status
				// Bad Packet send back a Status to the client
				_ = se.New("errcode=badpacket", session, 0, 0, nil)
				fmt.Println("Bad MetaData:", rxerr, " from ",
					UDPinfo(remoteAddr), " session ", session)
				goto next
			} // Handle the data
			session := binary.BigEndian.Uint32(frame[4:8])
			if errcode := rxmetadata(remoteAddr, session, &m); errcode != "success" {

			}
		case "status":
			var s status.Status
			if rxerr := s.Get(frame); rxerr != nil {
				session := binary.BigEndian.Uint32(frame[4:8])
				var se status.Status
				// Bad Packet send back a Status to the client
				_ = se.New("errcode=badpacket", session, 0, 0, nil)
				fmt.Println("Bad Status:", rxerr, " from ",
					UDPinfo(remoteAddr), " session ", session)
				goto next
			} // Handle the status
			session := binary.BigEndian.Uint32(frame[4:8])
			if errcode := rxstatus(remoteAddr, session, &s); errcode != "success" {

			}
		default:
			// Bad Packet send back a Status to the client
			// We can't even know the session # so use 0
			var se status.Status
			_ = se.New("errcode=badpacket", 0, 0, 0, nil)
			fmt.Println("Bad Header in Saratoga Frame from ",
				UDPinfo(remoteAddr))
		}
	}
	fmt.Println("Sarataga listener failed - ", err)
	quit <- struct{}{}
}

// SetMulticastLoop - Set the Multicast Loopback address OK for Rx Multicasts
func SetMulticastLoop(conn net.PacketConn, v4orv6 string) {
	file, _ := conn.(*net.UDPConn).File()
	fd := int(file.Fd())
	if v4orv6 == "IPv4" {
		if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_MULTICAST_LOOP, 1); err != nil {
			panic(err)
		}
	}
	if v4orv6 == "IPv6" {
		if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_IPV6, syscall.IPV6_MULTICAST_LOOP, 1); err != nil {
			panic(err)
		}
	}
	//syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
}
