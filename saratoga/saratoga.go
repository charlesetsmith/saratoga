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

// Cinfo - Information held on the cmd view
var Cinfo sarwin.Cmdinfo

func setCurrentViewOnTop(g *gocui.Gui, name string) (*gocui.View, error) {
	if _, err := g.SetCurrentView(name); err != nil {
		return nil, err
	}
	_, err := g.SetViewOnTop(name)
	if showpacket {
		g.SetViewOnTop("packet")
	}
	return nil, err
}

// Rotate through the views - CtrlSpace
func switchView(g *gocui.Gui, v *gocui.View) error {
	var err error
	var view string

	switch v.Name() {
	case "cmd":
		view = "msg"
	case "msg":
		if showpacket {
			view = "packet"
		} else {
			view = "err"
		}
	case "err":
		view = "cmd"
	case "packet":
		view = "err"
	}
	if _, err = setCurrentViewOnTop(g, view); err != nil {
		return err
	}
	return nil
}

// Backspace or Delete
func backSpace(g *gocui.Gui, v *gocui.View) error {
	switch v.Name() {
	case "cmd":
		cx, _ := v.Cursor()
		if cx <= promptlen(Cinfo) { // Dont move we are at the prompt
			return nil
		}
		// Delete rune backwards
		v.EditDelete(true)
	case "msg", "packet":
		return nil
	}
	return nil
}

// Handle Left Arrow Move -- All good
func cursorLeft(g *gocui.Gui, v *gocui.View) error {
	switch v.Name() {
	case "cmd":
		cx, cy := v.Cursor()
		if cx <= promptlen(Cinfo) { // Dont move we are at the prompt
			return nil
		}
		// Move back a character
		if err := v.SetCursor(cx-1, cy); err != nil {
			sarwin.ErrPrintln(g, "white_black", v.Name(), "LeftArrow:", "cx=", cx, "cy=", cy, "error=", err)
		}
	case "msg", "packet":
		return nil
	}
	return nil
}

// Handle Right Arrow Move - All good
func cursorRight(g *gocui.Gui, v *gocui.View) error {
	switch v.Name() {
	case "cmd":
		cx, cy := v.Cursor()
		line, _ := v.Line(cy)
		if cx >= len(line)-1 { // We are at the end of line do nothing
			v.SetCursor(len(line), cy)
			return nil
		}
		// Move forward a character
		if err := v.SetCursor(cx+1, cy); err != nil {
			sarwin.ErrPrintln(g, "white_red", "RightArrow:", "cx=", cx, "cy=", cy, "error=", err)
		}
	case "msg", "packet":
		return nil
	}
	return nil
}

// Handle down cursor -- All good!
func cursorDown(g *gocui.Gui, v *gocui.View) error {
	ox, oy := v.Origin()
	cx, cy := v.Cursor()
	// Don't move down if we are at the last line in current views Bufferlines
	if oy+cy == len(v.BufferLines())-1 {
		sarwin.ErrPrintf(g, "white_black", "%s Down oy=%d cy=%d lines=%d\n",
			v.Name(), oy, cy, len(v.BufferLines()))
		return nil
	}
	if err := v.SetCursor(cx, cy+1); err != nil {
		sarwin.ErrPrintf(g, "magenta_black", "%s Down oy=%d cy=%d lines=%d err=%s\n",
			v.Name(), oy, cy, len(v.BufferLines()), err.Error())
		// ox, oy = v.Origin()
		if err := v.SetOrigin(ox, oy+1); err != nil {
			sarwin.ErrPrintf(g, "cyan_black", "%s Down oy=%d cy=%d lines=%d err=%s\n",
				v.Name(), oy, cy, len(v.BufferLines()), err.Error())
			return err
		}
	}
	sarwin.ErrPrintf(g, "green_black", "%s Down oy=%d cy=%d lines=%d\n",
		v.Name(), oy, cy, len(v.BufferLines()))
	return nil
}

func cursorUp(g *gocui.Gui, v *gocui.View) error {
	ox, oy := v.Origin()
	cx, cy := v.Cursor()
	if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
		sarwin.ErrPrintf(g, "magenta_black", "%s Up oy=%d cy=%d lines=%d err=%s\n",
			v.Name(), oy, cy, len(v.BufferLines()), err.Error())
		if err := v.SetOrigin(ox, oy-1); err != nil {
			sarwin.ErrPrintf(g, "cyan_black", "%s Up oy=%d cy=%d lines=%d err=%s\n",
				v.Name(), oy, cy, len(v.BufferLines()), err.Error())
			return err
		}
	}
	_, cy = v.Cursor()
	sarwin.ErrPrintf(g, "green_black", "%s Up oy=%d cy=%d lines=%d\n",
		v.Name(), oy, cy, len(v.BufferLines()))
	return nil
}

/*
// Handle up cursor
func oldcursorUp(g *gocui.Gui, v *gocui.View) error {
	ox, oy := v.Origin()
	cx, cy := v.Cursor()
	err := v.SetCursor(cx, cy-1)
	if err != nil && oy > 0 { // Reset the origin
		if err := v.SetOrigin(ox, oy-1); err != nil { // changed ox to 0
			sarwin.MsgPrintf(g, "white_red", "SetOrigin error=%s", err)
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
*/

// Return the length of the prompt
func promptlen(v sarwin.Cmdinfo) int {
	return len(v.Prompt) + len(strconv.Itoa(v.Curline)) + v.Ppad
}

// Display the prompt
func prompt(g *gocui.Gui, v *gocui.View) {
	if g == nil || v == nil || v.Name() != "cmd" {
		log.Fatal("prompt must be in cmd view")
	}
	_, oy := v.Origin()
	_, cy := v.Cursor()
	// Only display it if it is on the next new line
	if oy+cy == Cinfo.Curline {
		if FirstPass { // Just the prompt no precedin \n as we are the first line
			sarwin.CmdPrintf(g, "yellow_black", "%s[%d]:", Cinfo.Prompt, Cinfo.Curline)
			v.SetCursor(promptlen(Cinfo), cy)
		} else { // End the last command by going to new lin \n then put up the new prompt
			Cinfo.Curline++
			sarwin.CmdPrintf(g, "yellow_black", "\n%s[%d]:", Cinfo.Prompt, Cinfo.Curline)
			_, cy := v.Cursor()
			v.SetCursor(promptlen(Cinfo), cy)
			if err := cursorDown(g, v); err != nil {
				sarwin.MsgPrintln(g, "red_black", "Cannot move to next line")
			}
			_, cy = v.Cursor()
			v.SetCursor(promptlen(Cinfo), cy+1)
		}
	}
}

// Commands - cli commands entered
var Commands []string

// Sarwg - Wait group for commands to run/finish - We dont quit till this is 0
var Sarwg sync.WaitGroup

// This is where we process command line inputs after a CR entered
func getLine(g *gocui.Gui, v *gocui.View) error {
	if g == nil || v == nil {
		log.Fatal("getLine - g or v is nil")
	}
	switch v.Name() {
	case "cmd":
		c := Cmdptr
		// Find out where we are
		_, cy := v.Cursor()
		// Get the line
		line, _ := v.Line(cy)

		command := strings.SplitN(line, ":", 2)
		if command[1] == "" { // We have just hit enter - do nothing
			return nil
		}
		// Save the command into history
		Cinfo.Commands = append(Cinfo.Commands, command[1])

		// Spawn a go to run the command
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
		prompt(g, v)
	case "msg", "packet", "err":
		return cursorDown(g, v)
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

// ShowPacket - Show Packet trace info
var showpacket bool = false

// Turn on/off the Packet View
func showPacket(g *gocui.Gui, v *gocui.View) error {
	var err error

	if g == nil || v == nil {
		log.Fatal("showPacket g is nil")
	}
	showpacket = !showpacket
	if showpacket {
		_, err = g.SetViewOnTop("packet")
	} else {
		_, err = g.SetViewOnTop("msg")
	}
	return err
}

// Bind keys to function handlers
func keybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("", gocui.KeyCtrlSpace, gocui.ModNone, switchView); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowLeft, gocui.ModNone, cursorLeft); err != nil {
		return nil
	}
	if err := g.SetKeybinding("", gocui.KeyArrowRight, gocui.ModNone, cursorRight); err != nil {
		return nil
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlP, gocui.ModNone, showPacket); err != nil {
		return nil
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, getLine); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyBackspace, gocui.ModNone, backSpace); err != nil {
		return nil
	}
	if err := g.SetKeybinding("", gocui.KeyBackspace2, gocui.ModNone, backSpace); err != nil {
		return nil
	}
	if err := g.SetKeybinding("", gocui.KeyDelete, gocui.ModNone, backSpace); err != nil {
		return nil
	}
	return nil
}

// FirstPass -- First time around layout we don;t put \n at end of prompt
var FirstPass = true

func layout(g *gocui.Gui) error {
	var err error
	var cmd *gocui.View
	var msg *gocui.View
	var packet *gocui.View

	ratio := 4 // Ratio of cmd to msg views

	// Maximum size of x and y
	maxx, maxy := g.Size()
	// This is the command line input view -- cli inputs and return messages go here
	if cmd, err = g.SetView("cmd", 0, maxy-(maxy/ratio)+1, maxx/2-1, maxy-1); err != nil {
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
		cmd.Autoscroll = true
	}
	// This is the error msg view -- mic errors go here
	if cmd, err = g.SetView("err", maxx/2, maxy-(maxy/ratio)+1, maxx-1, maxy-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		cmd.Title = "Errors"
		cmd.Highlight = false
		cmd.BgColor = gocui.ColorBlack
		cmd.FgColor = gocui.ColorGreen
		cmd.Editable = false
		cmd.Overwrite = false
		cmd.Wrap = true
		cmd.Autoscroll = true
	}
	// This is the packet trace window - packet trace history goes here
	// Toggles on/off with CtrlP
	if packet, err = g.SetView("packet", maxx-maxx/4, 1, maxx-2, maxy-maxy/ratio-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		packet.Title = "Packets"
		packet.Highlight = false
		packet.BgColor = gocui.ColorBlack
		packet.FgColor = gocui.ColorMagenta
		packet.Editable = false
		packet.Wrap = true
		packet.Overwrite = false
		packet.Autoscroll = true
	}

	// This is the message view window - Status & error messages go here
	if msg, err = g.SetView("msg", 0, 0, maxx-1, maxy-maxy/ratio); err != nil {
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

	// Display the prompt without the \n first time around
	if FirstPass {
		// All inputs happen via the cmd view
		if cmd, err = g.SetCurrentView("cmd"); err != nil {
			return err
		}
		cmd.SetCursor(0, 0)
		Cinfo.Curline = 0
		cmdv, _ := g.View("cmd")
		prompt(g, cmdv)
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
				sarwin.MsgPrintln(g, "yellow_black", "Created Request", reqtype, "from",
					sarnet.UDPinfo(remoteAddr),
					"session", r.Session)
				return "success"
			}
			sarwin.MsgPrintln(g, "red_black", "Cannot create Request", reqtype, "from",
				sarnet.UDPinfo(remoteAddr),
				"session", r.Session, err)
			return "badrequest"
		}
		// Request is currently in progress
		sarwin.MsgPrintln(g, "red_black", "Request", reqtype, "from",
			sarnet.UDPinfo(remoteAddr),
			"for session", r.Session, "already in progress")
		return "badrequest"
	default:
		sarwin.MsgPrintln(g, "red_black", "Invalid Request from",
			sarnet.UDPinfo(remoteAddr),
			"session", r.Session)
		return "badrequest"
	}
}

// Metadata handler for Server
func metrxhandler(g *gocui.Gui, m metadata.MetaData, remoteAddr *net.UDPAddr) string {
	// Handle the metadata
	var t *transfer.STransfer
	if t = transfer.SMatch(remoteAddr.IP.String(), m.Session); t != nil {
		if err := t.SChange(g, m); err != nil { // Size of file has changed!!!
			return "unspecified"
		}
		sarwin.MsgPrintln(g, "yellow_black", "Changed Transfer", m.Session, "from",
			sarnet.UDPinfo(remoteAddr))
		return "success"
	}
	// Request is currently in progress
	sarwin.MsgPrintln(g, "red_black", "Metadata received for no such transfer as", m.Session, "from",
		sarnet.UDPinfo(remoteAddr))
	return "badpacket"
}

// Data handler for server
func datrxhandler(g *gocui.Gui, d data.Data, conn *net.UDPConn, remoteAddr *net.UDPAddr) string {
	// Handle the data
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
		sarwin.MsgPrintln(g, "yellow_black", "Server Recieved Data Len:", len(d.Payload), "Pos:", d.Offset)
		transfer.Strmu.Unlock()
		sarwin.MsgPrintln(g, "yellow_black", "Changed Transfer", d.Session, "from",
			sarnet.UDPinfo(remoteAddr))
		return "success"
	}
	// No transfer is currently in progress
	sarwin.MsgPrintln(g, "red_black", "Data received for no such transfer as", d.Session, "from",
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
			sarwin.MsgPrintln(g, "red_black", "Sarataga listener failed:", err)
			quit <- err
			return
		}
		sarwin.MsgPrintln(g, "green_black", "Listen read", framelen, "bytes from", remoteAddr.String())

		// Very basic frame checks before we get into what it is
		if framelen < 8 {
			// Saratoga packet too small
			sarwin.MsgPrintln(g, "red_black", "Rx Saratoga Frame too short from ",
				sarnet.UDPinfo(remoteAddr))
			continue
		}
		if framelen > maxframesize {
			sarwin.MsgPrintln(g, "red_black", "Rx Saratoga Frame too long", framelen,
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
			sarwin.MsgPrintln(g, "red_black", "Header is not Saratoga v1")
			if _, err = sarflags.Set(0, "errno", "badpacket"); err != nil {
				// Bad Packet send back a Status to the client
				var se status.Status
				sinfo := status.Sinfo{Session: 0, Progress: 0, Inrespto: 0, Holes: nil}
				if frames.New(&se, "errcode=badpacket", &sinfo) != nil {
					sarwin.MsgPrintln(g, "red_black", "Cannot create badpacket status")
				}
				sarwin.MsgPrintln(g, "red_black", "Not Saratoga Version 1 Frame from ",
					sarnet.UDPinfo(remoteAddr))
			}
			continue
		}
		sarwin.MsgPrintln(g, "white_black", "Received: ", sarflags.GetStr(header, "frametype"))

		// Process the frame
		switch sarflags.GetStr(header, "frametype") {
		case "beacon":
			var rxb beacon.Beacon
			if rxerr := frames.Decode(&rxb, frame); rxerr != nil {
				// if rxerr := rxb.Decode(frame); rxerr != nil {
				// We just drop bad beacons
				sarwin.MsgPrintln(g, "red_black", "Bad Beacon:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr))
				continue
			}
			// Handle the beacon
			if errcode := rxb.Handler(g, remoteAddr); errcode != "success" {
				sarwin.MsgPrintln(g, "red_black", "Bad Beacon:", errcode, " from ",
					sarnet.UDPinfo(remoteAddr))
				continue
			}
			sarwin.PacketPrintln(g, "green_black", "Rx", rxb.ShortPrint())

		case "request":
			// Handle incoming request
			var r request.Request
			var rxerr error
			if rxerr = frames.Decode(&r, frame); rxerr != nil {
				// We just drop bad requests
				sarwin.MsgPrintln(g, "red_black", "Bad Request:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr))
				continue
			}
			sarwin.PacketPrintln(g, "green_black", "Rx", r.ShortPrint())

			// Create a status to the client to tell it the error or that we have accepted the transfer
			session := binary.BigEndian.Uint32(frame[4:8])
			errcode := reqrxhandler(g, r, remoteAddr)                         // process the request
			stheader := "descriptor=" + sarflags.GetStr(header, "descriptor") // echo the descriptor
			stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,"
			stheader += "errcode=" + errcode

			if errcode != "success" {
				transfer.WriteErrStatus(g, stheader, session, conn, remoteAddr)
				sarwin.MsgPrintln(g, "red_black", "Bad Status:", rxerr, " from ",
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
			sarwin.MsgPrintln(g, "white_black", "Data frame length is:", len(frame))
			if rxerr = frames.Decode(&d, frame); rxerr != nil {
				session := binary.BigEndian.Uint32(frame[4:8])
				// Bad Packet send back a Status to the client
				stheader := "descriptor=" + sarflags.GetStr(header, "descriptor")
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,errcode=badpacket"
				transfer.WriteErrStatus(g, stheader, session, conn, remoteAddr)
				sarwin.MsgPrintln(g, "red_black", "Bad Data:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", session)
				continue
			}
			sarwin.PacketPrintln(g, "green_black", "Rx", d.ShortPrint())

			// sarwin.MsgPrintln(g, "white_black", "Decoded data ", d.Print())
			session := binary.BigEndian.Uint32(frame[4:8])
			errcode := datrxhandler(g, d, conn, remoteAddr) // process the data
			if errcode != "success" {                       // If we have a error send back a status with it
				stheader := "descriptor=" + sarflags.GetStr(header, "descriptor") // echo the descriptor
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,"
				stheader += "errcode=" + errcode
				// Send back a status to the client to tell it the error or that we have a success with creating the transfer
				var st status.Status
				sinfo := status.Sinfo{Session: session, Progress: 0, Inrespto: 0, Holes: nil}
				if frames.New(&st, stheader, &sinfo) != nil {
					sarwin.MsgPrintln(g, "red_black", "Cannot asemble status")
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
					sarwin.MsgPrintln(g, "red_black", "Cannot assemble status")
				}
				sarwin.MsgPrintln(g, "red_black", "Bad MetaData:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", session)
				continue
			}
			sarwin.PacketPrintln(g, "green_black", "Rx", m.ShortPrint())

			session := binary.BigEndian.Uint32(frame[4:8])
			errcode := metrxhandler(g, m, remoteAddr) // process the metadata
			if errcode != "success" {                 // If we have a error send back a status with it
				stheader := "descriptor=" + sarflags.GetStr(header, "descriptor") // echo the descriptor
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,"
				stheader += "errcode=" + errcode
				transfer.WriteErrStatus(g, stheader, session, conn, remoteAddr)
				sarwin.MsgPrintln(g, "red_black", "Bad Metadata:", rxerr, " from ",
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
					sarwin.MsgPrintln(g, "red_black", "Cannot assemble status")
				}
				sarwin.MsgPrintln(g, "red_black", "Bad Status:", rxerr, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", session)
				continue
			} // Handle the status
			sarwin.PacketPrintln(g, "green_black", "Rx", s.ShortPrint())
			errcode := starxhandler(g, s, conn, remoteAddr) // process the status
			if errcode != "success" {                       // If we have a error send back a status with it
				stheader := "descriptor=" + sarflags.GetStr(header, "descriptor") // echo the descriptor
				stheader += ",metadatarecvd=no,allholes=yes,reqholes=requested,"
				stheader += "errcode=" + errcode
				transfer.WriteErrStatus(g, stheader, s.Session, conn, remoteAddr)
				sarwin.MsgPrintln(g, "red_black", "Bad Status:", errcode, " from ",
					sarnet.UDPinfo(remoteAddr), " session ", s.Session)
			}
			continue

		default:
			// Bad Packet drop it
			sarwin.MsgPrintln(g, "red_black", "Invalid Saratoga Frame Recieved from ",
				sarnet.UDPinfo(remoteAddr))
		}
	}
}

var Cmdptr *sarflags.Cliflags

// Main
func main() {

	// var c *sarflags.Cliflags

	if len(os.Args) != 3 {
		fmt.Println("usage:", "saratoga <config> <iface>")
		fmt.Println("Eg. go run saratoga.go saratoga.json en0 (Interface says where to listen for multicast joins")
		return
	}

	// The Command line interface commands, help & usage to be read from saratoga.json
	Cmdptr = new(sarflags.Cliflags)

	// Read in JSON config file and parse it into the Config structure.
	if err := sarflags.ReadConfig(os.Args[1], Cmdptr); err != nil {
		fmt.Println("Cannot open saratoga config file we have a Readconf error", os.Args[1], err)
		return
	}

	// Grab my process ID
	// Pid := os.Getpid()

	// Set global variable pointing to sarflags.Cliflags structure
	// We have to have a global pointer as we cannot pass c directly into gocui

	Cinfo.Prompt = Cmdptr.Prompt
	Cinfo.Ppad = Cmdptr.Ppad

	// Move to saratoga working directory
	if err := os.Chdir(Cmdptr.Sardir); err != nil {
		log.Fatal(fmt.Errorf("no such directory SARDIR=%s", Cmdptr.Sardir))
	}

	var fs syscall.Statfs_t
	if err := syscall.Statfs(Cmdptr.Sardir, &fs); err != nil {
		log.Fatal(errors.New("cannot stat saratoga working directory"))
	}

	// What Interface are we receiving Multicasts on
	var iface *net.Interface
	var err error

	iface, err = net.InterfaceByName(os.Args[2])
	if err != nil {
		fmt.Println("Saratoga Unable to lookup interfacebyname:", os.Args[1])
		log.Fatal(err)
	}
	// Set the Mtu to Interface we are using
	sarflags.MtuSet(iface.MTU)

	// Set up the gocui interface and start the mainloop
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		fmt.Printf("Cannot run gocui user interface")
		log.Fatal(err)
	}
	defer g.Close()

	g.Cursor = true
	g.Highlight = true
	g.SelFgColor = gocui.ColorRed
	g.SelBgColor = gocui.ColorWhite
	g.SetManagerFunc(layout)
	if err := keybindings(g); err != nil {
		log.Panicln(err)
	}

	// Show Host Interfaces & Address's
	ifis, ierr := net.Interfaces()
	if ierr != nil {
		sarwin.MsgPrintln(g, "green_black", ierr.Error())
	}
	for _, ifi := range ifis {
		if ifi.Name == os.Args[2] { // || ifi.Name == "lo0" {
			sarwin.MsgPrintln(g, "green_black", ifi.Name, "MTU", ifi.MTU, ifi.Flags.String(), ":")
			adrs, _ := ifi.Addrs()
			for _, adr := range adrs {
				if strings.Contains(adr.Network(), "ip") {
					sarwin.MsgPrintln(g, "green_black", "\t Unicast ", adr.String(), adr.Network())

				}
			}
			/*
				madrs, _ := ifi.MulticastAddrs()
				for _, madr := range madrs {
					if strings.Contains(madr.Network(), "ip") {
						sarwin.MsgPrintln(g, "green_black", "\t Multicast ", madr.String(),
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
		log.Println("Saratoga Unable to Listen on IPv6 Multicast", v6mcastaddr.IP, v6mcastaddr.Port)
		log.Fatal(err)
	} else {
		if err := sarnet.SetMulticastLoop(v6mcastcon, "IPv6"); err != nil {
			log.Fatal(err)
		}
		go listen(g, v6mcastcon, v6listenquit)
		sarwin.MsgPrintln(g, "green_black", "Saratoga IPv6 Multicast Listener started on",
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
		log.Println("Saratoga Unable to Listen on IPv4 Multicast", v4mcastaddr.IP, v4mcastaddr.Port)
		log.Fatal(err)
	} else {
		if err := sarnet.SetMulticastLoop(v4mcastcon, "IPv4"); err != nil {
			log.Fatal(err)
		}
		go listen(g, v4mcastcon, v4listenquit)
		sarwin.MsgPrintln(g, "green_black", "Saratoga IPv4 Multicast Listener started on",
			sarnet.UDPinfo(&v4mcastaddr))
	}

	// v4unicastcon, err := net.ListenUDP("udp4", iface, &v4addr)
	sarwin.MsgPrintf(g, "green_black", "Saratoga Directory is %s\n", Cmdptr.Sardir)
	sarwin.MsgPrintf(g, "green_black", "Available space is %d MB\n",
		(uint64(fs.Bsize)*fs.Bavail)/1024/1024)

	/*
		sarwin.MsgPrintln(g, "green_black", "MaxInt=", sarflags.MaxInt)
		sarwin.MsgPrintln(g, "green_black", "MaxUint=", sarflags.MaxUint)
		sarwin.MsgPrintln(g, "green_black", "MaxInt16=", sarflags.MaxInt16)
		sarwin.MsgPrintln(g, "green_black", "MaxUint16=", sarflags.MaxUint16)
		sarwin.MsgPrintln(g, "green_black", "MaxInt32=", sarflags.MaxInt32)
		sarwin.MsgPrintln(g, "green_black", "MaxUint32=", sarflags.MaxUint32)
		sarwin.MsgPrintln(g, "green_black", "MaxInt64=", sarflags.MaxInt64)
		sarwin.MsgPrintln(g, "green_black", "MaxUint64=", sarflags.MaxUint64)
	*/

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
