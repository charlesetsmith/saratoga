// Network Handler

package sarnet

import (
	// "bytes"
	// "encoding/gob"
	"errors"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// SaratogaPort - IANA allocated Saratoga UDP & TCP Port is 7542
const Port int = 7542

// IPv4Multicast -- IANA allocated Saratoga IPV4 all-hosts Multicast Address is "224.0.0.108"
const IPv4Multicast string = "224.0.0.108"

// IPv6Multicast -- IANA allocated Saratoga IPV6 link-local Multicast Address is "FF02:0:0:0:0:0:0:6c" or "FF02::6c"
const IPv6Multicast string = "FF02::6c"

// MaxFrameSize -- Maximum Saratoga Frame Size
// Move this to Network Section & Calculate it
// const MaxFrameSize = 1500 - 60 // After MTU & IPv6 Header

// Return if address type is IPv4
func isIPv4(address string) bool {
	return strings.Count(address, ".") == 3
}

// Return if address type is IPv6
func isIPv6(address string) bool {
	return strings.Count(address, ":") >= 2 // (::)
}

// Given a valid IPv4 or IPv6 address
// Returns the ipv4 or ipv6 resolved UDPAddr with the saratoga Port ready to "dial"
func UDPAddress(s string) (*net.UDPAddr, error) {
	var a string

	if isIPv6(s) {
		a = "[" + s + "]"
	} else if isIPv4(s) {
		a = s
	} else {
		return nil, errors.New("Invalid IPAddress:" + s)
	}
	a = a + ":" + strconv.Itoa(Port)
	if udpad, err := net.ResolveUDPAddr("udp", a); err == nil {
		return udpad, nil
	}
	return nil, errors.New("Cannot ResolveUDPAddr: " + a)
}

// UDPinfo - Return string of IP Address and Port #
func UDPinfo(addr *net.UDPAddr) string {
	if addr == nil {
		return ""
	}
	return addr.String()
}

func removeUDPAddrIndex(a []net.UDPAddr, index int) []net.UDPAddr {
	ret := make([]net.UDPAddr, 0)
	ret = append(ret, a[:index]...)
	return append(ret, a[index+1:]...)
}

// removeUDPAddrValue -- Remove all entries in slice of UDPAddr matching val
func RemoveUDPAddrValue(a []net.UDPAddr, val *net.UDPAddr) []net.UDPAddr {
	for i := 0; i < len(a); i++ {
		if a[i].String() == val.String() {
			a = removeUDPAddrIndex(a, i)
			a = RemoveUDPAddrValue(a, val) // Too recurse is divine. Call me again to remove dupes
		}
	}
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
func SetMulticastLoop(conn net.PacketConn) error {
	file, _ := conn.(*net.UDPConn).File()
	fd := int(file.Fd())
	addr := conn.LocalAddr().String()
	if isIPv4(addr) {
		if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_MULTICAST_LOOP, 1); err != nil {
			return err
		}
	} else if isIPv6(addr) {
		if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_IPV6, syscall.IPV6_MULTICAST_LOOP, 1); err != nil {
			return err
		}
	} else {
		err := "invalid - should be IPv4 or IPv6 address"
		return errors.New(err)
	}
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		return err
	}
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEPORT, 1); err != nil {
		return err
	}
	return nil
}

// ****************************************************************************************************

// Code from github - See https://holwech.github.io/blog/Creating-a-simple-UDP-module
/*
func broadcast(send chan CommData, localIP string, port string) {
	log.Fatal("COMM: Broadcasting message to ", broadcast_addr, port)
	broadcastAddress, err := net.ResolveUDPAddr("udp", broadcast_addr+port)
	log.Fatal("ResolvingUDPAddr in Broadcast failed.", err)
	localAddress, err := net.ResolveUDPAddr("udp", GetLocalIP())
	connection, err := net.DialUDP("udp", localAddress, broadcastAddress)
	log.Fatal("DialUDP in Broadcast failed.", err)

	localhostAddress, err := net.ResolveUDPAddr("udp", "localhost"+port)
	log.Fatal("ResolvingUDPAddr in Broadcast localhost failed.", err)
	lConnection, err := net.DialUDP("udp", localAddress, localhostAddress)
	log.Fatal("DialUDP in Broadcast localhost failed.", err)
	defer connection.Close()

	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	for {
		message := <-send
		err := encoder.Encode(message)
		log.Fatal("Encode error in broadcast: ", err)
		_, err = connection.Write(buffer.Bytes())
		if err != nil {
			_, err = lConnection.Write(buffer.Bytes())
			log.Fatal("Write in broadcast localhost failed", err)
		}
		buffer.Reset()
	}
}

func listen(receive chan CommData, port string) {
	localAddress, err := net.ResolveUDPAddr("udp", port)
	if err != nil {
		log.Fatal(err, ":Cant resolve UDP Address ", port)
	}
	connection, err := net.ListenUDP("udp", localAddress)
	defer connection.Close()
	var message CommData

	for {
		inputBytes := make([]byte, 4096)
		length, _, err := connection.ReadFromUDP(inputBytes)
		if err != nil {
			log.Fatal(err, ":Cant ReadFromUDP")
		}
		buffer := bytes.NewBuffer(inputBytes[:length])
		decoder := gob.NewDecoder(buffer)
		err = decoder.Decode(&message)
		if message.Key == com_id {
			receive <- message
		}
	}
}

func Init(readPort string, writePort string) (<-chan frame.Frame, chan<- frame.Frame) {
	receive := make(chan frame.Frame, 10)
	send := make(chan Frame, 10)
	go listen(receive, readPort)
	go broadcast(send, writePort)
	return receive, send
}

func mcastOpen(bindAddr net.IP, port int, ifname string) (*ipv4.PacketConn, *net.UDPConn, error) {
	s, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
	if err != nil {
		log.Fatal(err)
	}
	if err := syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		log.Fatal(err)
	}
	//syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_REUSEPORT, 1)
	if err := syscall.SetsockoptString(s, syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, ifname); err != nil {
		log.Fatal(err)
	}

	lsa := syscall.SockaddrInet4{Port: port}
	copy(lsa.Addr[:], bindAddr.To4())

	if err := syscall.Bind(s, &lsa); err != nil {
		syscall.Close(s)
		log.Fatal(err)
	}
	f := os.NewFile(uintptr(s), "")
	c, err := net.FilePacketConn(f)
	f.Close()
	if err != nil {
		log.Fatal(err)
	}
	u := c.(*net.UDPConn)
	p := ipv4.NewPacketConn(c)

	return p, u, nil
}
*/
