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
	"github.com/charlesetsmith/saratoga/frames"
	"github.com/charlesetsmith/saratoga/metadata"
	"github.com/charlesetsmith/saratoga/request"
	"github.com/charlesetsmith/saratoga/sarflags"
	"github.com/charlesetsmith/saratoga/sarnet"
	"github.com/charlesetsmith/saratoga/sarwin"
	"github.com/charlesetsmith/saratoga/status"
	"github.com/charlesetsmith/saratoga/transfer"
	"github.com/jroimartin/gocui"
)

// Request Handler
func reqrxhandler(g *gocui.Gui, r request.Request, remoteAddr *net.UDPAddr) string {
	// Handle the request

	reqtype := sarflags.GetStr(r.Header, "reqtype")
	switch reqtype {
	case "noaction", "get", "put", "getdelete", "delete", "getdir":
		var t *transfer.Transfer
		if t = transfer.Lookup(transfer.Responder, r.Session, remoteAddr.IP.String()); t == nil {
			// No matching request so add a new transfer
			var err error
			if t, err = transfer.NewResponder(g, r, remoteAddr.String()); err == nil {
				sarwin.MsgPrintln(g, "yellow_black", "New Transfer ", reqtype, " from ",
					sarnet.UDPinfo(remoteAddr))
				return "success"
			}
			sarwin.ErrPrintln(g, "red_black", "Cannot create Transfer ", reqtype, " from ",
				sarnet.UDPinfo(remoteAddr),
				"session", r.Session, err)
			return "badrequest"
		}
		// Request is currently in progress
		sarwin.ErrPrintln(g, "red_black", "Request ", reqtype, " from ",
			sarnet.UDPinfo(remoteAddr),
			" for session ", r.Session, " already in progress")
		return "badrequest"
	default:
		sarwin.ErrPrintln(g, "red_black", "Invalid Request from ",
			sarnet.UDPinfo(remoteAddr),
			" session ", r.Session)
		return "badrequest"
	}
}

// Metadata handler for Responder
func metrxhandler(g *gocui.Gui, m metadata.MetaData, remoteAddr *net.UDPAddr) string {
	// Handle the metadata
	var t *transfer.Transfer
	if t = transfer.Lookup(transfer.Responder, m.Session, remoteAddr.IP.String()); t != nil {
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

// Data handler for responder
func datrxhandler(g *gocui.Gui, d data.Data, conn *net.UDPConn, remoteAddr *net.UDPAddr) string {
	// Handle the data
	var t *transfer.Transfer
	if t = transfer.Lookup(transfer.Responder, d.Session, remoteAddr.IP.String()); t != nil {
		// t.SData(g, d, conn, remoteAddr) // The data handler for the transfer
		transfer.Trmu.Lock()
		if sarflags.GetStr(d.Header, "reqtstamp") == "yes" { // Grab the timestamp from data
			t.Tstamp = d.Tstamp
		}
		// Copy the data in this frame to the transfer buffer
		// THIS IS BAD WE HAVE AN int NOT A uint64!!!
		if (d.Offset)+(uint64)(len(d.Payload)) > (uint64)(len(t.Data)) {
			return "badoffset"
		}
		copy(t.Data[d.Offset:], d.Payload)
		t.Dcount++
		if t.Dcount%100 == 0 { // Send back a status every 100 data frames recieved
			t.Dcount = 0
		}
		if t.Dcount == 0 || sarflags.GetStr(d.Header, "reqstatus") == "yes" || !t.Havemeta { // Send a status back
			stheader := "descriptor=" + sarflags.GetStr(d.Header, "descriptor") // echo the descriptor
			stheader += ",allholes=yes,reqholes=requested,errcode=success,"
			if !t.Havemeta {
				stheader += "metadatarecvd=no"
			} else {
				stheader += "metadatarecvd=yes"
			}
			// Send back a status to the client to tell it a success with creating the transfer
			transfer.WriteStatus(g, t, stheader)
		}
		sarwin.MsgPrintln(g, "yellow_black", "Responder Received Data Len:", len(d.Payload), " Pos:", d.Offset)
		transfer.Strmu.Unlock()
		sarwin.MsgPrintln(g, "yellow_black", "Changed Transfer ", d.Session, " from ",
			sarnet.UDPinfo(remoteAddr))
		return "success"
	}
	// No transfer is currently in progress
	sarwin.ErrPrintln(g, "red_black", "Data received for no such transfer as ", d.Session, " from ",
		sarnet.UDPinfo(remoteAddr))
	return "badpacket"
}

// Status handler for server
func starxhandler(g *gocui.Gui, s status.Status, conn *net.UDPConn, remoteAddr *net.UDPAddr) string {
	var t *transfer.STransfer
	if t = transfer.SMatch(remoteAddr.IP.String(), s.Session); t == nil { // No existing transfer
		return "badstatus"
	}
	// Update transfers Inrespto & Progress indicators
	transfer.Strmu.Lock()
	defer transfer.Strmu.Unlock()
	t.Inrespto = s.Inrespto
	t.Progress = s.Progress
	// Resend the data requested by the Holes

	return "success"
}

// Listen -- Go routine for recieving IPv4 & IPv6 for incoming frames and shunt them off to the
// correct frame handlers
func listen(g *gocui.Gui, conn *net.UDPConn, quit chan error) {

	// Investigate using conn.PathMTU()
	maxframesize := sarflags.Mtu() - 60   // Handles IPv4 & IPv6 header
	buf := make([]byte, maxframesize+100) // Just in case...
	framelen := 0
	err := error(nil)
	var remoteAddr *net.UDPAddr = new(net.UDPAddr)
	for err == nil { // Loop forever grabbing frames
		// Read into buf
		framelen, remoteAddr, err = conn.ReadFromUDP(buf)
		if err != nil {
			sarwin.ErrPrintln(g, "red_black", "Sarataga listener failed:", err)
			quit <- err
			return
		}
		sarwin.MsgPrintln(g, "green_black", "Listen read ", framelen, " bytes from ", remoteAddr.String())

		// Very basic frame checks before we get into what it is
		if framelen < 8 {
			// Saratoga packet too small
			sarwin.ErrPrintln(g, "red_black", "Rx Saratoga Frame too short from ",
				sarnet.UDPinfo(remoteAddr))
			continue
		}
		if framelen > maxframesize {
			sarwin.ErrPrintln(g, "red_black", "Rx Saratoga Frame too long ", framelen,
				" from ", sarnet.UDPinfo(remoteAddr))
			continue
		}

		// OK so we might have a valid frame so copy it to to the frame byte slice
		frame := make([]byte, framelen)
		copy(frame, buf[:framelen])
		// fmt.Println("We have a frame of length:", framelen)
		// Grab the Saratoga Header which is the first 32 bits
		header := binary.BigEndian.Uint32(frame[:4])
		// Check the version
		if sarflags.GetStr(header, "version") != "v1" { // Make sure we are Version 1
			// If a bad version received then send back a Status to the client
			var st status.Status
			sinfo := status.Sinfo{Session: 0, Progress: 0, Inrespto: 0, Holes: nil}
			sarwin.ErrPrintln(g, "red_black", "Not Saratoga Version 1 Frame from ",
				sarnet.UDPinfo(remoteAddr))
			if frames.New(&st, "errcode=badpacket", &sinfo) != nil {
				sarwin.ErrPrintln(g, "red_black", "Cannot create badpacket status")
			}
			// OK Send it out the connection
			err := st.UDPWrite(conn)
			if err != "success" {
				sarwin.ErrPrintln(g, "red_black", err+" unable to send status")
			}
			continue
		}
		sarwin.MsgPrintln(g, "white_black", "Received: ", sarflags.GetStr(header, "frametype"))

		// Process the received frame
		switch sarflags.GetStr(header, "frametype") {
		case "beacon":
			// We have received a beacon on the connection
			var rxb beacon.Beacon
			if rxerr := rxb.Decode(frame); rxerr != nil {
				// We just drop bad beacons
				sarwin.ErrPrintln(g, "red_black", "Bad Beacon:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr))
				continue
			}
			// Handle the beacon
			if errcode := frames.RxHandler(&rxb, conn); errcode != "success" {
				sarwin.ErrPrintln(g, "red_black", "Bad Beacon:", errcode, "  from ",
					sarnet.UDPinfo(remoteAddr))
			}
			if rxb.NewPeer(conn) {
				sarwin.MsgPrintln(g, "Received new or updated beacon from ", sarnet.UDPinfo(remoteAddr))
			}
			sarwin.PacketPrintln(g, "green_black", "Listen Rx ", rxb.ShortPrint())
			continue

		case "request":
			// Handle incoming request
			var r request.Request
			var rxerr error
			if rxerr = r.Decode(frame); rxerr != nil {
				// We just drop bad requests
				sarwin.ErrPrintln(g, "red_black", "Bad Request:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr))
				continue
			}
			sarwin.PacketPrintln(g, "green_black", "Listen Rx ", r.ShortPrint())

			// Create a status to the client to tell it the error or that we have accepted the transfer
			session := binary.BigEndian.Uint32(frame[4:8])
			// t := transfer.Lookup(transfer.Initiator, session, remoteAddr.String())
			errcode := reqrxhandler(g, r, remoteAddr) // process the request
			stheader := "descriptor=" + sarflags.GetStr(header, "descriptor") +
				",metadatarecvd=no,allholes=yes,reqholes=requested," +
				"errcode=" + errcode // and the handler result

			if errcode != "success" {
				transfer.WriteErrStatus(g, stheader, session, conn, remoteAddr)
				sarwin.ErrPrintln(g, "red_black", "Bad Status:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", session)
			} else {
				// Acknowledge the request to the peer
				var t *transfer.Transfer
				if t = transfer.Lookup(transfer.Initiator, session, remoteAddr.String()); t != nil {
					t.WriteStatus(g, stheader)
				} else {
					sarwin.ErrPrintln(g, "red_black", "Cannot find existing transfer")
				}
			}
			continue

		case "data":
			// Handle incoming data
			var d data.Data
			var rxerr error
			sarwin.MsgPrintln(g, "white_black", "Data frame length is:", len(frame))
			if rxerr = frames.Decode(&d, frame); rxerr != nil {
				session := binary.BigEndian.Uint32(frame[4:8])
				// Bad Packet send back a Status to the client
				stheader := "descriptor=" + sarflags.GetStr(header, "descriptor")
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,errcode=badpacket"
				transfer.WriteErrStatus(g, stheader, session, conn, remoteAddr)
				sarwin.ErrPrintln(g, "red_black", "Bad Data:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", session)
				continue
			}
			sarwin.PacketPrintln(g, "green_black", "Listen Rx ", d.ShortPrint())

			// sarwin.MsgPrintln(g, "white_black", "Decoded data ", d.Print())
			session := binary.BigEndian.Uint32(frame[4:8])
			if errcode := frames.RxHandler(&d, conn); errcode != "success" {
				sarwin.ErrPrintln(g, "red_black", "Bad Data:", errcode, "  from ",
					sarnet.UDPinfo(remoteAddr))
			}
			errcode := datrxhandler(g, d, conn, remoteAddr) // process the data
			if errcode != "success" {                       // If we have a error send back a status with it
				stheader := "descriptor=" + sarflags.GetStr(header, "descriptor") // echo the descriptor
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,"
				stheader += "errcode=" + errcode
				// Send back a status to the client to tell it the error or that we have a success with creating the transfer
				var st status.Status
				sinfo := status.Sinfo{Session: session, Progress: 0, Inrespto: 0, Holes: nil}
				if frames.New(&st, stheader, &sinfo) != nil {
					sarwin.ErrPrintln(g, "red_black", "Cannot asemble status")
				}
				var wframe []byte
				var txerr error
				if wframe, txerr = frames.Encode(&st); txerr == nil {
					conn.WriteToUDP(wframe, remoteAddr)
				}
			}
			continue

		case "metadata":
			// Handle incoming metadata
			var m metadata.MetaData
			var rxerr error
			if rxerr = frames.Decode(&m, frame); rxerr != nil {
				session := binary.BigEndian.Uint32(frame[4:8])
				var se status.Status
				// Bad Packet send back a Status to the client
				stheader := "descriptor=" + sarflags.GetStr(header, "descriptor")
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,"
				stheader += "errcode=badpacket"
				sinfo := status.Sinfo{Session: session, Progress: 0, Inrespto: 0, Holes: nil}
				if frames.New(&se, stheader, &sinfo) != nil {
					sarwin.ErrPrintln(g, "red_black", "Cannot assemble status")
				}
				sarwin.ErrPrintln(g, "red_black", "Bad MetaData:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", session)
				continue
			}
			sarwin.PacketPrintln(g, "green_black", "Listen Rx ", m.ShortPrint())

			session := binary.BigEndian.Uint32(frame[4:8])
			if errcode := frames.RxHandler(&m, conn); errcode != "success" {
				sarwin.ErrPrintln(g, "red_black", "Bad Metadata:", errcode, "  from ",
					sarnet.UDPinfo(remoteAddr))
			}
			errcode := metrxhandler(g, m, remoteAddr) // process the metadata
			if errcode != "success" {                 // If we have a error send back a status with it
				stheader := "descriptor=" + sarflags.GetStr(header, "descriptor") // echo the descriptor
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,"
				stheader += "errcode=" + errcode
				transfer.WriteErrStatus(g, stheader, session, conn, remoteAddr)
				sarwin.ErrPrintln(g, "red_black", "Bad Metadata:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", session)
			}
			continue

		case "status":
			// Handle incoming status
			var s status.Status
			if rxerr := frames.Decode(&s, frame); rxerr != nil {
				session := binary.BigEndian.Uint32(frame[4:8])
				var se status.Status
				// Bad Packet send back a Status to the client
				stheader := "descriptor=" + sarflags.GetStr(header, "descriptor")
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,"
				stheader += "errcode=badpacket"
				sinfo := status.Sinfo{Session: session, Progress: 0, Inrespto: 0, Holes: nil}
				transfer.WriteErrStatus(g, stheader, s.Session, conn, remoteAddr)
				if frames.New(&se, stheader, &sinfo) != nil {
					sarwin.ErrPrintln(g, "red_black", "Cannot assemble status")
				}
				sarwin.ErrPrintln(g, "red_black", "Bad Status:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", session)
				continue
			} // Handle the status
			sarwin.PacketPrintln(g, "green_black", "Listen Rx ", s.ShortPrint())
			if errcode := frames.RxHandler(&s, conn); errcode != "success" {
				sarwin.ErrPrintln(g, "red_black", "Bad Status:", errcode, "  from ",
					sarnet.UDPinfo(remoteAddr))
			}
			errcode := starxhandler(g, s, conn, remoteAddr) // process the status
			if errcode != "success" {                       // If we have a error send back a status with it
				stheader := "descriptor=" + sarflags.GetStr(header, "descriptor") // echo the descriptor
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,"
				stheader += "errcode=" + errcode
				transfer.WriteErrStatus(g, stheader, s.Session, conn, remoteAddr)
				sarwin.ErrPrintln(g, "red_black", "Bad Status:", errcode, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", s.Session)
			}
			continue

		default:
			// Bad Packet drop it
			sarwin.ErrPrintln(g, "red_black", "Listen Invalid Saratoga Frame Recieved from ",
				sarnet.UDPinfo(remoteAddr))
		}
	}
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
	Cmdptr = new(sarflags.Cliflags)

	var err error
	// Read in JSON config file and parse it into the Config structure.
	if Cmdptr, err = sarflags.ReadConfig(os.Args[argnumb]); err != nil {
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
	g.SetManagerFunc(sarwin.Layout)
	if err := sarwin.Keybindings(g); err != nil {
		log.Panicln(err)
	}

	// Show Host Interfaces & Address's
	ifis, ierr := net.Interfaces()
	if ierr != nil {
		sarwin.MsgPrintln(g, "green_black", ierr.Error())
	}
	argnumb++
	for _, ifi := range ifis {
		if ifi.Name == os.Args[argnumb] { // || ifi.Name == "lo0" {
			sarwin.MsgPrintln(g, "green_black", ifi.Name, " MTU ", ifi.MTU, " ", ifi.Flags.String(), ":")
			adrs, _ := ifi.Addrs()
			for _, adr := range adrs {
				if strings.Contains(adr.Network(), "ip") {
					sarwin.MsgPrintln(g, "green_black", "\t Unicast ", adr.String(), " ", adr.Network())

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
		if err := sarnet.SetMulticastLoop(v6mcastcon); err != nil {
			log.Fatal(err)
		}
		go listen(g, v6mcastcon, v6listenquit)
		sarwin.MsgPrintln(g, "green_black", "Saratoga IPv6 Multicast Listener started on ",
			sarnet.UDPinfo(&v6mcastaddr))
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
		if err := sarnet.SetMulticastLoop(v4mcastcon); err != nil {
			log.Fatal(err)
		}
		go listen(g, v4mcastcon, v4listenquit)
		sarwin.MsgPrintln(g, "green_black", "Saratoga IPv4 Multicast Listener started on ",
			sarnet.UDPinfo(&v4mcastaddr))
	}

	// v4unicastcon, err := net.ListenUDP("udp4", iface, &v4addr)
	sarwin.MsgPrintf(g, "green_black", "Saratoga Directory is %s\n", Cmdptr.Sardir)
	sarwin.MsgPrintf(g, "green_black", "Available space is %d MB\n",
		(uint64(fs.Bsize)*fs.Bavail)/1024/1024)

	sarwin.ErrPrintln(g, "green_black", "MaxInt=", sarflags.MaxInt)
	sarwin.ErrPrintln(g, "green_black", "MaxUint=", sarflags.MaxUint)
	sarwin.ErrPrintln(g, "green_black", "MaxInt16=", sarflags.MaxInt16)
	sarwin.ErrPrintln(g, "green_black", "MaxUint16=", sarflags.MaxUint16)
	sarwin.ErrPrintln(g, "green_black", "MaxInt32=", sarflags.MaxInt32)
	sarwin.ErrPrintln(g, "green_black", "MaxUint32=", sarflags.MaxUint32)
	sarwin.ErrPrintln(g, "green_black", "MaxInt64=", sarflags.MaxInt64)
	sarwin.ErrPrintln(g, "green_black", "MaxUint64=", sarflags.MaxUint64)

	sarwin.MsgPrintln(g, "green_black", "Maximum Descriptor is:", sarflags.MaxDescriptor)

	sarwin.ErrPrintln(g, "white_black", "^P - Toggle Packet View")
	sarwin.ErrPrintln(g, "white_black", "^Space - Rotate/Change View")

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
func mainloop(g *gocui.Gui, done chan error, c *sarflags.Cliflags) {
	err := g.MainLoop()
	// fmt.Println(err.Error())
	done <- err
}
