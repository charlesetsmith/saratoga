// Saratoga Interactive Client - Main

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

	"github.com/jroimartin/gocui"
)

// Request Handler for Server
func reqrxhandler(g *gocui.Gui, r Request, remoteAddr *net.UDPAddr) string {
	// Handle the request

	reqtype := FlagGetStr(r.Header, "reqtype")
	switch reqtype {
	case "noaction", "get", "put", "getdelete", "delete", "getdir":
		var t *Transfer
		if t = TranLookup(r.Session, remoteAddr.IP.String()); t == nil {
			// No matching request so add a new transfer
			var err error
			a := remoteAddr.String()
			if _, err = NewServer(g, r, a); err == nil {
				MsgPrintln(g, "yellow_black", "Created Transfer ", reqtype, " from ",
					UDPinfo(remoteAddr))
				return "success"
			}
			ErrPrintln(g, "red_black", "Cannot create Transfer ", reqtype, " from ",
				UDPinfo(remoteAddr),
				"session", r.Session, err)
			return "badrequest"
		}
		// Request is currently in progress
		ErrPrintln(g, "red_black", "Request ", reqtype, " from ",
			UDPinfo(remoteAddr),
			" for session ", r.Session, " already in progress")
		return "badrequest"
	default:
		ErrPrintln(g, "red_black", "Invalid Request from ",
			UDPinfo(remoteAddr),
			" session ", r.Session)
		return "badrequest"
	}
}

// Metadata handler for Server
func metrxhandler(g *gocui.Gui, m MetaData, remoteAddr *net.UDPAddr) string {
	// Handle the metadata
	var t *Transfer
	if t = TranLookup(m.Session, remoteAddr.IP.String()); t != nil {
		if err := t.TranChange(g, m); err != nil { // Size of file has changed!!!
			return "unspecified"
		}
		MsgPrintln(g, "yellow_black", "Changed Transfer ", m.Session, " from ",
			UDPinfo(remoteAddr))
		return "success"
	}
	// Request is currently in progress
	ErrPrintln(g, "red_black", "Metadata received for no such transfer as ", m.Session, " from ",
		UDPinfo(remoteAddr))
	return "badpacket"
}

// Data handler for server
func datrxhandler(g *gocui.Gui, d Data, conn *net.UDPConn, remoteAddr *net.UDPAddr) string {
	// Handle the data
	var t *Transfer
	if t = TranLookup(d.Session, remoteAddr.IP.String()); t != nil {
		// t.SData(g, d, conn, remoteAddr) // The data handler for the transfer
		Trmu.Lock()
		if FlagGetStr(d.Header, "reqtstamp") == "yes" { // Grab the timestamp from data
			t.tstamptype = TimeGetStr(d.Tstamp.header)
		}
		// Copy the data in this frame to the transfer buffer
		// THIS IS BAD WE HAVE AN int NOT A uint64!!!
		if (d.Offset)+(uint64)(len(d.Payload)) > (uint64)(len(t.data)) {
			return "badoffset"
		}
		copy(t.data[d.Offset:], d.Payload)
		t.framecount++
		if t.framecount%100 == 0 || FlagGetStr(d.Header, "reqstatus") == "yes" || !t.havemeta { // Send a status back
			stheader := "descriptor=" + FlagGetStr(d.Header, "descriptor") // echo the descriptor
			stheader += ",allholes=yes,reqholes=requested,errcode=success,"
			if !t.havemeta {
				stheader += "metadatarecvd=no"
			} else {
				stheader += "metadatarecvd=yes"
			}
			// Send back a status to the client to tell it a success with creating the transfer
			t.WriteStatus(g, stheader)
		}
		MsgPrintln(g, "yellow_black", "Server Received Data Len:", len(d.Payload), " Pos:", d.Offset)
		Strmu.Unlock()
		MsgPrintln(g, "yellow_black", "Changed Transfer ", d.Session, " from ",
			UDPinfo(remoteAddr))
		return "success"
	}
	// No transfer is currently in progress
	ErrPrintln(g, "red_black", "Data received for no such transfer as ", d.Session, " from ",
		UDPinfo(remoteAddr))
	return "badpacket"
}

// Status handler for server
func starxhandler(g *gocui.Gui, s Status, conn *net.UDPConn, remoteAddr *net.UDPAddr) string {
	var t *STransfer
	if t = SMatch(remoteAddr.IP.String(), s.Session); t == nil { // No existing transfer
		return "badstatus"
	}
	// Update transfers Inrespto & Progress indicators
	Strmu.Lock()
	defer Strmu.Unlock()
	t.Inrespto = s.Inrespto
	t.Progress = s.Progress
	// Resend the data requested by the Holes

	return "success"
}

// Listen -- Go routine for recieving IPv4 & IPv6 for an incoming frames and shunt them off to the
// correct frame handlers
func listen(g *gocui.Gui, conn *net.UDPConn, quit chan error) {

	// Investigate using conn.PathMTU()
	maxframesize := Mtu() - 60            // Handles IPv4 & IPv6 header
	buf := make([]byte, maxframesize+100) // Just in case...
	framelen := 0
	err := error(nil)
	var remoteAddr *net.UDPAddr = new(net.UDPAddr)
	for err == nil { // Loop forever grabbing frames
		// Read into buf
		framelen, remoteAddr, err = conn.ReadFromUDP(buf)
		if err != nil {
			ErrPrintln(g, "red_black", "Sarataga listener failed:", err)
			quit <- err
			return
		}
		MsgPrintln(g, "green_black", "Listen read ", framelen, " bytes from ", remoteAddr.String())

		// Very basic frame checks before we get into what it is
		if framelen < 8 {
			// Saratoga packet too small
			ErrPrintln(g, "red_black", "Rx Saratoga Frame too short from ",
				UDPinfo(remoteAddr))
			continue
		}
		if framelen > maxframesize {
			ErrPrintln(g, "red_black", "Rx Saratoga Frame too long ", framelen,
				" from ", UDPinfo(remoteAddr))
			continue
		}

		// OK so we might have a valid frame so copy it to to the frame byte slice
		frame := make([]byte, framelen)
		copy(frame, buf[:framelen])
		// fmt.Println("We have a frame of length:", framelen)
		// Grab the Saratoga Header
		header := binary.BigEndian.Uint32(frame[:4])
		if FlagGetStr(header, "version") != "v1" { // Make sure we are Version 1
			ErrPrintln(g, "red_black", "Header is not Saratoga v1")
			if _, err = FlagSet(0, "errno", "badpacket"); err != nil {
				// Bad Packet send back a Status to the client
				var st Status
				sinfo := Sinfo{Session: 0, Progress: 0, Inrespto: 0, Holes: nil}
				if NewFrame(&st, "errcode=badpacket", &sinfo) != nil {
					ErrPrintln(g, "red_black", "Cannot create badpacket status")
				}
				ErrPrintln(g, "red_black", "Not Saratoga Version 1 Frame from ",
					UDPinfo(remoteAddr))
			}
			continue
		}
		MsgPrintln(g, "white_black", "Received: ", FlagGetStr(header, "frametype"))

		// Process the frame
		switch FlagGetStr(header, "frametype") {
		case "beacon":
			var rxb Beacon
			if rxerr := FrameDecode(&rxb, frame); rxerr != nil {
				// if rxerr := rxb.Decode(frame); rxerr != nil {
				// We just drop bad beacons
				ErrPrintln(g, "red_black", "Bad Beacon:", rxerr, " from ",
					UDPinfo(remoteAddr))
				continue
			}
			// Handle the beacon
			if errcode := RxHandler(&rxb, g, conn); errcode != "success" {
				ErrPrintln(g, "red_black", "Bad Beacon:", errcode, "  from ",
					UDPinfo(remoteAddr))
			}
			PacketPrintln(g, "green_black", "Listen Rx ", rxb.ShortPrint())

		case "request":
			// Handle incoming request
			var r Request
			var rxerr error
			if rxerr = FrameDecode(&r, frame); rxerr != nil {
				// We just drop bad requests
				ErrPrintln(g, "red_black", "Bad Request:", rxerr, " from ",
					UDPinfo(remoteAddr))
				continue
			}
			PacketPrintln(g, "green_black", "Listen Rx ", r.ShortPrint())

			// Create a status to the client to tell it the error or that we have accepted the transfer
			session := binary.BigEndian.Uint32(frame[4:8])
			// t := transfer.Lookup(transfer.Client, session, remoteAddr.String())
			if errcode := RxHandler(&r, g, conn); errcode != "success" {
				ErrPrintln(g, "red_black", "Bad Request:", errcode, "  from ",
					UDPinfo(remoteAddr))
			}
			errcode := reqrxhandler(g, r, remoteAddr)                    // process the request
			stheader := "descriptor=" + FlagGetStr(header, "descriptor") // echo the descriptor
			stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,"
			stheader += "errcode=" + errcode // and the handler result

			if errcode != "success" {
				WriteErrStatus(g, stheader, session, conn, remoteAddr)
				ErrPrintln(g, "red_black", "Bad Status:", rxerr, " from ",
					UDPinfo(remoteAddr), " session ", session)
			} else {
				var t *Transfer
				if t = TranLookup(session, remoteAddr.String()); t != nil {
					t.WriteStatus(g, stheader)
				}
			}
			continue

		case "data":
			// Handle incoming data
			var d Data
			var rxerr error
			MsgPrintln(g, "white_black", "Data frame length is:", len(frame))
			if rxerr = FrameDecode(&d, frame); rxerr != nil {
				session := binary.BigEndian.Uint32(frame[4:8])
				// Bad Packet send back a Status to the client
				stheader := "descriptor=" + FlagGetStr(header, "descriptor")
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,errcode=badpacket"
				WriteErrStatus(g, stheader, session, conn, remoteAddr)
				ErrPrintln(g, "red_black", "Bad Data:", rxerr, " from ",
					UDPinfo(remoteAddr), " session ", session)
				continue
			}
			PacketPrintln(g, "green_black", "Listen Rx ", d.ShortPrint())

			// sarwin.MsgPrintln(g, "white_black", "Decoded data ", d.Print())
			session := binary.BigEndian.Uint32(frame[4:8])
			if errcode := RxHandler(&d, g, conn); errcode != "success" {
				ErrPrintln(g, "red_black", "Bad Data:", errcode, "  from ",
					UDPinfo(remoteAddr))
			}
			errcode := datrxhandler(g, d, conn, remoteAddr) // process the data
			if errcode != "success" {                       // If we have a error send back a status with it
				stheader := "descriptor=" + FlagGetStr(header, "descriptor") // echo the descriptor
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,"
				stheader += "errcode=" + errcode
				// Send back a status to the client to tell it the error or that we have a success with creating the transfer
				var st Status
				sinfo := Sinfo{Session: session, Progress: 0, Inrespto: 0, Holes: nil}
				if NewFrame(&st, stheader, &sinfo) != nil {
					ErrPrintln(g, "red_black", "Cannot asemble status")
				}
				var wframe []byte
				var txerr error
				if wframe, txerr = FrameEncode(&st); txerr == nil {
					conn.WriteToUDP(wframe, remoteAddr)
				}
			}
			continue

		case "metadata":
			// Handle incoming metadata
			var m MetaData
			var rxerr error
			if rxerr = FrameDecode(&m, frame); rxerr != nil {
				session := binary.BigEndian.Uint32(frame[4:8])
				var se Status
				// Bad Packet send back a Status to the client
				stheader := "descriptor=" + FlagGetStr(header, "descriptor")
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,"
				stheader += "errcode=badpacket"
				sinfo := Sinfo{Session: session, Progress: 0, Inrespto: 0, Holes: nil}
				if NewFrame(&se, stheader, &sinfo) != nil {
					ErrPrintln(g, "red_black", "Cannot assemble status")
				}
				ErrPrintln(g, "red_black", "Bad MetaData:", rxerr, " from ",
					UDPinfo(remoteAddr), " session ", session)
				continue
			}
			PacketPrintln(g, "green_black", "Listen Rx ", m.ShortPrint())

			session := binary.BigEndian.Uint32(frame[4:8])
			if errcode := RxHandler(&m, g, conn); errcode != "success" {
				ErrPrintln(g, "red_black", "Bad Metadata:", errcode, "  from ",
					UDPinfo(remoteAddr))
			}
			errcode := metrxhandler(g, m, remoteAddr) // process the metadata
			if errcode != "success" {                 // If we have a error send back a status with it
				stheader := "descriptor=" + FlagGetStr(header, "descriptor") // echo the descriptor
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,"
				stheader += "errcode=" + errcode
				WriteErrStatus(g, stheader, session, conn, remoteAddr)
				ErrPrintln(g, "red_black", "Bad Metadata:", rxerr, " from ",
					UDPinfo(remoteAddr), " session ", session)
			}
			continue

		case "status":
			// Handle incoming status
			var s Status
			if rxerr := FrameDecode(&s, frame); rxerr != nil {
				session := binary.BigEndian.Uint32(frame[4:8])
				var se Status
				// Bad Packet send back a Status to the client
				stheader := "descriptor=" + FlagGetStr(header, "descriptor")
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,"
				stheader += "errcode=badpacket"
				sinfo := Sinfo{Session: session, Progress: 0, Inrespto: 0, Holes: nil}
				WriteErrStatus(g, stheader, s.Session, conn, remoteAddr)
				if NewFrame(&se, stheader, &sinfo) != nil {
					ErrPrintln(g, "red_black", "Cannot assemble status")
				}
				ErrPrintln(g, "red_black", "Bad Status:", rxerr, " from ",
					UDPinfo(remoteAddr), " session ", session)
				continue
			} // Handle the status
			PacketPrintln(g, "green_black", "Listen Rx ", s.ShortPrint())
			if errcode := RxHandler(&s, g, conn); errcode != "success" {
				ErrPrintln(g, "red_black", "Bad Status:", errcode, "  from ",
					UDPinfo(remoteAddr))
			}
			errcode := starxhandler(g, s, conn, remoteAddr) // process the status
			if errcode != "success" {                       // If we have a error send back a status with it
				stheader := "descriptor=" + FlagGetStr(header, "descriptor") // echo the descriptor
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,"
				stheader += "errcode=" + errcode
				WriteErrStatus(g, stheader, s.Session, conn, remoteAddr)
				ErrPrintln(g, "red_black", "Bad Status:", errcode, " from ",
					UDPinfo(remoteAddr), " session ", s.Session)
			}
			continue

		default:
			// Bad Packet drop it
			ErrPrintln(g, "red_black", "Listen Invalid Saratoga Frame Recieved from ",
				UDPinfo(remoteAddr))
		}
	}
}

var Cmdptr *Cliflags

// Main
func main() {

	if len(os.Args) != 3 {
		fmt.Println("usage:saratoga <config> <iface>")
		fmt.Println("e.g.: go run saratoga.go saratoga.json en0 (Interface says where to listen for multicast joins")
		return
	}

	// The Command line interface commands, help & usage to be read from saratoga.json
	Cmdptr = new(Cliflags)

	var err error
	// Read in JSON config file and parse it into the Config structure.
	if Cmdptr, err = ReadConfig(os.Args[1]); err != nil {
		fmt.Println("Cannot open saratoga config file we have a Readconf error ", os.Args[1], " ", err)
		return
	}

	// Grab my process ID
	// Pid := os.Getpid()

	// Set global variable pointing to sarflags.Cliflags structure
	// We have to have a global pointer as we cannot pass c directly into gocui

	Cinfo.Prompt = Cmdptr.Prompt
	Cinfo.Ppad = Cmdptr.Ppad

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

	iface, err = net.InterfaceByName(os.Args[2])
	if err != nil {
		fmt.Println("Saratoga Unable to lookup interfacebyname:", os.Args[2])
		log.Fatal(err)
	}
	// Set the Mtu to Interface we are using
	MtuSet(iface.MTU)

	// Set up the gocui interface and start the mainloop
	g, gerr := gocui.NewGui(gocui.OutputNormal)
	if gerr != nil {
		fmt.Printf("Cannot run gocui user interface")
		log.Fatal(err)
	}
	defer g.Close()
	g.SetManagerFunc(Layout)
	if err := Keybindings(g); err != nil {
		log.Panicln(err)
	}

	// Show Host Interfaces & Address's
	ifis, ierr := net.Interfaces()
	if ierr != nil {
		MsgPrintln(g, "green_black", ierr.Error())
	}
	for _, ifi := range ifis {
		if ifi.Name == os.Args[2] { // || ifi.Name == "lo0" {
			MsgPrintln(g, "green_black", ifi.Name, " MTU ", ifi.MTU, " ", ifi.Flags.String(), ":")
			adrs, _ := ifi.Addrs()
			for _, adr := range adrs {
				if strings.Contains(adr.Network(), "ip") {
					MsgPrintln(g, "green_black", "\t Unicast ", adr.String(), " ", adr.Network())
				}
			}
			/*
				madrs, _ := ifi.MulticastAddrs()
				for _, madr := range madrs {
					if strings.Contains(madr.Network(), "ip") {
						sarwin.ErrPrintln(g, "green_black", "\t Multicast ", madr.String(),
						 "Net:", madr.Network())
					}
				}
			*/
		}
	}
	// When will we return from listening for v6 frames
	v6listenquit := make(chan error)

	// Open up V6 sockets for listening on the Saratoga Port
	v6mcastaddr := net.UDPAddr{
		Port: Cmdptr.Port,
		IP:   net.ParseIP(Cmdptr.V6Multicast),
	}
	// Listen to Multicast v6
	v6mcastcon, err := net.ListenMulticastUDP("udp6", iface, &v6mcastaddr)
	if err != nil {
		log.Println("Saratoga Unable to Listen on IPv6 Multicast ", v6mcastaddr.IP, " ", v6mcastaddr.Port)
		log.Fatal(err)
	} else {
		if err := SetMulticastLoop(v6mcastcon); err != nil {
			log.Fatal(err)
		}
		go listen(g, v6mcastcon, v6listenquit)
		MsgPrintln(g, "green_black", "Saratoga IPv6 Multicast Listener started on ",
			UDPinfo(&v6mcastaddr))
	}

	// When will we return from listening for v4 frames
	v4listenquit := make(chan error)

	// Open up V4 sockets for listening on the Saratoga Port
	v4mcastaddr := net.UDPAddr{
		Port: Cmdptr.Port,
		IP:   net.ParseIP(Cmdptr.V4Multicast),
	}

	// Listen to Multicast v4
	v4mcastcon, err := net.ListenMulticastUDP("udp4", iface, &v4mcastaddr)
	if err != nil {
		log.Println("Saratoga Unable to Listen on IPv4 Multicast ", v4mcastaddr.IP, " ", v4mcastaddr.Port)
		log.Fatal(err)
	} else {
		if err := SetMulticastLoop(v4mcastcon); err != nil {
			log.Fatal(err)
		}
		go listen(g, v4mcastcon, v4listenquit)
		MsgPrintln(g, "green_black", "Saratoga IPv4 Multicast Listener started on ",
			UDPinfo(&v4mcastaddr))
	}

	// v4unicastcon, err := net.ListenUDP("udp4", iface, &v4addr)
	MsgPrintf(g, "green_black", "Saratoga Directory is %s\n", Cmdptr.Sardir)
	MsgPrintf(g, "green_black", "Available space is %d MB\n",
		(uint64(fs.Bsize)*fs.Bavail)/1024/1024)

	ErrPrintln(g, "green_black", "MaxInt=", MaxInt)
	ErrPrintln(g, "green_black", "MaxUint=", MaxUint)
	ErrPrintln(g, "green_black", "MaxInt16=", MaxInt16)
	ErrPrintln(g, "green_black", "MaxUint16=", MaxUint16)
	ErrPrintln(g, "green_black", "MaxInt32=", MaxInt32)
	ErrPrintln(g, "green_black", "MaxUint32=", MaxUint32)
	ErrPrintln(g, "green_black", "MaxInt64=", MaxInt64)
	ErrPrintln(g, "green_black", "MaxUint64=", MaxUint64)

	MsgPrintln(g, "green_black", "Maximum Descriptor is:", MaxDescriptor)

	ErrPrintln(g, "white_black", "^P - Toggle Packet View")
	ErrPrintln(g, "white_black", "^Space - Rotate/Change View")

	// The Base calling functions for Saratoga live in cli.go so look there first!
	errflag := make(chan error, 1)
	go mainloop(g, errflag, Cmdptr)
	select {
	case v4err := <-v4listenquit:
		fmt.Println("Saratoga v4 listener has quit with error:", v4err)
	case v6err := <-v6listenquit:
		fmt.Println("Saratoga v6 listener has quit with error:", v6err)
	case err := <-errflag:
		fmt.Println("Mainloop has quit with error:", err.Error())
	}
}

// Go routine for command line loop
func mainloop(g *gocui.Gui, done chan error, c *Cliflags) {
	err := g.MainLoop()
	// fmt.Println(err.Error())
	done <- err
}
