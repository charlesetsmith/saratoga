// Test Saratoga Flags

package main

import (
	"log"
	"strconv"
	"strings"

	"cli"
	"screen"

	"github.com/charlesetsmith/saratoga/src/sarflags"
	"github.com/jroimartin/gocui"
)

// *******************************************************************

func switchView(g *gocui.Gui, v *gocui.View) error {
	if v == nil || v.Name() == "msg" {
		_, err := g.SetCurrentView("cmd")
		return err
	}
	_, err := g.SetCurrentView("msg")
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
			screen.Fprintln(screen.Msg, "red_black", "cursorDown() x out of range We should never see this!")
		}
	}
	return nil
}

// Up Arrow
func cursorUp(g *gocui.Gui, v *gocui.View) error {
	cx, cy := v.Cursor()
	screen.Fprintf(screen.Msg, "green_black", "CursorUp Position x=%d y=%d\n", cx, cy)

	if line, lineerr := v.Line(cy - 1); lineerr == nil {
		screen.Fprintln(screen.Msg, "red_black", "No Line ERROR!")
		if err := v.SetCursor(len(line), cy-1); err != nil {
			screen.Fprintln(screen.Msg, "blue_black", "SetCursor Out of range for cx=", cx, "cy=", cy-1)
			ox, oy := v.Origin()
			screen.Fprintln(screen.Msg, "blue_black", "Origin Reset ox=", ox, "oy=", oy)
			if err := v.SetOrigin(ox, oy); err != nil {
				screen.Fprintln(screen.Msg, "blue_black", "SetOrigin Out of range for ox=", ox, "oy=", oy)
				return nil
			}
			line, _ := v.Line(oy)
			if err := v.SetCursor(len(line), oy); err != nil {
				screen.Fprintln(screen.Msg, "red_black", "We should never EVER see this!")
			}
		} else {
			line, _ := v.Line(cy - 1)
			if err := v.SetCursor(len(line), cy-1); err != nil {
				screen.Fprintln(screen.Msg, "red_black", "We should never see this!")
			}
		}
		return nil
	}
	screen.Fprintln(screen.Msg, "red_black", "LINE ERROR!")

	return nil
}

// Commands - cli commands entered
var Commands []string

// This is where we process command line inputs after a CR entered
func getLine(g *gocui.Gui, v *gocui.View) error {
	if v.Name() == "msg" { // DOn;t do anyhting with return if we are in the Mwg View
		return nil
	}
	if FirstPass {
		cli.CurLine = 0
		screen.Fprintf(v, "yellow_black", "%s[%d]:", cli.Cprompt, cli.CurLine)
		screen.Cmd.SetCursor(len(cli.Cprompt)+3+len(strconv.Itoa(cli.CurLine)), 0)
		return nil
	}
	cx, cy := v.Cursor()
	line, _ := v.Line(cy)
	command := strings.SplitN(line, ":", 2)
	if command[1] == "" { // We have just hit enter - do nothing
		return nil
	}

	if err := cli.Docmd(command[1]); err != nil {
		screen.Fprintln(screen.Msg, "red_black", "Invalid Command: ", command[1])
	}
	if command[1] == "exit" || command[1] == "quit" {
		return quit(g, v)
	}
	Commands = append(Commands, command[1])

	cli.CurLine++
	screen.Fprintf(screen.Msg, "magenta_black", "CurLine=%d <%s>\n", cli.CurLine, command[1])
	screen.Fprintf(screen.Msg, "green_black", "cx=%d, cy=%d\n", cx, cy)

	// Have we scrolled past the length of v, if so reset the origin
	if err := v.SetCursor(len(cli.Cprompt)+len(strconv.Itoa(cli.CurLine))+3, cy+1); err != nil {
		ox, oy := v.Origin()
		if err := v.SetOrigin(ox, oy+1); err != nil {
			screen.Fprintln(screen.Msg, "red_black", "SetOrigin Error:", err)
			return err
		}
		// Reset the cursor to last line in v
		_ = v.SetCursor(len(cli.Cprompt)+len(strconv.Itoa(cli.CurLine))+3, cy)
	}
	screen.Fprintf(v, "yellow_black", "\n%s[%d]:", cli.Cprompt, cli.CurLine)
	/*
		bframe, _ := frames.BeaconMake("descriptor=d64,stream=no,txwilling=yes,rxwilling=yes,udplite=no,freespace=yes", "EID", 9000)
		binfo, _ := frames.BeaconGet(bframe)
		screen.Fprintln(screen.Msg, "yellow_black", frames.BeaconPrint(binfo))

		tframe, _ := frames.TimeStampNow("posix32")
		tinfo, _ := frames.TimeStampGet(tframe)
		screen.Fprintln(screen.Msg, "magenta_black", frames.TimeStampPrint(tinfo))

		tframe, _ = frames.TimeStampNow("posix64")
		tinfo, _ = frames.TimeStampGet(tframe)
		screen.Fprintln(screen.Msg, "magenta_black", frames.TimeStampPrint(tinfo))

		tframe, _ = frames.TimeStampNow("posix32_32")
		tinfo, _ = frames.TimeStampGet(tframe)
		screen.Fprintln(screen.Msg, "magenta_black", frames.TimeStampPrint(tinfo))

		tframe, _ = frames.TimeStampNow("posix64_32")
		tinfo, _ = frames.TimeStampGet(tframe)
		screen.Fprintln(screen.Msg, "magenta_black", frames.TimeStampPrint(tinfo))

		tframe, _ = frames.TimeStampNow("epoch2000_32")
		tinfo, _ = frames.TimeStampGet(tframe)
		screen.Fprintln(screen.Msg, "magenta_black", frames.TimeStampPrint(tinfo))

		var deframe []byte
		var derr error
		var deinfo frames.DirEnt

		if deframe, derr = frames.DirEntMake("descriptor=d32", "test.txt"); derr != nil {
			screen.Fprintln(screen.Msg, "red_black", "DirEntMake:", derr)
			return nil
		}
		if _, derr = frames.DirEntGet(deframe); derr != nil {
			screen.Fprintln(screen.Msg, "red_black", "DirentGet:", derr)
			return nil
		}
		screen.Fprintln(screen.Msg, "magenta_black", frames.DirEntPrint(deinfo))

		spay := "this is the payload"
		payload := make([]byte, len(spay))
		copy(payload, spay)
		var dframe []byte
		var err error

		if dframe, err = frames.DataMake("descriptor=d32,reqtstamp=yes,posix64", 100, 1001, payload); err != nil {
			return err
		}
		dinfo, _ := frames.DataGet(dframe)
		screen.Fprintln(screen.Msg, "magenta_black", frames.DataPrint(dinfo))

		if dframe, err = frames.DataMake("descriptor=d64", 100, 1002, payload); err != nil {
			return err
		}
		ndinfo, _ := frames.DataGet(dframe)
		screen.Fprintln(screen.Msg, "magenta_black", frames.DataPrint(ndinfo))

		var rframe []byte
		fname := "ThisIsFileName"
		var auth []byte

		if rframe, err = frames.RequestMake("descriptor=d16,reqtype=get,fileordir=directory", 9998, fname, auth); err != nil {
			return err
		}
		rinfo, _ := frames.RequestGet(rframe)
		screen.Fprintln(screen.Msg, "magenta_black", frames.RequestPrint(rinfo))

		var nauth = make([]byte, 3)
		copy(nauth, "abc")

		var nrframe []byte
		if nrframe, err = frames.RequestMake("descriptor=d32,reqtype=put,fileordir=directory", 9998, "NewName", nauth); err != nil {
			return err
		}
		nrinfo, _ := frames.RequestGet(nrframe)
		screen.Fprintln(screen.Msg, "magenta_black", frames.RequestPrint(nrinfo))

			var stframe []byte
			var ho []frames.Hole
			var stinfo frames.Status


			ah := frames.Hole{Start: 1, End: 2}
			ho = append(ho, ah)
			bh := frames.Hole{Start: 3, End: 4}
			ho = append(ho, bh)

			if stframe, err = frames.StatusMake("descriptor=d16,reqtstamp=yes,posix64,allholes=yes,metadatarecvd=no", 111, 999, 499, ho); err != nil {
				screen.Fprintln(screen.Msg, "red_black", err)
				return err
			}
			if stinfo, err = frames.StatusGet(stframe); err != nil {
				screen.Fprintln(screen.Msg, "red_black", err)
				return err
			}
			screen.Fprintln(screen.Msg, "magenta_black", frames.StatusPrint(stinfo))


				var mdframe []byte
				var mdinfo frames.MetaData

				if mdframe, err = frames.MetaDataMake("transfer=file,descriptor=d16,csumtype=md5", 111, "test.txt"); err != nil {
					screen.Fprintln(screen.Msg, "red_black", "MetaDataMake:", err)
					return nil
				}
				if mdinfo, err = frames.MetaDataGet(mdframe); err != nil {
					screen.Fprintln(screen.Msg, "red_black", "MetadataGet:", err)
					return nil
				}
				screen.Fprintln(screen.Msg, "magenta_black", frames.MetaDataPrint(mdinfo))
	*/
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
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
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

/*

func msgdisplay(g *gocui.Gui) error {

	// var x uint32
	// var sarflag uint32 = 0x0

	// All inputs happen via the cmd view
	if _, err := g.SetCurrentView("cmd"); err != nil {
		return err
	}

	// screen.Fprintf(screen.Msg, "cyan_black", "<%s>", screen.Cmd.Buffer())

		sarflag = sarflags.Set(sarflag, "version", "v1")
		x = sarflags.Get(sarflag, "version")
		screen.Fprintf(screen.Msg, "green_black", "Sarflag=%032b version=%032b\n", sarflag, x)

		sarflag = sarflags.Set(sarflag, "frametype", "data")
		x = sarflags.Get(sarflag, "frametype")
		screen.Fprintf(screen.Msg, "red_black", "Sarflag=%032b frametype=%032b\n", sarflag, x)

		sarflag = sarflags.Set(sarflag, "descriptor", "d64")
		x = sarflags.Get(sarflag, "descriptor")
		screen.Fprintf(screen.Msg, "cyan_black", "Sarflag =%032b descriptor=%032b\n", sarflag, x)

		screen.Fprintln(screen.Msg, "magenta_black", "Sarflag frametype=beacon")

		sarflags.Test(sarflag, "frametype", "data")
		screen.Fprintln(screen.Msg, "yellow_black", "descriptor=", sarflags.Name(sarflag, "descriptor"))

		// **********************************************************

		var y uint16
		var dflag uint16 = 0x0

		dflag = sarflags.SetD(dflag, "sod", "startofdirectory")
		y = sarflags.GetD(dflag, "sod")
		fmt.Fprintf(screen.Msg, "Dflag =%016b sod=%016b\n", dflag, y)

		dflag = sarflags.SetD(dflag, "properties", "normalfile")
		y = sarflags.GetD(dflag, "properties")
		fmt.Fprintf(screen.Msg, "Dflag =%016b properties=%016b\n", dflag, y)

		dflag = sarflags.SetD(dflag, "descriptor", "d32")
		y = sarflags.GetD(dflag, "descriptor")
		fmt.Fprintf(screen.Msg, "Dflag =%016b descriptor=%016b\n", dflag, y)

		fmt.Fprintln(screen.Msg, "Directory Properties=normalfile", sarflags.TestD(dflag, "properties", "normalfile"))
		fmt.Fprintln(screen.Msg, "properties=", sarflags.NameD(dflag, "properties"))

		// ******************************************************

		var z uint8
		var tflag uint8 = 0x0

		tflag = sarflags.SetT(tflag, "timestamp", "posix32_32")
		z = sarflags.GetT(tflag, "timestamp")

		screen.Fprintf(screen.Msg, "green_black", "Tflag =%08b timestamp=%08b\n", tflag, z)

		screen.Fprintln(screen.Msg, "green_black", "Timestamp=posix32_32", sarflags.TestT(tflag, "timestamp", "posix32_32"))
		screen.Fprintln(screen.Msg, "green_black", "timestamp=", sarflags.NameT(tflag, "timestamp"))

		re := sarflags.Frame("request")
		screen.Fprintln(screen.Msg, "blue_black", "request flags: ", re)

		st := sarflags.Frame("status")
		screen.Fprintln(screen.Msg, "cyan_black", "status flags: ", st)

		md := sarflags.Frame("metadata")
		screen.Fprintln(screen.Msg, "yellow_black", "metadata flags:", md)

		da := sarflags.Frame("data")
		screen.Fprintln(screen.Msg, "red_black", "data flags:", da)

		be := sarflags.Frame("beacon")
		screen.Fprintln(screen.Msg, "bright_white_black", "beacon flags:", be)

	return nil
}
*/

// FirstPass -- First time around layout we don;t put \n at end of prompt
var FirstPass = true

func layout(g *gocui.Gui) error {

	var err error

	ratio := 4 // Ratio of cmd to err views
	maxX, maxY := g.Size()
	// This is the command line input view -- cli inputs and return messages go here
	if screen.Cmd, err = g.SetView("cmd", 0, maxY-(maxY/ratio)+1, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		screen.Cmd.Title = "Command Line"
		screen.Cmd.Highlight = false
		screen.Cmd.BgColor = gocui.ColorBlack
		screen.Cmd.FgColor = gocui.ColorGreen
		screen.Cmd.Editable = true
		screen.Cmd.Overwrite = true
		screen.Cmd.Wrap = false
	}
	// This is the message view window - All sorts of status & error messages go here
	if screen.Msg, err = g.SetView("messages", 0, 0, maxX-1, maxY-maxY/ratio); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		screen.Msg.Title = "Messages"
		screen.Msg.Highlight = false
		screen.Msg.BgColor = gocui.ColorBlack
		screen.Msg.FgColor = gocui.ColorYellow
		screen.Msg.Editable = false
		screen.Msg.Wrap = false
		screen.Msg.Autoscroll = true
	}

	// All inputs happen via the cmd view
	if _, err := g.SetCurrentView("cmd"); err != nil {
		return err
	}

	// Display the prompt without the \n first time around
	if FirstPass {
		_ = getLine(g, screen.Cmd)
		FirstPass = false
	}
	return nil
}

// Global - map of fields and flags
var Global map[string]string

func main() {

	// Global Flags set in cli
	Global = make(map[string]string)
	// Give them some defaults
	Global["descriptor"] = "d64"
	Global["csumtype"] = "none"
	Global["freespace"] = "no"
	Global["txwilling"] = "yes"
	Global["rxwilling"] = "yes"
	Global["stream"] = "no"
	Global["reqtstamp"] = "no"
	Global["reqstatus"] = "no"

	for f := range Global {
		if !sarflags.Valid(f, Global[f]) {
			panic("Invalid Flag:", f, "=", Global[f])
		}
	}

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
