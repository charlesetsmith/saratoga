/*
 * Handle screen outputs for views in colours for Saratoga
 * We have msg,error,cli and packet windows
 */

package sarwin

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/jroimartin/gocui"

	"github.com/charlesetsmith/saratoga/beacon"
	"github.com/charlesetsmith/saratoga/dirent"
	"github.com/charlesetsmith/saratoga/holes"
	"github.com/charlesetsmith/saratoga/metadata"
	"github.com/charlesetsmith/saratoga/request"
	"github.com/charlesetsmith/saratoga/sarflags"
	"github.com/charlesetsmith/saratoga/sarnet"
	"github.com/charlesetsmith/saratoga/status"
	"github.com/charlesetsmith/saratoga/timestamp"
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

// Does the foreground colour exist in our map of fg colours
func fgexists(col string) bool {
	for c := range fg {
		if col == c {
			return true
		}
	}
	return false
}

// Does the background colour exist in our map of bg colours
func bgexists(col string) bool {
	for c := range bg {
		if col == c {
			return true
		}
	}
	return false
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
		if fgexists(sequence[0]) && bgexists(sequence[1]) {
			return ansiprefix + fg[sequence[0]] + ansiseparator + bg[sequence[1]] + ansipostfix
		}
		return ansiprefix + fg["white"] + ansiseparator + bg["bred"] + ansipostfix
	case 1: // fg and "black" bg
		if fgexists(sequence[0]) {
			return ansiprefix + fg[sequence[0]] + ansiseparator + bg["black"] + ansipostfix
		}
		return ansiprefix + fg["white"] + ansiseparator + bg["bred"] + ansipostfix
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
		// RUN THE COMMAND ENTERED!!!
		Run(g, command[1])
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
func prhelp(cf string) string {
	for key, val := range sarflags.Commands {
		if key == cf {
			return key + ":" + val.Help
		}
	}
	return "Invalid Command"
}

// prusage -- return command usage string
func prusage(cf string) string {
	for key, val := range sarflags.Commands {
		if key == cf {
			return "usage:" + val.Usage
		}
	}
	return "Invalid Command"
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
func sendbeacons(g *gocui.Gui, flags string, count uint, interval uint, addr *net.UDPAddr) {
	// We have a hostname maybe with multiple addresses
	var err error
	var txb beacon.Beacon // The assembled beacon to transmita
	var conn *net.UDPConn
	b := &txb

	errflag := make(chan string, 1) // The return channel holding the saratoga errflag

	MsgPrintln(g, "cyan_black", "Sending beacon to ", addr.String())
	binfo := beacon.Binfo{Freespace: 0, Eid: ""}
	if err = txb.New(flags, &binfo); err == nil {
		if conn, err = net.DialUDP("udp", nil, addr); err != nil {
			ErrPrintln(g, "red_black", "Error:", err,
				"Unable to Dial", addr.String())
			return
		}
		txb.Sendmore(conn, count, interval, errflag)
		errcode := <-errflag
		if errcode != "success" {
			ErrPrintln(g, "red_black", "Error:", errcode,
				"Unable to send beacon to ", addr)
			return
		}
		PacketPrintln(g, "cyan_black", "Tx ", b.ShortPrint())
		return
	}
	ErrPrintln(g, "red_black", "cannot create beacon in txb.New:", err.Error())
}

/* ********************************************************************************* */
//  Transfer handlers
/* ********************************************************************************* */

// Create new Session number
func newsession() uint32 {
	smu.Lock()
	defer smu.Unlock()

	if sessionid == 0 {
		sessionid = uint32(os.Getpid()) + 1
	} else {
		sessionid++
	}
	return sessionid
}

// FileDescriptor - Get the appropriate descriptor flag size based on file length
/*
func filedescriptor(fname string) string {
	if fi, err := os.Stat(fname); err == nil {
		size := uint64(fi.Size())
		if size <= sarflags.MaxUint16 {
			return "descriptor=d16"
		}
		if size <= sarflags.MaxUint32 {
			return "descriptor=d32"
		}
		if size <= sarflags.MaxUint64 {
			return "descriptor=d64"
		}
	}
	// Just send back the maximum supported descriptor
	if sarflags.MaxUint <= sarflags.MaxUint16 {
		return "descriptor=d16"
	}
	if sarflags.MaxUint <= sarflags.MaxUint32 {
		return "descriptor=d32"
	}
	return "descriptor=d64"
}
*/

// Work out the maximum payload in data.Data frame given flagsa
/*
func maxpaylen(flags string) int {

	plen := sarflags.Mtu() - 60 - 8 // 60 for IP header, 8 for UDP header
	plen -= 8                       // Saratoga Header + Offset

	flags = strings.Replace(flags, " ", "", -1) // Get rid of extra spaces in flags
	// Grab the flags and set the frame header
	flag := strings.Split(flags, ",") // The name=val of the flag
	for fl := range flag {
		f := strings.Split(flag[fl], "=") // f[0]=name f[1]=val
		switch f[0] {
		case "descriptor":
			switch f[1] {
			case "d16":
				plen -= 2
			case "d32":
				plen -= 4
			case "d64":
				plen -= 8
			case "d128":
				plen -= 16
			default:
				return 0
			}
		case "reqtstamp":
			if f[1] == "yes" {
				plen -= 16
			}
		default:
		}
	}
	return plen
}
*/

// Work out the maximum payload in status.Status frame given flags
func stpaylen(flags string) int {
	var hsize int
	var plen int

	plen = sarflags.Mtu() - 60 - 8 // 60 for IP header, 8 for UDP header
	plen -= 8                      // Saratoga Header + Session

	flags = strings.Replace(flags, " ", "", -1) // Get rid of extra spaces in flags
	// Grab the flags and set the frame header
	flag := strings.Split(flags, ",") // The name=val of the flag
	for fl := range flag {
		f := strings.Split(flag[fl], "=") // f[0]=name f[1]=val
		switch f[0] {
		case "descriptor":
			switch f[1] {
			case "d16":
				hsize = 4
			case "d32":
				hsize = 8
			case "d64":
				hsize = 16
			case "d128":
				hsize = 32
			default:
				return 0
			}
		case "reqtstamp":
			if f[1] == "yes" {
				plen -= 16
			}
		default:
		}
	}
	plen -= hsize       // For Progress & Inrespto
	return plen / hsize // Max holes we can now have in the frame
}

// Ttypes - Transfer types
var Ttypes = []string{"get", "getrm", "getdir", "put", "putblind", "putrm", "rm", "rmdir"}

// Transfer direction we are an initiator of a transfer or a respondant to a request for a transfer
const Initiator bool = true
const Responder bool = false

var Directions = map[bool]string{true: "Initiator", false: "Responder"}

// current protected session number
var smu sync.Mutex
var sessionid uint32

type Transfer struct {
	Direction  bool                // Am I the Initiator of; or Responder to a transfer
	Session    uint32              // Session ID - This is the unique key
	Conn       *net.UDPConn        // The connection to the remote peer holds ip address and port of the peer
	Ttype      string              // Transfer type "get,getrm,put,putrm,putblind,rm"
	Tstamp     timestamp.Timestamp // Latest timestamp received from Data
	Tstamptype string              // Timestamp type "localinterp,posix32,posix64,posix32_32,posix64_32,epoch2000_32"
	Filename   string              // Local File name to receive or remove from remote host or send from local host
	Fp         *os.File            // File pointer for local file
	// frames    [][]byte           // Frames to process
	// holes     holes.Holes        // Holes to process
	Version    string               // Flag
	Fileordir  string               // Flag
	Udplite    string               // Flag
	Descriptor string               // Flag
	Stream     string               // Flag
	Csumtype   string               // What type of checksum are we using
	Havemeta   bool                 // Have we recieved a metadata yet
	Checksum   []byte               // Checksum of the remote file to be get/put if requested
	Dir        *dirent.DirEnt       // Directory entry info of the file to get/put
	Fileinfo   *dirent.FileMetaData // File metadata of the local file
	Data       []byte               // Buffered data
	Dcount     uint64               // Number Data frames sent/recieved
	Framecount uint64               // Total number frames received in this transfer (so we can schedule status)
	Progress   uint64               // Current Progress indicator
	Inrespto   uint64               // In respose to indicator
	Curfills   holes.Holes          // What has been received
	Cliflags   *sarflags.Cliflags   // Global flags used in this transfer
}

// Transfers - protected transfers in progress
var Trmu sync.Mutex
var Transfers = []Transfer{}

// Lookup - Return a pointer to the transfer if we find it in Transfers, nil otherwise
func Lookup(direction bool, session uint32, peer string) *Transfer {
	for _, t := range Transfers {
		// Check if direction (Initiator or Responder), session # and IP address match in our current
		// list of transfers (if so then return a pointer to it)

		if direction == t.Direction && session == t.Session && t.Conn.RemoteAddr().String() == peer {
			return &t
		}
	}
	return nil
}

// Lookup a host & session and return transfer pointer or nil if it does not exist
func Match(addr string, session uint32) *Transfer {
	for i := len(Transfers) - 1; i >= 0; i-- {
		if addr == Transfers[i].Conn.RemoteAddr().String() && session == Transfers[i].Session {
			return &Transfers[i]
		}
	}
	return nil
}

// CNew - Add a new transfer to the Transfers list
func NewInitiator(g *gocui.Gui, ttype string, peer *net.UDPAddr, fname string, c *sarflags.Cliflags) (*Transfer, error) {
	// screen.Fprintln(g,  "red_black", "Addtran for ", ip.String(), " ", fname, " ", flags)
	for _, i := range Transfers { // Don't add duplicates (ie dont try act on same fname)
		// Make sure we have connections to all the current Transfers
		if i.Conn == nil {
			emsg := "No connection exists to peer: " + peer.String()
			ErrPrintln(g, "red_black", emsg)
			// Remove the transfer as we have no connection to a peer
			if err := i.Remove(); err != nil {
				ErrPrintln(g, "red_black", "Can't remove transfer of", i.Session)
			}
			return nil, errors.New(emsg)
		}
		if peer.String() == i.Conn.RemoteAddr().String() && fname == i.Filename { // We can't write to same file
			emsg := fmt.Sprintf("Initiator Transfer for %s to %s is currently in progress, cannnot add transfer",
				fname, peer.String())
			ErrPrintln(g, "red_black", emsg)
			return nil, errors.New(emsg)
		}
	}

	// Lock it as we are going to add a new transfer slice
	Trmu.Lock()
	defer Trmu.Unlock()
	t := new(Transfer)
	t.Direction = Initiator
	t.Ttype = ttype
	t.Tstamptype = c.Timestamp
	t.Session = newsession()

	var err error
	// Dial the peer to create the connection
	if t.Conn, err = net.DialUDP("udp", nil, peer); err != nil {
		ErrPrintln(g, "red_black", "Cannot dial peer "+peer.String()+" "+err.Error())
	}

	t.Filename = fname

	// Copy the FLAGS to t.cliflags
	if t.Cliflags, err = c.CopyCliflags(); err != nil {
		panic(err)
	}
	msg := fmt.Sprintf("Initiator Added %s Transfer to %s %s",
		t.Ttype, t.Conn.RemoteAddr().String(), t.Filename)
	Transfers = append(Transfers, *t)
	MsgPrintln(g, "green_black", msg)
	return t, nil
}

// New - Add a new transfer to the Transfers list upon receipt of a request
// when we receive a request we are therefore a "server"
func NewResponder(g *gocui.Gui, r request.Request, peer string) error {

	var err error
	if Lookup(Responder, r.Session, peer) != nil {
		emsg := fmt.Sprintf("Transfer %s for session %d to %s is currently in progress, cannnot duplicate transfer",
			Directions[Responder], r.Session, peer)
		ErrPrintln(g, "red_black", emsg)
		return errors.New(emsg)
	}
	var udpaddr *net.UDPAddr
	if udpaddr, err = net.ResolveUDPAddr("udp", peer); err != nil {
		return err
	}

	var tc *net.UDPConn
	// Dial the peer to create the connection
	if tc, err = net.DialUDP("udp", nil, udpaddr); err != nil {
		ErrPrintln(g, "red_black", "Cannot dial peer "+udpaddr.String()+" "+err.Error())
		return err
	}

	// Create the transfer record
	Trmu.Lock()
	defer Trmu.Unlock()
	t := new(Transfer)

	t.Conn = tc
	// Lock it as we are going to add a new transfer
	t.Direction = Responder // We are the Responder
	t.Session = r.Session
	// The Header flags set for the transfer
	t.Version = sarflags.GetStr(r.Header, "version")       // What version of saratoga
	t.Ttype = sarflags.GetStr(r.Header, "reqtype")         // What is the request type "get,getrm,put,putrm,putblind,rm"
	t.Fileordir = sarflags.GetStr(r.Header, "fileordir")   // Are we handling a file or directory entry
	t.Udplite = sarflags.GetStr(r.Header, "udplite")       // Should always be "no"
	t.Stream = sarflags.GetStr(r.Header, "stream")         // Denotes a named pipe
	t.Descriptor = sarflags.GetStr(r.Header, "descriptor") // What descriptor we use for the transfer

	t.Havemeta = false
	t.Framecount = 0 // No data yet. count of data frames
	t.Csumtype = ""  // We don't know checksum type until we get a metadata
	t.Checksum = nil // Nor do we know what it is
	t.Filename = r.Fname
	t.Tstamptype = "" // Filled out with status or data frame "localinterp,posix32,posix64,posix32_32,posix64_32,epoch2000_32"
	t.Progress = 0    // Current progress indicator
	t.Inrespto = 0    // Cururent In response to indicator
	t.Dir = nil       //

	// flags
	// property - normalfile, normaldirectory, specialfile, specialdirectory
	// descriptor - d16, d32, d64, d128
	// reliability - yes, no
	var flags string

	switch t.Ttype {
	case "get", "getrm", "rm": // We are acting on a file local to this system
		// Find the file metadata to get it's properties
		t.Fileinfo = new(dirent.FileMetaData)
		if err = t.Fileinfo.FileMeta(t.Filename); err != nil {
			t.Conn.Close()
			return err
		}
		if t.Fileinfo.IsDir {
			flags = sarflags.AddFlag("", "property", "normaldirectory")
			flags = sarflags.AddFlag(flags, "reliability", "yes")
		} else if t.Fileinfo.IsRegular {
			flags = sarflags.AddFlag("", "property", "normalfile")
			flags = sarflags.AddFlag(flags, "reliability", "yes")
		} else { // specialfile (no such thing as a "specialdirectory")
			flags = sarflags.AddFlag("", "property", "specialfile")
			flags = sarflags.AddFlag(flags, "reliability", "no")
		}
		flags = sarflags.AddFlag(flags, "descriptor", t.Descriptor)

	case "put", "putrm": // FIX THIS!!!!!!! We are creating/deleting a file local to this system
		flags = sarflags.AddFlag(flags, "reliability", "yes")
	case "putblind": // We are putting a file onto this system
		flags = sarflags.AddFlag(flags, "reliability", "no")
	}
	t.Dir = new(dirent.DirEnt)
	if err = t.Dir.New(flags, t.Filename); err != nil {
		t.Conn.Close()
		return err
	}
	t.Curfills = nil
	if t.Cliflags, err = sarflags.Cliflag.CopyCliflags(); err != nil {
		t.Conn.Close()
		return errors.New("cannot copy CLI flags for transfer")
	}
	t.Data = nil // Buffered data

	msg := fmt.Sprintf("Added %s Transfer to %s session %d",
		Directions[t.Direction], peer, r.Session)
	Transfers = append(Transfers, *t)
	MsgPrintln(g, "green_black", msg)
	return nil
}

// Info - List transfers in progress to msg window
func Info(g *gocui.Gui, ttype string) {
	var tinfo []Transfer

	for i := range Transfers {
		if ttype == "" || Transfers[i].Ttype == ttype {
			tinfo = append(tinfo, Transfers[i])
		}
	}
	if len(tinfo) > 0 {
		var maxaddrlen, maxfname int // Work out the width for the table
		for key := range tinfo {
			if len(tinfo[key].Conn.RemoteAddr().String()) > maxaddrlen {
				maxaddrlen = len(tinfo[key].Conn.RemoteAddr().String())
			}
			if len(tinfo[key].Filename) > maxfname {
				maxfname = len(tinfo[key].Conn.RemoteAddr().String())
			}
		}
		// Table format
		sfmt := fmt.Sprintf("|%%6s|%%8s|%%%ds|%%%ds|\n", maxaddrlen, maxfname)
		sborder := fmt.Sprintf(sfmt, strings.Repeat("-", 6), strings.Repeat("-", 8),
			strings.Repeat("-", maxaddrlen), strings.Repeat("-", maxfname))

		var sslice sort.StringSlice
		for key := range tinfo {
			sslice = append(sslice, tinfo[key].FmtPrint(sfmt))
		}
		sort.Sort(sslice)

		sbuf := sborder
		sbuf += fmt.Sprintf(sfmt, "Direct", "Tran Typ", "IP", "Fname")
		sbuf += sborder
		for key := 0; key < len(sslice); key++ {
			sbuf += sslice[key]
		}
		sbuf += sborder
		MsgPrintln(g, "magenta_black", sbuf)
	} else {
		msg := fmt.Sprintf("No %s transfers currently in progress", ttype)
		MsgPrintln(g, "green_black", msg)
	}
}

// WriteStatus -- compose & send status frames
// Our connection to the client is conn
// We assemble Status using sflags
// We transmit status immediately
// We send back a string holding the status error code or "success" keeps transfer alivea
func (t *Transfer) WriteStatus(g *gocui.Gui, sflags string) string {

	if t.Conn == nil {
		MsgPrintln(g, "cyan_black", "No Connection to write to")
		return "badstatus"
	}
	MsgPrintln(g, "cyan_black", "Responder Assemble & Send status to ", t.Conn.RemoteAddr().String())
	var maxholes = stpaylen(sflags) // Work out maximum # holes we can put in a single status frame

	errf := sarflags.FlagValue(sflags, "errcode")
	var lasthole int
	h := t.Curfills.Getholes()
	if errf == "success" {
		lasthole = len(h) // How many holes do we have
	}

	var framecnt int // Number of status frames we will need (at least 1)
	flags := sflags
	if lasthole <= maxholes {
		framecnt = 1
		flags = sarflags.ReplaceFlag(sflags, "allholes", "yes")
	} else {
		h := t.Curfills.Getholes()
		framecnt = len(h)/maxholes + 1
		flags = sarflags.ReplaceFlag(sflags, "allholes", "no")
	}

	// Loop through creating and sending the status frames with the holes in them
	for fc := 0; fc < framecnt; fc++ {
		starthole := fc * maxholes
		endhole := starthole + maxholes
		if endhole > lasthole {
			endhole = lasthole
		}

		var st status.Status
		h := t.Curfills.Getholes()
		sinfo := status.Sinfo{Session: t.Session, Progress: t.Progress, Inrespto: t.Inrespto, Holes: h}
		if st.New(flags, &sinfo) != nil {
			ErrPrintln(g, "red_black", "Cannot asemble status")
			return "badstatus"
		}
		if se := st.Send(t.Conn); se != nil {
			ErrPrintln(g, "red_black", se.Error())
			return "badstatus"
		}
		PacketPrintln(g, "cyan_black", "Tx ", st.ShortPrint())
		MsgPrintln(g, "cyan_black", "Responder Sent Status:", st.Print(),
			" to ", t.Conn.RemoteAddr().String())
	}
	return "success"
}

// Change - Add metadata information to the Transfer in Transfers list upon receipt of a metadata
func (t *Transfer) Change(g *gocui.Gui, m metadata.MetaData) error {
	// Lock it as we are going to add a new transfer slice
	Trmu.Lock()
	defer Trmu.Unlock()
	t.Csumtype = sarflags.GetStr(m.Header, "csumtype")
	t.Checksum = make([]byte, len(m.Checksum))
	copy(t.Checksum, m.Checksum)
	t.Dir = m.Dir.Copy()
	// Create the file buffer for the transfer
	// AT THE MOMENT WE ARE HOLDING THE WHOLE FILE IN A MEMORY BUFFER!!!!
	// OF COURSE WE NEED TO SORT THIS OUT LATER
	if len(t.Data) == 0 { // Create the buffer only once
		t.Data = make([]byte, t.Dir.Size)
	}
	if len(t.Data) != (int)(m.Dir.Size) {
		emsg := fmt.Sprintf("Size of File Differs - Old=%d New=%d",
			len(t.Data), m.Dir.Size)
		return errors.New(emsg)
	}
	t.Havemeta = true
	MsgPrintln(g, "yellow_black", "Added metadata to transfer and file buffer size ", len(t.Data))
	return nil
}

// Remove - Remove a Transfer from the Transfers
func (t *Transfer) Remove() error {
	Trmu.Lock()
	defer Trmu.Unlock()

	for i := len(Transfers) - 1; i >= 0; i-- {
		if Lookup(t.Direction, t.Session, t.Conn.RemoteAddr().String()) != nil {
			Transfers = append(Transfers[:i], Transfers[i+1:]...)
			return nil
		}
	}
	emsg := fmt.Sprintf("Cannot remove %s Transfer for session %d to %s",
		Directions[t.Direction], t.Session, t.Conn.RemoteAddr().String())
	return errors.New(emsg)
}

// FmtPrint - String of relevant transfer info
func (t *Transfer) FmtPrint(sfmt string) string {
	return fmt.Sprintf(sfmt, "Initiator",
		t.Ttype,
		t.Conn.RemoteAddr().String(),
		t.Filename)
}

// Print - String of relevant transfer info
func (t *Transfer) Print() string {
	return fmt.Sprintf("%s|%s|%s|%s", Directions[t.Direction],
		t.Ttype,
		t.Conn.RemoteAddr().String(),
		t.Filename)
}

// YEAH!!!! Not Doing anything yet
func (t *Transfer) Do(g *gocui.Gui, e chan error) {
	MsgPrintln(g, "Doing command - Well actually not doing it!!!!!")
	ErrPrintln(g, "red_black", "Charles write some code to Do a transfer!!!!")
	if e != nil {
		e = nil
	}
	<-e
}

/* ************************************************************************************ */
// All of the different command line input handlers
// These handle I/O to the Screen, write to Err and Msg Windows, read from Cmd Window
/* ************************************************************************************ */

// Beacon CLI Info
// beacon <off|V4|V6i|ipaddr> [count]
type Beaconcmd struct {
	flags    string        // Header Flags set for beacons
	count    uint          // How many beacons to send 0|1 == 1
	interval uint          // interval in seconds between beacons 0|1 == 1
	v4mcast  bool          // Sending beacons to V4 Multicast
	v6mcast  bool          // Sending beacons to V6 Multicast
	hosts    []net.UDPAddr // List of addresses to send beacons to
}

var clibeacon Beaconcmd

// cmdBeacon - Beacon commands
func cmdBeacon(g *gocui.Gui, args []string) {

	// var bmu sync.Mutex // Protects beacon.Beacon structure (EID)
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()
	clibeacon.flags = sarflags.Setglobal("beacon", sarflags.Cliflag) // Initialise Global Beacon flags
	clibeacon.interval = sarflags.Cliflag.Timeout.Binterval          // Set up the correct interval
	clibeacon.count = 1                                              // Default is always to send a single beacon

	disable := false
	for argcnt := 0; argcnt < len(args); argcnt++ {
		switch argcnt {
		case 0:
			if len(args) == 1 {
				// Show current Cbeacon flags and lists - beacon
				if clibeacon.count != 0 && clibeacon.count != 1 {
					MsgPrintln(g, "yellow_black", clibeacon.count, " beacons to be sent")
				} else {
					clibeacon.count = 1
					MsgPrintln(g, "yellow_black", "Single Beacon to be sent")
				}
				if clibeacon.v4mcast {
					MsgPrintln(g, "yellow_black", "Sending IPv4 multicast beacons")
				}
				if clibeacon.v6mcast {
					MsgPrintln(g, "yellow_black", "Sending IPv6 multicast beacons")
				}
				if len(clibeacon.hosts) > 0 {
					MsgPrintln(g, "cyan_black", "Sending beacons to:")
					for _, h := range clibeacon.hosts {
						MsgPrintln(g, "cyan_black", "\t", h.String())
					}
				}
				if !clibeacon.v4mcast && !clibeacon.v6mcast && len(clibeacon.hosts) == 0 {
					MsgPrintln(g, "yellow_black", "No beacons currently being sent")
				}
				return
			}
		case 1:
			if len(args) == 2 {
				switch args[1] {
				case "?":
					MsgPrintln(g, "magenta_black", prhelp("beacon"))
					MsgPrintln(g, "green_black", prusage("beacon"))
					return
				case "off":
					clibeacon.flags = sarflags.Setglobal("beacon", sarflags.Cliflag)
					clibeacon.count = 0
					clibeacon.interval = sarflags.Cliflag.Timeout.Binterval
					MsgPrintln(g, "green_black", "Beacons Disabled")
					// remove and disable all beacons
					clibeacon.hosts = nil
					return
				case "v4":
					// V4 Multicast
					MsgPrintln(g, "cyan_black", "Sending beacon to IPv4 Multicast")
					clibeacon.flags = sarflags.Setglobal("beacon", sarflags.Cliflag)
					clibeacon.v4mcast = true
					clibeacon.count = 1
					// Start up the beacon client sending count IPv4 beacons
					if addr, err := sarnet.UDPAddress(sarflags.Cliflag.V4Multicast); err == nil {
						go sendbeacons(g, clibeacon.flags, clibeacon.count, clibeacon.interval, addr)
					}
					return
				case "v6":
					// V6 Multicast
					MsgPrintln(g, "cyan_black", "Sending beacon to IPv6 Multicast")
					clibeacon.flags = sarflags.Setglobal("beacon", sarflags.Cliflag)
					clibeacon.v6mcast = true
					clibeacon.count = 1
					if addr, err := sarnet.UDPAddress(sarflags.Cliflag.V6Multicast); err == nil {
						// Start up the beacon client sending count IPv6 beacons
						go sendbeacons(g, clibeacon.flags, clibeacon.count, clibeacon.interval, addr)
					}
					return
				default:
					// beacon <count> or beacon <ipaddr>
					if n, err := strconv.ParseUint(args[1], 10, 32); err == nil {
						// We have a number so it is the counter
						clibeacon.count = uint(n)
						if clibeacon.count == 0 {
							clibeacon.count = 1
						}
						return
					}
					// We have an IP Address so send it a beacon
					if addr, err := sarnet.UDPAddress(args[1]); err == nil {
						MsgPrintln(g, "green_black", "Sending ", clibeacon.count, " beacons to ", addr.String())
						go sendbeacons(g, clibeacon.flags, clibeacon.count, clibeacon.interval, addr)
						return
					}
					ErrPrintln(g, "red_black", "Invalid IP Address:", args[1])
					ErrPrintln(g, "red_black", prusage("beacon"))
					return
				}
			}
			// Otherwise we have more args than 1 so send the beacon from the first arg
			if addr, err := sarnet.UDPAddress(args[1]); err == nil {
				MsgPrintln(g, "green_black", "Sending ", clibeacon.count, " beacons to ", addr.String())
				go sendbeacons(g, clibeacon.flags, clibeacon.count, clibeacon.interval, addr)
				return
			} else {
				ErrPrintln(g, "blue_black", err.Error())
			}
			if args[1] == "off" {
				disable = true
			}
		default:
			if addr, err := sarnet.UDPAddress(args[argcnt]); err == nil {
				if disable {
					// Remove the host from the list
					clibeacon.hosts = sarnet.RemoveUDPAddrValue(clibeacon.hosts, addr)
					return
				}
				// Send the beacons to the host
				MsgPrintln(g, "green_black", "Sending ", clibeacon.count, " beacons to ", addr.String())
				go sendbeacons(g, clibeacon.flags, clibeacon.count, clibeacon.interval, addr)
				return
			}
			ErrPrintln(g, "red_black", "Invalid IP Address:", args[argcnt])
			ErrPrintln(g, "red_black", prusage("beacon"))
		}
	}
}

func cmdCancel(g *gocui.Gui, args []string) {
	MsgPrintln(g, "green_black", args)
}

func cmdChecksum(g *gocui.Gui, args []string) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()
	switch len(args) {
	case 1:
		MsgPrintln(g, "white_black", "Checksum ", sarflags.Cliflag.Global["csumtype"])
		return
	case 2:
		switch args[1] {
		case "?": // usage
			MsgPrintln(g, "magenta_black", prhelp("checksum"))
			MsgPrintln(g, "green_black", prusage("checksum"))
			return
		case "off", "none":
			sarflags.Cliflag.Global["csumtype"] = "none"
		case "crc32":
			sarflags.Cliflag.Global["csumtype"] = "crc32"
		case "md5":
			sarflags.Cliflag.Global["csumtype"] = "md5"
		case "sha1":
			sarflags.Cliflag.Global["csumtype"] = "sha1"
		default:
			ErrPrintln(g, "green_red", prusage("checksum"))
			return
		}
		MsgPrintln(g, "green_black", "Checksum ", sarflags.Cliflag.Global["csumtype"])
		return
	default:
		ErrPrintln(g, "green_red", prusage("checksum"))
	}
	ErrPrintln(g, "green_red", prusage("checksum"))
}

// cmdDescriptor -- set descriptor size 16,32,64,128 bits
func cmdDescriptor(g *gocui.Gui, args []string) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		MsgPrintln(g, "green_black", "Descriptor ", sarflags.Cliflag.Global["descriptor"])
		return
	case 2:
		switch args[1] {
		case "?": // usage
			MsgPrintln(g, "magenta_black", prhelp("descriptor"))
			MsgPrintln(g, "green_black", prusage("descriptor"))
			return
		case "auto":
			if sarflags.MaxUint <= sarflags.MaxUint16 {
				sarflags.Cliflag.Global["descriptor"] = "d16"
				break
			}
			if sarflags.MaxUint <= sarflags.MaxUint32 {
				sarflags.Cliflag.Global["descriptor"] = "d32"
				break
			}
			if sarflags.MaxUint <= sarflags.MaxUint64 {
				sarflags.Cliflag.Global["descriptor"] = "d64"
				break
			}
			MsgPrintln(g, "red_black", "128 bit descriptors not supported on this platform")
		case "d16":
			if sarflags.MaxUint > sarflags.MaxUint16 {
				sarflags.Cliflag.Global["descriptor"] = "d16"
			} else {
				MsgPrintln(g, "red_black", "16 bit descriptors not supported on this platform")
			}
		case "d32":
			if sarflags.MaxUint > sarflags.MaxUint32 {
				sarflags.Cliflag.Global["descriptor"] = "d32"
			} else {
				MsgPrintln(g, "red_black", "32 bit descriptors not supported on this platform")
			}
		case "d64":
			if sarflags.MaxUint <= sarflags.MaxUint64 {
				sarflags.Cliflag.Global["descriptor"] = "d64"
			} else {
				MsgPrintln(g, "red_black", "64 bit descriptors are not supported on this platform")
				MsgPrintln(g, "red_black", "MaxUint=", sarflags.MaxUint,
					" <= MaxUint64=", sarflags.MaxUint64)
			}
		case "d128":
			MsgPrintln(g, "red_black", "128 bit descriptors not supported on this platform")
		default:
			ErrPrintln(g, "red_black", "usage:", prusage("descriptor"))
		}
		MsgPrintln(g, "green_black", "Descriptor size is ", sarflags.Cliflag.Global["descriptor"])
		return
	}
	ErrPrintln(g, "red_black", "usage:", prusage("descriptor"))
}

// Cexit = Exit level to quit from saratoga
var Cexit = -1

// cmdExit -- Quit saratoga
func cmdExit(g *gocui.Gui, args []string) {
	switch len(args) {
	case 1: // exit 0
		Cexit = 0
		MsgPrintln(g, "green_black", "Good Bye!")
		return
	case 2:
		switch args[1] {
		case "?": // Usage
			MsgPrintln(g, "magenta_black", prhelp("exit"))
			MsgPrintln(g, "green_black", prusage("exit"))
		case "0": // exit 0
			Cexit = 0
			MsgPrintln(g, "green_black", "Good Bye!")
		case "1": // exit 1
			Cexit = 1
			MsgPrintln(g, "green_black", "Good Bye!")
		default: // Help
			ErrPrintln(g, "red_black", prusage("exit"))
		}
	default:
		ErrPrintln(g, "red_black", prusage("exit"))
	}
}

// cmdFiiles -- show currently open files and transfers in progress
func cmdFiles(g *gocui.Gui, args []string) {

	switch len(args) {
	case 1:
		Info(g, "")
		return
	case 2:
		if args[1] == "?" { // usage
			MsgPrintln(g, "magenta_black", prhelp("files"))
			MsgPrintln(g, "green_black", prusage("files"))
			return
		}
	}
	ErrPrintln(g, "red_black", prusage("files"))
}

func cmdFreespace(g *gocui.Gui, args []string) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		if sarflags.Cliflag.Global["freespace"] == "yes" {
			MsgPrintln(g, "green_black", "Free space is advertised")
		} else {
			MsgPrintln(g, "green_black", "Free space is not advertised")
		}
		return
	case 2:
		switch args[1] {
		case "?": // usage
			MsgPrintln(g, "magenta_black", prhelp("freespace"))
			MsgPrintln(g, "green_black", prusage("freespace"))
			return
		case "yes":
			MsgPrintln(g, "green_black", "freespace is advertised")
			sarflags.Cliflag.Global["freespace"] = "yes"
			return
		case "no":
			MsgPrintln(g, "green_black", "freespace is not advertised")
			sarflags.Cliflag.Global["freespace"] = "no"
			return
		}
	}
	ErrPrintln(g, "red_black", "usage:", prusage("freespace"))
}

// Initiator _get_
func cmdGet(g *gocui.Gui, args []string) {
	switch len(args) {
	case 1:
		Info(g, "get")
		return
	case 2:
		if args[1] == "?" {
			MsgPrintln(g, "magenta_black", prhelp("get"))
			MsgPrintln(g, "green_black", prusage("get"))
			return
		}
	case 3:
		// var t transfer.CTransfer

		if udpad, err := sarnet.UDPAddress(args[1]); err == nil {
			if _, err := NewInitiator(g, "get", udpad, args[2], sarflags.Cliflag); err != nil {
				return
			}
		}
		return
	}
	ErrPrintln(g, "red_black", prusage("get"))
}

// Initiator _getdir_
func cmdGetdir(g *gocui.Gui, args []string) {
	switch len(args) {
	case 1:
		Info(g, "getdir")
		return
	case 2:
		if args[1] == "?" {
			MsgPrintln(g, "magenta_black", prhelp("getdir"))
			MsgPrintln(g, "green_black", prusage("getdir"))
			return
		}
	case 3:
		if udpad, err := sarnet.UDPAddress(args[1]); err == nil {
			if _, err := NewInitiator(g, "getdir", udpad, args[2], sarflags.Cliflag); err != nil {
				MsgPrintln(g, "magenta_black", prhelp("getdir"))
				ErrPrintln(g, "green_black", prusage("getdir"))
			}
		} else {
			ErrPrintln(g, "green_black", "Invalid IP Address:", args[1])
		}
		return
	}
	ErrPrintln(g, "red_black", prusage("getdir"))
}

// Initiator _get_ then _delete_
func cmdGetrm(g *gocui.Gui, args []string) {
	switch len(args) {
	case 1:
		Info(g, "getrm")
		return
	case 2:
		if args[1] == "?" {
			MsgPrintln(g, "magenta_black", prhelp("getrm"))
			MsgPrintln(g, "green_black", prusage("getrm"))
			return
		}
	case 3:
		if udpad, err := sarnet.UDPAddress(args[1]); err == nil {
			if _, err := NewInitiator(g, "getrm", udpad, args[2], sarflags.Cliflag); err != nil {
				MsgPrintln(g, "magenta_black", prhelp("getrm"))
				ErrPrintln(g, "green_black", prusage("getrm"))
			}
		} else {
			ErrPrintln(g, "green_black", "Invalid IP Address:", args[1])
		}
		return
	}
	ErrPrintln(g, "red_black", prusage("getrm"))
}

func cmdHelp(g *gocui.Gui, args []string) {
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

func cmdInterval(g *gocui.Gui, args []string) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		if sarflags.Cliflag.Timeout.Binterval == 0 {
			MsgPrintln(g, "yellow_black", "Single Beacon Interation")
		} else {
			MsgPrintln(g, "yellow_black", "Beacons sent every ",
				sarflags.Cliflag.Timeout.Binterval, " seconds")
		}
		return
	case 2:
		switch args[1] {
		case "?":
			MsgPrintln(g, "magenta_black", prhelp("interval"))
			MsgPrintln(g, "green_black", prusage("interval"))
			return
		case "off":
			sarflags.Cliflag.Timeout.Binterval = 0
			return
		default:
			if n, err := strconv.Atoi(args[1]); err == nil && n >= 0 {
				sarflags.Cliflag.Timeout.Binterval = uint(n)
				return
			}
		}
		ErrPrintln(g, "red_black", prusage("interval"))
	}
	ErrPrintln(g, "red_black", prusage("interval"))
}

func cmdHistory(g *gocui.Gui, args []string) {
	switch len(args) {
	case 1:
		MsgPrintln(g, "green_black", "History not implemented yet")
		return
	case 2:
		switch args[1] {
		case "?":
			MsgPrintln(g, "magenta_black", prhelp("history"))
			MsgPrintln(g, "green_black", prusage("history"))
			return
		default:
			MsgPrintln(g, "green_black", "History not implemented yet")
			return
		}
	}
	ErrPrintln(g, "red_black", prusage("history"))
}

func cmdHome(g *gocui.Gui, args []string) {
	switch len(args) {
	case 1:
		MsgPrintln(g, "green_black", "Home not implemented yet")
		return
	case 2:
		switch args[1] {
		case "?":
			MsgPrintln(g, "magenta_black", prhelp("home"))
			MsgPrintln(g, "green_black", prusage("home"))
			return
		}
	}
	ErrPrintln(g, "red_black", prusage("home"))
}

func cmdLs(g *gocui.Gui, args []string) {
	if len(args) != 0 {
		ErrPrintln(g, "red_bblack", prusage("ls"))
		return
	}
	switch args[1] {
	case "?":
		MsgPrintln(g, "magenta_black", prhelp("ls"))
		MsgPrintln(g, "green_black", prusage("ls"))
		return
	}
	MsgPrintln(g, "green_black", "ls not implemented yet")
}

// Display all of the peer information learned frm beacons
func cmdPeers(g *gocui.Gui, args []string) {
	switch len(args) {
	case 1:
		if len(beacon.Peers) == 0 {
			MsgPrintln(g, "green_black", "No Peers")
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

		bfmt := fmt.Sprintf("+%%%ds+%%6s+%%%ds+%%3s+%%%ds+%%%ds+\n",
			addrlen, eidlen, dcrelen, dmodlen)
		sborder := fmt.Sprintf(bfmt,
			strings.Repeat("-", addrlen),
			strings.Repeat("-", 6),
			strings.Repeat("-", eidlen),
			strings.Repeat("-", 3),
			strings.Repeat("-", dcrelen),
			strings.Repeat("-", dmodlen))

		sfmt := fmt.Sprintf("|%%%ds|%%6s|%%%ds|%%3s|%%%ds|%%%ds|\n",
			addrlen, eidlen, dcrelen, dmodlen)
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
		sbuf += fmt.Sprintf(sfmt, "IP", "GB", "EID", "Des", "Created", "Modified")
		sbuf += sborder
		for key := 0; key < len(sslice); key++ {
			sbuf += sslice[key]
		}
		sbuf += sborder
		MsgPrintln(g, "green_black", sbuf)
		return
	case 2:
		if args[1] == "?" {
			MsgPrintln(g, "magenta_black", prhelp("peers"))
			MsgPrintln(g, "green_black", prusage("peers"))
			return
		}
	default:
		MsgPrintln(g, "magenta_black", prhelp("peers"))
		ErrPrintln(g, "red_black", prusage("peers"))
	}
}

// Initiator _put_
// send a file to a destination
func cmdPut(g *gocui.Gui, args []string) {

	switch len(args) {
	case 1:
		Info(g, "put")
		return
	case 2:
		if args[1] == "?" {
			MsgPrintln(g, "magenta_black", prhelp("put"))
			MsgPrintln(g, "green_black", prusage("put"))
			return
		}
	case 3:
		if udpad, err := sarnet.UDPAddress(args[1]); err == nil {
			if t, err := NewInitiator(g, "put", udpad, args[2], sarflags.Cliflag); err == nil && t != nil {
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
				return
			}
			MsgPrintln(g, "magenta_black", prhelp("getrm"))
			ErrPrintln(g, "green_black", prusage("getrm"))
			return
		}
		ErrPrintln(g, "green_black", "Invalid IP Address:", args[1])
	}
	ErrPrintln(g, "red_black", prusage("put"))
}

// Initiator _put_
// blind send a file to a destination not expecting return _status_ from Responder
func cmdPutblind(g *gocui.Gui, args []string) {

	switch len(args) {
	case 1:
		Info(g, "putblind")
		return
	case 2:
		if args[1] == "?" {
			MsgPrintln(g, "magenta_black", prhelp("putblind"))
			MsgPrintln(g, "green_black", prusage("putblind"))
			return
		}
	case 3:
		// We send the Metadata and do not bother with request/status exchange
		if udpad, err := sarnet.UDPAddress(args[1]); err == nil {
			if t, err := NewInitiator(g, "putblind", udpad, args[2], sarflags.Cliflag); err == nil && t != nil {
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
				MsgPrintln(g, "green_black", "putblind completed closing channel")
				close(errflag)
				return
			}
			MsgPrintln(g, "magenta_black", prhelp("putblind"))
			ErrPrintln(g, "green_black", prusage("putblind"))
			return
		}
		ErrPrintln(g, "green_black", "Invalid IP Address:", args[1])
	}
	ErrPrintln(g, "red_black", prusage("putblind"))
}

// Initiator _put_
// send a file file to a remote destination then remove it from the origin
func cmdPutrm(g *gocui.Gui, args []string) {

	switch len(args) {
	case 1:
		Info(g, "putrm")
		return
	case 2:
		if args[1] == "?" {
			MsgPrintln(g, "magenta_black", prhelp("putrm"))
			MsgPrintln(g, "green_black", prusage("putrm"))
			return
		}
	case 3:
		// var t *transfer.Transfer
		if udpad, err := sarnet.UDPAddress(args[1]); err == nil {
			if t, err := NewInitiator(g, "putrm", udpad, args[2], sarflags.Cliflag); err == nil && t != nil {
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
				MsgPrintln(g, "green_black", "putrm completed closing channel")
				close(errflag)
				MsgPrintln(g, "red_black", "Put and now removing (NOT) (ADD MORE CODE  HERE!!!!) file:", t.Print())
				return
			}
			ErrPrintln(g, "green_black", "Invalid IP Address:", args[1])
		}
		ErrPrintln(g, "red_black", prusage("putblind"))
	}
	ErrPrintln(g, "red_black", prusage("putrm"))
}

func cmdReqtstamp(g *gocui.Gui, args []string) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		if sarflags.Cliflag.Global["reqtstamp"] == "yes" {
			MsgPrintln(g, "green_black", "Time stamps requested")
		} else {
			MsgPrintln(g, "green_black", "Time stamps not requested")
		}
		return
	case 2:
		switch args[1] {
		case "?": // usage
			MsgPrintln(g, "magenta_black", prhelp("reqtstamp"))
			MsgPrintln(g, "green_black", prusage("reqtstamp"))
			return
		case "yes":
			sarflags.Cliflag.Global["reqtstamp"] = "yes"
			return
		case "no":
			sarflags.Cliflag.Global["reqtstamp"] = "no"
			return
		}
	}
	ErrPrintln(g, "red_black", "usage:", prusage("reqtstamp"))
}

// Initiator _delete_
// remove a file from a remote destination
func cmdRm(g *gocui.Gui, args []string) {

	switch len(args) {
	case 1:
		Info(g, "rm")
		return
	case 2:
		if args[1] == "?" {
			MsgPrintln(g, "magenta_black", prhelp("rm"))
			MsgPrintln(g, "green_black", prusage("rm"))
			return
		}
	case 3:
		if udpad, err := sarnet.UDPAddress(args[1]); err == nil {
			if t, err := NewInitiator(g, "rm", udpad, args[2], sarflags.Cliflag); err == nil && t != nil {
				errflag := make(chan error, 1) // The return channel holding the saratoga errflag
				go t.Do(g, errflag)            // Actually do the transfer
				errcode := <-errflag
				if errcode != nil {
					ErrPrintln(g, "red_black", "Error:", errcode,
						" Unable to remove file:", t.Print())
					if derr := t.Remove(); derr != nil {
						MsgPrintln(g, "red_black", "Unable to remove transfer:", t.Print())
					}
				}
				MsgPrintln(g, "green_black", "rm completed closing channel")
				close(errflag)
				return
			}
			MsgPrintln(g, "magenta_black", prhelp("rm"))
			ErrPrintln(g, "green_black", prusage("rm"))
			return
		}
		ErrPrintln(g, "green_black", "Invalid IP Address:", args[1])
	}
	ErrPrintln(g, "red_black", prusage("rm"))
}

// Initiator _getdir_, _delete_ ...
// remove a directory from a remote destination
func cmdRmdir(g *gocui.Gui, args []string) {

	switch len(args) {
	case 1:
		Info(g, "rmdir")
		return
	case 2:
		if args[1] == "?" {
			MsgPrintln(g, "magenta_black", prhelp("rmdir"))
			MsgPrintln(g, "green_black", prusage("rmdir"))
			return
		}
	case 3:
		if udpad, err := sarnet.UDPAddress(args[1]); err == nil {
			if t, err := NewInitiator(g, "rmdir", udpad, args[2], sarflags.Cliflag); err == nil && t != nil {
				errflag := make(chan error, 1) // The return channel holding the saratoga errflag
				go t.Do(g, errflag)            // Actually do the transfer
				errcode := <-errflag
				if errcode != nil {
					ErrPrintln(g, "red_black", "Error:", errcode,
						" Unable to remove file:", t.Print())
					if derr := t.Remove(); derr != nil {
						MsgPrintln(g, "red_black", "Unable to remove transfer:", t.Print())
					}
				}
				MsgPrintln(g, "green_black", "rmdir completed closing channel")
				close(errflag)
				return
			}
			MsgPrintln(g, "magenta_black", prhelp("rmdir"))
			ErrPrintln(g, "green_black", prusage("rmdir"))
			return
		}
		ErrPrintln(g, "green_black", "Invalid IP Address:", args[1])
	}
	ErrPrintln(g, "red_black", prusage("rmdir"))
}

func cmdRmtran(g *gocui.Gui, args []string) {

	switch len(args) {
	case 1:
		MsgPrintln(g, "magenta_black", prhelp("rmtran"))
		ErrPrintln(g, "red_black", prusage("rmtran"))
		return
	case 2:
		if args[1] == "?" {
			MsgPrintln(g, "magenta_black", prhelp("rmtran"))
			MsgPrintln(g, "green_black", prusage("rmtran"))
			return
		}
	case 4:
		ttype := args[1]
		addr := args[2]
		// We are unsigned so Atoi does not cut it
		if session, err := strconv.ParseUint(args[3], 10, 32); err == nil {
			if t := Match(addr, uint32(session)); t != nil {
				if err := t.Remove(); err != nil {
					MsgPrintln(g, "red_black", err.Error())
				}
				return
			}
			MsgPrintln(g, "red_black", "No such transfer:", ttype, " ", addr, " ", args[2])
		}
	}
	ErrPrintln(g, "red_black", prusage("rmtran"))
}

// Are we willing to transmit files
func cmdRxwilling(g *gocui.Gui, args []string) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		MsgPrintln(g, "green_black", "Receive Files:", sarflags.Cliflag.Global["rxwilling"])
		return
	case 2:
		switch args[1] {
		case "?":
			MsgPrintln(g, "magenta_black", prhelp("rxwilling"))
			MsgPrintln(g, "green_black", prusage("rxwilling"))
			return
		case "on":
			sarflags.Cliflag.Global["rxwilling"] = "yes"
			return
		case "off":
			sarflags.Cliflag.Global["rxwilling"] = "no"
			return
		case "capable":
			sarflags.Cliflag.Global["rxwilling"] = "capable"
			return
		}
	}
	ErrPrintln(g, "red_black", prusage("rxwilling"))
}

// Initiator _put_ not expecting _status_
// source is a named pipe not a file
func cmdStream(g *gocui.Gui, args []string) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		if sarflags.Cliflag.Global["stream"] == "yes" {
			MsgPrintln(g, "green_black", "Can stream")
		} else {
			MsgPrintln(g, "green_black", "Cannot stream")
		}
		return
	case 2:
		switch args[1] {
		case "?":
			MsgPrintln(g, "magenta_black", prhelp("stream"))
			MsgPrintln(g, "green_black", prusage("stream"))
			return
		case "yes":
			sarflags.Cliflag.Global["stream"] = "yes"
			return
		case "no":
			sarflags.Cliflag.Global["stream"] = "no"
			return
		}
	}
	ErrPrintln(g, "red_black", prusage("stream"))
}

// Timeout - set timeouts for responses to request/status/transfer in seconds
func cmdTimeout(g *gocui.Gui, args []string) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		if sarflags.Cliflag.Timeout.Metadata == 0 {
			MsgPrintln(g, "green_black", "metadata:No Timeout")
		} else {
			MsgPrintln(g, "green_black", "metadata:", sarflags.Cliflag.Timeout.Metadata, " sec")
		}
		if sarflags.Cliflag.Timeout.Request == 0 {
			MsgPrintln(g, "green_black", "request:No Timeout")
		} else {
			MsgPrintln(g, "green_black", "request:", sarflags.Cliflag.Timeout.Request, " sec")
		}
		if sarflags.Cliflag.Timeout.Status == 0 {
			MsgPrintln(g, "green_black", "status:No Timeout")
		} else {
			MsgPrintln(g, "green_black", "status:", sarflags.Cliflag.Timeout.Status, " sec")
		}
		if sarflags.Cliflag.Timeout.Datacounter == 0 {
			sarflags.Cliflag.Timeout.Datacounter = 100
			MsgPrintln(g, "green_black", "Data Counter every 100 frames")
		} else {
			MsgPrintln(g, "green_black", "datacounter:", sarflags.Cliflag.Timeout.Datacounter, " frames")
		}
		if sarflags.Cliflag.Timeout.Transfer == 0 {
			MsgPrintln(g, "green_black", "transfer:No Timeout")
		} else {
			MsgPrintln(g, "green_black", "transfer:", sarflags.Cliflag.Timeout.Transfer, " sec")
		}
		return
	case 2:
		switch args[1] {
		case "?":
			MsgPrintln(g, "magenta_black", prhelp("timeout"))
			MsgPrintln(g, "green_black", prusage("timeout"))
		case "request":
			if sarflags.Cliflag.Timeout.Request == 0 {
				MsgPrintln(g, "green_black", "request:No Timeout")
			} else {
				MsgPrintln(g, "green_black", "request:", sarflags.Cliflag.Timeout.Request, " sec")
			}
		case "metadata":
			if sarflags.Cliflag.Timeout.Request == 0 {
				MsgPrintln(g, "green_black", "metadata:No Timeout")
			} else {
				MsgPrintln(g, "green_black", "metadata:", sarflags.Cliflag.Timeout.Metadata, " sec")
			}
		case "status":
			if sarflags.Cliflag.Timeout.Status == 0 {
				MsgPrintln(g, "green_black", "status:No Timeout")
			} else {
				MsgPrintln(g, "green_black", "status:", sarflags.Cliflag.Timeout.Status, " sec")
			}
		case "datacounter":
			if sarflags.Cliflag.Timeout.Datacounter == 0 {
				sarflags.Cliflag.Timeout.Datacounter = 100
				MsgPrintln(g, "green_black", "datacounter:Never")
			} else {
				MsgPrintln(g, "green_black", "datacounter:", sarflags.Cliflag.Timeout.Datacounter, " frames")
			}
		case "transfer":
			if sarflags.Cliflag.Timeout.Transfer == 0 {
				MsgPrintln(g, "green_black", "transfer:No Timeout")
			} else {
				MsgPrintln(g, "green_black", "transfer:", sarflags.Cliflag.Timeout.Transfer, " sec")
			}
		default:
			ErrPrintln(g, "red_black", prusage("timeout"))
		}
		return
	case 3:
		if n, err := strconv.Atoi(args[2]); err == nil && n >= 0 {
			switch args[1] {
			case "metadata":
				sarflags.Cliflag.Timeout.Metadata = n
				if sarflags.Cliflag.Timeout.Metadata == 0 {
					MsgPrintln(g, "green_black", "metadata:No Timeout")
				} else {
					MsgPrintln(g, "green_black", "metadata:", sarflags.Cliflag.Timeout.Metadata, " sec")
				}
			case "request":
				sarflags.Cliflag.Timeout.Request = n
				if sarflags.Cliflag.Timeout.Request == 0 {
					MsgPrintln(g, "green_black", "request:No Timeout")
				} else {
					MsgPrintln(g, "green_black", "request:", sarflags.Cliflag.Timeout.Request, " sec")
				}
			case "status":
				sarflags.Cliflag.Timeout.Status = n
				if sarflags.Cliflag.Timeout.Status == 0 {
					MsgPrintln(g, "green_black", "status:No Timeout")
				} else {
					MsgPrintln(g, "green_black", "status:", sarflags.Cliflag.Timeout.Status, " sec")
				}
			case "datacounter":
				sarflags.Cliflag.Timeout.Datacounter = n
				if sarflags.Cliflag.Timeout.Datacounter == 0 {
					sarflags.Cliflag.Timeout.Datacounter = 100
					MsgPrintln(g, "green_black", "datacounter:Default set to every 100 frames")
				} else {
					MsgPrintln(g, "green_black", "datacounter:", sarflags.Cliflag.Timeout.Datacounter, " frames")
				}
			case "transfer":
				sarflags.Cliflag.Timeout.Transfer = n
				if sarflags.Cliflag.Timeout.Transfer == 0 {
					MsgPrintln(g, "green_black", "transfer:No Timeout")
				} else {
					MsgPrintln(g, "green_black", "transfer:", sarflags.Cliflag.Timeout.Transfer, " sec")
				}
			default:
				ErrPrintln(g, "red_black", prusage("timeout"))
			}
			return
		}
		if args[2] == "off" {
			switch args[1] {
			case "metadata":
				sarflags.Cliflag.Timeout.Metadata = 60
				MsgPrintln(g, "green_black", "metadata:", sarflags.Cliflag.Timeout.Metadata, " sec")
			case "request":
				sarflags.Cliflag.Timeout.Request = 60
				MsgPrintln(g, "green_black", "request:", sarflags.Cliflag.Timeout.Request, " sec")
			case "status":
				sarflags.Cliflag.Timeout.Status = 60
				MsgPrintln(g, "green_black", "status:", sarflags.Cliflag.Timeout.Status, " sec")
			case "datacounter":
				sarflags.Cliflag.Timeout.Datacounter = 100
				MsgPrintln(g, "green_black", "datacounter:", sarflags.Cliflag.Timeout.Datacounter, " frames")
			case "transfer":
				sarflags.Cliflag.Timeout.Transfer = 60
				MsgPrintln(g, "green_black", "transfer:", sarflags.Cliflag.Timeout.Transfer, " sec")
			}
			return
		}
		ErrPrintln(g, "red_black", prusage("timeout"))
		return
	}
	ErrPrintln(g, "red_black", prusage("timeout"))
}

// set the timestamp type we are using
func cmdTimestamp(g *gocui.Gui, args []string) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		MsgPrintln(g, "green_black", "Timestamps  are",
			sarflags.Cliflag.Global["reqtstamp"], " and ", sarflags.Cliflag.Timestamp)
		return
	case 2:
		switch args[1] {
		case "?":
			MsgPrintln(g, "magenta_black", prhelp("timestamp"))
			MsgPrintln(g, "green_black", prusage("timestamp"))
		case "off":
			sarflags.Cliflag.Global["reqtstamp"] = "no"
			// Don't change the TGlobal from what it was
		case "32":
			sarflags.Cliflag.Global["reqtstamp"] = "yes"
			sarflags.Cliflag.Timestamp = "posix32"
		case "32_32":
			sarflags.Cliflag.Global["reqtstamp"] = "yes"
			sarflags.Cliflag.Timestamp = "posix32_32"
		case "64":
			sarflags.Cliflag.Global["reqtstamp"] = "yes"
			sarflags.Cliflag.Timestamp = "posix64"
		case "64_32":
			sarflags.Cliflag.Global["reqtstamp"] = "yes"
			sarflags.Cliflag.Timestamp = "posix64_32"
		case "32_y2k":
			sarflags.Cliflag.Global["reqtstamp"] = "yes"
			sarflags.Cliflag.Timestamp = "epoch2000_32"
		case "local":
			sarflags.Cliflag.Global["reqtstamp"] = "yes"
			sarflags.Cliflag.Timestamp = "localinterp"
		default:
			ErrPrintln(g, "red_black", prusage("timestamp"))
		}
		return
	}
	ErrPrintln(g, "red_black", prusage("timestamp"))
}

// set the timezone we use for logs local or utc
func cmdTimezone(g *gocui.Gui, args []string) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		MsgPrintln(g, "green_black", "Timezone:", sarflags.Cliflag.Timezone)
		return
	case 2:
		switch args[1] {
		case "?":
			MsgPrintln(g, "magenta_black", prhelp("timezone"))
			MsgPrintln(g, "green_black", prusage("timezone"))
		case "local":
			sarflags.Cliflag.Timezone = "local"
		case "utc":
			sarflags.Cliflag.Timezone = "utc"
		default:
			ErrPrintln(g, "red_black", prusage("timezone"))
		}
		return
	}
	ErrPrintln(g, "red_black", prusage("timezone"))
}

// show current transfers in progress & % completed
func cmdTran(g *gocui.Gui, args []string) {
	switch len(args) {
	case 1:
		Info(g, "")
		return
	case 2:
		switch args[1] {
		case "?":
			MsgPrintln(g, "magenta_black", prhelp("tran"))
			MsgPrintln(g, "green_black", prusage("tran"))
		default:
			for _, tt := range Ttypes {
				if args[1] == tt {
					Info(g, args[1])
					return
				}
			}
			ErrPrintln(g, "red_black", prusage("tran"))
		}
		return
	}
	ErrPrintln(g, "red_black", prusage("tran"))
}

// we are willing to transmit files
func cmdTxwilling(g *gocui.Gui, args []string) {
	sarflags.Climu.Lock()
	defer sarflags.Climu.Unlock()

	switch len(args) {
	case 1:
		MsgPrintln(g, "green_black", "Transmit Files:", sarflags.Cliflag.Global["txwilling"])
		return
	case 2:
		switch args[1] {
		case "?":
			MsgPrintln(g, "magenta_black", prhelp("txwilling"))
			MsgPrintln(g, "green_black", prusage("txwilling"))
			return
		case "on":
			sarflags.Cliflag.Global["txwilling"] = "on"
			return
		case "off":
			sarflags.Cliflag.Global["txwilling"] = "off"
			return
		case "capable":
			sarflags.Cliflag.Global["txwilling"] = "capable"
			return
		}
	}
	ErrPrintln(g, "red_black", prusage("txwilling"))
}

// Show all commands usage
func cmdUsage(g *gocui.Gui, args []string) {
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

type cmdfunc func(*gocui.Gui, []string)

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
func Run(g *gocui.Gui, name string) bool {
	if name == "" { // Handle just return
		return true
	}
	// for k, v := range c.Global {
	//	MsgPrintln(g, "white_red", k, "=", v)
	// }
	// MsgPrintln(g, "white_black", "Commands=", sarflags.Commands)
	// Get rid of leading and trailing whitespace
	s := strings.TrimSpace(name)
	vals := strings.Fields(s)
	// Lookup the cmd and index func to run via cmdhandler map
	for key := range sarflags.Commands {
		if key == vals[0] {
			fn, ok := cmdhandler[key]
			if ok {
				go fn(g, vals)
				return true
			}
			ErrPrintln(g, "red_black", "Invalid command:", vals[0])
		}
	}
	ErrPrintln(g, "red_black", "Invalid command:", vals[0])
	return false
}
