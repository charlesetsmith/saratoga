// Saratoga Interactive Client - Main

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

	"github.com/charlesetsmith/saratoga/beacon"
	"github.com/charlesetsmith/saratoga/cli"
	"github.com/charlesetsmith/saratoga/data"
	"github.com/charlesetsmith/saratoga/metadata"
	"github.com/charlesetsmith/saratoga/request"
	"github.com/charlesetsmith/saratoga/sarflags"
	"github.com/charlesetsmith/saratoga/sarnet"
	"github.com/charlesetsmith/saratoga/sarscreen"
	"github.com/charlesetsmith/saratoga/status"
	"github.com/charlesetsmith/saratoga/transfer"
	"github.com/jroimartin/gocui"
)

// Cinfo - Information held on the cmd view
var Cinfo sarscreen.Viewinfo

// Minfo - Information held on the msg view
var Minfo sarscreen.Viewinfo

// Return the length of the prompt
func promptlen(v sarscreen.Viewinfo) int {
	return len(v.Prompt) + len(strconv.Itoa(v.Curline)) + v.Ppad
}

func switchView(g *gocui.Gui, v *gocui.View) error {
	var err error

	switch v.Name() {
	case "cmd":
		_, err = g.SetCurrentView("msg")
	case "msg":
		_, err = g.SetCurrentView("cmd")
	}
	return err
}

// Backspace or Delete
func backSpace(g *gocui.Gui, v *gocui.View) error {
	if g == nil || v == nil {
		log.Fatal("backSpace - g or v is nil")
	}
	cx, _ := v.Cursor()
	if cx <= promptlen(Cinfo) { // Dont move we are at the prompt
		return nil
	}
	// Delete rune backwards
	v.EditDelete(true)
	return nil
}

// Handle Left Arrow Move -- All good
func cursorLeft(g *gocui.Gui, v *gocui.View) error {
	if g == nil || v == nil {
		log.Fatal("cursorLeft - g or v is nil")
	}
	cx, cy := v.Cursor()
	if cx <= promptlen(Cinfo) { // Dont move
		return nil
	}
	// Move back a character
	if err := v.SetCursor(cx-1, cy); err != nil {
		sarscreen.Fprintln(g, "msg", "bwhite_black", "LeftArrow:", "cx=", cx, "cy=", cy, "error=", err)
	}
	return nil
}

// Handle Right Arrow Move - All good
func cursorRight(g *gocui.Gui, v *gocui.View) error {
	if g == nil || v == nil {
		log.Fatal("cursorRight - g or v is nil")
	}
	cx, cy := v.Cursor()
	line, _ := v.Line(cy)
	if cx >= len(line)-1 { // We are at the end of line do nothing
		v.SetCursor(len(line), cy)
		return nil
	}
	// Move forward a character
	if err := v.SetCursor(cx+1, cy); err != nil {
		sarscreen.Fprintln(g, "msg", "bwhite_red", "RightArrow:", "cx=", cx, "cy=", cy, "error=", err)
	}
	return nil
}

// Handle down cursor -- All good!
// well not quite, still issue if we scroll down before hitting return
func cursorDown(g *gocui.Gui, v *gocui.View) error {
	if g == nil || v == nil {
		log.Fatal("cursorDown - g or v is nil")
	}
	_, oy := v.Origin()
	cx, cy := v.Cursor()

	// Don't move down if we are at the last line in current views Bufferlines
	if oy+cy >= len(v.BufferLines())-1 {
		return nil
	}
	err := v.SetCursor(cx, cy+1)
	if err != nil { // Reset the origin
		if err := v.SetOrigin(0, oy+1); err != nil { // changed ox to 0
			sarscreen.Fprintf(g, "msg", "bwhite_red", "SetOrigin error=%s", err)
			return err
		}

	}
	// Move the cursor to the end of the current line
	_, cy = v.Cursor()
	if line, err := v.Line(cy); err == nil {
		v.SetCursor(len(line), cy)
	}
	return nil
}

// Handle up cursor -- All good!
func cursorUp(g *gocui.Gui, v *gocui.View) error {
	if g == nil || v == nil {
		log.Fatal("cursorUp - g or v is nil")
	}
	_, oy := v.Origin()
	cx, cy := v.Cursor()
	err := v.SetCursor(cx, cy-1)
	if err != nil && oy > 0 { // Reset the origin
		if err := v.SetOrigin(0, oy-1); err != nil { // changed ox to 0
			sarscreen.Fprintf(g, "msg", "bwhite_red", "SetOrigin error=%s", err)
			return err
		}
	}
	// Move the cursor to the end of the current line
	_, cy = v.Cursor()
	if line, err := v.Line(cy); err == nil {
		v.SetCursor(len(line), cy)
	}
	return nil
}

// Commands - cli commands entered
var Commands []string

// Sarwg - Wait group for commands to run/finish - We dont quit till this is 0
var Sarwg sync.WaitGroup

// This is where we process command line inputs after a CR entered
func getLine(g *gocui.Gui, v *gocui.View, c *sarflags.Cliflags) error {
	if g == nil || v == nil {
		log.Fatal("getLine - g or v is nil")
	}
	// Find out where we are
	_, cy := v.Cursor()
	// Get the line
	line, _ := v.Line(cy)
	// sarscreen.Fprintf(g, "msg", "red_black", "cx=%d cy=%d lines=%d line=%s\n",
	//      len(v.BufferLines()), cx, cy, line)
	command := strings.SplitN(line, ":", 2)
	if command[1] == "" { // We have just hit enter - do nothing
		return nil
	}
	// Save the command into history
	Cinfo.Commands = append(Cinfo.Commands, command[1])

	// Spawn a go and run the command
	go func(*gocui.Gui, string) {
		// defer Sarwg.Done()
		cli.Docmd(g, command[1], c)
	}(g, command[1])

	if command[1] == "exit" || command[1] == "quit" {
		// Sarwg.Wait()
		err := quit(g, v)
		// THIS IS A KLUDGE FIX IT WITH A CHANNEL
		log.Fatal("\nGocui Exit. Bye!\n", err)
	}

	Cinfo.Curline++
	xpos := promptlen(Cinfo)
	// Have we scrolled past the length of v, if so reset the origin

	if err := v.SetCursor(xpos, cy+1); err != nil {
		sarscreen.Fprintln(g, "msg", "red_black", "We Scrolled past length of v", err)
		_, oy := v.Origin()
		// sarscreen.Fprintf(g, "msg", "red_black", "Origin reset ox=%d oy=%d\n", ox, oy)
		if err := v.SetOrigin(0, oy+1); err != nil { // changed xpos to 0
			// sarscreen.Fprintln(g, "msg", "red_black", "SetOrigin Error:", err)
			return err
		}
		// Set the cursor to last line in v
		if verr := v.SetCursor(xpos, cy); verr != nil {
			sarscreen.Fprintln(g, "msg", "bwite_red", "Setcursor out of bounds:", verr)
		}
		cx, cy := v.Cursor()
		sarscreen.Fprintf(g, "msg", "red_black", "cx=%d cy=%d line=%s\n", cx, cy, line)
	}
	// Put up the new prompt on the next line
	sarscreen.Fprintf(g, "cmd", "yellow_black", "\n%s[%d]:", Cinfo.Prompt, Cinfo.Curline)
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
	if err := g.SetKeybinding("cmd", gocui.KeyArrowLeft, gocui.ModNone, cursorLeft); err != nil {
		return nil
	}
	if err := g.SetKeybinding("cmd", gocui.KeyArrowRight, gocui.ModNone, cursorRight); err != nil {
		return nil
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
		cmd.Wrap = true
	}
	// This is the message view window - Status & error messages go here
	if msg, err = g.SetView("msg", 0, 0, maxX-1, maxY-maxY/ratio); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		msg.Title = "Messages"
		msg.Highlight = false
		msg.BgColor = gocui.ColorBlack
		msg.FgColor = gocui.ColorYellow
		msg.Editable = false
		msg.Wrap = true
		msg.Overwrite = false
		msg.Autoscroll = true
	}

	// All inputs happen via the cmd view
	if cmd, err = g.SetCurrentView("cmd"); err != nil {
		return err
	}
	// Display the prompt without the \n first time around
	if FirstPass {
		cmd.SetCursor(0, 0)
		Cinfo.Curline = 0
		prompt := fmt.Sprintf("%s[%d]:", Cinfo.Prompt, Cinfo.Curline)
		sarscreen.Fprintf(g, "cmd", "yellow_black", prompt)
		cmd.SetCursor(promptlen(Cinfo), 0)
		FirstPass = false
	}
	return nil
}

// Request Handler for Server
func reqrxhandler(g *gocui.Gui, r request.Request, remoteAddr *net.UDPAddr) string {

	// Handle the request
	reqtype := sarflags.GetStr(r.Header, "reqtype")
	switch reqtype {
	case "noaction", "get", "put", "getdelete", "delete", "getdir":
		var t *transfer.STransfer
		if t = transfer.SMatch(remoteAddr.IP.String(), r.Session); t == nil {
			// No matching request so add a new transfer
			var err error
			if err = transfer.SNew(g, reqtype, r, remoteAddr.IP.String(), r.Session); err == nil {
				sarscreen.Fprintln(g, "msg", "yellow_black", "Created Request", reqtype, "from",
					sarnet.UDPinfo(remoteAddr),
					"session", r.Session)
				return "success"
			}
			sarscreen.Fprintln(g, "msg", "red_black", "Cannot create Request", reqtype, "from",
				sarnet.UDPinfo(remoteAddr),
				"session", r.Session, err)
			return "badrequest"
		}
		// Request is currently in progress
		sarscreen.Fprintln(g, "msg", "red_black", "Request", reqtype, "from",
			sarnet.UDPinfo(remoteAddr),
			"for session", r.Session, "already in progress")
		return "badrequest"
	default:
		sarscreen.Fprintln(g, "msg", "red_black", "Invalid Request from",
			sarnet.UDPinfo(remoteAddr),
			"session", r.Session)
		return "badrequest"
	}
}

// Metadata handler for Server
func metrxhandler(g *gocui.Gui, m metadata.MetaData, remoteAddr *net.UDPAddr) string {
	// Handle the metadata
	// sarscreen.Fprintln(g, "msg", "green_black", m.Print())
	var t *transfer.STransfer
	if t = transfer.SMatch(remoteAddr.IP.String(), m.Session); t != nil {
		if err := t.SChange(g, m); err != nil { // Size of file has changed!!!
			return "unspecified"
		}
		sarscreen.Fprintln(g, "msg", "yellow_black", "Changed Transfer", m.Session, "from",
			sarnet.UDPinfo(remoteAddr))
		return "success"
	}
	// Request is currently in progress
	sarscreen.Fprintln(g, "msg", "red_black", "Metadata received for no such transfer as", m.Session, "from",
		sarnet.UDPinfo(remoteAddr))
	return "badpacket"
}

// Data handler for server
func datrxhandler(g *gocui.Gui, d data.Data, conn *net.UDPConn, remoteAddr *net.UDPAddr) string {
	// Handle the data
	// sarscreen.Fprintln(g, "msg", "green_black", m.Print())
	var t *transfer.STransfer
	if t = transfer.SMatch(remoteAddr.IP.String(), d.Session); t != nil {
		// t.SData(g, d, conn, remoteAddr) // The data handler for the transfer
		transfer.Strmu.Lock()
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
			transfer.WriteStatus(g, t, stheader, conn, remoteAddr)
		}
		sarscreen.Fprintln(g, "msg", "yellow_black", "Server Recieved Data Len:", len(d.Payload), "Pos:", d.Offset)
		transfer.Strmu.Unlock()
		sarscreen.Fprintln(g, "msg", "yellow_black", "Changed Transfer", d.Session, "from",
			sarnet.UDPinfo(remoteAddr))
		return "success"
	}
	// No transfer is currently in progress
	sarscreen.Fprintln(g, "msg", "red_black", "Data received for no such transfer as", d.Session, "from",
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

// Listen -- Go routine for recieving IPv4 & IPv6 for an incoming frames and shunt them off to the
// correct frame handlers
func listen(g *gocui.Gui, conn *net.UDPConn, quit chan error) {

	buf := make([]byte, sarnet.MaxFrameSize+100) // Just in case...
	framelen := 0
	err := error(nil)
	remoteAddr := new(net.UDPAddr)
	sarscreen.Fprintln(g, "msg", "green_black", "Listen Remote Address:", remoteAddr)
	for err == nil { // Loop forever grabbing frames
		// Read into buf
		framelen, remoteAddr, err = conn.ReadFromUDP(buf)
		if err != nil {
			sarscreen.Fprintln(g, "msg", "red_black", "Sarataga listener failed:", err)
			quit <- err
			return
		}

		// Very basic frame checks before we get into what it is
		if framelen < 8 {
			// Saratoga packet too small
			sarscreen.Fprintln(g, "msg", "red_black", "Rx Saratoga Frame too short from ",
				sarnet.UDPinfo(remoteAddr))
			continue
		}
		if framelen > sarnet.MaxFrameSize {
			sarscreen.Fprintln(g, "msg", "red_black", "Rx Saratoga Frame too long", framelen,
				"from", sarnet.UDPinfo(remoteAddr))
			continue
		}

		// OK so we might have a valid frame so copy it to to the frame byte slice
		frame := make([]byte, framelen)
		copy(frame, buf[:framelen])
		// fmt.Println("We have a frame of length:", framelen)
		// Grab the Saratoga Header
		header := binary.BigEndian.Uint32(frame[:4])
		if sarflags.GetStr(header, "version") != "v1" { // Make sure we are Version 1
			if _, err = sarflags.Set(0, "errno", "badpacket"); err != nil {
				// Bad Packet send back a Status to the client
				var se status.Status
				_ = se.New("errcode=badpacket", 0, 0, 0, nil)
				sarscreen.Fprintln(g, "msg", "red_black", "Not Saratoga Version 1 Frame from ",
					sarnet.UDPinfo(remoteAddr))
			}
			continue
		}

		// Process the frame
		switch sarflags.GetStr(header, "frametype") {
		case "beacon":
			var rxb beacon.Beacon
			if rxerr := rxb.Get(frame); rxerr != nil {
				// We just drop bad beacons
				sarscreen.Fprintln(g, "msg", "red_black", "Bad Beacon:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr))
				continue
			}
			// Handle the beacon
			if errcode := rxb.Handler(g, remoteAddr); errcode != "success" {
				sarscreen.Fprintln(g, "msg", "red_black", "Bad Beacon:", errcode, " from ",
					sarnet.UDPinfo(remoteAddr))
				continue
			}

		case "request":
			// Handle incoming request
			var r request.Request
			var rxerr error
			if rxerr = r.Get(frame); rxerr != nil {
				// We just drop bad requests
				sarscreen.Fprintln(g, "msg", "red_black", "Bad Request:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr))
				continue
			}

			// Create a status to the client to tell it the error or that we have accepted the transfer
			session := binary.BigEndian.Uint32(frame[4:8])
			errcode := reqrxhandler(g, r, remoteAddr)                         // process the request
			stheader := "descriptor=" + sarflags.GetStr(header, "descriptor") // echo the descriptor
			stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,"
			stheader += "errcode=" + errcode

			if errcode != "success" {
				transfer.WriteErrStatus(g, stheader, session, conn, remoteAddr)
				sarscreen.Fprintln(g, "msg", "red_black", "Bad Status:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", session)
			} else {
				var t *transfer.STransfer
				if t = transfer.SMatch(remoteAddr.IP.String(), session); t != nil {
					transfer.WriteStatus(g, t, stheader, conn, remoteAddr)
				}
			}
			continue

		case "data":
			// Handle incoming data
			var d data.Data
			var rxerr error
			if rxerr = d.Get(frame); rxerr != nil {
				session := binary.BigEndian.Uint32(frame[4:8])
				// Bad Packet send back a Status to the client
				stheader := "descriptor=" + sarflags.GetStr(header, "descriptor")
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,errcode=badpacket"
				transfer.WriteErrStatus(g, stheader, session, conn, remoteAddr)
				sarscreen.Fprintln(g, "msg", "red_black", "Bad Data:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", session)
				continue
			}
			session := binary.BigEndian.Uint32(frame[4:8])
			errcode := datrxhandler(g, d, conn, remoteAddr) // process the data
			if errcode != "success" {                       // If we have a error send back a status with it
				stheader := "descriptor=" + sarflags.GetStr(header, "descriptor") // echo the descriptor
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,"
				stheader += "errcode=" + errcode
				// Send back a status to the client to tell it the error or that we have a success with creating the transfer
				var st status.Status
				_ = st.New(stheader, session, 0, 0, nil)
				var wframe []byte
				var txerr error
				if wframe, txerr = st.Put(); txerr == nil {
					conn.WriteToUDP(wframe, remoteAddr)
				}
			}
			continue

		case "metadata":
			// Handle incoming metadata
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
				sarscreen.Fprintln(g, "msg", "red_black", "Bad MetaData:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", session)
				continue
			}
			session := binary.BigEndian.Uint32(frame[4:8])
			errcode := metrxhandler(g, m, remoteAddr) // process the metadata
			if errcode != "success" {                 // If we have a error send back a status with it
				stheader := "descriptor=" + sarflags.GetStr(header, "descriptor") // echo the descriptor
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,"
				stheader += "errcode=" + errcode
				transfer.WriteErrStatus(g, stheader, session, conn, remoteAddr)
				sarscreen.Fprintln(g, "msg", "red_black", "Bad Metadata:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", session)
			}
			continue

		case "status":
			// Handle incoming status
			var s status.Status
			if rxerr := s.Get(frame); rxerr != nil {
				session := binary.BigEndian.Uint32(frame[4:8])
				var se status.Status
				// Bad Packet send back a Status to the client
				stheader := "descriptor=" + sarflags.GetStr(header, "descriptor")
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,"
				stheader += "errcode=badpacket"
				_ = se.New(stheader, session, 0, 0, nil)
				sarscreen.Fprintln(g, "msg", "red_black", "Bad Status:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", session)
				continue
			} // Handle the status
			errcode := starxhandler(g, s, conn, remoteAddr) // process the status
			if errcode != "success" {                       // If we have a error send back a status with it
				stheader := "descriptor=" + sarflags.GetStr(header, "descriptor") // echo the descriptor
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,"
				stheader += "errcode=" + errcode
				transfer.WriteErrStatus(g, stheader, s.Session, conn, remoteAddr)
				sarscreen.Fprintln(g, "msg", "red_black", "Bad Status:", errcode, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", s.Session)
			}
			continue

		default:
			// Bad Packet drop it
			sarscreen.Fprintln(g, "msg", "red_black", "Invalid Saratoga Frame Recieved from ",
				sarnet.UDPinfo(remoteAddr))
		}
	}
}

// Main
func main() {

	var c *sarflags.Cliflags

	if len(os.Args) != 3 {
		fmt.Println("usage:", "saratoga <config> <iface>")
		fmt.Println("Eg. go run saratoga.go saratoga.json en0 (Interface says where to listen for multicast joins")
		return
	}

	// Read in JSON config file and parse it into the Config structure.
	if c = sarflags.ReadConfig(os.Args[1]); c != nil {
		fmt.Println("Cannot open saratoga config file", os.Args[1])
		return
	}
	for xx := range c.Cmds {
		fmt.Println(c.Cmds[xx].Cmd)
	}
	// panic("WE ARE OK DONE!!!!!")

	// Grab my process ID
	// Pid := os.Getpid()

	Cinfo.Prompt = c.Prompt
	Cinfo.Ppad = c.Ppad

	// Move to saratoga working directory
	if err := os.Chdir(c.Sardir); err != nil {
		log.Fatal(fmt.Errorf("no such directory SARDIR=%s", c.Sardir))
	}

	var fs syscall.Statfs_t
	if err := syscall.Statfs(c.Sardir, &fs); err != nil {
		log.Fatal(errors.New("cannot stat saratoga working directory"))
	}

	// Open up V4 & V6 sockets for listening on the Saratoga Port
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
	var err error

	iface, err = net.InterfaceByName(os.Args[2])
	if err != nil {
		fmt.Println("Saratoga Unable to lookup interfacebyname:", os.Args[1])
		log.Fatal(err)
	}
	sarflags.MTU = iface.MTU

	// Set up the gocui interface and start the mainloop
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		fmt.Printf("Cannot run gocui user interface")
		log.Fatal(err)
	}
	defer g.Close()

	g.Cursor = true
	g.SetManagerFunc(layout)
	if err := keybindings(g); err != nil {
		log.Panicln(err)
	}

	// When will we return from listening for v6 frames
	v6listenquit := make(chan error)

	// Listen to Unicast & Multicast v6
	v6mcastcon, err := net.ListenMulticastUDP("udp6", iface, &v6mcastaddr)
	if err != nil {
		sarscreen.Fprintln(g, "msg", "green_black", "Saratoga Unable to Listen on IPv6 Multicast")
		log.Fatal(err)
	} else {
		sarnet.SetMulticastLoop(v6mcastcon, "IPv6")
		go listen(g, v6mcastcon, v6listenquit)
		sarscreen.Fprintln(g, "msg", "green_black", "Saratoga IPv6 Multicast Listener started on",
			sarnet.UDPinfo(&v6mcastaddr))
	}

	// When will we return from listening for v4 frames
	v4listenquit := make(chan error)

	// Listen to Unicast & Multicast v4
	v4mcastcon, err := net.ListenMulticastUDP("udp4", iface, &v4mcastaddr)
	if err != nil {
		sarscreen.Fprintln(g, "msg", "green_black", "Saratoga Unable to Listen on IPv4 Multicast")
		log.Fatal(err)
	} else {
		sarnet.SetMulticastLoop(v4mcastcon, "IPv4")
		go listen(g, v4mcastcon, v4listenquit)
		sarscreen.Fprintln(g, "msg", "green_black", "Saratoga IPv4 Multicast Listener started on",
			sarnet.UDPinfo(&v4mcastaddr))
	}

	sarscreen.Fprintf(g, "msg", "green_black", "Saratoga Directory is %s\n", c.Sardir)
	sarscreen.Fprintf(g, "msg", "green_black", "Available space is %d MB\n",
		(uint64(fs.Bsize)*fs.Bavail)/1024/1024)
	sarscreen.Fprintf(g, "msg", "green_black", "Sizes of Ints is %d\n", sarflags.MaxUint)

	// Show Host Interfaces & Address's
	ifis, _ := net.Interfaces()
	for _, ifi := range ifis {
		if ifi.Name == os.Args[1] || ifi.Name == "lo0" {
			sarscreen.Fprintln(g, "msg", "green_black", ifi.Name, "MTU", ifi.MTU, ifi.Flags.String(), ":")
			adrs, _ := ifi.Addrs()
			for _, adr := range adrs {
				if strings.Contains(adr.Network(), "ip") {
					sarscreen.Fprintln(g, "msg", "green_black", "\t Unicast ", adr.String(), adr.Network())
				}
			}
			madrs, _ := ifi.MulticastAddrs()
			for _, madr := range madrs {
				if strings.Contains(madr.Network(), "ip") {
					sarscreen.Fprintln(g, "msg", "green_black", "\t Multicast ", madr.String(), madr.Network())
				}
			}
		}
	}

	// The Base calling functions for Saratoga live in cli.go so look there first!
	errflag := make(chan error, 1)
	go mainloop(g, errflag, c)

	select {
	case v6err := <-v6listenquit:
		fmt.Println("Saratoga v6 listener has quit:", v6err)
	case v4err := <-v4listenquit:
		fmt.Println("Saratoga v4 lisntener has quit:", v4err)
	case err := <-errflag:
		fmt.Println("Mainloop has quit:", err.Error())
	}
}

// Go routine for command line loop
func mainloop(g *gocui.Gui, done chan error, c *sarflags.Cliflags) {
	var err error

	if err = g.MainLoop(); err != nil && err != gocui.ErrQuit {
		fmt.Printf("%s", err.Error())
	}
	done <- err
}
