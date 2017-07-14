package tty

import (
	"gopheros/device"
	"gopheros/device/video/console"
	"gopheros/kernel"
	"io"
)

// VT implements a terminal supporting scrollback. The terminal interprets the
// following special characters:
//  - \r (carriage-return)
//  - \n (line-feed)
//  - \b (backspace)
//  - \t (tab; expanded to tabWidth spaces)
type VT struct {
	cons console.Device

	// Terminal dimensions
	termWidth      uint32
	termHeight     uint32
	viewportWidth  uint32
	viewportHeight uint32

	// The number of additional lines of output that are buffered by the
	// terminal to support scrolling up.
	scrollback uint32

	// The terminal contents. Each character occupies 3 bytes and uses the
	// format: (ASCII char, fg, bg)
	data []uint8

	// Terminal state.
	tabWidth         uint8
	defaultFg, curFg uint8
	defaultBg, curBg uint8
	cursorX          uint32
	cursorY          uint32
	viewportY        uint32
	dataOffset       uint
	state            State
}

// NewVT creates a new virtual terminal device. The tabWidth parameter controls
// tab expansion whereas the scrollback parameter defines the line count that
// gets buffered by the terminal to provide scrolling beyond the console
// height.
func NewVT(tabWidth uint8, scrollback uint32) *VT {
	return &VT{
		tabWidth:   tabWidth,
		scrollback: scrollback,
		cursorX:    1,
		cursorY:    1,
	}
}

// AttachTo connects a TTY to a console instance.
func (t *VT) AttachTo(cons console.Device) {
	if cons == nil {
		return
	}

	t.cons = cons
	t.viewportWidth, t.viewportHeight = cons.Dimensions(console.Characters)
	t.viewportY = 0
	t.defaultFg, t.defaultBg = cons.DefaultColors()
	t.curFg, t.curBg = t.defaultFg, t.defaultBg
	t.termWidth, t.termHeight = t.viewportWidth, t.viewportHeight+t.scrollback
	t.cursorX, t.cursorY = 1, 1

	// Allocate space for the contents and fill it with empty characters
	// using the default fg/bg colors for the attached console.
	t.data = make([]uint8, t.termWidth*t.termHeight*3)
	for i := 0; i < len(t.data); i += 3 {
		t.data[i] = ' '
		t.data[i+1] = t.defaultFg
		t.data[i+2] = t.defaultBg
	}
}

// State returns the TTY's state.
func (t *VT) State() State {
	return t.state
}

// SetState updates the TTY's state.
func (t *VT) SetState(newState State) {
	if t.state == newState {
		return
	}

	t.state = newState

	// If the terminal became active, update the console with its contents
	if t.state == StateActive && t.cons != nil {
		for y := uint32(1); y <= t.viewportHeight; y++ {
			offset := (y - 1 + t.viewportY) * (t.viewportWidth * 3)
			for x := uint32(1); x <= t.viewportWidth; x, offset = x+1, offset+3 {
				t.cons.Write(t.data[offset], t.data[offset+1], t.data[offset+2], x, y)
			}
		}
	}
}

// CursorPosition returns the current cursor position.
func (t *VT) CursorPosition() (uint32, uint32) {
	return t.cursorX, t.cursorY
}

// SetCursorPosition sets the current cursor position to (x,y).
func (t *VT) SetCursorPosition(x, y uint32) {
	if t.cons == nil {
		return
	}

	if x < 1 {
		x = 1
	} else if x > t.viewportWidth {
		x = t.viewportWidth
	}

	if y < 1 {
		y = 1
	} else if y > t.viewportHeight {
		y = t.viewportHeight
	}

	t.cursorX, t.cursorY = x, y
	t.updateDataOffset()
}

// Write implements io.Writer.
func (t *VT) Write(data []byte) (int, error) {
	for count, b := range data {
		err := t.WriteByte(b)
		if err != nil {
			return count, err
		}
	}

	return len(data), nil
}

// WriteByte implements io.ByteWriter.
func (t *VT) WriteByte(b byte) error {
	if t.cons == nil {
		return io.ErrClosedPipe
	}

	switch b {
	case '\r':
		t.cr()
	case '\n':
		t.lf(true)
	case '\b':
		if t.cursorX > 1 {
			t.SetCursorPosition(t.cursorX-1, t.cursorY)
			t.doWrite(' ', false)
		}
	case '\t':
		for i := uint8(0); i < t.tabWidth; i++ {
			t.doWrite(' ', true)
		}
	default:
		t.doWrite(b, true)
	}

	return nil
}

// doWrite writes the specified character together with the current fg/bg
// attributes at the current data offset advancing the cursor position if
// advanceCursor is true. If the terminal is active, then doWrite also writes
// the character to the attached console.
func (t *VT) doWrite(b byte, advanceCursor bool) {
	if t.state == StateActive {
		t.cons.Write(b, t.curFg, t.curBg, t.cursorX, t.cursorY)
	}

	t.data[t.dataOffset] = b
	t.data[t.dataOffset+1] = t.curFg
	t.data[t.dataOffset+2] = t.curBg

	if advanceCursor {
		// Advance x position and handle wrapping when the cursor reaches the
		// end of the current line
		t.dataOffset += 3
		t.cursorX++
		if t.cursorX > t.viewportWidth {
			t.lf(true)
		}
	}
}

// cr resets the x coordinate of the terminal cursor to 0.
func (t *VT) cr() {
	t.cursorX = 1
	t.updateDataOffset()
}

// lf advances the y coordinate of the terminal cursor by one line scrolling
// the terminal contents if the end of the last terminal line is reached.
func (t *VT) lf(withCR bool) {
	if withCR {
		t.cursorX = 1
	}

	switch {
	// Cursor has not reached the end of the viewport
	case t.cursorY+1 <= t.viewportHeight:
		t.cursorY++
	default:
		// Check if the viewport can be scrolled down
		if t.viewportY+t.viewportHeight < t.termHeight {
			t.viewportY++
		} else {
			// We have reached the bottom of the terminal buffer.
			// We need to scroll its contents up and clear the last line
			var stride = int(t.viewportWidth * 3)
			var startOffset = int(t.viewportY) * stride
			var endOffset = int(t.viewportY+t.viewportHeight-1) * stride

			for offset := startOffset; offset < endOffset; offset++ {
				t.data[offset] = t.data[offset+stride]
			}

			for offset := endOffset; offset < endOffset+stride; offset += 3 {
				t.data[offset+0] = ' '
				t.data[offset+1] = t.defaultFg
				t.data[offset+2] = t.defaultBg
			}
		}

		// Sync console
		if t.state == StateActive {
			t.cons.Scroll(console.ScrollDirUp, 1)
			t.cons.Fill(1, t.cursorY, t.termWidth, 1, t.defaultFg, t.defaultBg)
		}
	}

	t.updateDataOffset()
}

// updateDataOffset calculates the offset in the data buffer taking into account
// the cursor position and the viewportY value.
func (t *VT) updateDataOffset() {
	t.dataOffset = uint((t.viewportY+(t.cursorY-1))*(t.viewportWidth*3) + ((t.cursorX - 1) * 3))
}

// DriverName returns the name of this driver.
func (t *VT) DriverName() string {
	return "vt"
}

// DriverVersion returns the version of this driver.
func (t *VT) DriverVersion() (uint16, uint16, uint16) {
	return 0, 0, 1
}

// DriverInit initializes this driver.
func (t *VT) DriverInit(_ io.Writer) *kernel.Error { return nil }

func probeForVT() device.Driver {
	return NewVT(DefaultTabWidth, DefaultScrollback)
}

func init() {
	ProbeFuncs = append(ProbeFuncs, probeForVT)
}
