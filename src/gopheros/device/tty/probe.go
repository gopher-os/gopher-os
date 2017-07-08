package tty

import "gopheros/device"

var (
	// ProbeFuncs is a slice of device probe functions
	// that is used by the hal package to probe for TTY
	// hardware. Each driver should use an init() block
	// to append its probe function to this list.
	ProbeFuncs []device.ProbeFn
)
