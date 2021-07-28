// Network Handler

package sarnet

import (
	"log"
	"net"
	"os"
	"strconv"
	"syscall"
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
