package vmm

import (
	"github.com/achilleasa/gopher-os/kernel"
	"github.com/achilleasa/gopher-os/kernel/cpu"
	"github.com/achilleasa/gopher-os/kernel/irq"
	"github.com/achilleasa/gopher-os/kernel/kfmt/early"
	"github.com/achilleasa/gopher-os/kernel/mem/pmm"
)

var (
	// frameAllocator points to a frame allocator function registered using
	// SetFrameAllocator.
	frameAllocator FrameAllocatorFn

	// the following functions are mocked by tests and are automatically
	// inlined by the compiler.
	panicFn                   = kernel.Panic
	handleExceptionWithCodeFn = irq.HandleExceptionWithCode
	readCR2Fn                 = cpu.ReadCR2
)

// FrameAllocatorFn is a function that can allocate physical frames.
type FrameAllocatorFn func() (pmm.Frame, *kernel.Error)

// SetFrameAllocator registers a frame allocator function that will be used by
// the vmm code when new physical frames need to be allocated.
func SetFrameAllocator(allocFn FrameAllocatorFn) {
	frameAllocator = allocFn
}

func pageFaultHandler(errorCode uint64, frame *irq.Frame, regs *irq.Regs) {
	early.Printf("\nPage fault while accessing address: 0x%16x\nReason: ", readCR2Fn())
	switch {
	case errorCode == 0:
		early.Printf("read from non-present page")
	case errorCode == 1:
		early.Printf("page protection violation (read)")
	case errorCode == 2:
		early.Printf("write to non-present page")
	case errorCode == 3:
		early.Printf("page protection violation (write)")
	case errorCode == 4:
		early.Printf("page-fault in user-mode")
	case errorCode == 8:
		early.Printf("page table has reserved bit set")
	case errorCode == 16:
		early.Printf("instruction fetch")
	default:
		early.Printf("unknown")
	}

	early.Printf("\n\nRegisters:\n")
	regs.Print()
	frame.Print()

	// TODO: Revisit this when user-mode tasks are implemented
	panicFn(nil)
}

func generalProtectionFaultHandler(_ uint64, frame *irq.Frame, regs *irq.Regs) {
	early.Printf("\nGeneral protection fault while accessing address: 0x%x\n", readCR2Fn())
	early.Printf("Registers:\n")
	regs.Print()
	frame.Print()

	// TODO: Revisit this when user-mode tasks are implemented
	panicFn(nil)
}

// Init initializes the vmm system and installs paging-related exception
// handlers.
func Init() *kernel.Error {
	handleExceptionWithCodeFn(irq.PageFaultException, pageFaultHandler)
	handleExceptionWithCodeFn(irq.GPFException, generalProtectionFaultHandler)

	return nil
}
