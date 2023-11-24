/*
 * Handle screen outputs for views in colours for Saratoga
 * We have msg,error,cli and packet windows
 */

package sarwin

import (
	"fmt"
	"log"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/jroimartin/gocui"

	"github.com/charlesetsmith/saratoga/beacon"
	"github.com/charlesetsmith/saratoga/frames"
	"github.com/charlesetsmith/saratoga/sarflags"

	"github.com/charlesetsmith/saratoga/transfer"
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

		if command[1] == "exit" || command[1] == "quit" {
			// Sarwg.Wait()
			err := Quit(g, v)
			// THIS IS A KLUDGE FIX IT WITH A CHANNEL
			log.Fatal("\nGocui Exit. Bye!\n", err)
		}
		Run(g, c, command[1])
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
// All of the different command line input handlers
/* ********************************************************************************* */

// Beacon CLI Info
type Beaconcmd struct {
	flags    string   // Header Flags set for beacons
	count    uint     // How many beacons to send 0|1 == 1
	interval uint     // interval in seconds between beacons 0|1 == 1
	v4mcast  bool     // Sending beacons to V4 Multicast
	v6mcast  bool     // Sending beacons to V6 Multicast
	host     []string // Send unicast beacon to List of hosts
}

var clibeacon Beaconcmd

// cmdBeacon - Beacon commands
func cmdBeacon(g *gocui.Gui, args []string, c *sarflags.Cliflags) {

	// var bmu sync.Mutex // Protects beacon.Beacon structure (EID)
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()
	clibeacon.flags = sarflags.Setglobal("beacon", c) // Initialise Global Beacon flags
	clibeacon.interval = c.Timeout.Binterval          // Set up the correct interval

	switch len(args) {
	// Show current Cbeacon flags and lists - beacon
	case 1:
		if clibeacon.count != 0 {
			MsgPrintln(g, "yellow_black", clibeacon.count, "Beacons to be sent every %d secs",
				clibeacon.interval)
		} else {
			MsgPrintln(g, "yellow_black", "Single Beacon to be sent")
		}
		if clibeacon.v4mcast {
			MsgPrintln(g, "yellow_black", "Sending IPv4 multicast beacons")
		}
		if clibeacon.v6mcast {
			MsgPrintln(g, "yellow_black", "Sending IPv6 multicast beacons")
		}
		if len(clibeacon.host) > 0 {
			MsgPrintln(g, "cyan_black", "Sending beacons to:")
			for _, i := range clibeacon.host {
				MsgPrintln(g, "cyan_black", "\t", i)
			}
		}
		if !clibeacon.v4mcast && !clibeacon.v6mcast &&
			len(clibeacon.host) == 0 {
			MsgPrintln(g, "yellow_black", "No beacons currently being sent")
		}
		return
	case 2:
		switch args[1] {
		case "?": // usage
			MsgPrintln(g, "green_black", prusage("beacon", c))
			MsgPrintln(g, "green_black", prhelp("beacon", c))
			return
		case "off": // remove and disable all beacons
			clibeacon.flags = sarflags.Setglobal("beacon", c)
			clibeacon.count = 0
			clibeacon.interval = c.Timeout.Binterval
			clibeacon.host = nil
			MsgPrintln(g, "green_black", "Beacons Disabled")
			return
		case "v4": // V4 Multicast
			MsgPrintln(g, "cyan_black", "Sending beacon to IPv4 Multicast")
			clibeacon.flags = sarflags.Setglobal("beacon", c)
			clibeacon.v4mcast = true
			clibeacon.count = 1
			// Start up the beacon client sending count IPv4 beacons
			go sendbeacons(g, clibeacon.flags, clibeacon.count, clibeacon.interval, c.V4Multicast, c.Port)
			return
		case "v6": // V6 Multicast
			MsgPrintln(g, "cyan_black", "Sending beacon to IPv6 Multicast")
			clibeacon.flags = sarflags.Setglobal("beacon", c)
			clibeacon.v6mcast = true
			clibeacon.count = 1
			// Start up the beacon client sending count IPv6 beacons
			go sendbeacons(g, clibeacon.flags, clibeacon.count, clibeacon.interval, c.V6Multicast, c.Port)
			return
		default: // beacon <count> or beacon <ipaddr>
			if n, err := strconv.ParseUint(args[1], 10, 32); err == nil {
				// We have a number so it is a timer
				clibeacon.count = uint(n)
				MsgPrintln(g, "green_black", "Beacons timer set to ", clibeacon.count, " seconds")
			} else {
				MsgPrintln(g, "cyan_black", "Sending ", clibeacon.count, " beacons to ", args[1])
				go sendbeacons(g, clibeacon.flags, clibeacon.count, clibeacon.interval, args[1], c.Port)
			}
			return
		}
	}

	// beacon off <ipaddr> ...
	if args[1] == "off" && len(args) > 2 { // turn off following addresses
		MsgPrintf(g, "green_black", "%s ", "Beacons turned off to")
		for i := 2; i < len(args); i++ { // Remove Address'es from lists
			if net.ParseIP(args[i]) != nil { // Do We have a valid IP Address
				clibeacon.host = removeValue(clibeacon.host, args[i])
				MsgPrintf(g, "green_black", "%s ", args[i])
				if i == len(args)-1 {
					MsgPrintln(g, "green_black", "")
				}
			} else {
				MsgPrintln(g, "red_black", "Invalid IP Address:", args[i])
				CmdPrintln(g, "red_black", prusage("beacon", c))
			}
		}
		return
	}

	// beacon <count> <ipaddr> ...
	var addrstart = 1
	u32, err := strconv.ParseUint(args[1], 10, 32)
	if err == nil { // We have a number so it is a timer
		clibeacon.count = uint(u32)
		MsgPrintln(g, "green_black", "Beacon counter set to ", clibeacon.count)
		addrstart = 2
	}
	// beacon [count] <ipaddr> ...
	MsgPrintf(g, "cyan_black", "Sending %d beacons to:",
		clibeacon.count)
	for i := addrstart; i < len(args); i++ { // Add Address'es to lists
		MsgPrintf(g, "cyan_black", "%s ", args[i])
		switch args[i] {
		case "v4":
			go sendbeacons(g, clibeacon.flags, clibeacon.count,
				clibeacon.interval, c.V4Multicast, c.Port)
		case "v6":
			go sendbeacons(g, clibeacon.flags, clibeacon.count,
				clibeacon.interval, c.V6Multicast, c.Port)
		default:
			go sendbeacons(g, clibeacon.flags, clibeacon.count,
				clibeacon.interval, args[i], c.Port)
		}
	}
	MsgPrintln(g, "green_black", "")
}

func cmdCancel(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	MsgPrintln(g, "green_black", args)
}

func cmdChecksum(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		MsgPrintln(g, "green_black", "Checksum ", c.Global["csumtype"])
		return
	case 2:
		switch args[1] {
		case "?": // usage
			MsgPrintln(g, "green_black", prusage("checksum", c))
			MsgPrintln(g, "green_black", prhelp("checksum", c))
			return
		case "off", "none":
			c.Global["csumtype"] = "none"
		case "crc32":
			c.Global["csumtype"] = "crc32"
		case "md5":
			c.Global["csumtype"] = "md5"
		case "sha1":
			c.Global["csumtype"] = "sha1"
		default:
			CmdPrintln(g, "green_red", prusage("checksum", c))
		}
		return
	}
	CmdPrintln(g, "green_red", prusage("checksum", c))
}

// cmdDescriptor -- set descriptor size 16,32,64,128 bits
func cmdDescriptor(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		MsgPrintln(g, "green_black", "Descriptor ", c.Global["descriptor"])
		return
	case 2:
		switch args[1] {
		case "?": // usage
			MsgPrintln(g, "green_black", prusage("descriptor", c))
			MsgPrintln(g, "green_black", prhelp("descriptor", c))
			return
		case "auto":
			if sarflags.MaxUint <= sarflags.MaxUint16 {
				c.Global["descriptor"] = "d16"
				break
			}
			if sarflags.MaxUint <= sarflags.MaxUint32 {
				c.Global["descriptor"] = "d32"
				break
			}
			if sarflags.MaxUint <= sarflags.MaxUint64 {
				c.Global["descriptor"] = "d64"
				break
			}
			MsgPrintln(g, "red_black", "128 bit descriptors not supported on this platform")
		case "d16":
			if sarflags.MaxUint > sarflags.MaxUint16 {
				c.Global["descriptor"] = "d16"
			} else {
				MsgPrintln(g, "red_black", "16 bit descriptors not supported on this platform")
			}
		case "d32":
			if sarflags.MaxUint > sarflags.MaxUint32 {
				c.Global["descriptor"] = "d32"
			} else {
				MsgPrintln(g, "red_black", "32 bit descriptors not supported on this platform")
			}
		case "d64":
			if sarflags.MaxUint <= sarflags.MaxUint64 {
				c.Global["descriptor"] = "d64"
			} else {
				MsgPrintln(g, "red_black", "64 bit descriptors are not supported on this platform")
				MsgPrintln(g, "red_black", "MaxUint=", sarflags.MaxUint,
					" <= MaxUint64=", sarflags.MaxUint64)
			}
		case "d128":
			MsgPrintln(g, "red_black", "128 bit descriptors not supported on this platform")
		default:
			MsgPrintln(g, "red_black", "usage:", prusage("descriptor", c))
		}
		MsgPrintln(g, "green_black", "Descriptor size is ", c.Global["descriptor"])
		return
	}
	MsgPrintln(g, "red_black", "usage:", prusage("descriptor", c))
}

// Cexit = Exit level to quit from saratoga
var Cexit = -1

// cmdExit -- Quit saratoga
func cmdExit(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	switch len(args) {
	case 1: // exit 0
		Cexit = 0
		MsgPrintln(g, "green_black", "Good Bye!")
		return
	case 2:
		switch args[1] {
		case "?": // Usage
			MsgPrintln(g, "green_black", prusage("exit", c))
			MsgPrintln(g, "green_black", prhelp("exit", c))
		case "0": // exit 0
			Cexit = 0
			MsgPrintln(g, "green_black", "Good Bye!")
		case "1": // exit 1
			Cexit = 1
			MsgPrintln(g, "green_black", "Good Bye!")
		default: // Help
			MsgPrintln(g, "red_black", prusage("exit", c))
		}
	default:
		MsgPrintln(g, "red_black", prusage("exit", c))
	}
}

// MORE WORK TO DO HERE!!!!! USE TRANSFERS LIST
// cmdFiiles -- show currently open files
func cmdFiles(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	var flist []string

	switch len(args) {
	case 1:
		if len(flist) == 0 {
			MsgPrintln(g, "green_black", "No currently open files")
			return
		}
		for _, i := range flist {
			MsgPrintln(g, "green_black", i)
		}
		return
	case 2:
		if args[1] == "?" { // usage
			MsgPrintln(g, "green_black", prusage("files", c))
			MsgPrintln(g, "green_black", prhelp("files", c))
			return
		}
	}
	MsgPrintln(g, "red_black", prusage("files", c))
}

func cmdFreespace(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		if c.Global["freespace"] == "yes" {
			MsgPrintln(g, "green_black", "Free space is advertised")
		} else {
			MsgPrintln(g, "green_black", "Free space is not advertised")
		}
		return
	case 2:
		switch args[1] {
		case "?": // usage
			MsgPrintln(g, "green_black", prusage("freespace", c))
			MsgPrintln(g, "green_black", prhelp("freespace", c))
			return
		case "yes":
			MsgPrintln(g, "green_black", "freespace is advertised")
			c.Global["freespace"] = "yes"
			return
		case "no":
			MsgPrintln(g, "green_black", "freespace is not advertised")
			c.Global["freespace"] = "no"
			return
		}
	}
	MsgPrintln(g, "red_black", "usage:", prusage("freespace", c))
}

// Initiator _get_
func cmdGet(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	switch len(args) {
	case 1:
		transfer.Info(g, "get")
		return
	case 2:
		if args[1] == "?" {
			MsgPrintln(g, "green_black", prusage("get", c))
			MsgPrintln(g, "green_black", prhelp("get", c))
			return
		}
	case 3:
		// var t transfer.CTransfer
		if _, err := transfer.NewInitiator(g, "get", args[1], args[2], c); err != nil {
			return
		}
		return
	}
	MsgPrintln(g, "red_black", prusage("get", c))
}

// Initiator _getdir_
func cmdGetdir(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	switch len(args) {
	case 1:
		transfer.Info(g, "getdir")
		return
	case 2:
		if args[1] == "?" {
			MsgPrintln(g, "green_black", prusage("getdir", c))
			MsgPrintln(g, "green_black", prhelp("getdir", c))
			return
		}
	case 3:
		if _, err := transfer.NewInitiator(g, "getdir", args[1], args[2], c); err != nil {
			MsgPrintln(g, "green_black", prusage("getdir", c))
			MsgPrintln(g, "green_black", prhelp("getdir", c))
		}
		return
	}
	MsgPrintln(g, "red_black", prusage("getdir", c))
}

// Initiator _get_ then _delete_
func cmdGetrm(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	switch len(args) {
	case 1:
		transfer.Info(g, "getrm")
		return
	case 2:
		if args[1] == "?" {
			MsgPrintln(g, "green_black", prusage("getrm", c))
			MsgPrintln(g, "green_black", prhelp("getrm", c))
			return
		}
	case 3:
		if _, err := transfer.NewInitiator(g, "getrm", args[1], args[2], c); err != nil {
			MsgPrintln(g, "green_black", prusage("getrm", c))
			MsgPrintln(g, "green_black", prhelp("getrm", c))
			return
		}
		return
	}
	MsgPrintln(g, "red_black", prusage("getrm", c))
}

func cmdHelp(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	switch len(args) {
	case 1:
		var sslice sort.StringSlice
		for key, val := range sarflags.Commands {
			sslice = append(sslice, fmt.Sprintf("%s - %s",
				key,
				val.Help))
		}
		sort.Sort(sslice)
		var sbuf string
		for key := 0; key < len(sslice); key++ {
			sbuf += fmt.Sprintf("%s\n", sslice[key])
		}
		MsgPrintln(g, "magenta_black", sbuf)
		return
	case 2:
		if args[1] == "?" {
			var sslice sort.StringSlice
			for key, val := range sarflags.Commands {
				sslice = append(sslice, fmt.Sprintf("%s - %s\n  %s",
					key,
					val.Help,
					val.Usage))
			}
			sort.Sort(sslice)
			var sbuf string
			for key := 0; key < len(sslice); key++ {
				sbuf += fmt.Sprintf("%s\n", sslice[key])
			}
			MsgPrintln(g, "magenta_black", sbuf)
			return
		}
	}
	for key, val := range sarflags.Commands {
		if key == "help" {
			MsgPrintln(g, "red_black", fmt.Sprintf("%s - %s",
				key,
				val.Help))
		}
	}
}

func cmdInterval(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		if c.Timeout.Binterval == 0 {
			MsgPrintln(g, "yellow_black", "Single Beacon Interation")
		} else {
			MsgPrintln(g, "yellow_black", "Beacons sent every ",
				c.Timeout.Binterval, " seconds")
		}
		return
	case 2:
		switch args[1] {
		case "?":
			MsgPrintln(g, "green_black", prusage("interval", c))
			MsgPrintln(g, "green_black", prhelp("interval", c))
			return
		case "off":
			c.Timeout.Binterval = 0
			return
		default:
			if n, err := strconv.Atoi(args[1]); err == nil && n >= 0 {
				c.Timeout.Binterval = uint(n)
				return
			}
		}
		MsgPrintln(g, "red_black", prusage("interval", c))
	}
	MsgPrintln(g, "red_black", prusage("interval", c))
}

func cmdHistory(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	switch len(args) {
	case 1:
		MsgPrintln(g, "green_black", "History not implemented yet")
		return
	case 2:
		switch args[1] {
		case "?":
			MsgPrintln(g, "green_black", prusage("history", c))
			MsgPrintln(g, "green_black", prhelp("history", c))
			return
		default:
			MsgPrintln(g, "green_black", "History not implemented yet")
			return
		}
	}
	MsgPrintln(g, "red_black", prusage("history", c))
}

func cmdHome(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	switch len(args) {
	case 1:
		MsgPrintln(g, "green_black", "Home not implemented yet")
		return
	case 2:
		switch args[1] {
		case "?":
			MsgPrintln(g, "green_black", prusage("home", c))
			MsgPrintln(g, "green_black", prhelp("home", c))
			return
		}
	}
	MsgPrintln(g, "red_black", prusage("home", c))
}

func cmdLs(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	if len(args) != 0 {
		MsgPrintln(g, "red_bblack", prusage("ls", c))
		return
	}
	switch args[1] {
	case "?":
		MsgPrintln(g, "green_black", prusage("ls", c))
		MsgPrintln(g, "green_black", prhelp("ls", c))
		return
	}
	MsgPrintln(g, "green_black", "ls not implemented yet")
}

// Display all of the peer information learned frm beacons
func cmdPeers(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	if len(args) != 1 {
		MsgPrintln(g, "red_bblack", prusage("peers", c))
		return
	}

	if len(beacon.Peers) == 0 {
		MsgPrintln(g, "magenta_black", "No Peers")
		return
	}
	// Table format
	// Work out the max length of each field
	var addrlen, eidlen, dcrelen, dmodlen int
	for p := range beacon.Peers {
		if len(beacon.Peers[p].Addr) > addrlen {
			addrlen = len(beacon.Peers[p].Addr)
		}
		if len(beacon.Peers[p].Eid) > eidlen {
			eidlen = len(beacon.Peers[p].Eid)
		}
		if len(beacon.Peers[p].Created.Print()) > dcrelen {
			dcrelen = len(beacon.Peers[p].Created.Print())
		}
		if len(beacon.Peers[p].Created.Print()) > dmodlen {
			dmodlen = len(beacon.Peers[p].Updated.Print())
		}
	}
	if eidlen < 3 {
		eidlen = 3
	}

	sfmt := fmt.Sprintf("|%%%ds|%%6s|%%%ds|%%3s|%%%ds|%%%ds|\n",
		addrlen, eidlen, dcrelen, dmodlen)
	sborder := fmt.Sprintf(sfmt,
		strings.Repeat("-", addrlen),
		strings.Repeat("-", 6),
		strings.Repeat("-", eidlen),
		strings.Repeat("-", 3),
		strings.Repeat("-", dcrelen),
		strings.Repeat("-", dmodlen))

	var sslice sort.StringSlice
	for key := range beacon.Peers {
		pinfo := fmt.Sprintf(sfmt, beacon.Peers[key].Addr,
			strconv.Itoa(int(beacon.Peers[key].Freespace/1024/1024)),
			beacon.Peers[key].Eid,
			beacon.Peers[key].Maxdesc,
			beacon.Peers[key].Created.Print(),
			beacon.Peers[key].Updated.Print())
		sslice = append(sslice, pinfo)
	}
	sort.Sort(sslice)

	sbuf := sborder
	sbuf += fmt.Sprintf(sfmt, "IP", "GB", "EID", "Des", "Date Created", "Date Modified")
	sbuf += sborder
	for key := 0; key < len(sslice); key++ {
		sbuf += sslice[key]
	}
	sbuf += sborder
	MsgPrintln(g, "magenta_black", sbuf)
}

// Initiator _put_
// send a file to a destination
func cmdPut(g *gocui.Gui, args []string, c *sarflags.Cliflags) {

	switch len(args) {
	case 1:
		transfer.Info(g, "put")
		return
	case 2:
		if args[1] == "?" {
			MsgPrintln(g, "green_black", prusage("put", c))
			MsgPrintln(g, "green_black", prhelp("put", c))
			return
		}
	case 3:
		if t, err := transfer.NewInitiator(g, "put", args[1], args[2], c); err == nil && t != nil {
			errflag := make(chan error, 1) // The return channel holding the saratoga errflag
			go t.Do(g, errflag)            // Actually do the transfer
			errcode := <-errflag
			if errcode != nil {
				ErrPrintln(g, "red_black", "Error:", errcode,
					" Unable to send file:", t.Print())
				if derr := t.Remove(); derr != nil {
					MsgPrintln(g, "red_black", "Unable to remove transfer:", t.Print())
				}
			}
			MsgPrintln(g, "green_black", "put completed closing channel")
			close(errflag)
		} else {
			MsgPrintln(g, "red_black", "Cannot add transfer:", err.Error())
		}
		return
	}
	MsgPrintln(g, "red_black", prusage("put", c))
}

// Initiator _put_
// blind send a file to a destination not expecting return _status_ from Responder
func cmdPutblind(g *gocui.Gui, args []string, c *sarflags.Cliflags) {

	errflag := make(chan error, 1) // The return channel holding the saratoga errflag

	switch len(args) {
	case 1:
		transfer.Info(g, "putrm")
		return
	case 2:
		if args[1] == "?" {
			MsgPrintln(g, "green_black", prusage("putblind", c))
			MsgPrintln(g, "green_black", prhelp("putblind", c))
			return
		}
	case 3:
		// We send the Metadata and do not bother with request/status exchange
		if t, err := transfer.NewInitiator(g, "putblind", args[1], args[2], c); err == nil && t != nil {
			go t.Do(g, errflag)
			errcode := <-errflag
			if errcode != nil {
				ErrPrintln(g, "red_black", "Error:", errcode,
					"Unable to send file:", t.Print())
			}
		} else {
			ErrPrintln(g, "red_black", "Cannot create Transfer:", error.Error(err))
		}
		return
	}
	MsgPrintln(g, "red_black", prusage("putblind", c))
}

// Initiator _put_
// send a file file to a remote destination then remove it from the origin
func cmdPutrm(g *gocui.Gui, args []string, c *sarflags.Cliflags) {

	errflag := make(chan error, 1) // The return channel holding the saratoga errflag

	switch len(args) {
	case 1:
		transfer.Info(g, "putrm")
		return
	case 2:
		if args[1] == "?" {
			MsgPrintln(g, "green_black", prusage("putrm", c))
			MsgPrintln(g, "green_black", prhelp("putrm", c))
			return
		}
	case 3:
		// var t *transfer.Transfer
		if t, err := transfer.NewInitiator(g, "putrm", args[1], args[2], c); err == nil && t != nil {
			go t.Do(g, errflag)
			errcode := <-errflag
			if errcode != nil {
				ErrPrintln(g, "red_black", "Error:", errcode,
					" Unable to send file:", t.Print())
			} else {
				MsgPrintln(g, "red_black",
					"Put and now removing (NOT) (ADD MORE CODE  HERE!!!!) file:", t.Print())
			}
		}
		return
	}
	MsgPrintln(g, "red_black", prusage("putrm", c))
}

func cmdReqtstamp(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		if c.Global["reqtstamp"] == "yes" {
			MsgPrintln(g, "green_black", "Time stamps requested")
		} else {
			MsgPrintln(g, "green_black", "Time stamps not requested")
		}
		return
	case 2:
		switch args[1] {
		case "?": // usage
			MsgPrintln(g, "green_black", prusage("reqtstamp", c))
			MsgPrintln(g, "green_black", prhelp("reqtstamp", c))
			return
		case "yes":
			c.Global["reqtstamp"] = "yes"
			return
		case "no":
			c.Global["reqtstamp"] = "no"
			return
		}
	}
	MsgPrintln(g, "red_black", "usage:", prusage("reqtstamp", c))
}

// Initiator _delete_
// remove a file from a remote destination
func cmdRm(g *gocui.Gui, args []string, c *sarflags.Cliflags) {

	switch len(args) {
	case 1:
		transfer.Info(g, "rm")
		return
	case 2:
		if args[1] == "?" {
			MsgPrintln(g, "green_black", prusage("rm", c))
			MsgPrintln(g, "green_black", prhelp("rm", c))
			return
		}
	case 3:
		if _, err := transfer.NewInitiator(g, "rm", args[1], args[2], c); err != nil {
			MsgPrintln(g, "green_black", prusage("rm", c))
			MsgPrintln(g, "green_black", prhelp("rm", c))
			return
		}
	}
	MsgPrintln(g, "red_black", prusage("rm", c))
}

// Initiator _getdir_, _delete_ ...
// remove a directory from a remote destination
func cmdRmdir(g *gocui.Gui, args []string, c *sarflags.Cliflags) {

	switch len(args) {
	case 1:
		transfer.Info(g, "rmdir")
		return
	case 2:
		if args[1] == "?" {
			MsgPrintln(g, "green_black", prusage("rmdir", c))
			MsgPrintln(g, "green_black", prhelp("rmdir", c))
			return
		}
	case 3:
		if _, err := transfer.NewInitiator(g, "rmdir", args[1], args[2], c); err != nil {
			MsgPrintln(g, "green_black", prusage("rmdir", c))
			MsgPrintln(g, "green_black", prhelp("rmdir", c))
			return
		}
	}
	MsgPrintln(g, "red_black", prusage("rmdir", c))
}

func cmdRmtran(g *gocui.Gui, args []string, c *sarflags.Cliflags) {

	switch len(args) {
	case 1:
		MsgPrintln(g, "green_black", prusage("rmtran", c))
		MsgPrintln(g, "green_black", prhelp("rmtran", c))
		return
	case 2:
		if args[1] == "?" {
			MsgPrintln(g, "green_black", prusage("rmtran", c))
			MsgPrintln(g, "green_black", prhelp("rmtran", c))
			return
		}
	case 4:
		ttype := args[1]
		addr := args[2]
		fname := args[3]
		if t := transfer.Match(addr, fname); t != nil {
			if err := t.Remove(); err != nil {
				MsgPrintln(g, "red_black", err.Error())
			}
		} else {
			MsgPrintln(g, "red_black", "No such transfer:", ttype, " ", addr, " ", fname)
		}
		return
	}
	MsgPrintln(g, "red_black", prusage("rmtran", c))
}

// Are we willing to transmit files
func cmdRxwilling(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		MsgPrintln(g, "green_black", "Receive Files:", c.Global["rxwilling"])
		return
	case 2:
		switch args[1] {
		case "?":
			MsgPrintln(g, "green_black", prusage("rxwilling", c))
			MsgPrintln(g, "green_black", prhelp("rxwilling", c))
			return
		case "on":
			c.Global["rxwilling"] = "yes"
			return
		case "off":
			c.Global["rxwilling"] = "no"
			return
		case "capable":
			c.Global["rxwilling"] = "capable"
			return
		}
	}
	MsgPrintln(g, "red_black", prusage("rxwilling", c))
}

// Initiator _put_ not expecting _status_
// source is a named pipe not a file
func cmdStream(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		if c.Global["stream"] == "yes" {
			MsgPrintln(g, "green_black", "Can stream")
		} else {
			MsgPrintln(g, "green_black", "Cannot stream")
		}
		return
	case 2:
		switch args[1] {
		case "?":
			MsgPrintln(g, "green_black", prusage("stream", c))
			MsgPrintln(g, "green_black", prhelp("stream", c))
			return
		case "yes":
			c.Global["stream"] = "yes"
			return
		case "no":
			c.Global["stream"] = "no"
			return
		}
	}
	MsgPrintln(g, "red_black", prusage("stream", c))
}

// Timeout - set timeouts for responses to request/status/transfer in seconds
func cmdTimeout(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		if c.Timeout.Metadata == 0 {
			MsgPrintln(g, "green_black", "metadata:No Timeout")
		} else {
			MsgPrintln(g, "green_black", "metadata:", c.Timeout.Metadata, " sec")
		}
		if c.Timeout.Request == 0 {
			MsgPrintln(g, "green_black", "request:No Timeout")
		} else {
			MsgPrintln(g, "green_black", "request:", c.Timeout.Request, " sec")
		}
		if c.Timeout.Status == 0 {
			MsgPrintln(g, "green_black", "status:No Timeout")
		} else {
			MsgPrintln(g, "green_black", "status:", c.Timeout.Status, " sec")
		}
		if c.Timeout.Datacounter == 0 {
			c.Timeout.Datacounter = 100
			MsgPrintln(g, "green_black", "Data Counter every 100 frames")
		} else {
			MsgPrintln(g, "green_black", "Datacnt:", c.Timeout.Datacounter, " frames")
		}
		if c.Timeout.Transfer == 0 {
			MsgPrintln(g, "green_black", "transfer:No Timeout")
		} else {
			MsgPrintln(g, "green_black", "transfer:", c.Timeout.Transfer, " sec")
		}
		return
	case 2:
		switch args[1] {
		case "?":
			MsgPrintln(g, "green_black", prusage("timeout", c))
			MsgPrintln(g, "green_black", prhelp("timeout", c))
		case "request":
			if c.Timeout.Request == 0 {
				MsgPrintln(g, "green_black", "request:No Timeout")
			} else {
				MsgPrintln(g, "green_black", "request:", c.Timeout.Request, " sec")
			}
		case "metadata":
			if c.Timeout.Request == 0 {
				MsgPrintln(g, "green_black", "metadata:No Timeout")
			} else {
				MsgPrintln(g, "green_black", "metadata:", c.Timeout.Metadata, " sec")
			}
		case "status":
			if c.Timeout.Status == 0 {
				MsgPrintln(g, "green_black", "status:No Timeout")
			} else {
				MsgPrintln(g, "green_black", "status:", c.Timeout.Status, " sec")
			}
		case "datacounter":
			if c.Timeout.Datacounter == 0 {
				c.Timeout.Datacounter = 100
				MsgPrintln(g, "green_black", "Datacounter:Never")
			} else {
				MsgPrintln(g, "green_black", "Datacounter:", c.Timeout.Datacounter, " frames")
			}
		case "transfer":
			if c.Timeout.Transfer == 0 {
				MsgPrintln(g, "green_black", "transfer:No Timeout")
			} else {
				MsgPrintln(g, "green_black", "transfer:", c.Timeout.Transfer, " sec")
			}
		default:
			MsgPrintln(g, "red_black", prusage("stream", c))
		}
		return
	case 3:
		if n, err := strconv.Atoi(args[2]); err == nil && n >= 0 {
			switch args[1] {
			case "metadata":
				c.Timeout.Metadata = n
			case "request":
				c.Timeout.Request = n
			case "status":
				c.Timeout.Status = n
			case "datacnt":
				if n == 0 {
					n = 100
				}
				c.Timeout.Datacounter = n
			case "transfer":
				c.Timeout.Transfer = n
			}
			return
		}
		if args[2] == "off" {
			switch args[1] {
			case "metadata":
				c.Timeout.Metadata = 60
			case "request":
				c.Timeout.Request = 60
			case "status":
				c.Timeout.Status = 60
			case "datacounter":
				c.Timeout.Datacounter = 100
			case "transfer":
				c.Timeout.Transfer = 60
			}
			return
		}
	}
	MsgPrintln(g, "red_black", prusage("timeout", c))
}

// set the timestamp type we are using
func cmdTimestamp(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		MsgPrintln(g, "green_black", "Timestamps  are",
			c.Global["reqtstamp"], " and ", c.Timestamp)
		return
	case 2:
		switch args[1] {
		case "?":
			MsgPrintln(g, "green_black", prusage("timestamp", c))
			MsgPrintln(g, "green_black", prhelp("timestamp", c))
		case "off":
			c.Global["reqtstamp"] = "no"
			// Don't change the TGlobal from what it was
		case "32":
			c.Global["reqtstamp"] = "yes"
			c.Timestamp = "posix32"
		case "32_32":
			c.Global["reqtstamp"] = "yes"
			c.Timestamp = "posix32_32"
		case "64":
			c.Global["reqtstamp"] = "yes"
			c.Timestamp = "posix64"
		case "64_32":
			c.Global["reqtstamp"] = "yes"
			c.Timestamp = "posix64_32"
		case "32_y2k":
			c.Global["reqtstamp"] = "yes"
			c.Timestamp = "epoch2000_32"
		case "local":
			c.Global["reqtstamp"] = "yes"
			c.Timestamp = "localinterp"
		default:
			MsgPrintln(g, "red_black", prusage("timestamp", c))
		}
		return
	}
	MsgPrintln(g, "red_black", prusage("timestamp", c))
}

// set the timezone we use for logs local or utc
func cmdTimezone(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		MsgPrintln(g, "green_black", "Timezone:", c.Timezone)
		return
	case 2:
		switch args[1] {
		case "?":
			MsgPrintln(g, "green_black", prusage("timezone", c))
			MsgPrintln(g, "green_black", prhelp("timezone", c))
		case "local":
			c.Timezone = "local"
		case "utc":
			c.Timezone = "utc"
		default:
			MsgPrintln(g, "red_black", prusage("timezone", c))
		}
		return
	}
	MsgPrintln(g, "red_black", prusage("timezone", c))
}

// show current transfers in progress & % completed
func cmdTran(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	switch len(args) {
	case 1:
		transfer.Info(g, "")
		return
	case 2:
		switch args[1] {
		case "?":
			MsgPrintf(g, "green_black", "%s\n  %s\n",
				prusage("tran", c), prhelp("tran", c))
		default:
			for _, tt := range transfer.Ttypes {
				if args[1] == tt {
					transfer.Info(g, args[1])
					return
				}
			}
			MsgPrintln(g, "green_black", prusage("tran", c))
		}
		return
	}
	MsgPrintln(g, "green_black", prusage("tran", c))
}

// we are willing to transmit files
func cmdTxwilling(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		MsgPrintln(g, "green_black", "Transmit Files:", c.Global["txwilling"])
		return
	case 2:
		switch args[1] {
		case "?":
			MsgPrintln(g, "green_black", prusage("txwilling", c))
			MsgPrintln(g, "green_black", prhelp("txwilling", c))
			return
		case "on":
			c.Global["txwilling"] = "on"
			return
		case "off":
			c.Global["txwilling"] = "off"
			return
		case "capable":
			c.Global["txwilling"] = "capable"
			return
		}
	}
	MsgPrintln(g, "red_black", prusage("txwilling", c))
}

// Show all commands usage
func cmdUsage(g *gocui.Gui, args []string, c *sarflags.Cliflags) {
	var sslice sort.StringSlice

	for key, val := range sarflags.Commands {
		sslice = append(sslice, fmt.Sprintf("%s - %s",
			key,
			val.Usage))
	}

	sort.Sort(sslice)
	var sbuf string
	for key := 0; key < len(sslice); key++ {
		sbuf += fmt.Sprintf("%s\n", sslice[key])
	}
	MsgPrintln(g, "magenta_black", sbuf)
}

// *********************************************************************************************************

type cmdfunc func(*gocui.Gui, []string, *sarflags.Cliflags)

// Commands and function pointers to handle them
var cmdhandler = map[string]cmdfunc{
	"?":          cmdHelp,
	"beacon":     cmdBeacon,
	"cancel":     cmdCancel,
	"checksum":   cmdChecksum,
	"descriptor": cmdDescriptor,
	"exit":       cmdExit,
	"files":      cmdFiles,
	"freespace":  cmdFreespace,
	"get":        cmdGet,    // _get_
	"getdir":     cmdGetdir, // _getdir_
	"getrm":      cmdGetrm,  // _get_,_delete_
	"help":       cmdHelp,
	"history":    cmdHistory,
	"home":       cmdHome,
	"interval":   cmdInterval,
	"ls":         cmdLs,
	"peers":      cmdPeers,
	"put":        cmdPut,      // _put_
	"putblind":   cmdPutblind, // _put_ (no _status_)
	"putrm":      cmdPutrm,    // _put_ (delete local)
	"quit":       cmdExit,
	"reqtstamp":  cmdReqtstamp,
	"rm":         cmdRm, // _delete_
	"rmtran":     cmdRmtran,
	"rmdir":      cmdRmdir, // _getdir_, _delete_ ...
	"rxwilling":  cmdRxwilling,
	"stream":     cmdStream,
	"timeout":    cmdTimeout,
	"timestamp":  cmdTimestamp,
	"timezone":   cmdTimezone,
	"tran":       cmdTran,
	"txwilling":  cmdTxwilling,
	"usage":      cmdUsage,
}

// Lookup and run the command
func Run(g *gocui.Gui, c *sarflags.Cliflags, name string) bool {
	if name == "" { // Handle just return
		return true
	}
	// Get rid of leading and trailing whitespace
	s := strings.TrimSpace(name)
	vals := strings.Fields(s)
	for key := range sarflags.Commands {
		if key == name {
			fn, ok := cmdhandler[name]
			if ok {
				fn(g, vals, c)
				return true
			}
			return false
		}
	}
	return false
}
