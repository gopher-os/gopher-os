package vmm

import (
	"unsafe"

	"github.com/achilleasa/gopher-os/kernel"
	"github.com/achilleasa/gopher-os/kernel/mem"
	"github.com/achilleasa/gopher-os/kernel/mem/pmm"
)

var (
	// activePDTFn is used by tests to override calls to activePDT which
	// will cause a fault if called in user-mode.
	activePDTFn = activePDT

	// switchPDTFn is used by tests to override calls to switchPDT which
	// will cause a fault if called in user-mode.
	switchPDTFn = switchPDT

	// mapFn is used by tests and is automatically inlined by the compiler.
	mapFn = Map

	// mapTemporaryFn is used by tests and is automatically inlined by the compiler.
	mapTemporaryFn = MapTemporary

	// unmapmFn is used by tests and is automatically inlined by the compiler.
	unmapFn = Unmap
)

// PageDirectoryTable describes the top-most table in a multi-level paging scheme.
type PageDirectoryTable struct {
	pdtFrame pmm.Frame
}

// Init sets up the page table directory starting at the supplied physical
// address. If the supplied frame does not match the currently active PDT, then
// Init assumes that this is a new page table directory that needs
// bootstapping. In such a case, a temporary mapping is established so that
// Init can:
//  - call mem.Memset to clear the frame contents
//  - setup a recursive mapping for the last table entry to the page itself.
func (pdt *PageDirectoryTable) Init(pdtFrame pmm.Frame, allocFn FrameAllocatorFn) *kernel.Error {
	pdt.pdtFrame = pdtFrame

	// Check active PDT physical address. If it matches the input pdt then
	// nothing more needs to be done
	activePdtAddr := activePDTFn()
	if pdtFrame.Address() == activePdtAddr {
		return nil
	}

	// Create a temporary mapping for the pdt frame so we can work on it
	pdtPage, err := mapTemporaryFn(pdtFrame, allocFn)
	if err != nil {
		return err
	}

	// Clear the page contents and setup recursive mapping for the last PDT entry
	mem.Memset(pdtPage.Address(), 0, mem.PageSize)
	lastPdtEntry := (*pageTableEntry)(unsafe.Pointer(pdtPage.Address() + (((1 << pageLevelBits[0]) - 1) << mem.PointerShift)))
	*lastPdtEntry = 0
	lastPdtEntry.SetFlags(FlagPresent | FlagRW)
	lastPdtEntry.SetFrame(pdtFrame)

	// Remove temporary mapping
	unmapFn(pdtPage)

	return nil
}

// Map establishes a mapping between a virtual page and a physical memory frame
// using this PDT. This method behaves in a similar fashion to the global Map()
// function with the difference that it also supports inactive page PDTs by
// establishing a temporary mapping so that Map() can access the inactive PDT
// entries.
func (pdt PageDirectoryTable) Map(page Page, frame pmm.Frame, flags PageTableEntryFlag, allocFn FrameAllocatorFn) *kernel.Error {
	var (
		activePdtFrame   = pmm.Frame(activePDTFn() >> mem.PageShift)
		lastPdtEntryAddr uintptr
		lastPdtEntry     *pageTableEntry
	)
	// If this table is not active we need to temporarily map it to the
	// last entry in the active PDT so we can access it using the recursive
	// virtual address scheme.
	if activePdtFrame != pdt.pdtFrame {
		lastPdtEntryAddr = activePdtFrame.Address() + (((1 << pageLevelBits[0]) - 1) << mem.PointerShift)
		lastPdtEntry = (*pageTableEntry)(unsafe.Pointer(lastPdtEntryAddr))
		lastPdtEntry.SetFrame(pdt.pdtFrame)
		flushTLBEntryFn(lastPdtEntryAddr)
	}

	err := mapFn(page, frame, flags, allocFn)

	if activePdtFrame != pdt.pdtFrame {
		lastPdtEntry.SetFrame(activePdtFrame)
		flushTLBEntryFn(lastPdtEntryAddr)
	}

	return err
}

// Unmap removes a mapping previousle installed by a call to Map() on this PDT.
// This method behaves in a similar fashion to the global Unmap() function with
// the difference that it also supports inactive page PDTs by establishing a
// temporary mapping so that Unmap() can access the inactive PDT entries.
func (pdt PageDirectoryTable) Unmap(page Page) *kernel.Error {
	var (
		activePdtFrame   = pmm.Frame(activePDTFn() >> mem.PageShift)
		lastPdtEntryAddr uintptr
		lastPdtEntry     *pageTableEntry
	)
	// If this table is not active we need to temporarily map it to the
	// last entry in the active PDT so we can access it using the recursive
	// virtual address scheme.
	if activePdtFrame != pdt.pdtFrame {
		lastPdtEntryAddr = activePdtFrame.Address() + (((1 << pageLevelBits[0]) - 1) << mem.PointerShift)
		lastPdtEntry = (*pageTableEntry)(unsafe.Pointer(lastPdtEntryAddr))
		lastPdtEntry.SetFrame(pdt.pdtFrame)
		flushTLBEntryFn(lastPdtEntryAddr)
	}

	err := unmapFn(page)

	if activePdtFrame != pdt.pdtFrame {
		lastPdtEntry.SetFrame(activePdtFrame)
		flushTLBEntryFn(lastPdtEntryAddr)
	}

	return err
}

// Activate enables this page directory table and flushes the TLB
func (pdt PageDirectoryTable) Activate() {
	switchPDTFn(pdt.pdtFrame.Address())
}
