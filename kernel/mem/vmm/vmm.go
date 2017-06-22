package vmm

import (
	"github.com/achilleasa/gopher-os/kernel"
	"github.com/achilleasa/gopher-os/kernel/cpu"
	"github.com/achilleasa/gopher-os/kernel/irq"
	"github.com/achilleasa/gopher-os/kernel/kfmt/early"
	"github.com/achilleasa/gopher-os/kernel/mem"
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
	var (
		faultAddress = uintptr(readCR2Fn())
		faultPage    = PageFromAddress(faultAddress)
		pageEntry    *pageTableEntry
	)

	// Lookup entry for the page where the fault occurred
	walk(faultPage.Address(), func(pteLevel uint8, pte *pageTableEntry) bool {
		nextIsPresent := pte.HasFlags(FlagPresent)

		if pteLevel == pageLevels-1 && nextIsPresent {
			pageEntry = pte
		}

		// Abort walk if the next page table entry is missing
		return nextIsPresent
	})

	// CoW is supported for RO pages with the CoW flag set
	if pageEntry != nil && !pageEntry.HasFlags(FlagRW) && pageEntry.HasFlags(FlagCopyOnWrite) {
		var (
			copy    pmm.Frame
			tmpPage Page
			err     *kernel.Error
		)

		if copy, err = frameAllocator(); err != nil {
			nonRecoverablePageFault(faultAddress, errorCode, frame, regs, err)
		} else if tmpPage, err = mapTemporaryFn(copy); err != nil {
			nonRecoverablePageFault(faultAddress, errorCode, frame, regs, err)
		} else {
			// Copy page contents, mark as RW and remove CoW flag
			mem.Memcopy(faultPage.Address(), tmpPage.Address(), mem.PageSize)
			unmapFn(tmpPage)

			// Update mapping to point to the new frame, flag it as RW and
			// remove the CoW flag
			pageEntry.ClearFlags(FlagCopyOnWrite)
			pageEntry.SetFlags(FlagPresent | FlagRW)
			pageEntry.SetFrame(copy)
			flushTLBEntryFn(faultPage.Address())

			// Fault recovered; retry the instruction that caused the fault
			return
		}
	}

	nonRecoverablePageFault(faultAddress, errorCode, frame, regs, nil)
}

func nonRecoverablePageFault(faultAddress uintptr, errorCode uint64, frame *irq.Frame, regs *irq.Regs, err *kernel.Error) {
	early.Printf("\nPage fault while accessing address: 0x%16x\nReason: ", faultAddress)
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
	panicFn(err)
}

func generalProtectionFaultHandler(_ uint64, frame *irq.Frame, regs *irq.Regs) {
	early.Printf("\nGeneral protection fault while accessing address: 0x%x\n", readCR2Fn())
	early.Printf("Registers:\n")
	regs.Print()
	frame.Print()

	// TODO: Revisit this when user-mode tasks are implemented
	panicFn(nil)
}

// reserveZeroedFrame reserves a physical frame to be used together with
// FlagCopyOnWrite for lazy allocation requests.
func reserveZeroedFrame() *kernel.Error {
	var (
		err      *kernel.Error
		tempPage Page
	)

	if ReservedZeroedFrame, err = frameAllocator(); err != nil {
		return err
	} else if tempPage, err = mapTemporaryFn(ReservedZeroedFrame); err != nil {
		return err
	}
	mem.Memset(tempPage.Address(), 0, mem.PageSize)
	unmapFn(tempPage)

	// From this point on, ReservedZeroedFrame cannot be mapped with a RW flag
	protectReservedZeroedPage = true
	return nil
}

// Init initializes the vmm system and installs paging-related exception
// handlers.
func Init() *kernel.Error {
	if err := reserveZeroedFrame(); err != nil {
		return err
	}

	handleExceptionWithCodeFn(irq.PageFaultException, pageFaultHandler)
	handleExceptionWithCodeFn(irq.GPFException, generalProtectionFaultHandler)
	return nil
}
