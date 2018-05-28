package vmm

import (
	"gopheros/kernel"
	"gopheros/kernel/irq"
	"gopheros/kernel/kfmt"
	"gopheros/kernel/mm"
)

func pageFaultHandler(errorCode uint64, frame *irq.Frame, regs *irq.Regs) {
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
			nonRecoverablePageFault(faultAddress, errorCode, frame, regs, err)
		} else if tmpPage, err = mapTemporaryFn(copy); err != nil {
			nonRecoverablePageFault(faultAddress, errorCode, frame, regs, err)
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

	nonRecoverablePageFault(faultAddress, errorCode, frame, regs, errUnrecoverableFault)
}

func nonRecoverablePageFault(faultAddress uintptr, errorCode uint64, frame *irq.Frame, regs *irq.Regs, err *kernel.Error) {
	kfmt.Printf("\nPage fault while accessing address: 0x%16x\nReason: ", faultAddress)
	switch {
	case errorCode == 0:
		kfmt.Printf("read from non-present page")
	case errorCode == 1:
		kfmt.Printf("page protection violation (read)")
	case errorCode == 2:
		kfmt.Printf("write to non-present page")
	case errorCode == 3:
		kfmt.Printf("page protection violation (write)")
	case errorCode == 4:
		kfmt.Printf("page-fault in user-mode")
	case errorCode == 8:
		kfmt.Printf("page table has reserved bit set")
	case errorCode == 16:
		kfmt.Printf("instruction fetch")
	default:
		kfmt.Printf("unknown")
	}

	kfmt.Printf("\n\nRegisters:\n")
	regs.Print()
	frame.Print()

	// TODO: Revisit this when user-mode tasks are implemented
	panic(err)
}

func generalProtectionFaultHandler(_ uint64, frame *irq.Frame, regs *irq.Regs) {
	kfmt.Printf("\nGeneral protection fault while accessing address: 0x%x\n", readCR2Fn())
	kfmt.Printf("Registers:\n")
	regs.Print()
	frame.Print()

	// TODO: Revisit this when user-mode tasks are implemented
	panic(errUnrecoverableFault)
}
