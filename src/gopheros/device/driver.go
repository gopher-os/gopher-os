package device

import (
	"gopheros/kernel"
	"io"
)

// Driver is an interface implemented by all drivers.
type Driver interface {
	// DriverName returns the name of the driver.
	DriverName() string

	// DriverVersion returns the driver version.
	DriverVersion() (major uint16, minor uint16, patch uint16)

	// DriverInit initializes the device driver. If the driver init code
	// needs to log some output, it can use the supplied io.Writer in
	// conjunction with a call to kfmt.Fprint.
	DriverInit(io.Writer) *kernel.Error
}

// ProbeFn is a function that scans for the presence of a particular
// piece of hardware and returns a driver for it.
type ProbeFn func() Driver
