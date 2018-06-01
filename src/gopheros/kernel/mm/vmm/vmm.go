package vmm

import (
	"gopheros/kernel"
	"gopheros/kernel/cpu"
	"gopheros/kernel/mm"
)

var (
	// the following functions are mocked by tests and are automatically
	// inlined by the compiler.
	readCR2Fn   = cpu.ReadCR2
	translateFn = Translate

	errUnrecoverableFault = &kernel.Error{Module: "vmm", Message: "page/gpf fault"}
)

// Init initializes the vmm system, creates a granular PDT for the kernel and
// installs paging-related exception handlers.
func Init(kernelPageOffset uintptr) *kernel.Error {
	if err := setupPDTForKernel(kernelPageOffset); err != nil {
		return err
	}

	// Install arch-specific handlers for vmm-related faults.
	installFaultHandlers()

	return reserveZeroedFrame()
}

// reserveZeroedFrame reserves a physical frame to be used together with
// FlagCopyOnWrite for lazy allocation requests.
func reserveZeroedFrame() *kernel.Error {
	var (
		err      *kernel.Error
		tempPage mm.Page
	)

	if ReservedZeroedFrame, err = mm.AllocFrame(); err != nil {
		return err
	} else if tempPage, err = mapTemporaryFn(ReservedZeroedFrame); err != nil {
		return err
	}
	kernel.Memset(tempPage.Address(), 0, mm.PageSize)
	_ = unmapFn(tempPage)

	// From this point on, ReservedZeroedFrame cannot be mapped with a RW flag
	protectReservedZeroedPage = true
	return nil
}
