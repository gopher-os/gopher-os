package tty

import "github.com/achilleasa/gopher-os/kernel/driver/video/console"

const (
	defaultFg = console.LightGrey
	defaultBg = console.Black
)

// Vt implements a simple terminal that can process LF and CR characters. The
// terminal uses a console device for its output.
type Vt struct {
	// Go interfaces will not work before we can get memory allocation working.
	// Till then we need to use concrete types instead.
	cons *console.Vga

	width  uint16
	height uint16

	curX    uint16
	curY    uint16
	curAttr console.Attr
}

func (t *Vt) Init(cons *console.Vga) {
	t.cons = cons
	t.width, t.height = cons.Dimensions()
	t.curX = 0
	t.curY = 0

	// Default to lightgrey on black text.
	t.curAttr = makeAttr(defaultFg, defaultBg)

}

// Clear clears the terminal.
func (t *Vt) Clear() {
	t.cons.Lock()
	defer t.cons.Unlock()

	t.clear()
}

// Position returns the current cursor position (x, y).
func (t *Vt) Position() (uint16, uint16) {
	t.cons.Lock()
	defer t.cons.Unlock()

	return t.curX, t.curY
}

// SetPosition sets the current cursor position to (x,y).
func (t *Vt) SetPosition(x, y uint16) {
	t.cons.Lock()
	defer t.cons.Unlock()

	if x >= t.width {
		x = t.width - 1
	}

	if y >= t.height {
		y = t.height - 1
	}

	t.curX, t.curY = x, y
}

// Write implements io.Writer.
func (t *Vt) Write(data []byte) (int, error) {
	t.cons.Lock()
	defer t.cons.Unlock()

	attr := t.curAttr
	for _, b := range data {
		switch b {
		case '\r':
			t.cr()
		case '\n':
			t.cr()
			t.lf()
		default:
			t.cons.Write(b, attr, t.curX, t.curY)
			t.curX++
			if t.curX == t.width {
				t.lf()
			}
		}
	}

	return len(data), nil
}

// cls clears the terminal.
func (t *Vt) clear() {
	t.cons.Clear(0, 0, t.width, t.height)
}

// cr resets the x coordinate of the terminal cursor to 0.
func (t *Vt) cr() {
	t.curX = 0
}

// lf advances the y coordinate of the terminal cursor by one line scrolling
// the terminal contents if the end of the last terminal line is reached.
func (t *Vt) lf() {
	if t.curY+1 < t.height {
		t.curY++
		return
	}

	t.cons.Scroll(console.Up, 1)
	t.cons.Clear(0, t.height-1, t.width, 1)
	return
}

func makeAttr(fg, bg console.Attr) console.Attr {
	return (bg << 4) | (fg & 0xF)
}
