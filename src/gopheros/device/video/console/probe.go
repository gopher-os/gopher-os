package console

import (
	"gopheros/device"
	"gopheros/kernel/cpu"
	"gopheros/kernel/hal/multiboot"
	"gopheros/kernel/mem/vmm"
)

var (
	mapRegionFn          = vmm.MapRegion
	portWriteByteFn      = cpu.PortWriteByte
	getFramebufferInfoFn = multiboot.GetFramebufferInfo

	// ProbeFuncs is a slice of device probe functions that is used by
	// the hal package to probe for console device hardware. Each driver
	// should use an init() block to append its probe function to this list.
	ProbeFuncs []device.ProbeFn
)
