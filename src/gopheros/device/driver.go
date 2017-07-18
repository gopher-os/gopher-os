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

// DetectOrder specifies when each driver's probe function will be invoked
// by the hal package.
type DetectOrder int8

const (
	// DetectOrderEarly specifies that the driver's probe function should
	// be executed at the beginning of the HW detection phase. It is used
	// by some of the console and TTY device drivers.
	DetectOrderEarly DetectOrder = -128

	// DetectOrderBeforeACPI specifies that the driver's probe function
	// should be executed before attempting any ACPI-based HW detection but
	// after any drivers with DetectOrderEarly.
	DetectOrderBeforeACPI = -127

	// DetectOrderACPI specifies that the driver's probe function should
	// be executed after parsing the ACPI tables. This is the default (zero
	// value) for all drivers.
	DetectOrderACPI = 0

	// DetectOrderLast specifies that the driver's probe function should
	// be executed at the end of the HW detection phase.
	DetectOrderLast = 127
)

// DriverInfo is a driver-defined struct that is passed to calls to RegisterDriver.
type DriverInfo struct {
	// Order specifies at which stage of the HW detection step should
	// the probe function be invoked.
	Order DetectOrder

	// Probe is a function that checks for the presence of a particular
	// piece of hardware and returns back a driver for it.
	Probe ProbeFn
}

// DriverInfoList is a list of registered drivers that implements sort.Sort.
type DriverInfoList []*DriverInfo

// Len returns the length of the driver info list.
func (l DriverInfoList) Len() int { return len(l) }

// Swap exchanges 2 elements in the driver info list.
func (l DriverInfoList) Swap(i, j int) { l[i], l[j] = l[j], l[i] }

// Less compares 2 elements of the driver info list.
func (l DriverInfoList) Less(i, j int) bool { return l[i].Order < l[j].Order }

var (
	// registeredDrivers tracks the drivers registered via a call to
	// RegisterDriver.
	registeredDrivers DriverInfoList
)

// RegisterDriver adds the supplied driver info to the list of registered
// drivers.  The list can be retrieved by a call to DriverList().
func RegisterDriver(info *DriverInfo) {
	registeredDrivers = append(registeredDrivers, info)
}

// DriverList returns the list of registered drivers.
func DriverList() DriverInfoList {
	return registeredDrivers
}
