package tty

import "io"

// Tty is implemented by objects that can register themselves as ttys.
type Tty interface {
	io.Writer
	io.ByteWriter

	// Position returns the current cursor position (x, y).
	Position() (uint16, uint16)

	// SetPosition sets the current cursor position to (x,y). Console implementations
	// must clip the provided cursor position if it exceeds the console dimensions.
	SetPosition(x, y uint16)

	// Clear clears the terminal.
	Clear()
}
