package tty

import "gopheros/device"

// HWProbes returns a slice of device.ProbeFn that can be used by the hal
// package to probe for TTY device hardware.
func HWProbes() []device.ProbeFn {
	return []device.ProbeFn{
		probeForVT,
	}
}
