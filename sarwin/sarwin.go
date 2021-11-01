// Handle screen outputs for views in colours for Saratoga

package sarwin

import (
	"fmt"
	"log"
	"strings"

	"github.com/jroimartin/gocui"
)

// Ansi Colour Escape Sequences
const ansiprefix = "\033["
const ansipostfix = "m"
const ansiseparator = ";"
const ansioff = "0" // "\033[0m"

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
	"black":   "40",
	"red":     "41",
	"green":   "42",
	"yellow":  "43",
	"blue":    "44",
	"magenta": "45",
	"cyan":    "46",
	"white":   "47",
	// Bright background (this just makes foreground lighter)
	"bblack":   "40;1",
	"bred":     "41;1",
	"bgreen":   "42;1",
	"byellow":  "43;1",
	"bblue":    "44;1",
	"bmagenta": "45;1",
	"bcyan":    "46;1",
	"bwhite":   "47;1",
}

// Viewinfo -- Data and info on views (cmd & msg)
type Viewinfo struct {
	Commands []string // History of commands
	Prompt   string   // Command line prompt prefix
	Curline  int      // What is my current line #
	Ppad     int      // Number of pad characters around prompt e.g. prompt[99]: would be 3 for the []:
	Numlines int      // How many lines do we have
}

// Create ansi sequence for colour change with c format of fg_bg (e.g. red_black)
func setcolour(c string) string {

	var fgok bool
	var bgok bool

	if c == "off" {
		return ansiprefix + ansioff + ansipostfix
	}
	sequence := strings.Split(c, "_")

	// Check that the colors are OK
	if len(sequence) == 2 {
		for c := range fg {
			if sequence[0] == c {
				fgok = true
				break
			}
		}
		for c := range bg {
			if sequence[1] == c {
				bgok = true
				break
			}
		}
	}
	if fgok && bgok {
		return ansiprefix + fg[sequence[0]] + ansiseparator + bg[sequence[1]] + ansipostfix
	}
	// Error so make it jump out at us
	return ansiprefix + fg["bwhite"] + ansiseparator + bg["red"] + ansipostfix
}

// Fprintf out in ANSII escape sequenace colour
// If colour is undefined then still print it out but in bright red to show there is an issue
func Fprintf(g *gocui.Gui, vname string, colour string, format string, args ...interface{}) {

	g.Update(func(g *gocui.Gui) error {
		v, err := g.View(vname)
		if err != nil {
			e := fmt.Sprintf("\nView Fprintf invalid view: %s", vname)
			log.Fatal(e)
		}
		colfmt := setcolour(colour) + format + setcolour("off")
		fmt.Fprintf(v, colfmt, args...)
		return nil
	})
}

// Fprintln out in ANSII escape sequenace colour
// If colour is undefined then still print it out but in bright red to show there is an issue
func Fprintln(g *gocui.Gui, vname string, colour string, args ...interface{}) {

	g.Update(func(g *gocui.Gui) error {
		v, err := g.View(vname)
		if err != nil {
			e := fmt.Sprintf("\nView Fprintln invalid view: %s", vname)
			log.Fatal(e)
		}
		fmt.Fprintf(v, "%s", setcolour(colour))
		fmt.Fprintln(v, args...)
		fmt.Fprintf(v, "%s", setcolour("off"))
		return nil
	})
}

// Send formatted output to "msg"  window
func MsgPrintf(g *gocui.Gui, colour string, format string, args ...interface{}) {
	Fprintf(g, "msg", colour, format, args...)
}

// Send unformatted output to "msg" window
func MsgPrintln(g *gocui.Gui, colour string, args ...interface{}) {
	Fprintln(g, "msg", colour, args...)
}

// Send formatted output to "cmd" window
func CmdPrintf(g *gocui.Gui, colour string, format string, args ...interface{}) {
	Fprintf(g, "cmd", colour, format, args...)
}

// Send unformatted output to "cmd" window
func CmdPrintln(g *gocui.Gui, colour string, args ...interface{}) {
	Fprintln(g, "cmd", colour, args...)
}

// Send formatted output to "cmd" window
func PacketPrintf(g *gocui.Gui, colour string, format string, args ...interface{}) {
	Fprintf(g, "packet", colour, format, args...)
}

// Send unformatted output to "cmd" window
func PacketPrintln(g *gocui.Gui, colour string, args ...interface{}) {
	Fprintln(g, "packet", colour, args...)
}
