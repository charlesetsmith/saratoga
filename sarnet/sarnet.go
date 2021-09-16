// Network Handler

package sarnet

import (
	"errors"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// SaratogaPort - IANA allocated Saratoga UDP & TCP Port is 7542

// IPv4Multicast -- IANA allocated Saratoga IPV4 all-hosts Multicast Address is "224.0.0.108"

// IPv6Multicast -- IANA allocated Saratoga IPV6 link-local Multicast Address is "FF02:0:0:0:0:0:0:6c" or "FF02::6c"

// MaxFrameSize -- Maximum Saratoga Frame Size
// Move this to Network Section & Calculate it
// const MaxFrameSize = 1500 - 60 // After MTU & IPv6 Header

// UDPinfo - Return string of IP Address and Port #
func UDPinfo(addr *net.UDPAddr) string {
	if strings.Contains(addr.IP.String(), ":") { // IPv6
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
func SetMulticastLoop(conn net.PacketConn, v4orv6 string) error {
	file, _ := conn.(*net.UDPConn).File()
	fd := int(file.Fd())
	switch v4orv6 {
	case "IPv4":
		if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_MULTICAST_LOOP, 1); err != nil {
			return err
		}
		return nil
	case "IPv6":
		if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_IPV6, syscall.IPV6_MULTICAST_LOOP, 1); err != nil {
			return err
		}
		return nil
	default:
		err := "invalid - should be IPv4 or IPv6:" + v4orv6
		return errors.New(err)
	}
	//syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
}
