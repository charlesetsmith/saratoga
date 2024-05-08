// Saratoga Interactive Initiator - Main

package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"syscall"

	"github.com/charlesetsmith/saratoga/beacon"
	"github.com/charlesetsmith/saratoga/data"
	"github.com/charlesetsmith/saratoga/metadata"
	"github.com/charlesetsmith/saratoga/request"
	"github.com/charlesetsmith/saratoga/sarflags"
	"github.com/charlesetsmith/saratoga/sarnet"
	"github.com/charlesetsmith/saratoga/sarwin" // Most of the cmd input and transfer logic is here
	"github.com/charlesetsmith/saratoga/status"
	"github.com/charlesetsmith/saratoga/timestamp"
	"github.com/charlesetsmith/saratoga/trans"

	"github.com/jroimartin/gocui"
)

/*
// Request Handler for the Responder
func reqrxhandler(g *gocui.Gui, r request.Request, remoteAddr *net.UDPAddr) string {
	// Handle the request

	reqtype := sarflags.GetStr(r.Header, "reqtype")
	switch reqtype {
	case "noaction", "get", "put", "getdelete", "delete", "getdir":
		// See if we already have the Transfer
		var t *sarwin.Transfer
		if t = sarwin.Lookup(sarwin.Responder, r.Session, remoteAddr.String()); t != nil {
			if t.Ttype != reqtype {
				sarwin.ErrPrintln(g, "red_black", "Request does not match Request Type ",
					sarnet.UDPinfo(remoteAddr),
					" session ", r.Session)
				return "badrequest"
			}
			return "success"
		}
		// No matching Transfer so add a new one
		var err error
		if err = sarwin.NewResponder(g, r, remoteAddr.String()); err == nil {
			sarwin.MsgPrintln(g, "yellow_black", "New Transfer ", reqtype, " from ",
				sarnet.UDPinfo(remoteAddr))
			return "success"
		}
		sarwin.ErrPrintln(g, "red_black", "Cannot create Transfer ", reqtype, " from ",
			sarnet.UDPinfo(remoteAddr),
			"session", r.Session, err)
		return "badrequest"
	default:
		sarwin.ErrPrintln(g, "red_black", "Invalid Request from ",
			sarnet.UDPinfo(remoteAddr),
			" session ", r.Session)
		return "badrequest"
	}
}
*/

/*
// Metadata handler for Responder
func metrxhandler(g *gocui.Gui, m metadata.MetaData, remoteAddr *net.UDPAddr) string {
	// Handle the metadata
	var t *sarwin.Transfer
	if t = sarwin.Lookup(sarwin.Responder, m.Session, remoteAddr.String()); t != nil {
		if err := t.Change(g, m); err != nil { // Size of file has changed!!!
			return "unspecified"
		}
		sarwin.MsgPrintln(g, "yellow_black", "Changed Transfer ", m.Session, " from ",
			sarnet.UDPinfo(remoteAddr))
		return "success"
	}
	// Request is currently in progress
	sarwin.ErrPrintln(g, "red_black", "Metadata received for no such transfer as ", m.Session, " from ",
		sarnet.UDPinfo(remoteAddr))
	return "badpacket"
}
*/

/*
// Data handler for responder
func datrxhandler(g *gocui.Gui, d data.Data, conn *net.UDPConn, remoteAddr *net.UDPAddr) string {
	// Handle the data
	var t *sarwin.Transfer
	if t = sarwin.Lookup(sarwin.Responder, d.Session, remoteAddr.String()); t != nil {
		// t.SData(g, d, conn, remoteAddr) // The data handler for the transfer
		sarwin.Trmu.Lock()
		if sarflags.GetStr(d.Header, "reqtstamp") == "yes" { // Grab the latest timestamp from data
			t.Tstamp = d.Tstamp
		}
		// Copy the data in this frame to the transfer buffer
		// THIS IS BAD WE HAVE AN int NOT A uint64!!!
		if (d.Offset)+(uint64)(len(d.Payload)) > (uint64)(len(t.Data)) {
			return "badoffset"
		}
		copy(t.Data[d.Offset:], d.Payload)
		if t.Dcount%100 == 0 || sarflags.GetStr(d.Header, "reqstatus") == "yes" || !t.Havemeta { // Send a status back
			stheader := "descriptor=" + sarflags.GetStr(d.Header, "descriptor") + // echo the descriptor
				",allholes=yes,reqholes=requested,errcode=success,"
			if !t.Havemeta {
				stheader += "metadatarecvd=no"
			} else {
				stheader += "metadatarecvd=yes"
			}
			// Send back a status to the client to tell it a success so far
			var st status.Status
			sinfo := status.Sinfo{Session: d.Session, Progress: d.Offset, Inrespto: d.Offset, Holes: nil}
			if st.New(stheader, &sinfo) != nil {
				sarwin.ErrPrintln(g, "red_black", "Cannot create badpacket status")
				return "badstatus"
			}
			// Send it out the connection to the peer
			if err := st.Send(conn, remoteAddr); err != nil {
				sarwin.ErrPrintln(g, "red_black", err.Error())
				return "badstatus"
			}
		}
		t.Dcount++
		sarwin.MsgPrintln(g, "yellow_black", "Responder Received Data Len:", len(d.Payload), " Pos:", d.Offset)
		sarwin.Trmu.Unlock()
		sarwin.MsgPrintln(g, "yellow_black", "Changed Transfer ", d.Session, " from ",
			sarnet.UDPinfo(remoteAddr))
		return "success"
	}
	// No transfer is currently in progress
	sarwin.ErrPrintln(g, "red_black", "Data received for no such transfer as ", d.Session, " from ",
		sarnet.UDPinfo(remoteAddr))
	return "badpacket"
}
*/

/*
// Status handler for responder - LOTS OF CODE TO WRWRITE HERE!!!!
func starxhandler(g *gocui.Gui, s status.Status, conn *net.UDPConn, remoteAddr *net.UDPAddr) string {
	var t *sarwin.Transfer
	if conn == nil || g == nil { // just a placeholder so Xcode does not complain
		return "badstatus"
	}
	if t = sarwin.Lookup(sarwin.Initiator, s.Session, remoteAddr.String()); t == nil { // No existing transfer
		return "badstatus"
	}

	// Update transfers Inrespto & Progress indicators
	sarwin.Trmu.Lock()
	defer sarwin.Trmu.Unlock()
	t.Inrespto = s.Inrespto
	t.Progress = s.Progress
	// Resend the data requested by the Holes

	return "success"
}
*/

// Listen -- Go routine for recieving IPv4 & IPv6 for incoming frames and shunt them off to the
// correct frame handlers
// incoming frames are via rx channel interface,
// outgoing frames are to the tx channel interface -- These are status frames on a read error
func listen(g *gocui.Gui, conn *net.UDPConn, rx chan interface{}, tx chan interface{}, quit chan error) {

	// Investigate using conn.PathMTU()
	maxframesize := sarflags.Mtu() - 60   // Handles IPv4 & IPv6 header
	buf := make([]byte, maxframesize+100) // Just in case...
	framelen := 0
	err := error(nil)
	// Create some space to hold the remote address in
	var remoteAddr *net.UDPAddr = new(net.UDPAddr)

	// THE MAIN LOOP HERE
	for err == nil { // Loop forever grabbing frames
		// Actually Read the frame into buf
		framelen, remoteAddr, err = conn.ReadFromUDP(buf)
		if err != nil {
			// We failed to read.. That is pretty bad, so lets tell caller we have failed
			sarwin.ErrPrintln(g, "red_black", "Sarataga listener failed:", err)
			quit <- err
			return
		}

		// sarwin.MsgPrintln(g, "green_black", "Listen read ", framelen, " bytes from ", remoteAddr.String())

		// Very basic frame checks before we get into what it is
		if framelen < 8 {
			// Saratoga packet too small
			sarwin.ErrPrintln(g, "red_black", "Rx Saratoga Frame too short from ",
				sarnet.UDPinfo(remoteAddr))
			// drop it and just continue reading
			continue
		}
		if framelen > maxframesize {
			sarwin.ErrPrintln(g, "red_black", "Rx Saratoga Frame too long ", framelen,
				" from ", sarnet.UDPinfo(remoteAddr))
			// drop it and just continue reading
			continue
		}

		// OK so we might have a valid frame so copy it to to the frame byte slice
		framebuf := make([]byte, framelen)
		copy(framebuf, buf[:framelen])

		// Grab the Saratoga Header which is the first 32 bits
		header := binary.BigEndian.Uint32(framebuf[:4])
		// Check the version
		if sarflags.GetStr(header, "version") != "v1" { // Make sure we are Version 1
			// If a bad version received then send back a Status errcode to the initiator
			var st status.Status
			sinfo := status.Sinfo{Session: 0, Progress: 0, Inrespto: 0, Holes: nil}
			sarwin.ErrPrintln(g, "red_black", "Not Saratoga Version 1 Frame from ",
				sarnet.UDPinfo(remoteAddr))
			if st.New("errcode=badpacket", &sinfo) != nil {
				sarwin.ErrPrintln(g, "red_black", "Cannot create badpacket status")
				continue // drop it
			}
			tx <- st.Val(remoteAddr)
			continue
		}

		// Process the received frame
		switch sarflags.GetStr(header, "frametype") {
		case "beacon":
			// We have received a beacon on the connection
			rxb := new(beacon.Beacon)
			if rxerr := rxb.Decode(framebuf); rxerr != nil {
				// We just drop bad beacons
				sarwin.ErrPrintln(g, "red_black", "Bad Beacon:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr))
				// Ignore and throw it away, we dont send status frames for beacons
				continue
			}
			rx <- rxb.Val(remoteAddr)
			continue

		case "request":
			// Handle incoming request
			var req request.Request
			var rxerr error
			if rxerr = req.Decode(framebuf); rxerr != nil {
				// We just drop bad requests
				sarwin.ErrPrintln(g, "red_black", "Bad Request:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr))
				continue
			}
			rx <- req.Val(remoteAddr)
			continue
			// sarwin.PacketPrintln(g, "green_black", "Listen Rx ", req.ShortPrint())
			// sarwin.MsgPrintln(g, "white_black", "Decoded request ", req.Print())
			// Pass it up to the rx channel for further handling
			// Move this out of here
			// errcode := reqrxhandler(g, req, remoteAddr) // process the request
			// Create the status header with the errcode to send back
			// stheader := "descriptor=" + sarflags.GetStr(header, "descriptor") +
			// 	",metadatarecvd=no,allholes=yes,reqholes=requested,errcode=" + errcode
			// if errcode != "success" {
			// 	var st status.Status
			// 	sinfo := status.Sinfo{Session: req.Session, Progress: 0, Inrespto: 0, Holes: nil}
			// 	if st.New(stheader, &sinfo) != nil {
			// 		sarwin.ErrPrintln(g, "red_black", "Cannot create badpacket status")
			// 		continue
			// 	}
			// Send it out the connecton
			// Move the send out of here!!!
			// 	if err := st.Send(conn, remoteAddr); err != nil {
			// 		sarwin.ErrPrintln(g, "red_black", err.Error())
			// 	}
			// Just send an error status back, dont create a transfer
			// 	sarwin.ErrPrintln(g, "red_black", "Bad Status:", rxerr, " from ",
			// 		sarnet.UDPinfo(remoteAddr), " session ", req.Session)
			// 		tx <- st.Val(remoteAddr)
			//	continue
			//  }
			// Send it out the connection
			// move the Send and Lookup out of here!!!
			// Lookup or create the Transfer and Acknowledge the request to the Initiator
			// var t *sarwin.Transfer
			// if t = sarwin.Lookup(sarwin.Responder, req.Session, remoteAddr.String()); t != nil {
			// 	// Write off the status to the Initiator
			// 	t.WriteStatus(g, stheader)

		case "data":
			// Handle incoming data
			var d data.Data
			var rxerr error
			sarwin.MsgPrintln(g, "white_black", "Data frame length is:", len(framebuf))
			if rxerr = d.Decode(framebuf); rxerr != nil {
				// Bad Packet send back a Status to the sender with current offset and session
				var st status.Status
				stheader := "descriptor=" + sarflags.GetStr(header, "descriptor") +
					",metadatarecvd=no,allholes=yes,reqholes=requested,errcode=badpacket"
				sinfo := status.Sinfo{Session: d.Session, Progress: d.Offset, Inrespto: 0, Holes: nil}
				if st.New(stheader, &sinfo) != nil {
					sarwin.ErrPrintln(g, "red_black", "Cannot create badpacket status")
					continue
				}
				sarwin.ErrPrintln(g, "red_black", "Bad Data:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", d.Session)
				// Send a badpacket status back to the sender
				tx <- st.Val(remoteAddr)
				continue
			}
			rx <- d.Val(remoteAddr)
			continue
			// Move this out of here
			// Ok we have received a good data frame, process it
			// sarwin.PacketPrintln(g, "green_black", "Listen Rx ", d.ShortPrint())
			// sarwin.MsgPrintln(g, "white_black", "Decoded data ", d.Print())
			// Pass it up to the rx channel for further handling
			// errcode := datrxhandler(g, d, conn, remoteAddr) // process the data
			// if errcode != "success" {                       // If we have a error send back a status with it
			// 	stheader := "descriptor=" + sarflags.GetStr(header, "descriptor") + // echo the descriptor
			// 		",metadatarecvd=no,allholes=yes,reqholes=requested," +
			// 		"errcode=" + errcode
			// 	// Send back a status to the client to tell it the error or that we have a success with creating the transfer
			// 	var st status.Status
			// 	sinfo := status.Sinfo{Session: d.Session, Progress: d.Offset, Inrespto: 0, Holes: nil}
			// 	if st.New(stheader, &sinfo) != nil {
			// 		sarwin.ErrPrintln(g, "red_black", "Cannot asemble status")
			// 	}
			// 	if se := st.Send(conn, remoteAddr); se != nil {
			// 		sarwin.ErrPrintln(g, "red_black", se.Error())
			// 	}

		case "metadata":
			// Handle incoming metadata
			var m metadata.MetaData
			var rxerr error
			if rxerr = m.Decode(framebuf); rxerr != nil {
				session := binary.BigEndian.Uint32(framebuf[4:8])
				var st status.Status
				// Bad Packet send back a Status to the client
				stheader := "descriptor=" + sarflags.GetStr(header, "descriptor") +
					",metadatarecvd=no,allholes=yes,reqholes=requested,errcode=badpacket"
				sinfo := status.Sinfo{Session: session, Progress: 0, Inrespto: 0, Holes: nil}
				if st.New(stheader, &sinfo) != nil {
					sarwin.ErrPrintln(g, "red_black", "Cannot assemble status")
					continue
				}
				sarwin.ErrPrintln(g, "red_black", "Bad MetaData:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", session)
				// Pass it up to the tx channel for further handling
				tx <- st.Val(remoteAddr)
				continue
			}
			rx <- m.Val(remoteAddr)
			continue

			// Move this out of here
			// sarwin.PacketPrintln(g, "green_black", "Listen Rx ", m.ShortPrint())
			// sarwin.MsgPrintln(g, "white_black", "Decoded metadata ", m.Print())
			// Pass it up to the rx channel for further handling
			// errcode := metrxhandler(g, m, remoteAddr) // process the metadata
			// if errcode != "success" {                 // If we have a error send back a status with it
			// 	var st status.Status
			// 	stheader := "descriptor=" + sarflags.GetStr(header, "descriptor") +
			// 		",metadatarecvd=no,allholes=yes,reqholes=requested,errcode=" + errcode
			// 	sinfo := status.Sinfo{Session: m.Session, Progress: 0, Inrespto: 0, Holes: nil}
			// 	if st.New(stheader, &sinfo) != nil {
			// 		sarwin.ErrPrintln(g, "red_black", "Cannot assemble status")
			// 	}
			// 	if se := st.Send(conn, remoteAddr); se != nil {
			// 		sarwin.ErrPrintln(g, "red_black", se.Error())
			// 	}
			// 	sarwin.ErrPrintln(g, "red_black", "Bad Metadata:", rxerr, " from ",
			// 		sarnet.UDPinfo(remoteAddr), " session ", m.Session)
			// 	tx <- st.Val(remoteAddr)
			// }

		case "status":
			// Handle incoming status
			var s status.Status
			if rxerr := s.Decode(framebuf); rxerr != nil {
				session := binary.BigEndian.Uint32(framebuf[4:8])
				var st status.Status
				// Bad Packet send back a Status to the sender
				stheader := "descriptor=" + sarflags.GetStr(header, "descriptor") +
					",metadatarecvd=no,allholes=yes,reqholes=requested,errcode=badpacket"
				sinfo := status.Sinfo{Session: session, Progress: 0, Inrespto: 0, Holes: nil}
				if st.New(stheader, &sinfo) != nil {
					sarwin.ErrPrintln(g, "red_black", "Cannot assemble status")
					continue
				}
				sarwin.ErrPrintln(g, "red_black", "Bad Status:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", session)
				// Pass it up to the tx channel for further handling
				tx <- st.Val(remoteAddr)
				continue
			} // Handle the status
			sarwin.PacketPrintln(g, "green_black", "Listen Rx ", s.ShortPrint())
			sarwin.MsgPrintln(g, "white_black", "Decoded status ", s.Print())
			// Pass it up to the rx channel for further handling
			rx <- s.Val(remoteAddr)
			continue

			// Move rest of this off out of here
			// errcode := starxhandler(g, s, conn, remoteAddr) // process the status
			// if errcode != "success" {                       // If we have a error send back a status with it
			// 	var st status.Status
			// 	stheader := "descriptor=" + sarflags.GetStr(header, "descriptor") + // echo the descriptor
			// 		",metadatarecvd=no,allholes=yes,reqholes=requested,errcode=" + errcode
			// 	sinfo := status.Sinfo{Session: s.Session, Progress: s.Progress, Inrespto: s.Inrespto, Holes: nil}
			// 	if st.New(stheader, &sinfo) != nil {
			// 		sarwin.ErrPrintln(g, "red_black", "Cannot assemble status")
			// 	}
			// 	if se := st.Send(conn, remoteAddr); se != nil {
			// 		sarwin.ErrPrintln(g, "red_black", se.Error())
			// 	}
			// 	sarwin.ErrPrintln(g, "red_black", "Bad Status:", errcode, " from ",
			// 		sarnet.UDPinfo(remoteAddr), " session ", s.Session)
			// 	tx <- st.Val(remoteAddr)
			// }

		default:
			// Bad Packet drop it
			sarwin.ErrPrintln(g, "red_black", "Listen Invalid Saratoga Frame Recieved from ",
				sarnet.UDPinfo(remoteAddr))
		}
	}
}

// Peer - beacon peer
type Peer struct {
	Addr      string              // The Peer IP Address. is format net.UDPAddr.IP.String()
	Freespace uint64              // 0 if freespace not advertised
	Eid       string              // Exactly who sent this and from what PID
	Maxdesc   string              // The maximum descriptor size of the peer
	Created   timestamp.Timestamp // When was this Peer created
	Updated   timestamp.Timestamp // When was this Peer last updated
}

var Cmdptr *sarflags.Cliflags

// Main
func main() {

	if len(os.Args) != 3 {
		fmt.Println("usage:saratoga <config> <iface>")
		fmt.Println("e.g.: go run saratoga.go saratoga.json en0 (Interface says where to listen for multicast joins")
		return
	}

	argnumb := 1

	// The Command line interface commands, help & usage to be read from saratoga.json
	Cmdptr := new(sarflags.Cliflags)
	sarflags.Cliflag = Cmdptr

	var err error
	// Read in JSON config file and parse it into the Config structure.
	if err = Cmdptr.ReadConfig(os.Args[argnumb]); err != nil {
		fmt.Println("Cannot open saratoga config file we have a Readconf error ", os.Args[argnumb], " ", err)
		return
	}

	// Grab my process ID
	// Pid := os.Getpid()

	// Set global variable pointing to sarflags.Cliflags structure
	// We have to have a global pointer as we cannot pass c directly into gocui

	sarwin.Cinfo.Prompt = Cmdptr.Prompt
	sarwin.Cinfo.Ppad = Cmdptr.Ppad

	// Move to saratoga working directory
	if err = os.Chdir(Cmdptr.Sardir); err != nil {
		log.Fatal(fmt.Errorf("no such directory SARDIR=%s", Cmdptr.Sardir))
	}

	var fs syscall.Statfs_t
	if err = syscall.Statfs(Cmdptr.Sardir, &fs); err != nil {
		log.Fatal(errors.New("cannot stat saratoga working directory"))
	}

	// What Interface are we receiving Multicasts on
	var iface *net.Interface

	argnumb++
	iface, err = net.InterfaceByName(os.Args[argnumb])
	if err != nil {
		fmt.Println("Saratoga Unable to lookup interfacebyname:", os.Args[argnumb-1])
		log.Fatal(err)
	}
	// Set the Mtu to Interface we are using
	sarflags.MtuSet(iface.MTU)

	// Set up the gocui interface and start the mainloop
	g, gerr := gocui.NewGui(gocui.OutputNormal)
	if gerr != nil {
		fmt.Printf("Cannot run gocui user interface")
		log.Fatal(err)
	}
	defer g.Close()

	// Get Host Interfaces & Address's
	ifis, ierr := net.Interfaces()
	if ierr != nil {
		fmt.Println(ierr.Error())
		log.Fatal(ierr)
	}
	g.SetManagerFunc(sarwin.Layout)
	if err := sarwin.Keybindings(g); err != nil {
		log.Panicln(err)
	}

	// Show Host Interfaces & Address's
	for _, ifi := range ifis {
		if ifi.Name == os.Args[argnumb] {
			sarwin.MsgPrintln(g, "green_black", ifi.Name, " MTU ", ifi.MTU, " ", ifi.Flags.String(), ":")
			adrs, _ := ifi.Addrs()
			for _, adr := range adrs {
				if strings.Contains(adr.Network(), "ip") {
					sarwin.MsgPrintln(g, "green_black", "\t Unicast ", adr.String(), " ", adr.Network())
				}
			}
			//
			madrs, _ := ifi.MulticastAddrs()
			for _, madr := range madrs {
				if strings.Contains(madr.Network(), "ip") {
					sarwin.MsgPrintln(g, "green_black", "Multicast ", madr.String(),
						" Net:", madr.Network())
				}
			}
			//
		}
	}
	// Open up V6 sockets for listening on the Saratoga Port
	v6mcastaddr := net.UDPAddr{
		Port: Cmdptr.Port,
		IP:   net.ParseIP(Cmdptr.V6Multicast),
	}
	// Set up for Listen to Multicast v6
	v6mcastcon, err := net.ListenMulticastUDP("udp6", iface, &v6mcastaddr)
	if err != nil {
		log.Println("Saratoga Unable to Listen on IPv6 Multicast ", v6mcastaddr.IP, " ", v6mcastaddr.Port)
		log.Fatal(err)
	}
	if err := sarnet.SetMulticastLoop(v6mcastcon); err != nil {
		log.Fatal(err)
	}

	// Open up V4 sockets for listening on the Saratoga Port
	v4mcastaddr := net.UDPAddr{
		Port: Cmdptr.Port,
		IP:   net.ParseIP(Cmdptr.V4Multicast),
	}

	// Set up for Listen to Multicast v4
	v4mcastcon, err := net.ListenMulticastUDP("udp4", iface, &v4mcastaddr)
	if err != nil {
		log.Println("Saratoga Unable to Listen on IPv4 Multicast ", v4mcastaddr.IP, " ", v4mcastaddr.Port)
		log.Fatal(err)
	}
	if err := sarnet.SetMulticastLoop(v4mcastcon); err != nil {
		log.Fatal(err)
	}

	// v4unicastcon, err := net.ListenUDP("udp4", iface, &v4addr)
	sarwin.MsgPrintf(g, "green_black", "Saratoga Directory is %s\n", Cmdptr.Sardir)
	sarwin.MsgPrintf(g, "green_black", "Available space is %d MB\n",
		(uint64(fs.Bsize)*fs.Bavail)/1024/1024)

	// Lets see what our integer sizes are on this system
	sarwin.MsgPrintln(g, "green_black", "MaxInt=", sarflags.MaxInt)
	sarwin.MsgPrintln(g, "green_black", "MaxUint=", sarflags.MaxUint)
	sarwin.MsgPrintln(g, "green_black", "MaxInt16=", sarflags.MaxInt16)
	sarwin.MsgPrintln(g, "green_black", "MaxUint16=", sarflags.MaxUint16)
	sarwin.MsgPrintln(g, "green_black", "MaxInt32=", sarflags.MaxInt32)
	sarwin.MsgPrintln(g, "green_black", "MaxUint32=", sarflags.MaxUint32)
	sarwin.MsgPrintln(g, "green_black", "MaxInt64=", sarflags.MaxInt64)
	sarwin.MsgPrintln(g, "green_black", "MaxUint64=", sarflags.MaxUint64)

	sarwin.MsgPrintln(g, "green_black", "Maximum Descriptor is:", sarflags.MaxDescriptor)

	sarwin.MsgPrintln(g, "white_black", "^P - Toggle Packet View")
	sarwin.MsgPrintln(g, "white_black", "^Space - Rotate/Change View")

	// Listen for incoming v6 frames
	v6listenquit := make(chan error)    // When will we return from listening for v6 frames
	rxv6frame := make(chan interface{}) // We receive v6 decoded frames on this channel
	txv6frame := make(chan interface{}) // We transmit v6 encoded frames on this channel
	go listen(g, v6mcastcon, rxv6frame, txv6frame, v6listenquit)
	sarwin.MsgPrintln(g, "green_black", "Saratoga IPv6 Multicast Listener started on ",
		sarnet.UDPinfo(&v6mcastaddr))

	// Listen for incoming v4 frames
	v4listenquit := make(chan error)    // When will we return from listening for v4 frames
	rxv4frame := make(chan interface{}) // Wee receive decoded v4 frames on this channel
	txv4frame := make(chan interface{}) // We transmit encoded v4 frames on this channel
	go listen(g, v4mcastcon, rxv4frame, txv4frame, v4listenquit)
	sarwin.MsgPrintln(g, "green_black", "Saratoga IPv4 Multicast Listener started on ",
		sarnet.UDPinfo(&v4mcastaddr))

	// The Base calling functions for Saratoga live in cli.go so look there first!
	errflag := make(chan error, 1)
	go gocuimainloop(g, errflag)

	// This is the major handler of incoming frames. First decode what type they are
	// and where they came from then handle them
	// sarwin.MsgPrintln(g, "red_black", "OK We got to the main select loop")
	for {
		select {
		case rxv4 := <-rxv4frame:
			switch rxv4.(type) {
			case beacon.Packet:
				// sarwin.MsgPrintln(g, "white_black", "Received v4 Saratoga BEACON Frame")
				pkt := rxv4.(beacon.Packet)
				sarwin.PacketPrintln(g, "white_black", "Rx", pkt.Info.ShortPrint())
				// Add a new or update an existing peer information from the beacon
				if beacon.NewPeer(&pkt.Info, &pkt.Addr) {
					sarwin.MsgPrintln(g, "yellow_black", "Added New Peer ", pkt.Addr.String())
				}
			case data.Packet:
				sarwin.MsgPrintln(g, "white_black", "Received v4 Saratoga DATA Frame")
				pkt := rxv4.(data.Packet)
				sarwin.PacketPrintln(g, "white_black", "Rx", pkt.Info.ShortPrint())
			case metadata.Packet:
				sarwin.MsgPrintln(g, "white_black", "Received v4 Saratoga METADATA Frame")
				pkt := rxv4.(metadata.Packet)
				sarwin.PacketPrintln(g, "white_black", "Rx", pkt.Info.ShortPrint())
			case request.Packet:
				sarwin.MsgPrintln(g, "white_black", "Received v4 Saratoga REQUEST Frame")
				pkt := rxv4.(request.Packet)
				sarwin.PacketPrintln(g, "white_black", "Rx", pkt.Info.ShortPrint())
				// We have received a request to send or receive a file or dir
				if trans.AddRx(g, &pkt.Info, &pkt.Addr) {
					sarwin.MsgPrintln(g, "yellow_black", "New transfer request from ", pkt.Addr.String())
				}
			case status.Packet:
				sarwin.MsgPrintln(g, "white_black", "Received v4 Saratoga STATUS Frame")
				pkt := rxv4.(status.Packet)
				sarwin.PacketPrintln(g, "white_black", "Rx", pkt.Info.ShortPrint())
			default:
				sarwin.ErrPrintln(g, "white_black", "Received v4 Saratoga INVALID Frame")
			}
		case rxv6 := <-rxv6frame:
			switch rxv6.(type) {
			case beacon.Packet:
				// sarwin.MsgPrintln(g, "white_black", "Received v6 Saratoga BEACON Frame")
				pkt := rxv6.(beacon.Packet)
				sarwin.PacketPrintln(g, "white_black", "Rx", pkt.Info.ShortPrint())
				if beacon.NewPeer(&pkt.Info, &pkt.Addr) {
					sarwin.MsgPrintln(g, "yellow_black", "Added New Peer ", pkt.Addr.String())
				}
			case data.Packet:
				sarwin.MsgPrintln(g, "white_black", "Received v6 Saratoga DATA Frame")
				pkt := rxv6.(data.Packet)
				sarwin.PacketPrintln(g, "white_black", "Rx", pkt.Info.ShortPrint())
			case metadata.Packet:
				sarwin.MsgPrintln(g, "white_black", "Received v6 Saratoga METADATA Frame")
				pkt := rxv6.(metadata.Packet)
				sarwin.PacketPrintln(g, "white_black", "Rx", pkt.Info.ShortPrint())
			case request.Packet:
				sarwin.MsgPrintln(g, "white_black", "Received v6 Saratoga REQUEST Frame")
				pkt := rxv6.(request.Packet)
				sarwin.PacketPrintln(g, "white_black", "Rx", pkt.Info.ShortPrint())
			case status.Packet:
				sarwin.MsgPrintln(g, "white_black", "Received v6 Saratoga STATUS Frame")
				pkt := rxv6.(status.Packet)
				sarwin.PacketPrintln(g, "white_black", "Rx", pkt.Info.ShortPrint())
			default:
				sarwin.ErrPrintln(g, "white_black", "Received v6 Saratoga INVALID Frame")
			}
		case txv4 := <-txv4frame:
			switch txv4.(type) {
			case beacon.Packet:
				sarwin.MsgPrintln(g, "blue_black", "Need to send v4 Saratoga BEACON Frame")
				pkt := txv4.(beacon.Packet)
				sarwin.MsgPrintln(g, "blue_black", pkt.Info.ShortPrint())
			case data.Packet:
				sarwin.MsgPrintln(g, "blue_black", "Need to send v4 Saratoga DATA Frame")
				pkt := txv4.(data.Packet)
				sarwin.MsgPrintln(g, "blue_black", pkt.Info.ShortPrint())
			case metadata.Packet:
				sarwin.MsgPrintln(g, "blue_black", "Need to send v4 Saratoga METADATA Frame")
				pkt := txv4.(metadata.Packet)
				sarwin.MsgPrintln(g, "blue_black", pkt.Info.ShortPrint())
			case request.Packet:
				sarwin.MsgPrintln(g, "blue_black", "Need to send v4 Saratoga REQUEST Frame")
				pkt := txv4.(request.Packet)
				sarwin.MsgPrintln(g, "blue_black", pkt.Info.ShortPrint())
			case status.Packet:
				sarwin.MsgPrintln(g, "blue_black", "Need to send v4 Saratoga STATUS Frame")
				pkt := txv4.(status.Packet)
				sarwin.MsgPrintln(g, "blue_black", pkt.Info.ShortPrint())
			default:
				sarwin.ErrPrintln(g, "red_black", "Need to send v4 Saratoga INVALID Frame")
			}
		case txv6 := <-txv6frame:
			switch txv6.(type) {
			case beacon.Packet:
				sarwin.MsgPrintln(g, "blue_black", "Need to send v6 Saratoga BEACON Frame")
				pkt := txv6.(beacon.Packet)
				sarwin.MsgPrintln(g, "blue_black", pkt.Info.ShortPrint())
			case data.Packet:
				sarwin.MsgPrintln(g, "blue_black", "Need to send v6 Saratoga DATA Frame")
				pkt := txv6.(data.Packet)
				sarwin.MsgPrintln(g, "blue_black", pkt.Info.ShortPrint())
			case metadata.Packet:
				sarwin.MsgPrintln(g, "blue_black", "Need to send v6 Saratoga METADATA Frame")
				pkt := txv6.(metadata.Packet)
				sarwin.MsgPrintln(g, "blue_black", pkt.Info.ShortPrint())
			case request.Packet:
				sarwin.MsgPrintln(g, "blue_black", "Need to send v6 Saratoga REQUEST Frame")
				pkt := txv6.(request.Packet)
				sarwin.MsgPrintln(g, "blue_black", pkt.Info.ShortPrint())
			case status.Packet:
				sarwin.MsgPrintln(g, "blue_black", "Need to send v6 Saratoga STATUS Frame")
				pkt := txv6.(status.Packet)
				sarwin.MsgPrintln(g, "blue_black", pkt.Info.ShortPrint())
			default:
				sarwin.ErrPrintln(g, "blue_black", "Need to send v6 Saratoga INVALID Frame")
			}
		case v4err := <-v4listenquit:
			log.Fatal("Saratoga v4 listener has quit with error:", v4err)
		case v6err := <-v6listenquit:
			log.Fatal("Saratoga v6 listener has quit with error:", v6err)
		case err := <-errflag:
			if err != nil {
				log.Fatal("Mainloop has quit with error:", err.Error())
			}
			log.Fatal("Saratoga has quit")
		}
	}
}

// Go routine for command line loop
func gocuimainloop(g *gocui.Gui, done chan error) {
	err := g.MainLoop()
	// fmt.Println(err.Error())
	done <- err
}
