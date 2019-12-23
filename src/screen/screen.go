// Handle screen outputs for views in colours

package screen

import (
	"fmt"
	"log"

	"github.com/jroimartin/gocui"
)

// Ansi Colour Map Escape Codes
var colours = map[string]string{
	"off":                  "\033[0m",
	"red_black":            "\033[31;40m",
	"green_black":          "\033[32;40m",
	"yellow_black":         "\033[33;40m",
	"blue_black":           "\033[34;40m",
	"magenta_black":        "\033[35;40m",
	"cyan_black":           "\033[36;40m",
	"white_black":          "\033[37;40m",
	"black_red":            "\033[30;41m",
	"black_green":          "\033[30;42m",
	"black_yellow":         "\033[30;43m",
	"black_blue":           "\033[30;44m",
	"black_magenta":        "\033[30;45m",
	"black_cyan":           "\033[30;46m",
	"black_white":          "\033[30;47m",
	"bright_red_black":     "\033[31;1;40m",
	"bright_green_black":   "\033[32;1;40m",
	"bright_yellow_black":  "\033[33;1;40m",
	"bright_blue_black":    "\033[34;1;40m",
	"bright_magenta_black": "\033[35;1;40m",
	"bright_cyan_black":    "\033[36;1;40m",
	"bright_white_black":   "\033[37;1;40m",
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
		for col := range colours {
			if col == colour {
				colfmt := colours[colour] + format + colours["off"]
				fmt.Fprintf(v, colfmt, args...)
				return nil
			}
		}
		colfmt := colours["bright_red_black"] + format + colours["off"]
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
		for col := range colours {
			if col == colour {
				fmt.Fprintf(v, "%s", colours[col])
				fmt.Fprintln(v, args...)
				fmt.Fprintf(v, "%s", colours["off"])
				return nil
			}
		}
		fmt.Fprintf(v, "%s", colours["bright_red_black"])
		fmt.Fprintln(v, args...)
		fmt.Fprintf(v, "%s", colours["off"])
		return nil
	})
}
