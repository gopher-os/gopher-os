package vmm

import (
	"gopheros/kernel"
	"gopheros/kernel/cpu"
	"gopheros/kernel/mm"
	"gopheros/multiboot"
	"unsafe"
)

var (
	// activePDTFn is used by tests to override calls to activePDT which
	// will cause a fault if called in user-mode.
	activePDTFn = cpu.ActivePDT

	// switchPDTFn is used by tests to override calls to switchPDT which
	// will cause a fault if called in user-mode.
	switchPDTFn = cpu.SwitchPDT

	// mapFn is used by tests and is automatically inlined by the compiler.
	mapFn = Map

	// mapTemporaryFn is used by tests and is automatically inlined by the compiler.
	mapTemporaryFn = MapTemporary

	// unmapmFn is used by tests and is automatically inlined by the compiler.
	unmapFn = Unmap

	// visitElfSectionsFn is used by tests and is automatically inlined
	// by the compiler.
	visitElfSectionsFn = multiboot.VisitElfSections

	// The granular PDT which is set up by the setupPDTForKernel call. It's
	// entries correspond to the various kernel section address/size tuples
	// as reported by the bootloader.
	kernelPDT PageDirectoryTable
)

// PageDirectoryTable describes the top-most table in a multi-level paging scheme.
type PageDirectoryTable struct {
	pdtFrame mm.Frame
}

// Init sets up the page table directory starting at the supplied physical
// address. If the supplied frame does not match the currently active PDT, then
// Init assumes that this is a new page table directory that needs
// bootstapping. In such a case, a temporary mapping is established so that
// Init can:
//  - call kernel.Memset to clear the frame contents
//  - setup a recursive mapping for the last table entry to the page itself.
func (pdt *PageDirectoryTable) Init(pdtFrame mm.Frame) *kernel.Error {
	pdt.pdtFrame = pdtFrame

	// Check active PDT physical address. If it matches the input pdt then
	// nothing more needs to be done
	activePdtAddr := activePDTFn()
	if pdtFrame.Address() == activePdtAddr {
		return nil
	}

	// Create a temporary mapping for the pdt frame so we can work on it
	pdtPage, err := mapTemporaryFn(pdtFrame)
	if err != nil {
		return err
	}

	// Clear the page contents and setup recursive mapping for the last PDT entry
	kernel.Memset(pdtPage.Address(), 0, mm.PageSize)
	lastPdtEntry := (*pageTableEntry)(unsafe.Pointer(pdtPage.Address() + (((1 << pageLevelBits[0]) - 1) << mm.PointerShift)))
	*lastPdtEntry = 0
	lastPdtEntry.SetFlags(FlagPresent | FlagRW)
	lastPdtEntry.SetFrame(pdtFrame)

	// Remove temporary mapping
	_ = unmapFn(pdtPage)

	return nil
}

// Map establishes a mapping between a virtual page and a physical memory frame
// using this PDT. This method behaves in a similar fashion to the global Map()
// function with the difference that it also supports inactive page PDTs by
// establishing a temporary mapping so that Map() can access the inactive PDT
// entries.
func (pdt PageDirectoryTable) Map(page mm.Page, frame mm.Frame, flags PageTableEntryFlag) *kernel.Error {
	var (
		activePdtFrame   = mm.Frame(activePDTFn() >> mm.PageShift)
		lastPdtEntryAddr uintptr
		lastPdtEntry     *pageTableEntry
	)
	// If this table is not active we need to temporarily map it to the
	// last entry in the active PDT so we can access it using the recursive
	// virtual address scheme.
	if activePdtFrame != pdt.pdtFrame {
		lastPdtEntryAddr = activePdtFrame.Address() + (((1 << pageLevelBits[0]) - 1) << mm.PointerShift)
		lastPdtEntry = (*pageTableEntry)(unsafe.Pointer(lastPdtEntryAddr))
		lastPdtEntry.SetFrame(pdt.pdtFrame)
		flushTLBEntryFn(lastPdtEntryAddr)
	}

	err := mapFn(page, frame, flags)

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
func (pdt PageDirectoryTable) Unmap(page mm.Page) *kernel.Error {
	var (
		activePdtFrame   = mm.Frame(activePDTFn() >> mm.PageShift)
		lastPdtEntryAddr uintptr
		lastPdtEntry     *pageTableEntry
	)
	// If this table is not active we need to temporarily map it to the
	// last entry in the active PDT so we can access it using the recursive
	// virtual address scheme.
	if activePdtFrame != pdt.pdtFrame {
		lastPdtEntryAddr = activePdtFrame.Address() + (((1 << pageLevelBits[0]) - 1) << mm.PointerShift)
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

// setupPDTForKernel queries the multiboot package for the ELF sections that
// correspond to the loaded kernel image and establishes a new granular PDT for
// the kernel's VMA using the appropriate flags (e.g. NX for data sections, RW
// for writable sections e.t.c).
func setupPDTForKernel(kernelPageOffset uintptr) *kernel.Error {
	// Allocate frame for the page directory and initialize it
	kernelPDTFrame, err := mm.AllocFrame()
	if err != nil {
		return err
	}

	if err = kernelPDT.Init(kernelPDTFrame); err != nil {
		return err
	}

	// Query the ELF sections of the kernel image and establish mappings
	// for each one using the appropriate flags
	var visitor = func(_ string, secFlags multiboot.ElfSectionFlag, secAddress uintptr, secSize uint64) {
		// Bail out if we have encountered an error; also ignore sections
		// not using the kernel's VMA
		if err != nil || secAddress < kernelPageOffset {
			return
		}

		flags := FlagPresent

		if (secFlags & multiboot.ElfSectionExecutable) == 0 {
			flags |= FlagNoExecute
		}

		if (secFlags & multiboot.ElfSectionWritable) != 0 {
			flags |= FlagRW
		}

		// Map the start and end VMA addresses for the section contents
		// into a start and end (inclusive) page number. To figure out
		// the physical start frame we just need to subtract the
		// kernel's VMA offset from the virtual address and round that
		// down to the nearest frame number.
		curPage := mm.PageFromAddress(secAddress)
		lastPage := mm.PageFromAddress(secAddress + uintptr(secSize-1))
		curFrame := mm.Frame((secAddress - kernelPageOffset) >> mm.PageShift)
		for ; curPage <= lastPage; curFrame, curPage = curFrame+1, curPage+1 {
			if err = kernelPDT.Map(curPage, curFrame, flags); err != nil {
				return
			}
		}
	}

	// Use the noescape hack to prevent the compiler from leaking the visitor
	// function literal to the heap.
	visitElfSectionsFn(
		*(*multiboot.ElfSectionVisitor)(noEscape(unsafe.Pointer(&visitor))),
	)

	// If an error occurred while maping the ELF sections bail out
	if err != nil {
		return err
	}

	// Ensure that any pages mapped by the mmory allocator using
	// EarlyReserveRegion are copied to the new page directory.
	for rsvAddr := earlyReserveLastUsed; rsvAddr < tempMappingAddr; rsvAddr += mm.PageSize {
		page := mm.PageFromAddress(rsvAddr)

		frameAddr, err := translateFn(rsvAddr)
		if err != nil {
			return err
		}

		if err = kernelPDT.Map(page, mm.Frame(frameAddr>>mm.PageShift), FlagPresent|FlagRW); err != nil {
			return err
		}
	}

	// Activate the new PDT. After this point, the identify mapping for the
	// physical mmory addresses where the kernel is loaded becomes invalid.
	kernelPDT.Activate()

	return nil
}

// noEscape hides a pointer from escape analysis. This function is copied over
// from runtime/stubs.go
//go:nosplit
func noEscape(p unsafe.Pointer) unsafe.Pointer {
	x := uintptr(p)
	return unsafe.Pointer(x ^ 0)
}

var (
	// ErrInvalidMapping is returned when trying to lookup a virtual memory address that is not yet mapped.
	ErrInvalidMapping = &kernel.Error{Module: "vmm", Message: "virtual address does not point to a mapped physical page"}
)

// PageTableEntryFlag describes a flag that can be applied to a page table entry.
type PageTableEntryFlag uintptr

// pageTableEntry describes a page table entry. These entries encode
// a physical frame address and a set of flags. The actual format
// of the entry and flags is architecture-dependent.
type pageTableEntry uintptr

// HasFlags returns true if this entry has all the input flags set.
func (pte pageTableEntry) HasFlags(flags PageTableEntryFlag) bool {
	return (uintptr(pte) & uintptr(flags)) == uintptr(flags)
}

// HasAnyFlag returns true if this entry has at least one of the input flags set.
func (pte pageTableEntry) HasAnyFlag(flags PageTableEntryFlag) bool {
	return (uintptr(pte) & uintptr(flags)) != 0
}

// SetFlags sets the input list of flags to the page table entry.
func (pte *pageTableEntry) SetFlags(flags PageTableEntryFlag) {
	*pte = (pageTableEntry)(uintptr(*pte) | uintptr(flags))
}

// ClearFlags unsets the input list of flags from the page table entry.
func (pte *pageTableEntry) ClearFlags(flags PageTableEntryFlag) {
	*pte = (pageTableEntry)(uintptr(*pte) &^ uintptr(flags))
}

// Frame returns the physical page frame that this page table entry points to.
func (pte pageTableEntry) Frame() mm.Frame {
	return mm.Frame((uintptr(pte) & ptePhysPageMask) >> mm.PageShift)
}

// SetFrame updates the page table entry to point the the given physical frame .
func (pte *pageTableEntry) SetFrame(frame mm.Frame) {
	*pte = (pageTableEntry)((uintptr(*pte) &^ ptePhysPageMask) | frame.Address())
}

// pteForAddress returns the final page table entry that correspond to a
// particular virtual address. The function performs a page table walk till it
// reaches the final page table entry returning ErrInvalidMapping if the page
// is not present.
func pteForAddress(virtAddr uintptr) (*pageTableEntry, *kernel.Error) {
	var (
		err   *kernel.Error
		entry *pageTableEntry
	)

	walk(virtAddr, func(pteLevel uint8, pte *pageTableEntry) bool {
		if !pte.HasFlags(FlagPresent) {
			entry = nil
			err = ErrInvalidMapping
			return false
		}

		entry = pte
		return true
	})

	return entry, err
}

var (
	// ptePointerFn returns a pointer to the supplied entry address. It is
	// used by tests to override the generated page table entry pointers so
	// walk() can be properly tested. When compiling the kernel this function
	// will be automatically inlined.
	ptePtrFn = func(entryAddr uintptr) unsafe.Pointer {
		return unsafe.Pointer(entryAddr)
	}
)

// pageTableWalker is a function that can be passed to the walk method. The
// function receives the current page level and page table entry as its
// arguments.  If the function returns false, then the page walk is aborted.
type pageTableWalker func(pteLevel uint8, pte *pageTableEntry) bool

// walk performs a page table walk for the given virtual address. It calls the
// suppplied walkFn with the page table entry that corresponds to each page
// table level. If walkFn returns an error then the walk is aborted and the
// error is returned to the caller.
func walk(virtAddr uintptr, walkFn pageTableWalker) {
	var (
		level                            uint8
		tableAddr, entryAddr, entryIndex uintptr
		ok                               bool
	)

	// tableAddr is initially set to the recursively mapped virtual address for the
	// last entry in the top-most page table. Dereferencing a pointer to this address
	// will allow us to access
	for level, tableAddr = uint8(0), pdtVirtualAddr; level < pageLevels; level, tableAddr = level+1, entryAddr {
		// Extract the bits from virtual address that correspond to the
		// index in this level's page table
		entryIndex = (virtAddr >> pageLevelShifts[level]) & ((1 << pageLevelBits[level]) - 1)

		// By shifting the table virtual address left by pageLevelShifts[level] we add
		// a new level of indirection to our recursive mapping allowing us to access
		// the table pointed to by the page entry
		entryAddr = tableAddr + (entryIndex << mm.PointerShift)

		if ok = walkFn(level, (*pageTableEntry)(ptePtrFn(entryAddr))); !ok {
			return
		}

		// Shift left by the number of bits for this paging level to get
		// the virtual address of the table pointed to by entryAddr
		entryAddr <<= pageLevelBits[level]
	}
}
