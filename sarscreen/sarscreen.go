// Handle screen outputs for views in colours

package sarscreen

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
	"black":   "30",
	"red":     "31",
	"green":   "32",
	"yellow":  "33",
	"blue":    "34",
	"magenta": "35",
	"cyan":    "36",
	"white":   "37",
	// Bright foreground
	"bblack":   "30;1",
	"bred":     "31;1",
	"bgreen":   "32;1",
	"byellow":  "33;1",
	"bblue":    "34;1",
	"bmagenta": "35;1",
	"bcyan":    "36;1",
	"bwhite":   "37;1",
}

// Background Colours (b=bright)
var bg = map[string]string{
	"black":   "40",
	"red":     "41",
	"green":   "42",
	"yellow":  "43",
	"blue":    "44",
	"magenta": "45",
	"cyan":    "46",
	"white":   "47",
	/* Bright background These do not work, they should
	"bblack":   "100",
	"bred":     "101",
	"bgreen":   "102",
	"byellow":  "103",
	"bblue":    "104",
	"bmagenta": "105",
	"bcyan":    "016",
	"bwhite":   "107",
	*/
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
