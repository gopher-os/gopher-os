package device

import "gopheros/kernel"

// Driver is an interface implemented by all drivers.
type Driver interface {
	// DriverName returns the name of the driver.
	DriverName() string

	// DriverVersion returns the driver version.
	DriverVersion() (major uint16, minor uint16, patch uint16)

	// DriverInit initializes the device driver.
	DriverInit() *kernel.Error
}
