/*
 * Handle screen outputs for views in colours for Saratoga
 * We have msg,error,cli and packet windows
 */

package sarwin

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/jroimartin/gocui"

	"github.com/charlesetsmith/saratoga/beacon"
	"github.com/charlesetsmith/saratoga/cmds"
	"github.com/charlesetsmith/saratoga/frames"
	"github.com/charlesetsmith/saratoga/sarflags"
)

// Ansi Colour Escape Sequences
var ansiprefix = "\033["
var ansipostfix = "m"
var ansiseparator = ";"
var ansioff = "\033[0m" // Turn ansii escape sequence off

// Foreground colours (b=bright)
var fg = map[string]string{
	// Normal
	"black":   "30;1",
	"red":     "31;1",
	"green":   "32;1",
	"yellow":  "33;1",
	"blue":    "34;1",
	"magenta": "35;1",
	"cyan":    "36;1",
	"white":   "37;1",
	// Underlined
	"ublack":   "30;4",
	"ured":     "31;4",
	"ugreen":   "32;4",
	"uyellow":  "33;4",
	"ublue":    "34;4",
	"umagenta": "35;4",
	"ucyan":    "36;4",
	"uwhite":   "37;4",
	// Invert
	"iblack":   "30;7",
	"ired":     "31;7",
	"igreen":   "32;7",
	"iyellow":  "33;7",
	"iblue":    "34;7",
	"imagenta": "35;7",
	"icyan":    "36;7",
	"iwhite":   "37;7",
}

// Background Colours (b=bright)
var bg = map[string]string{
	// Normal background
	"black":   "40;1",
	"red":     "41;1",
	"green":   "42;1",
	"yellow":  "43;1",
	"blue":    "44;1",
	"magenta": "45;1",
	"cyan":    "46;1",
	"white":   "47;1",
	// Bright background (this just makes foreground lighter)
	"bblack":   "40;4",
	"bred":     "41;4",
	"bgreen":   "42;4",
	"byellow":  "43;4",
	"bblue":    "44;4",
	"bmagenta": "45;4",
	"bcyan":    "46;4",
	"bwhite":   "47;4",
}

// Ensure multiple prints to screen don't interfere with eachother
var ViewMu sync.Mutex

// Viewinfo -- Data and info on views (cmd & msg)
type Cmdinfo struct {
	Commands []string // History of commands
	Prompt   string   // Command line prompt prefix
	Curline  int      // What is my current line #
	Ppad     int      // Number of pad characters around prompt e.g. prompt[99]: would be 3 for the []:
	Numlines int      // How many lines do we have
}

// Cinfo - Information held on the cmd view
var Cinfo Cmdinfo

// Send formatted output to "msg"  window
func MsgPrintf(g *gocui.Gui, colour string, format string, args ...interface{}) {
	fprintf(g, "msg", colour, format, args...)
}

// Send unformatted output to "msg" window
func MsgPrintln(g *gocui.Gui, colour string, args ...interface{}) {
	fprintln(g, "msg", colour, args...)
}

// Send formatted output to "cmd" window
func CmdPrintf(g *gocui.Gui, colour string, format string, args ...interface{}) {
	fprintf(g, "cmd", colour, format, args...)
}

// Send unformatted output to "cmd" window
func CmdPrintln(g *gocui.Gui, colour string, args ...interface{}) {
	fprintln(g, "cmd", colour, args...)
}

// Send formatted output to "err" window
func ErrPrintf(g *gocui.Gui, colour string, format string, args ...interface{}) {
	fprintf(g, "err", colour, format, args...)
}

// Send unformatted output to "err" window
func ErrPrintln(g *gocui.Gui, colour string, args ...interface{}) {
	fprintln(g, "err", colour, args...)
}

// Send formatted output to "packet" window
func PacketPrintf(g *gocui.Gui, colour string, format string, args ...interface{}) {
	fprintf(g, "packet", colour, format, args...)
}

// Send unformatted output to "packet" window
func PacketPrintln(g *gocui.Gui, colour string, args ...interface{}) {
	fprintln(g, "packet", colour, args...)
}

// FirstPass -- First time around layout we don;t put \n at end of prompt
var FirstPass = true

func Layout(g *gocui.Gui) error {
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
		cmd.Autoscroll = false
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
		cmd.Autoscroll = false
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
		packet.Autoscroll = false
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
		msg.Autoscroll = false
	}

	// Display the prompt without the \n first time around
	if FirstPass {
		g.Cursor = true
		g.Highlight = true
		g.SelFgColor = gocui.ColorRed
		g.SelBgColor = gocui.ColorWhite
		// All inputs happen via the cmd view
		if cmd, err = g.SetCurrentView("cmd"); err != nil {
			return err
		}
		cmd.SetCursor(0, 0)
		Cinfo.Curline = 0
		cmdv, _ := g.View("cmd")
		Prompt(g, cmdv)
		FirstPass = false
	}
	return nil
}

// Create ansi sequence for colour change with c format of fg_bg (e.g. red_black)
func setcolour(colour string) string {
	if colour == "none" || colour == "" {
		return ""
	}
	if colour == "off" {
		return ansioff
	}
	sequence := strings.Split(colour, "_")

	switch len(sequence) {
	case 2: // fg & bg
		if fg[sequence[0]] != "" && bg[sequence[1]] != "" {
			return ansiprefix + fg[sequence[0]] + ansiseparator + bg[sequence[1]] + ansipostfix
		}
		return ansiprefix + fg["white"] + ansiseparator + bg["red"] + ansipostfix
	case 1: // fg and "black" bg
		if fg[sequence[0]] != "" {
			return ansiprefix + fg[sequence[0]] + ansiseparator + bg["black"] + ansipostfix
		}
		return ansiprefix + fg[sequence[0]] + ansiseparator + bg[sequence[1]] + ansipostfix
	default: // Woops wrong fg or bg so scream at the screen
		// Error so make it jump out at us
		return ansiprefix + fg["white"] + ansiseparator + bg["bred"] + ansipostfix
	}
}

// fprintf out an ANSII escape sequence in colour to view
// If colour is undefined then still print it out but in bright red to show there is an issue
func fprintf(g *gocui.Gui, vname string, colour string, format string, args ...interface{}) {

	g.Update(func(g *gocui.Gui) error {
		ViewMu.Lock()
		defer ViewMu.Unlock()
		v, err := g.View(vname)
		if err != nil {
			e := fmt.Sprintf("\nView Fprintf invalid view: %s", vname)
			log.Fatal(e)
		}
		// gotolastrow(g, v)

		s := setcolour(colour)
		s += fmt.Sprintf(format, args...)
		if colour != "" {
			s += setcolour("off")
		}
		fmt.Fprint(v, s)
		return nil
	})
}

// Fprintln out in ANSII escape sequence in colour to view
// If colour is undefined then still print it out but in bright red to show there is an issue
func fprintln(g *gocui.Gui, vname string, colour string, args ...interface{}) {

	g.Update(func(g *gocui.Gui) error {
		ViewMu.Lock()
		defer ViewMu.Unlock()
		v, err := g.View(vname)
		if err != nil {
			e := fmt.Sprintf("\nView Fprintln invalid view: %s", vname)
			log.Fatal(e)
		}
		// gotolastrow(g, v)

		s := setcolour(colour)
		s += fmt.Sprint(args...)
		if colour != "" {
			s += setcolour("off")
		}
		fmt.Fprintln(v, s)
		return nil
	})
}

func setCurrentViewOnTop(g *gocui.Gui, name string) (*gocui.View, error) {
	var err error
	var v *gocui.View

	if v, err = g.SetCurrentView(name); err != nil {
		return v, err
	}
	if showpacket {
		return g.SetViewOnTop("packet")
	} else {
		return g.SetViewOnTop(name)
	}
}

// Jump to the last row in a view
func gotolastrow(g *gocui.Gui, v *gocui.View) {
	ox, oy := v.Origin()
	cx, cy := v.Cursor()

	lines := len(v.BufferLines())
	ErrPrintf(g, "white_black", "gotolastrow ox=%d oy=%d cx=%d cy=%d blines=%d\n",
		oy, oy, cx, cy, lines)
	// Don't move down if we already are at the last line in current views Bufferlines
	if oy+cy == lines-1 {
		return
	}
	if err := v.SetCursor(cx, lines-1); err != nil {
		_, sy := v.Size()
		v.SetOrigin(ox, lines-sy-1)
		v.SetCursor(cx, sy-1)
	}
}

// Rotate through the views - CtrlSpace
func switchView(g *gocui.Gui, v *gocui.View) error {
	var err error
	var view string
	var newv *gocui.View

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
	if newv, err = setCurrentViewOnTop(g, view); err != nil {
		return err
	}
	gotolastrow(g, newv)
	return nil
}

// Backspace or Delete
func BackSpace(g *gocui.Gui, v *gocui.View) error {
	switch v.Name() {
	case "cmd":
		cx, _ := v.Cursor()
		if cx <= promptlen(Cinfo) { // Dont move we are at the prompt
			return nil
		}
		// Delete rune backwards
		v.EditDelete(true)
	case "msg", "err", "packet":
		return nil
	}
	return nil
}

// Handle Left Arrow Move -- All good
func CursorLeft(g *gocui.Gui, v *gocui.View) error {
	switch v.Name() {
	case "cmd":
		cx, cy := v.Cursor()
		if cx <= promptlen(Cinfo) { // Dont move we are at the prompt
			return nil
		}
		// Move back a character
		if err := v.SetCursor(cx-1, cy); err != nil {
			return nil
			// ErrPrintln(g, "white_black", v.Name(), "LeftArrow:", "cx=", cx, "cy=", cy, "error=", err)
		}
	case "msg", "packet", "err":
		return nil
	}
	return nil
}

// Handle Right Arrow Move - All good
func CursorRight(g *gocui.Gui, v *gocui.View) error {
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
			return nil
			// ErrPrintln(g, "white_red", "RightArrow:", "cx=", cx, "cy=", cy, "error=", err)
		}
	case "msg", "packet", "err":
		return nil
	}
	return nil
}

// Handle down cursor -- All good!
func CursorDown(g *gocui.Gui, v *gocui.View) error {
	ox, oy := v.Origin()
	cx, cy := v.Cursor()
	// Don't move down if we are at the last line in current views Bufferlines
	if oy+cy == len(v.BufferLines())-1 {
		//ErrPrintf(g, "white_black", "%s Down oy=%d cy=%d lines=%d\n",
		//	v.Name(), oy, cy, len(v.BufferLines()))
		return nil
	}
	if err := v.SetCursor(cx, cy+1); err != nil {
		// ErrPrintf(g, "magenta_black", "%s Down oy=%d cy=%d lines=%d err=%s\n",
		//	v.Name(), oy, cy, len(v.BufferLines()), err.Error())
		// ox, oy = v.Origin()
		if err := v.SetOrigin(ox, oy+1); err != nil {
			// ErrPrintf(g, "cyan_black", "%s Down oy=%d cy=%d lines=%d err=%s\n",
			//	v.Name(), oy, cy, len(v.BufferLines()), err.Error())
			return err
		}
	}
	// ErrPrintf(g, "green_black", "%s Down oy=%d cy=%d lines=%d\n",
	//	v.Name(), oy, cy, len(v.BufferLines()))
	return nil
}

// Handle up cursor -- All good!
func CursorUp(g *gocui.Gui, v *gocui.View) error {
	ox, oy := v.Origin()
	cx, cy := v.Cursor()
	if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
		// ErrPrintf(g, "magenta_black", "%s Up oy=%d cy=%d lines=%d err=%s\n",
		// v.Name(), oy, cy, len(v.BufferLines()), err.Error())
		if err := v.SetOrigin(ox, oy-1); err != nil {
			// ErrPrintf(g, "cyan_black", "%s Up oy=%d cy=%d lines=%d err=%s\n",
			// v.Name(), oy, cy, len(v.BufferLines()), err.Error())
			return err
		}
	}
	//_, cy = v.Cursor()
	// ErrPrintf(g, "green_black", "%s Up oy=%d cy=%d lines=%d\n",
	//	v.Name(), oy, cy, len(v.BufferLines()))
	return nil
}

// Return the length of the prompt
func promptlen(v Cmdinfo) int {
	return len(v.Prompt) + len(strconv.Itoa(v.Curline)) + v.Ppad
}

// Cprompt - Command line prompt
// var Cprompt = "saratoga" // If not set in saratoga.json set it to saratoga

// Display the prompt
func Prompt(g *gocui.Gui, v *gocui.View) {
	if g == nil || v == nil || v.Name() != "cmd" {
		log.Fatal("prompt must be in cmd view")
	}
	_, oy := v.Origin()
	_, cy := v.Cursor()
	// Only display it if it is on the next new line
	if oy+cy == Cinfo.Curline {
		if FirstPass { // Just the prompt no precedin \n as we are the first line
			CmdPrintf(g, "yellow_black", "%s[%d]:", Cinfo.Prompt, Cinfo.Curline)
			v.SetCursor(promptlen(Cinfo), cy)
		} else { // End the last command by going to new lin \n then put up the new prompt
			Cinfo.Curline++
			CmdPrintf(g, "yellow_black", "\n%s[%d]:", Cinfo.Prompt, Cinfo.Curline)
			_, cy := v.Cursor()
			v.SetCursor(promptlen(Cinfo), cy)
			if err := CursorDown(g, v); err != nil {
				ErrPrintln(g, "red_black", "Cannot move to next line")
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

var Cmdptr *sarflags.Cliflags

func Quit(g *gocui.Gui, v *gocui.View) error {
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

// This is where we process command line inputs after a CR entered
func GetLine(g *gocui.Gui, v *gocui.View) error {
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
			cmds.Docmd(g, command[1], c)
		}(g, command[1])

		if command[1] == "exit" || command[1] == "quit" {
			// Sarwg.Wait()
			err := Quit(g, v)
			// THIS IS A KLUDGE FIX IT WITH A CHANNEL
			log.Fatal("\nGocui Exit. Bye!\n", err)
		}
		Prompt(g, v)
	case "msg", "packet", "err":
		return CursorDown(g, v)
	}
	return nil
}

// Bind keys to function handlers
func Keybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("", gocui.KeyCtrlSpace, gocui.ModNone, switchView); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowDown, gocui.ModNone, CursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowUp, gocui.ModNone, CursorUp); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowLeft, gocui.ModNone, CursorLeft); err != nil {
		return nil
	}
	if err := g.SetKeybinding("", gocui.KeyArrowRight, gocui.ModNone, CursorRight); err != nil {
		return nil
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlP, gocui.ModNone, showPacket); err != nil {
		return nil
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, Quit); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, GetLine); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyBackspace, gocui.ModNone, BackSpace); err != nil {
		return nil
	}
	if err := g.SetKeybinding("", gocui.KeyBackspace2, gocui.ModNone, BackSpace); err != nil {
		return nil
	}
	if err := g.SetKeybinding("", gocui.KeyDelete, gocui.ModNone, BackSpace); err != nil {
		return nil
	}
	return nil
}

// *********************************************************************************************************

// prhelp -- return command help string
func prhelp(cf string, c *sarflags.Cliflags) string {
	for key, val := range sarflags.Commands {
		if key == cf {
			return key + ":" + val.Help
		}
	}
	return "Invalid Command"
}

// prusage -- return command usage string
func prusage(cf string, c *sarflags.Cliflags) string {
	for key, val := range sarflags.Commands {
		if key == cf {
			return "usage:" + val.Usage
		}
	}
	return "Invalid Command"
}

// removeIndex -- Remove an entry in a slice of strings by index #
func removeIndex(s []string, index int) []string {
	ret := make([]string, 0)
	ret = append(ret, s[:index]...)
	return append(ret, s[index+1:]...)
}

// removeValue -- Remove all entries in slice of strings matching val
func removeValue(s []string, val string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == val {
			s = removeIndex(s, i)
			s = removeValue(s, val) // Call me again to remove dupes
		}
	}
	return s
}

// Only append to a string slice if it is unique
/* THIS IS NOT USED YET
func appendunique(slice []string, i string) []string {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}
*/

/* ********************************************************************************* */

// All of the different command line input handlers
// Send count beacons to host
func sendbeacons(g *gocui.Gui, flags string, count uint, interval uint, host string, port int) {
	// We have a hostname maybe with multiple addresses
	var addrs []string
	var err error
	var txb beacon.Beacon // The assembled beacon to transmit
	b := &txb

	errflag := make(chan string, 1) // The return channel holding the saratoga errflag

	if addrs, err = net.LookupHost(host); err != nil {
		MsgPrintln(g, "red_black", "Cannot resolve hostname:", err)
		return
	}
	// Loop thru the address(s) for the host and send beacons to them
	for _, addr := range addrs {
		MsgPrintln(g, "cyan_black", "Sending beacon to ", addr)
		binfo := beacon.Binfo{Freespace: 0, Eid: ""}
		if err := frames.New(b, flags, &binfo); err == nil {
			go txb.Send(addr, port, count, interval, errflag)
			errcode := <-errflag
			if errcode != "success" {
				ErrPrintln(g, "red_black", "Error:", errcode,
					"Unable to send beacon to ", addr)
			} else {
				PacketPrintln(g, "cyan_black", "Tx ", b.ShortPrint())
			}
		} else {
			ErrPrintln(g, "red_black", "cannot create beacon in txb.New:", err.Error())
		}
	}
}

/* ********************************************************************************* */
