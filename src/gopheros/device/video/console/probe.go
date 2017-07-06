package console

import "gopheros/device"
import "gopheros/kernel/hal/multiboot"

var (
	getFramebufferInfoFn = multiboot.GetFramebufferInfo
)

// HWProbes returns a slice of device.ProbeFn that can be used by the hal
// package to probe for console device hardware.
func HWProbes() []device.ProbeFn {
	return []device.ProbeFn{
		probeForVgaTextConsole,
	}
}
