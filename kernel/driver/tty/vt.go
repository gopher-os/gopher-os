package tty

import "github.com/achilleasa/gopher-os/kernel/driver/video/console"

const (
	defaultFg = console.LightGrey
	defaultBg = console.Black
	tabWidth  = 4
)

// Vt implements a simple terminal that can process LF and CR characters. The
// terminal uses a console device for its output.
type Vt struct {
	// Go interfaces will not work before we can get memory allocation working.
	// Till then we need to use concrete types instead.
	cons *console.Ega

	width  uint16
	height uint16

	curX    uint16
	curY    uint16
	curAttr console.Attr
}

// AttachTo links the terminal with the specified console device and updates
// the terminal's dimensions to match the ones reported by the attached device.
func (t *Vt) AttachTo(cons *console.Ega) {
	t.cons = cons
	t.width, t.height = cons.Dimensions()
	t.curX = 0
	t.curY = 0

	// Default to lightgrey on black text.
	t.curAttr = makeAttr(defaultFg, defaultBg)
}

// Clear clears the terminal.
func (t *Vt) Clear() {
	t.clear()
}

// Position returns the current cursor position (x, y).
func (t *Vt) Position() (uint16, uint16) {
	return t.curX, t.curY
}

// SetPosition sets the current cursor position to (x,y).
func (t *Vt) SetPosition(x, y uint16) {
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
	for _, b := range data {
		t.WriteByte(b)
	}

	return len(data), nil
}

// WriteByte implements io.ByteWriter.
func (t *Vt) WriteByte(b byte) error {
	switch b {
	case '\r':
		t.cr()
	case '\n':
		t.cr()
		t.lf()
	case '\b':
		if t.curX > 0 {
			t.cons.Write(' ', t.curAttr, t.curX, t.curY)
			t.curX--
		}
	case '\t':
		for i := 0; i < tabWidth; i++ {
			t.cons.Write(' ', t.curAttr, t.curX, t.curY)
			t.curX++
			if t.curX == t.width {
				t.cr()
				t.lf()
			}
		}
	default:
		t.cons.Write(b, t.curAttr, t.curX, t.curY)
		t.curX++
		if t.curX == t.width {
			t.cr()
			t.lf()
		}
	}

	return nil
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
}

func makeAttr(fg, bg console.Attr) console.Attr {
	return (bg << 4) | (fg & 0xF)
}
