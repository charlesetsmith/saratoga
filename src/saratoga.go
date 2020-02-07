// Saratoga Interactive Client

package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/charlesetsmith/saratoga/src/beacon"
	"github.com/charlesetsmith/saratoga/src/cli"
	"github.com/charlesetsmith/saratoga/src/data"
	"github.com/charlesetsmith/saratoga/src/metadata"
	"github.com/charlesetsmith/saratoga/src/request"
	"github.com/charlesetsmith/saratoga/src/sarflags"
	"github.com/charlesetsmith/saratoga/src/sarnet"
	"github.com/charlesetsmith/saratoga/src/screen"
	"github.com/charlesetsmith/saratoga/src/status"
	"github.com/charlesetsmith/saratoga/src/transfer"
	"github.com/jroimartin/gocui"
)

// *******************************************************************

func switchView(g *gocui.Gui, v *gocui.View) error {
	var err error

	if v.Name() == "cmd" {
		v, err = g.SetCurrentView("msg")
		return err
	}
	v, err = g.SetCurrentView("cmd")
	return err
}

// Backspace or Delete
func backSpace(g *gocui.Gui, v *gocui.View) error {
	cx, _ := v.Cursor()
	if cx <= len(cli.Cprompt)+len(strconv.Itoa(cli.CurLine))+3 { // Dont move
		return nil
	}
	// Delete backwards
	v.EditDelete(true)
	return nil
}

// Down Arrow
func cursorDown(g *gocui.Gui, v *gocui.View) error {
	cx, cy := v.Cursor()
	if err := v.SetCursor(cx, cy+1); err != nil {
		ox, oy := v.Origin()
		if err := v.SetOrigin(ox, oy+1); err != nil {
			return nil
		}
	} else {
		var line string
		var err error
		if line, err = v.Line(cy + 1); err != nil {
			v.SetCursor(len(line), cy)
			line, err = v.Line(cy)
			v.SetCursor(len(line), cy)
			return nil
		}
		if err = v.SetCursor(len(line), cy+1); err != nil {
			screen.Fprintln(g, "msg", "red_black", "cursorDown() x out of range We should never see this!")
		}
	}
	return nil
}

// Up Arrow
func cursorUp(g *gocui.Gui, v *gocui.View) error {
	cx, cy := v.Cursor()
	screen.Fprintf(g, "msg", "green_black", "CursorUp Position x=%d y=%d\n", cx, cy)

	if line, lineerr := v.Line(cy - 1); lineerr == nil {
		screen.Fprintln(g, "msg", "red_black", "No Line ERROR!")
		if err := v.SetCursor(len(line), cy-1); err != nil {
			screen.Fprintln(g, "msg", "blue_black", "SetCursor Out of range for cx=", cx, "cy=", cy-1)
			ox, oy := v.Origin()
			screen.Fprintln(g, "msg", "blue_black", "Origin Reset ox=", ox, "oy=", oy)
			if err := v.SetOrigin(ox, oy); err != nil {
				screen.Fprintln(g, "msg", "blue_black", "SetOrigin Out of range for ox=", ox, "oy=", oy)
				return nil
			}
			line, _ := v.Line(oy)
			if err := v.SetCursor(len(line), oy); err != nil {
				screen.Fprintln(g, "msg", "red_black", "We should never EVER see this!")
			}
		} else {
			line, _ := v.Line(cy - 1)
			if err := v.SetCursor(len(line), cy-1); err != nil {
				screen.Fprintln(g, "msg", "red_black", "We should never see this!")
			}
		}
		return nil
	}
	screen.Fprintln(g, "msg", "red_black", "LINE ERROR!")

	return nil
}

// Commands - cli commands entered
var Commands []string

// Sarwg - Wait group for commands to run/finish - We dont quit till this is 0
var Sarwg sync.WaitGroup

// This is where we process command line inputs after a CR entered
func getLine(g *gocui.Gui, v *gocui.View) error {

	if FirstPass {
		cli.CurLine = 0
		screen.Fprintf(g, "cmd", "yellow_black", "%s[%d]:", cli.Cprompt, cli.CurLine)
		v.SetCursor(len(cli.Cprompt)+3+len(strconv.Itoa(cli.CurLine)), 0)
		return nil
	}
	cx, cy := v.Cursor()
	line, _ := v.Line(cy)
	command := strings.SplitN(line, ":", 2)
	if command[1] == "" { // We have just hit enter - do nothing
		return nil
	}

	Sarwg.Add(1)
	go func(*gocui.Gui, string) {
		defer Sarwg.Done()
		cli.Docmd(g, command[1])
	}(g, command[1])
	// if err := cli.Docmd(g, command[1]); err != nil {
	// 	screen.Fprintln(g, "msg", "red_black", "Invalid Command: ", command[1])
	// }
	if command[1] == "exit" || command[1] == "quit" {
		Sarwg.Wait()
		return quit(g, v)
	}
	Commands = append(Commands, command[1])

	cli.CurLine++
	// screen.Fprintf(g, "msg", "magenta_black", "CurLine=%d <%s>\n", cli.CurLine, command[1])
	if cx > MaxX-2 { // We are about to move beyond X
		screen.Fprintln(g, "msg", "red_black", "cx too big", cx)
	}
	// screen.Fprintf(g, "msg", "green_black", "cx=%d, cy=%d\n", cx, cy)

	// Have we scrolled past the length of v, if so reset the origin
	if err := v.SetCursor(len(cli.Cprompt)+len(strconv.Itoa(cli.CurLine))+3, cy+1); err != nil {
		ox, oy := v.Origin()
		if err := v.SetOrigin(ox, oy+1); err != nil {
			screen.Fprintln(g, "msg", "red_black", "SetOrigin Error:", err)
			return err
		}
		// Reset the cursor to last line in v
		_ = v.SetCursor(len(cli.Cprompt)+len(strconv.Itoa(cli.CurLine))+3, cy)
	}
	// screen.Fprintln(g, "msg", "yellow_black", "MaxY=", MaxY, "Number Cmd View Lines=", CmdLines)
	screen.Fprintf(g, "cmd", "yellow_black", "\n%s[%d]:", cli.Cprompt, cli.CurLine)

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func keybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("cmd", gocui.KeyCtrlSpace, gocui.ModNone, switchView); err != nil {
		return err
	}

	if err := g.SetKeybinding("cmd", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("cmd", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		return err
	}
	if err := g.SetKeybinding("cmd", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}
	if err := g.SetKeybinding("cmd", gocui.KeyEnter, gocui.ModNone, getLine); err != nil {
		return err
	}
	if err := g.SetKeybinding("msg", gocui.KeyEnter, gocui.ModNone, getLine); err != nil {
		return err
	}
	if err := g.SetKeybinding("cmd", gocui.KeyBackspace, gocui.ModNone, backSpace); err != nil {
		return nil
	}
	if err := g.SetKeybinding("cmd", gocui.KeyBackspace2, gocui.ModNone, backSpace); err != nil {
		return nil
	}
	if err := g.SetKeybinding("cmd", gocui.KeyDelete, gocui.ModNone, backSpace); err != nil {
		return nil
	}
	return nil
}

// FirstPass -- First time around layout we don;t put \n at end of prompt
var FirstPass = true

// For working out screen positions in cli i/o

// CmdLines - Number of lines in Cmd View
var CmdLines int

// MaxX - Maximum screen X Value
var MaxX int

// MaxY - Maximum screen Y Value
var MaxY int

func layout(g *gocui.Gui) error {

	var err error
	var cmd *gocui.View
	var msg *gocui.View

	ratio := 4 // Ratio of cmd to err views
	maxX, maxY := g.Size()
	MaxX = maxX
	MaxY = maxY
	// This is the command line input view -- cli inputs and return messages go here
	if cmd, err = g.SetView("cmd", 0, maxY-(maxY/ratio)+1, maxX-1, maxY-1); err != nil {
		CmdLines = (maxY / ratio) - 3 // Number of input lines in cmd view
		if err != gocui.ErrUnknownView {
			return err
		}
		cmd.Title = "Command Line"
		cmd.Highlight = false
		cmd.BgColor = gocui.ColorBlack
		cmd.FgColor = gocui.ColorGreen
		cmd.Editable = true
		cmd.Overwrite = true
		cmd.Wrap = false
	}
	// This is the message view window - All sorts of status & error messages go here
	if msg, err = g.SetView("msg", 0, 0, maxX-1, maxY-maxY/ratio); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		msg.Title = "Messages"
		msg.Highlight = false
		msg.BgColor = gocui.ColorBlack
		msg.FgColor = gocui.ColorYellow
		msg.Editable = false
		msg.Wrap = false
		msg.Overwrite = false
		msg.Autoscroll = true
	}

	// Make sure we have valid Msg & Cmd views
	if msg, err = g.SetCurrentView("msg"); err != nil {
		return err
	}
	// All inputs happen via the cmd view
	if cmd, err = g.SetCurrentView("cmd"); err != nil {
		return err
	}

	// Display the prompt without the \n first time around
	if FirstPass {
		_ = getLine(g, cmd)
		FirstPass = false
	}
	return nil
}

// Request Handler for Server
func reqhandler(g *gocui.Gui, frame []byte, remoteAddr *net.UDPAddr) string {

	var session uint32
	var r request.Request

	if rxerr := r.Get(frame); rxerr != nil {
		session = binary.BigEndian.Uint32(frame[4:8])
		screen.Fprintln(g, "msg", "red_black", "Bad Request:", rxerr, " from ",
			sarnet.UDPinfo(remoteAddr), " session ", session)
		// Send back a Status to the client
		return "badpacket"
	}
	// Handle the request
	session = binary.BigEndian.Uint32(frame[4:8])
	// screen.Fprintln(g, "msg", "green_black", r.Print())

	reqtype := sarflags.GetStr(r.Header, "reqtype")
	switch reqtype {
	case "noaction", "get", "put", "getdelete", "delete", "getdir":
		var t *transfer.STransfer
		if t = transfer.SMatch(remoteAddr.IP.String(), session); t == nil {
			// No matching request so add a new transfer
			var err error
			if err = transfer.SNew(g, reqtype, r, remoteAddr.IP.String(), session); err == nil {
				screen.Fprintln(g, "msg", "yellow_black", "Created Request", reqtype, "from",
					sarnet.UDPinfo(remoteAddr),
					"session", session)
				return "success"
			}
			screen.Fprintln(g, "msg", "red_black", "Cannot create Request", reqtype, "from",
				sarnet.UDPinfo(remoteAddr),
				"session", session, err)
			return "badrequest"
		}
		// Request is currently in progress
		screen.Fprintln(g, "msg", "red_black", "Request", reqtype, "from",
			sarnet.UDPinfo(remoteAddr),
			"for session", session, "already in progress")
		return "badrequest"

	default:
		screen.Fprintln(g, "msg", "red_black", "Invalid Request from",
			sarnet.UDPinfo(remoteAddr),
			"session", session)
		return "badrequest"
	}
}

// Metadata handler for Server
func methandler(g *gocui.Gui, frame []byte, remoteAddr *net.UDPAddr) string {
	var session uint32
	var m metadata.MetaData

	if rxerr := m.Get(frame); rxerr != nil {
		session = binary.BigEndian.Uint32(frame[4:8])
		screen.Fprintln(g, "msg", "red_black", "Bad Metadata:", rxerr, " from ",
			sarnet.UDPinfo(remoteAddr), " session ", session)
		// Send back a Status to the client
		return "badpacket"
	}
	// Handle the metadata
	session = binary.BigEndian.Uint32(frame[4:8])
	// screen.Fprintln(g, "msg", "green_black", m.Print())
	var t *transfer.STransfer
	if t = transfer.SMatch(remoteAddr.IP.String(), session); t != nil {
		t.SChange(g, m)
		screen.Fprintln(g, "msg", "yellow_black", "Changed Transfer", session, "from",
			sarnet.UDPinfo(remoteAddr))
		return "success"
	}
	// Request is currently in progress
	screen.Fprintln(g, "msg", "red_black", "Metadata received for no such transfer as", session, "from",
		sarnet.UDPinfo(remoteAddr))
	return "badpacket"
}

// Data handler for server
func dathandler(g *gocui.Gui, frame []byte, conn *net.UDPConn, remoteAddr *net.UDPAddr) string {
	var session uint32
	var d data.Data

	if rxerr := d.Get(frame); rxerr != nil {
		session = binary.BigEndian.Uint32(frame[4:8])
		screen.Fprintln(g, "msg", "red_black", "Bad Data:", rxerr, " from ",
			sarnet.UDPinfo(remoteAddr), " session ", session)
		// Send back a Status to the client
		return "badpacket"
	}
	// Handle the data
	session = binary.BigEndian.Uint32(frame[4:8])
	// screen.Fprintln(g, "msg", "green_black", m.Print())
	var t *transfer.STransfer
	if t = transfer.SMatch(remoteAddr.IP.String(), session); t != nil {
		t.SData(g, d, conn, remoteAddr) // The data handler for the transfer
		screen.Fprintln(g, "msg", "yellow_black", "Changed Transfer", session, "from",
			sarnet.UDPinfo(remoteAddr))
		return "success"
	}
	// No transfer is currently in progress
	screen.Fprintln(g, "msg", "red_black", "Data received for no such transfer as", session, "from",
		sarnet.UDPinfo(remoteAddr))
	return "badpacket"
}

// Listen -- Go routing for recieving IPv4 & IPv6 for an incoming frames and shunt them off to the
// correct frame handlers
func listen(g *gocui.Gui, conn *net.UDPConn, quit chan struct{}) {

	buf := make([]byte, sarnet.MaxFrameSize+100) // Just in case...
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
			screen.Fprintln(g, "msg", "red_black", "Rx Saratoga Frame too short from ",
				sarnet.UDPinfo(remoteAddr))
			goto next
		}
		if framelen > sarnet.MaxFrameSize {
			screen.Fprintln(g, "msg", "red_black", "Rx Saratoga Frame too long", framelen,
				"from", sarnet.UDPinfo(remoteAddr))
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
				screen.Fprintln(g, "msg", "red_black", "Not Saratoga Version 1 Frame from ",
					sarnet.UDPinfo(remoteAddr))
			}
			goto next
		}

		// Process the frame
		switch sarflags.GetStr(header, "frametype") {
		case "beacon":
			var rxb beacon.Beacon
			if rxerr := rxb.Get(frame); rxerr != nil {
				// We just drop bad beacons
				screen.Fprintln(g, "msg", "red_black", "Bad Beacon:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr))
				goto next
			}
			// Handle the beacon
			if errcode := rxb.Handler(g, remoteAddr); errcode != "success" {
				screen.Fprintln(g, "msg", "red_black", "Bad Beacon:", errcode, " from ",
					sarnet.UDPinfo(remoteAddr))
				goto next
			}

		case "request":
			// Handle the request
			var req request.Request
			var rxerr error
			if rxerr = req.Get(frame); rxerr != nil {
				// We just drop bad requests
				screen.Fprintln(g, "msg", "red_black", "Bad Request:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr))
				goto next
			}

			// Create a status to the client to tell it the error or that we have accepted the transfer
			session := binary.BigEndian.Uint32(frame[4:8])
			errcode := reqhandler(g, frame, remoteAddr)                       // process the request
			stheader := "descriptor=" + sarflags.GetStr(header, "descriptor") // echo the descriptor
			stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,"
			stheader += "errcode=" + errcode

			if errcode != "success" {
				transfer.WriteErrStatus(g, stheader, session, conn, remoteAddr)
				screen.Fprintln(g, "msg", "red_black", "Bad Status:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", session)
			} else {
				var t *transfer.STransfer
				if t = transfer.SMatch(remoteAddr.IP.String(), session); t != nil {
					transfer.WriteStatus(g, t, stheader, conn, remoteAddr)
				}
			}
			goto next

		case "data":
			var d data.Data
			var rxerr error
			if rxerr = d.Get(frame); rxerr != nil {
				session := binary.BigEndian.Uint32(frame[4:8])
				// Bad Packet send back a Status to the client
				stheader := "descriptor=" + sarflags.GetStr(header, "descriptor")
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,errcode=badpacket"
				transfer.WriteErrStatus(g, stheader, session, conn, remoteAddr)
				screen.Fprintln(g, "msg", "red_black", "Bad Data:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", session)
				goto next
			} // Handle the data
			session := binary.BigEndian.Uint32(frame[4:8])
			errcode := dathandler(g, frame, conn, remoteAddr) // process the data
			if errcode != "success" {                         // If we have a error send back a status with it
				stheader := "descriptor=" + sarflags.GetStr(header, "descriptor") // echo the descriptor
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,"
				stheader += "errcode=" + errcode
				// Send back a status to the client to tell it the error or that we have a success with creating the transfer
				var st status.Status
				_ = st.New(stheader, session, 0, 0, nil)
				var wframe []byte
				var txerr error
				if wframe, txerr = st.Put(); txerr == nil {
					_, err := conn.WriteToUDP(wframe, remoteAddr)
					if err != nil || txerr != nil {
						// conn.Close()
						// errflag <- "cantsend"
					}
				}
			}
			goto next

		case "metadata":
			var m metadata.MetaData
			var rxerr error
			if rxerr = m.Get(frame); rxerr != nil {
				session := binary.BigEndian.Uint32(frame[4:8])
				var se status.Status
				// Bad Packet send back a Status to the client
				stheader := "descriptor=" + sarflags.GetStr(header, "descriptor")
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,"
				stheader += "errcode=badpacket"
				_ = se.New(stheader, session, 0, 0, nil)
				screen.Fprintln(g, "msg", "red_black", "Bad MetaData:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", session)
				goto next
			} // Handle the metadata
			session := binary.BigEndian.Uint32(frame[4:8])
			errcode := methandler(g, frame, remoteAddr) // process the metadata
			if errcode != "success" {                   // If we have a error send back a status with it
				stheader := "descriptor=" + sarflags.GetStr(header, "descriptor") // echo the descriptor
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,"
				stheader += "errcode=" + errcode
				transfer.WriteErrStatus(g, stheader, session, conn, remoteAddr)
				screen.Fprintln(g, "msg", "red_black", "Bad Metadata:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", session)
			}
			goto next

		case "status":
			var s status.Status
			if rxerr := s.Get(frame); rxerr != nil {
				session := binary.BigEndian.Uint32(frame[4:8])
				var se status.Status
				// Bad Packet send back a Status to the client
				stheader := "descriptor=" + sarflags.GetStr(header, "descriptor")
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,"
				stheader += "errcode=badpacket"
				_ = se.New(stheader, session, 0, 0, nil)
				screen.Fprintln(g, "msg", "red_black", "Bad Status:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", session)
				goto next
			} // Handle the status

			session := binary.BigEndian.Uint32(frame[4:8])
			if errcode := s.Handler(g, remoteAddr, session); errcode != "success" {

			}
		default:
			// Bad Packet drop it
		}
	}
	screen.Fprintln(g, "msg", "red_black", "Sarataga listener failed - ", err)
	quit <- struct{}{}
}

// Main
func main() {

	if len(os.Args) != 2 {
		fmt.Println("usage:", "saratoga <iface>")
		fmt.Println("Eg. go run saratoga.go en0 (Interface says where to listen for multicast joins")
		return
	}

	// Grab my process ID
	// Pid := os.Getpid()

	// Global Flags set in cli
	sarflags.Climu.Lock()
	sarflags.Cli.Global = make(map[string]string)
	// Give them some defaults
	// Find the maximum supported descriptor
	if sarflags.MaxUint <= sarflags.MaxUint16 {
		sarflags.Cli.Global["descriptor"] = "d16"
	} else if sarflags.MaxUint <= sarflags.MaxUint32 {
		sarflags.Cli.Global["descriptor"] = "d32"
	} else {
		sarflags.Cli.Global["descriptor"] = "d64"
	}
	sarflags.Cli.Global["csumtype"] = "none"
	sarflags.Cli.Global["freespace"] = "no"
	sarflags.Cli.Global["txwilling"] = "yes"
	sarflags.Cli.Global["rxwilling"] = "yes"
	sarflags.Cli.Global["stream"] = "no"
	sarflags.Cli.Global["reqtstamp"] = "no"
	sarflags.Cli.Global["reqstatus"] = "no"
	sarflags.Cli.Global["udplite"] = "no"
	sarflags.Cli.Timestamp = "posix64" // Default timestamp type to use

	/* for f := range sarflags.Cli.Global {
		if !sarflags.Valid(f, sarflags.Global[f]) {
			ps := "Invalid Flag:" + f + "=" + sarflags.Global[f]
			panic(ps)
		}
	} */
	sarflags.Cli.Timestamp = "posix64"
	sarflags.Cli.Timeout.Metadata = 60 // Seconds
	sarflags.Cli.Timeout.Request = 60  // Seconds
	sarflags.Cli.Timeout.Status = 60   // Seconds
	sarflags.Cli.Timeout.Transfer = 60 // Seconds
	sarflags.Cli.Datacnt = 100         // # Data frames between request for status
	sarflags.Cli.Timezone = "utc"      // TImezone to use for logs
	sarflags.Climu.Unlock()

	var sardir string

	// Get the default directory for sarotaga transfers from environment
	if sardir = os.Getenv("SARDIR"); sardir == "" {
		log.Fatal(errors.New("No Saratoga transfer directory SARDIR environment variable set"))
	}
	// Move to it
	if err := os.Chdir(sardir); err != nil {
		e := fmt.Sprintf("No such directory SARDIR=%s", sardir)
		log.Fatal(errors.New(e))
	}

	var fs syscall.Statfs_t
	if err := syscall.Statfs(sardir, &fs); err != nil {
		log.Fatal(errors.New("Cannot stat sardir"))
	}

	quit := make(chan struct{})

	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		fmt.Printf("Cannot run gocui user interface")
		log.Fatal(err)
	}
	defer g.Close()

	// Open up V4 & V6 scokets for listening on the Saratoga Port
	v4mcastaddr := net.UDPAddr{
		Port: sarnet.Port(),
		IP:   net.ParseIP(sarnet.IPv4Multicast),
	}

	v6mcastaddr := net.UDPAddr{
		Port: sarnet.Port(),
		IP:   net.ParseIP(sarnet.IPv6Multicast),
	}
	// What Interface are we receiving Multicasts on
	var iface *net.Interface

	iface, err = net.InterfaceByName(os.Args[1])
	if err != nil {
		fmt.Println("Saratoga Unable to lookup interfacebyname:", os.Args[1])
		log.Fatal(err)
	}
	sarflags.MTU = iface.MTU

	// Listen to Unicast & Multicast v6
	v6mcastcon, err := net.ListenMulticastUDP("udp6", iface, &v6mcastaddr)
	if err != nil {
		fmt.Println("Saratoga Unable to Listen on IPv6 Multicast")
		log.Fatal(err)
	} else {
		sarnet.SetMulticastLoop(v6mcastcon, "IPv6")
		go listen(g, v6mcastcon, quit)
		fmt.Println("Saratoga IPv6 Multicast Server started on", sarnet.UDPinfo(&v6mcastaddr))
	}

	// Listen to Unicast & Multicast v4
	v4mcastcon, err := net.ListenMulticastUDP("udp4", iface, &v4mcastaddr)
	if err != nil {
		fmt.Println("Saratoga Unable to Listen on IPv4 Multicast")
		log.Fatal(err)
	} else {
		sarnet.SetMulticastLoop(v4mcastcon, "IPv4")
		go listen(g, v4mcastcon, quit)
		fmt.Println("Saratoga IPv4 Multicast Server started on", sarnet.UDPinfo(&v4mcastaddr))
	}

	fmt.Printf("Saratoga Directory is %s\n", sardir)
	fmt.Printf("Available space is %d MB\n\n", (uint64(fs.Bsize)*fs.Bavail)/1024/1024)

	// Show Host Interfaces & Address's
	ifis, _ := net.Interfaces()
	for _, ifi := range ifis {
		if ifi.Name == os.Args[1] || ifi.Name == "lo0" {
			fmt.Println(ifi.Name, "MTU", ifi.MTU, ifi.Flags.String(), ":")
			adrs, _ := ifi.Addrs()
			for _, adr := range adrs {
				if strings.Contains(adr.Network(), "ip") {
					fmt.Println("\t Unicast ", adr.String(), adr.Network())
				}
			}
			madrs, _ := ifi.MulticastAddrs()
			for _, madr := range madrs {
				if strings.Contains(madr.Network(), "ip") {
					fmt.Println("\t Multicast ", madr.String(), madr.Network())
				}
			}
		}
	}

	fmt.Println("Sleeping for 5 seconds so you can check out the interfaces")
	time.Sleep(5 * time.Second)

	// Set up the gocui interface and start the mainloop
	g.Cursor = true
	g.SetManagerFunc(layout)
	if err := keybindings(g); err != nil {
		log.Panicln(err)
	}
	// The Base calling functions for Saratoga live in cli.go so look there first!
	errflag := make(chan error, 1)
	go mainloop(g, errflag)
	<-errflag
}

// Go routine for command line loop
func mainloop(g *gocui.Gui, done chan error) {
	var err error

	if err = g.MainLoop(); err != nil && err != gocui.ErrQuit {
		fmt.Printf("%s", err.Error())
	}
	done <- err
}
