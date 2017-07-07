package tty

import (
	"gopheros/device/video/console"
	"io"
)

const (
	// DefaultScrollback defines the terminal scrollback in lines.
	DefaultScrollback = 80

	// DefaultTabWidth defines the number of spaces that tabs expand to.
	DefaultTabWidth = 4
)

// State defines the supported terminal state values.
type State uint8

const (
	// StateInactive marks the terminal as inactive. Any writes will be
	// buffered and not synced to the attached console.
	StateInactive State = iota

	// StateActive marks the terminal as active. Any writes will be
	// buffered and also synced to the attached console.
	StateActive
)

// Device is implemented by objects that can be used as a terminal device.
type Device interface {
	io.Writer
	io.ByteWriter

	// AttachTo connects a TTY to a console instance.
	AttachTo(console.Device)

	// State returns the TTY's state.
	State() State

	// SetState updates the TTY's state.
	SetState(State)

	// CursorPosition returns the current cursor x,y coordinates. Both
	// coordinates are 1-based (top-left corner has coordinates 1,1).
	CursorPosition() (uint16, uint16)

	// SetCursorPosition sets the current cursor position to (x,y). Both
	// coordinates are 1-based (top-left corner has coordinates 1,1).
	// Implementations are expected to clip the cursor position to their
	// viewport.
	SetCursorPosition(x, y uint16)
}
