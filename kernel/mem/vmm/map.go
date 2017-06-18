package vmm

import (
	"unsafe"

	"github.com/achilleasa/gopher-os/kernel"
	"github.com/achilleasa/gopher-os/kernel/mem"
	"github.com/achilleasa/gopher-os/kernel/mem/pmm"
)

var (
	// nextAddrFn is used by used by tests to override the nextTableAddr
	// calculations used by Map. When compiling the kernel this function
	// will be automatically inlined.
	nextAddrFn = func(entryAddr uintptr) uintptr {
		return entryAddr
	}

	// flushTLBEntryFn is used by tests to override calls to flushTLBEntry
	// which will cause a fault if called in user-mode.
	flushTLBEntryFn = flushTLBEntry

	errNoHugePageSupport = &kernel.Error{Module: "vmm", Message: "huge pages are not supported"}
)

// FrameAllocatorFn is a function that can allocate physical frames.
type FrameAllocatorFn func() (pmm.Frame, *kernel.Error)

// Map establishes a mapping between a virtual page and a physical memory frame
// using the currently active page directory table. Calls to Map will use the
// supplied physical frame allocator to initialize missing page tables at each
// paging level supported by the MMU.
func Map(page Page, frame pmm.Frame, flags PageTableEntryFlag, allocFn FrameAllocatorFn) *kernel.Error {
	var err *kernel.Error

	walk(page.Address(), func(pteLevel uint8, pte *pageTableEntry) bool {
		// If we reached the last level all we need to do is to map the
		// frame in place and flag it as present and flush its TLB entry
		if pteLevel == pageLevels-1 {
			*pte = 0
			pte.SetFrame(frame)
			pte.SetFlags(FlagPresent | flags)
			flushTLBEntryFn(page.Address())
			return true
		}

		if pte.HasFlags(FlagHugePage) {
			err = errNoHugePageSupport
			return false
		}

		// Next table does not yet exist; we need to allocate a
		// physical frame for it map it and clear its contents.
		if !pte.HasFlags(FlagPresent) {
			var newTableFrame pmm.Frame
			newTableFrame, err = allocFn()
			if err != nil {
				return false
			}

			*pte = 0
			pte.SetFrame(newTableFrame)
			pte.SetFlags(FlagPresent | FlagRW)

			// The next pte entry becomes available but we need to
			// make sure that the new page is properly cleared
			nextTableAddr := (uintptr(unsafe.Pointer(pte)) << pageLevelBits[pteLevel+1])
			mem.Memset(nextAddrFn(nextTableAddr), 0, mem.PageSize)
		}

		return true
	})

	return err
}

// MapTemporary establishes a temporary RW mapping of a physical memory frame
// to a fixed virtual address overwriting any previous mapping. The temporary
// mapping mechanism is primarily used by the kernel to access and initialize
// inactive page tables.
func MapTemporary(frame pmm.Frame, allocFn FrameAllocatorFn) (Page, *kernel.Error) {
	if err := Map(PageFromAddress(tempMappingAddr), frame, FlagRW, allocFn); err != nil {
		return 0, err
	}

	return PageFromAddress(tempMappingAddr), nil
}

// Unmap removes a mapping previously installed via a call to Map or MapTemporary.
func Unmap(page Page) *kernel.Error {
	var err *kernel.Error

	walk(page.Address(), func(pteLevel uint8, pte *pageTableEntry) bool {
		// If we reached the last level all we need to do is to set the
		// page as non-present and flush its TLB entry
		if pteLevel == pageLevels-1 {
			pte.ClearFlags(FlagPresent)
			flushTLBEntryFn(page.Address())
			return true
		}

		// Next table is not present; this is an invalid mapping
		if !pte.HasFlags(FlagPresent) {
			err = ErrInvalidMapping
			return false
		}

		if pte.HasFlags(FlagHugePage) {
			err = errNoHugePageSupport
			return false
		}

		return true
	})

	return err
}
