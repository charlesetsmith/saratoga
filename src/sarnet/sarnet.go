package sarnet

import (
	"net"
)

// SaratogaPort - IANA allocated Saratoga UDP & TCP Port #'s
const saratogaport = 7542

// IPv4Multicast -- IANA allocated Saratoga IPV4 all-hosts Multicast Address
const IPv4Multicast = "224.0.0.108"

// IPv6Multicast -- IANA allocated Saratoga IPV6 link-local Multicast Address
const IPv6Multicast = "FF02:0:0:0:0:0:0:6c"

// Port -- Lookup saratoga official port number in /etc/services
func Port() int {

	var port int
	var err error

	if port, err = net.LookupPort("udp", "saratoga"); err != nil {
		port = saratogaport // Set to default if not in /etc/services
	}
	return port
}
