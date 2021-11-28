// Handle screen outputs for views in colours for Saratoga

package sarwin

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/jroimartin/gocui"
)

// Ansi Colour Escape Sequences
var ansiprefix = "\033["
var ansipostfix = "m"
var ansiseparator = ";"
var ansioff = "\033[0m" // Turn ansii escapesequence off

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

/*
// Jump to the last row in a view
func gotolastrow(g *gocui.Gui, v *gocui.View) {
	ox, oy := v.Origin()
	cx, cy := v.Cursor()

	lines := len(v.BufferLines())
	// ErrPrintf(g, "white_black", "gotolastrow ox=%d oy=%d cx=%d cy=%d blines=%d\n",
	//	oy, oy, cx, cy, lines)
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
*/

// fprintf out in ANSII escape sequence in colour to view
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
