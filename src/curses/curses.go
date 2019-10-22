package main

import (
	"unicode/utf8"

	runewidth "github.com/mattn/go-runewidth"
	termbox "github.com/nsf/termbox-go"
)

func tbprint(x, y int, fg, bg termbox.Attribute, msg string) {
	for _, c := range msg {
		termbox.SetCell(x, y, c, fg, bg)
		x += runewidth.RuneWidth(c)
	}
}

func fill(x, y, w, h int, cell termbox.Cell) {
	for ly := 0; ly < h; ly++ {
		for lx := 0; lx < w; lx++ {
			termbox.SetCell(x+lx, y+ly, cell.Ch, cell.Fg, cell.Bg)
		}
	}
}

func runeAdvanceLen(r rune, pos int) int {
	if r == '\t' {
		return tabstopLength - pos%tabstopLength
	}
	return runewidth.RuneWidth(r)
}

func voffsetCoffset(text []byte, boffset int) (voffset, coffset int) {
	text = text[:boffset]
	for len(text) > 0 {
		r, size := utf8.DecodeRune(text)
		text = text[size:]
		coffset++
		voffset += runeAdvanceLen(r, voffset)
	}
	return
}

func byteSliceGrow(s []byte, desiredcap int) []byte {
	if cap(s) < desiredcap {
		ns := make([]byte, len(s), desiredcap)
		copy(ns, s)
		return ns
	}
	return s
}

func byteSliceRemove(text []byte, from, to int) []byte {
	size := to - from
	copy(text[from:], text[to:])
	text = text[:len(text)-size]
	return text
}

func byteSliceInsert(text []byte, offset int, what []byte) []byte {
	n := len(text) + len(what)
	text = byteSliceGrow(text, n)
	text = text[:n]
	copy(text[offset+len(what):], text[offset:])
	copy(text[offset:], what)
	return text
}

const preferredHorizontalThreshold = 5
const tabstopLength = 8

// EditLine - structure holding text and offsets (one per line)
type EditLine struct {
	text          []byte
	lineVoffset   int
	cursorBoffset int // cursor offset in bytes
	cursorVoffset int // visual cursor offset in termbox cells
	cursorCoffset int // cursor offset in unicode code points
}

// Draw - Draws the EditLine in the given location, 'h' is the line we are drawing
func (el *EditLine) Draw(x, y, w, h int) {
	el.AdjustVOffset(w)

	const coldef = termbox.ColorDefault
	fill(x, y, w, h, termbox.Cell{Ch: ' '})

	t := el.text
	lx := 0
	tabstop := 0
	for {
		rx := lx - el.lineVoffset
		if len(t) == 0 {
			break
		}

		if lx == tabstop {
			tabstop += tabstopLength
		}

		if rx >= w {
			termbox.SetCell(x+w-1, y, '→',
				coldef, coldef)
			break
		}
		r, size := utf8.DecodeRune(t)
		if r == '\t' {
			for ; lx < tabstop; lx++ {
				rx = lx - el.lineVoffset
				if rx >= w {
					goto next
				}

				if rx >= 0 {
					termbox.SetCell(x+rx, y, ' ', coldef, coldef)
				}
			}
		} else {
			if rx >= 0 {
				termbox.SetCell(x+rx, y, r, coldef, coldef)
			}
			lx += runewidth.RuneWidth(r)
		}
	next:
		t = t[size:]
	}

	if el.lineVoffset != 0 {
		termbox.SetCell(x, y, '←', coldef, coldef)
	}
}

// AdjustVOffset - Adjusts line visual offset to a proper value depending on width
func (el *EditLine) AdjustVOffset(width int) {
	ht := preferredHorizontalThreshold
	maxHThreshold := (width - 1) / 2
	if ht > maxHThreshold {
		ht = maxHThreshold
	}

	threshold := width - 1
	if el.lineVoffset != 0 {
		threshold = width - ht
	}
	if el.cursorCoffset-el.lineVoffset >= threshold {
		el.lineVoffset = el.cursorVoffset + (ht - width + 1)
	}

	if el.lineVoffset != 0 && el.cursorVoffset-el.lineVoffset < ht {
		el.lineVoffset = el.cursorVoffset - ht
		if el.lineVoffset < 0 {
			el.lineVoffset = 0
		}
	}
}

// MoveCursorTo - move cursor to offset
func (el *EditLine) MoveCursorTo(boffset int) {
	el.cursorBoffset = boffset
	el.cursorVoffset, el.cursorCoffset = voffsetCoffset(el.text, boffset)
}

// RuneUnderCursor - What is the rune under the cursor
func (el *EditLine) RuneUnderCursor() (rune, int) {
	return utf8.DecodeRune(el.text[el.cursorBoffset:])
}

// RuneBeforeCursor - What is the rune before cursor
func (el *EditLine) RuneBeforeCursor() (rune, int) {
	return utf8.DecodeLastRune(el.text[:el.cursorBoffset])
}

// MoveCursorOneRuneBackward - Move backwards
func (el *EditLine) MoveCursorOneRuneBackward() {
	if el.cursorBoffset == 0 {
		return
	}
	_, size := el.RuneBeforeCursor()
	el.MoveCursorTo(el.cursorBoffset - size)
}

// MoveCursorOneRuneForward - Move forwards
func (el *EditLine) MoveCursorOneRuneForward() {
	if el.cursorBoffset == len(el.text) {
		return
	}
	_, size := el.RuneUnderCursor()
	el.MoveCursorTo(el.cursorBoffset + size)
}

// MoveCursorToBeginningOfTheLine - Move to the beggining of line
func (el *EditLine) MoveCursorToBeginningOfTheLine() {
	el.MoveCursorTo(0)
}

// MoveCursorToEndOfTheLine - Move to the end of line
func (el *EditLine) MoveCursorToEndOfTheLine() {
	el.MoveCursorTo(len(el.text))
}

// DeleteRuneBackward - Delete rune backward
func (el *EditLine) DeleteRuneBackward() {
	if el.cursorBoffset == 0 {
		return
	}

	el.MoveCursorOneRuneBackward()
	_, size := el.RuneUnderCursor()
	el.text = byteSliceRemove(el.text, el.cursorBoffset, el.cursorBoffset+size)
}

// DeleteRuneForward - Delete rune forward
func (el *EditLine) DeleteRuneForward() {
	if el.cursorBoffset == len(el.text) {
		return
	}
	_, size := el.RuneUnderCursor()
	el.text = byteSliceRemove(el.text, el.cursorBoffset, el.cursorBoffset+size)
}

// DeleteTheRestOfTheLine - Delete rest of line
func (el *EditLine) DeleteTheRestOfTheLine() {
	el.text = el.text[:el.cursorBoffset]
}

// InsertRune - Insert a rune
func (el *EditLine) InsertRune(r rune) {
	var buf [utf8.UTFMax]byte
	n := utf8.EncodeRune(buf[:], r)
	el.text = byteSliceInsert(el.text, el.cursorBoffset, buf[:n])
	el.MoveCursorOneRuneForward()
}

// CursorX - Please, keep in mind that cursor depends on the value of lineVoffset, which
// is being set on Draw() call, so.. call this method after the Draw() one.
func (el *EditLine) CursorX() int {
	return el.cursorVoffset - el.lineVoffset
}

// Imitialise the window environment
func initwin() {
	const EditlineWidth = 30
}

// Redraw screen
func redrawAll() {
	// Width of terminal Editline
	const EditLineWidth = 30
	const coldef = termbox.ColorDefault
	termbox.Clear(coldef, coldef)
	w, h := termbox.Size()

	initwin()

	// midy := h / 2
	// midx := (w - EditLineWidth) / 2

	// unicode box drawing chars around the edit box
	// termbox.SetCell(midx-1, midy, '│', coldef, coldef)
	// termbox.SetCell(midx+EditLineWidth, midy, '│', coldef, coldef)
	// termbox.SetCell(midx-1, midy-1, '┌', coldef, coldef)
	// termbox.SetCell(midx-1, midy+1, '└', coldef, coldef)
	// termbox.SetCell(midx+EditLineWidth, midy-1, '┐', coldef, coldef)
	// termbox.SetCell(midx+EditLineWidth, midy+1, '┘', coldef, coldef)
	// fill(midx, midy-1, EditLineWidth, 1, termbox.Cell{Ch: '─'})
	// fill(midx, midy+1, EditLineWidth, 1, termbox.Cell{Ch: '─'})

	// EditLine.Draw(midx, midy, EditLineWidth, 1)
	// termbox.SetCursor(midx+EditLine.CursorX(), midy)

	// tbprint(midx+6, midy+3, coldef, coldef, "Press ESC to quit")
	termbox.Flush()
}

// Win - A Win is a slice of Editline's
// Windows origin 0,0 is top left corner
type Win struct {
	el   []EditLine // Splices of lines in the Window
	topx int        // Top LH Corner of Window on the screen
	topy int
	botx int // Bottom RH Corner of Window on the screen
	boty int
	curx int // Current Cursor Position in Window on the screen
	cury int
}

// DrawBox - Draw a box around the OUTSIDE of a Win
func (w *Win) DrawBox(fg, bg termbox.Attribute) {
	termbox.SetCell(w.topx-1, w.topy-1, '┌', fg, bg)                          // Top Left
	termbox.SetCell(w.topx-1, w.boty+1, '└', fg, bg)                          // Bottom Left
	termbox.SetCell(w.botx+1, w.topy-1, '┐', fg, bg)                          // Top Right
	termbox.SetCell(w.botx+1, w.boty+1, '┘', fg, bg)                          // Bottom Right
	fill(w.topx, w.topy-1, w.botx-w.topx+1, 1, termbox.Cell{Ch: '─', Fg: fg}) // Top Row
	fill(w.topx, w.boty+1, w.botx-w.topx+1, 1, termbox.Cell{Ch: '─', Fg: fg}) // Bottom Row
	for i := w.topy; i <= w.boty; i++ {
		termbox.SetCell(w.topx-1, i, '│', fg, bg) // Left Column
		termbox.SetCell(w.botx+1, i, '│', fg, bg) // Right Column
	}
}

// NewWin - Create a new window within terminal position datum
func (w *Win) NewWin(startx, starty, endx, endy int, fg, bg termbox.Attribute) {
	w.topx = startx
	w.topy = starty
	w.botx = endx
	w.boty = endy
	w.curx = startx
	w.cury = starty

	height := w.boty - w.topy
	width := w.botx - w.topx

	w.DrawBox(fg, bg)
	for y := 0; y <= height; y++ {
		copy(w.el[y].text, "")
		w.el[y].lineVoffset = 0
		w.el[y].cursorBoffset = 0
		w.el[y].cursorVoffset = 0
		w.el[y].cursorCoffset = 0
		w.el[y].Draw(w.topx, y, w, h)
	}
	// Fill it all with spaces
	for y := w.topy; y <= w.boty; i++ {
		for x := w.topx; x <= w.botx; i++ {
			termbox.SetCell(x, y, ' ', fg, bg)
		}
	}
}

// ScrollUp - Scroll the Win contents up a line
func (w *Win) ScrollUp() {
	width := w.botx - w.topx
	height := w.boty - w.topy
	coldef := termbox.ColorDefault

	for y := 1; y <= height; y++ {
		for x := 0; x <= width; x++ {
			from := y*width + x
			to := (y-1)*width + x
			termbox.SetCell(x+w.topx, y+w.topy, w.buf[from].Ch, w.buf[from].Fg, w.buf[from].Bg)
			w.buf[to] = w.buf[from]
		}
	}
	// Set the last line
	for x := 0; x <= width; x++ {
		termbox.SetCell(x+w.topx, w.boty, 'J', coldef, coldef)
		w.buf[height*width+x] = termbox.Cell{Ch: 'J', Fg: coldef, Bg: coldef}
	}
	// And the Cursor
	w.curx = w.topx
	w.cury = w.boty
	termbox.SetCursor(w.curx, w.cury)
}

// Print out a msg at x, y 0,0 being top LH Corner of Win
func (w *Win) Print(x, y int, msg string, fg, bg termbox.Attribute) {
	width := w.botx - w.topx
	height := w.boty - w.topy
	for _, c := range msg {
		if x > width {
			x = 0
			y++
		}
		if y > height {
			w.ScrollUp()
			y = height
			x = 0

		}
		w.SetCell(x, y, c, fg, bg)
		x += runewidth.RuneWidth(c)
	}
	termbox.Flush()
}

// Move to x, y in the window
func (w *Win) Move(x, y int) (int, int) {
	width := w.botx - w.topx
	height := w.boty - w.topy

	if x > width {
		x = 0
		y++
	}
	if y > height {
		w.ScrollUp()
		y = height
		x = 0
	}
	return x, y
}

// SetCell - Place a rune in a position within a Win - 0,0 is top LH corner of a Win
// and handle nil, \n's and \t's
func (w *Win) SetCell(x, y int, ch rune, fg, bg termbox.Attribute) {
	var xpos, ypos int

	width := w.botx - w.topx
	height := w.boty - w.topy

	if ch == '\t' {
		for ts := 0; ts < tabstopLength; ts++ {
			x, y = w.Move(x, y)
			w.curx = w.topx + x
			w.cury = w.topy + y
			termbox.SetCell(w.curx, w.cury, 'T', fg, bg)
			to := y*width + x
			w.buf[to] = termbox.Cell{Ch: 'T', Fg: fg, Bg: bg}
			x++
		}
		termbox.SetCursor(w.curx, w.cury)
		return
	}

	x, y = w.Move(x, y)

	xpos = w.topx + x
	ypos = w.topy + y

	if ch != '\n' {
		to := y*width + x
		w.buf[to] = termbox.Cell{Ch: ch, Fg: fg, Bg: bg}
		termbox.SetCell(xpos, ypos, ch, fg, bg)
	} else {
		x, y = w.Move(x, y)
	}

	// Reposition the cursor
	if x < width {
		w.curx = xpos + 1
		w.cury = ypos
	} else if x == width {
		w.curx = w.topx
		if y < height {
			w.cury = ypos + 1
		} else {
			w.ScrollUp()
			w.cury = w.boty
		}
	}
	termbox.SetCursor(w.curx, w.cury)
}

func main() {

	var Cmdwin Win  // Cmmand window
	var Errwin Win  // Error msg output window
	var curwin *Win // The Current window Cmdwin or Errwin
	var curline int // The Current line in the curwin

	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()
	termbox.SetInputMode(termbox.InputEsc)

	const ratio int = 4
	const coldef = termbox.ColorDefault
	const colcmd = termbox.ColorGreen
	const colerr = termbox.ColorRed

	// What is the size of our terminal window
	width, height := termbox.Size()

	// Clear and Flush the terminal window
	termbox.Clear(coldef, coldef)
	termbox.Flush()

	// Error window is ratio of terminal at the top
	// Work out its size and draw a box around it
	errwidth := width - 1
	errheight := height - ((height - 2) / ratio) - 2
	errstartx := 1
	errstarty := 1
	errendy := errstarty + errheight - 2
	errendx := errstartx + errwidth - 2
	Errwin.NewWin(errstartx, errstarty, errendx, errendy, colerr, coldef)

	// Command window is 1/ratio of screen at bottom
	// Work out its size and draw a box around it
	cmdheight := (height - 2) / ratio
	cmdstartx := 1
	cmdstarty := errendy + 3
	cmdendy := cmdstarty + cmdheight - 2
	cmdendx := errendx
	Cmdwin.NewWin(cmdstartx, cmdstarty, cmdendx, cmdendy, colcmd, coldef)

	redrawAll()
	// Errwin.Print(0, 0, "ABCDEF", coldef, coldef)
	// Errwin.Print(Errwin.botx-Errwin.topx, Errwin.topy-Errwin.topy, "qweRTY", coldef, coldef)
	// Errwin.Print(Errwin.botx-Errwin.topx, 2, "H", coldef, coldef)
	// Errwin.Print(0, 2, "U", coldef, coldef)
	// Errwin.Print(Errwin.botx-Errwin.topx, Errwin.boty-Errwin.topy, "I", coldef, coldef)

	// Set the initial window to the error window
	curwin = &Errwin

	curline = 0
mainloop:
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			switch ev.Key {
			case termbox.KeyEsc:
				break mainloop
			case termbox.KeyArrowLeft, termbox.KeyCtrlB:
				curwin.el[curline].MoveCursorOneRuneBackward()
			case termbox.KeyArrowRight, termbox.KeyCtrlF:
				curwin.el[curline].MoveCursorOneRuneForward()
			case termbox.KeyArrowUp, termbox.KeyCtrlU:
				if curline > 0 {
					curline--
				}
			case termbox.KeyArrowDown, termbox.KeyCtrlD:
				if curline < curwin.boty-curwin.topy {
					curline++
				}
			case termbox.KeyBackspace, termbox.KeyBackspace2:
				curwin.el[curline].DeleteRuneBackward()
			case termbox.KeyDelete:
				curwin.el[curline].DeleteRuneForward()
			case termbox.KeyTab:
				curwin.el[curline].InsertRune('\t')
			case termbox.KeySpace:
				curwin.el[curline].InsertRune(' ')
			case termbox.KeyCtrlK:
				curwin.el[curline].DeleteTheRestOfTheLine()
			case termbox.KeyHome, termbox.KeyCtrlA:
				curwin.el[curline].MoveCursorToBeginningOfTheLine()
			case termbox.KeyEnd, termbox.KeyCtrlE:
				curwin.el[curline].MoveCursorToEndOfTheLine()
			default:
				if ev.Ch != 0 {
					curwin.el[curline].InsertRune(ev.Ch)
				}
			}
		case termbox.EventError:
			panic(ev.Err)
		}
		redrawAll()
	}
}
