package vmm

// flushTLBEntry flushes a TLB entry for a particular virtual address.
func flushTLBEntry(virtAddr uintptr)

// switchPDT sets the root page table directory to point to the specified
// physical address and flushes the TLB.
func switchPDT(pdtPhysAddr uintptr)

// activePDT returns the physical address of the currently active page table.
func activePDT() uintptr
