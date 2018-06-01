package vmm

import (
	"gopheros/kernel"
	"gopheros/kernel/gate"
	"gopheros/kernel/kfmt"
	"gopheros/kernel/mm"
)

var (
	// handleInterruptFn is used by tests.
	handleInterruptFn = gate.HandleInterrupt
)

func installFaultHandlers() {
	handleInterruptFn(gate.PageFaultException, 0, pageFaultHandler)
	handleInterruptFn(gate.GPFException, 0, generalProtectionFaultHandler)
}

// pageFaultHandler is invoked when a PDT or PDT-entry is not present or when a
// RW protection check fails.
func pageFaultHandler(regs *gate.Registers) {
	var (
		faultAddress = uintptr(readCR2Fn())
		faultPage    = mm.PageFromAddress(faultAddress)
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
			copy    mm.Frame
			tmpPage mm.Page
			err     *kernel.Error
		)

		if copy, err = mm.AllocFrame(); err != nil {
			nonRecoverablePageFault(faultAddress, regs, err)
		} else if tmpPage, err = mapTemporaryFn(copy); err != nil {
			nonRecoverablePageFault(faultAddress, regs, err)
		} else {
			// Copy page contents, mark as RW and remove CoW flag
			kernel.Memcopy(faultPage.Address(), tmpPage.Address(), mm.PageSize)
			_ = unmapFn(tmpPage)

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

	nonRecoverablePageFault(faultAddress, regs, errUnrecoverableFault)
}

// generalProtectionFaultHandler is invoked for various reasons:
// - segment errors (privilege, type or limit violations)
// - executing privileged instructions outside ring-0
// - attempts to access reserved or unimplemented CPU registers
func generalProtectionFaultHandler(regs *gate.Registers) {
	kfmt.Printf("\nGeneral protection fault while accessing address: 0x%x\n", readCR2Fn())
	kfmt.Printf("Registers:\n")
	regs.DumpTo(kfmt.GetOutputSink())

	// TODO: Revisit this when user-mode tasks are implemented
	panic(errUnrecoverableFault)
}

func nonRecoverablePageFault(faultAddress uintptr, regs *gate.Registers, err *kernel.Error) {
	kfmt.Printf("\nPage fault while accessing address: 0x%16x\nReason: ", faultAddress)
	switch {
	case regs.Info == 0:
		kfmt.Printf("read from non-present page")
	case regs.Info == 1:
		kfmt.Printf("page protection violation (read)")
	case regs.Info == 2:
		kfmt.Printf("write to non-present page")
	case regs.Info == 3:
		kfmt.Printf("page protection violation (write)")
	case regs.Info == 4:
		kfmt.Printf("page-fault in user-mode")
	case regs.Info == 8:
		kfmt.Printf("page table has reserved bit set")
	case regs.Info == 16:
		kfmt.Printf("instruction fetch")
	default:
		kfmt.Printf("unknown")
	}

	kfmt.Printf("\n\nRegisters:\n")
	regs.DumpTo(kfmt.GetOutputSink())

	// TODO: Revisit this when user-mode tasks are implemented
	panic(err)
}
