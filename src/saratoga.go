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

	if err := cli.Docmd(g, command[1]); err != nil {
		screen.Fprintln(g, "msg", "red_black", "Invalid Command: ", command[1])
	}
	if command[1] == "exit" || command[1] == "quit" {
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

// Listen -- IPv4 & IPv6 for an incoming frames and shunt them off to the
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
			var se status.Status
			// We don't know the session # so use 0
			_ = se.New("errcode=badpacket", 0, 0, 0, nil)
			screen.Fprintln(g, "msg", "red_black", "Rx Saratoga Frame too short from ",
				sarnet.UDPinfo(remoteAddr))
			goto next
		}
		if framelen > sarnet.MaxFrameSize {
			// Saratoga packet too long
			var se status.Status
			// We can't even know the session # so use 0
			_ = se.New("errcode=badpacket", 0, 0, 0, nil)
			screen.Fprintln(g, "msg", "red_black", "Rx Saratoga Frame too long from ",
				sarnet.UDPinfo(remoteAddr))
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
			var r request.Request
			if rxerr := r.Get(frame); rxerr != nil {
				session := binary.BigEndian.Uint32(frame[4:8])
				var se status.Status
				// Send back a Status to the client
				_ = se.New("errcode=badpacket", session, 0, 0, nil)
				screen.Fprintln(g, "msg", "red_black", "Bad Request:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", session)
				goto next
			} // Handle the request
			session := binary.BigEndian.Uint32(frame[4:8])
			if errcode := r.Handler(g, remoteAddr, session); errcode != "success" {

			}
		case "data":
			var d data.Data
			if rxerr := d.Get(frame); rxerr != nil {
				session := binary.BigEndian.Uint32(frame[4:8])
				var se status.Status
				// Bad Packet send back a Status to the client
				_ = se.New("errcode=badpacket", session, 0, 0, nil)
				screen.Fprintln(g, "msg", "red_black", "Bad Data:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", session)
				goto next
			} // Handle the data
			session := binary.BigEndian.Uint32(frame[4:8])
			if errcode := d.Handler(g, remoteAddr, session); errcode != "success" {

			}
		case "metadata":
			var m metadata.MetaData
			if rxerr := m.Get(frame); rxerr != nil {
				session := binary.BigEndian.Uint32(frame[4:8])
				var se status.Status
				// Bad Packet send back a Status to the client
				_ = se.New("errcode=badpacket", session, 0, 0, nil)
				screen.Fprintln(g, "msg", "red_black", "Bad MetaData:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", session)
				goto next
			} // Handle the data
			session := binary.BigEndian.Uint32(frame[4:8])
			if errcode := m.Handler(g, remoteAddr, session); errcode != "success" {

			}
		case "status":
			var s status.Status
			if rxerr := s.Get(frame); rxerr != nil {
				session := binary.BigEndian.Uint32(frame[4:8])
				var se status.Status
				// Bad Packet send back a Status to the client
				_ = se.New("errcode=badpacket", session, 0, 0, nil)
				screen.Fprintln(g, "msg", "red_black", "Bad Status:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", session)
				goto next
			} // Handle the status
			session := binary.BigEndian.Uint32(frame[4:8])
			if errcode := s.Handler(g, remoteAddr, session); errcode != "success" {

			}
		default:
			// Bad Packet send back a Status to the client
			// We can't even know the session # so use 0
			var se status.Status
			_ = se.New("errcode=badpacket", 0, 0, 0, nil)
			screen.Fprintln(g, "msg", "red_black", "Bad Header in Saratoga Frame from ",
				sarnet.UDPinfo(remoteAddr))
		}
	}
	screen.Fprintln(g, "msg", "red_black", "Sarataga listener failed - ", err)
	quit <- struct{}{}
}

func main() {

	if len(os.Args) != 2 {
		fmt.Println("usage:", "saratoga <iface>")
		fmt.Println("Eg. go run saratoga.go en0")
		return
	}

	// Grab my process ID
	// Pid := os.Getpid()

	// Global Flags set in cli
	sarflags.Global = make(map[string]string)
	// Give them some defaults
	sarflags.Global["descriptor"] = "d64"
	sarflags.Global["csumtype"] = "none"
	sarflags.Global["freespace"] = "no"
	sarflags.Global["txwilling"] = "yes"
	sarflags.Global["rxwilling"] = "yes"
	sarflags.Global["stream"] = "no"
	sarflags.Global["reqtstamp"] = "no"
	sarflags.Global["reqstatus"] = "no"
	sarflags.Global["udplite"] = "no"

	for f := range sarflags.Global {
		if !sarflags.Valid(f, sarflags.Global[f]) {
			ps := "Invalid Flag:" + f + "=" + sarflags.Global[f]
			panic(ps)
		}
	}
	var sardir string

	// Get the default directory for sarotaga transfers from environment
	if sardir = os.Getenv("SARDIR"); sardir == "" {
		panic(errors.New("No Saratoga transfer directory SARDIR environment variabe set"))
	}
	// Move to it
	if err := os.Chdir(sardir); err != nil {
		e := fmt.Sprintf("No such directory SARDIR=%s", sardir)
		panic(errors.New(e))
	}

	var fs syscall.Statfs_t
	if err := syscall.Statfs(sardir, &fs); err != nil {
		panic(errors.New("Cannot stat sardir"))
	}
	fmt.Printf("Saratoga Directory is %s\n", sardir)
	fmt.Printf("Available space is %d MB\n", (uint64(fs.Bsize)*fs.Bavail)/1024/1024)

	quit := make(chan struct{})

	time.Sleep(2 * time.Second)

	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		fmt.Printf("Cannot run gocui user interface")
		panic(err)
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
		panic(err)
	}

	// Listen to Unicast & Multicast
	v6mcastcon, err := net.ListenMulticastUDP("udp6", iface, &v6mcastaddr)
	if err != nil {
		fmt.Println("Saratoga Unable to Listen on IPv6 Multicast")
		panic(err)
	} else {
		sarnet.SetMulticastLoop(v6mcastcon, "IPv6")
		go listen(g, v6mcastcon, quit)
		fmt.Println("Saratoga IPv6 Multicast Listener started on", sarnet.UDPinfo(&v6mcastaddr))
	}

	v4mcastcon, err := net.ListenMulticastUDP("udp4", iface, &v4mcastaddr)
	if err != nil {
		fmt.Println("Saratoga Unable to Listen on IPv4 Multicast")
		panic(err)
	} else {
		sarnet.SetMulticastLoop(v4mcastcon, "IPv4")
		go listen(g, v4mcastcon, quit)
		fmt.Println("Saratoga IPv4 Multicast Listener started on", sarnet.UDPinfo(&v4mcastaddr))
	}

	// Show Host Interfaces & Address's
	ifis, _ := net.Interfaces()
	for _, ifi := range ifis {
		if ifi.Name == os.Args[1] || ifi.Name == "lo0" {
			fmt.Println(ifi.Name, ":")
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

	time.Sleep(30 * time.Second)

	g.Cursor = true

	g.SetManagerFunc(layout)

	if err := keybindings(g); err != nil {
		log.Panicln(err)
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}
