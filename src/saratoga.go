// Saratoga Interactive Client

package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"cli"
	"screen"

	"github.com/charlesetsmith/saratoga/src/sarflags"
	"github.com/charlesetsmith/saratoga/src/sarnet"
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
	screen.Fprintf(g, "msg", "magenta_black", "CurLine=%d <%s>\n", cli.CurLine, command[1])
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
	screen.Fprintln(g, "msg", "yellow_black", "MaxY=", MaxY, "Number Cmd View Lines=", CmdLines)
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

func main() {

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
	var en0 *net.Interface
	var err error

	en0, err = net.InterfaceByName("en0")
	if err != nil {
		log.Fatal(err)
	}

	// Listen to Unicast & Multicast
	v6mcastcon, err := net.ListenMulticastUDP("udp6", en0, &v6mcastaddr)
	if err != nil {
		fmt.Println("Saratoga Unable to Listen on IPv6 Multicast")
		panic(err)
	} else {
		setMulticastLoop(v6mcastcon, "IPv6")
		go listen(v6mcastcon, quit)
		fmt.Println("Saratoga IPv6 Multicast Listener started on", sarnet.UDPinfo(&v6mcastaddr))
	}

	v4mcastcon, err := net.ListenMulticastUDP("udp4", en0, &v4mcastaddr)
	if err != nil {
		fmt.Println("Saratoga Unable to Listen on IPv4 Multicast")
		panic(err)
	} else {
		setMulticastLoop(v4mcastcon, "IPv4")
		go listen(v4mcastcon, quit)
		fmt.Println("Saratoga IPv4 Multicast Listener started on", sarnet.UDPinfo(&v4mcastaddr))
	}

	quit := make(chan struct{})

	time.Sleep(2 * time.Second)

	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.Cursor = true

	g.SetManagerFunc(layout)

	if err := keybindings(g); err != nil {
		log.Panicln(err)
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}
