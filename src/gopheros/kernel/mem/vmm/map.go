package vmm

import (
	"gopheros/kernel"
	"gopheros/kernel/cpu"
	"gopheros/kernel/mem"
	"gopheros/kernel/mem/pmm"
	"unsafe"
)

// ReservedZeroedFrame is a special zero-cleared frame allocated by the
// vmm package's Init function. The purpose of this frame is to assist
// in implementing on-demand memory allocation when mapping it in
// conjunction with the CopyOnWrite flag. Here is an example of how it
// can be used:
//
//  func ReserveOnDemand(start vmm.Page, pageCount int) *kernel.Error {
//    var err *kernel.Error
//    mapFlags := vmm.FlagPresent|vmm.FlagCopyOnWrite
//    for page := start; pageCount > 0; pageCount, page = pageCount-1, page+1 {
//       if err = vmm.Map(page, vmm.ReservedZeroedFrame, mapFlags); err != nil {
//         return err
//       }
//    }
//    return nil
//  }
//
// In the above example, page mappings are set up for the requested number of
// pages but no physical memory is reserved for their contents. A write to any
// of the above pages will trigger a page-fault causing a new frame to be
// allocated, cleared (the blank frame is copied to the new frame) and
// installed in-place with RW permissions.
var ReservedZeroedFrame pmm.Frame

var (
	// protectReservedZeroedPage is set to true to prevent mapping to
	protectReservedZeroedPage bool

	// nextAddrFn is used by used by tests to override the nextTableAddr
	// calculations used by Map. When compiling the kernel this function
	// will be automatically inlined.
	nextAddrFn = func(entryAddr uintptr) uintptr {
		return entryAddr
	}

	// flushTLBEntryFn is used by tests to override calls to flushTLBEntry
	// which will cause a fault if called in user-mode.
	flushTLBEntryFn = cpu.FlushTLBEntry

	earlyReserveRegionFn = EarlyReserveRegion

	errNoHugePageSupport           = &kernel.Error{Module: "vmm", Message: "huge pages are not supported"}
	errAttemptToRWMapReservedFrame = &kernel.Error{Module: "vmm", Message: "reserved blank frame cannot be mapped with a RW flag"}
)

// Map establishes a mapping between a virtual page and a physical memory frame
// using the currently active page directory table. Calls to Map will use the
// supplied physical frame allocator to initialize missing page tables at each
// paging level supported by the MMU.
//
// Attempts to map ReservedZeroedFrame with a RW flag will result in an error.
func Map(page Page, frame pmm.Frame, flags PageTableEntryFlag) *kernel.Error {
	if protectReservedZeroedPage && frame == ReservedZeroedFrame && (flags&FlagRW) != 0 {
		return errAttemptToRWMapReservedFrame
	}

	var err *kernel.Error

	walk(page.Address(), func(pteLevel uint8, pte *pageTableEntry) bool {
		// If we reached the last level all we need to do is to map the
		// frame in place and flag it as present and flush its TLB entry
		if pteLevel == pageLevels-1 {
			*pte = 0
			pte.SetFrame(frame)
			pte.SetFlags(flags)
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
			newTableFrame, err = frameAllocator()
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

// MapRegion establishes a mapping to the physical memory region which starts
// at the given frame and ends at frame + pages(size). The size argument is
// always rounded up to the nearest page boundary. MapRegion reserves the next
// available region in the active virtual address space, establishes the
// mapping and returns back the Page that corresponds to the region start.
func MapRegion(frame pmm.Frame, size mem.Size, flags PageTableEntryFlag) (Page, *kernel.Error) {
	// Reserve next free block in the address space
	size = (size + (mem.PageSize - 1)) & ^(mem.PageSize - 1)
	startPage, err := earlyReserveRegionFn(size)
	if err != nil {
		return 0, err
	}

	pageCount := size >> mem.PageShift
	for page := PageFromAddress(startPage); pageCount > 0; pageCount, page, frame = pageCount-1, page+1, frame+1 {
		if err := mapFn(page, frame, flags); err != nil {
			return 0, err
		}
	}

	return PageFromAddress(startPage), nil
}

// MapTemporary establishes a temporary RW mapping of a physical memory frame
// to a fixed virtual address overwriting any previous mapping. The temporary
// mapping mechanism is primarily used by the kernel to access and initialize
// inactive page tables.
//
// Attempts to map ReservedZeroedFrame will result in an error.
func MapTemporary(frame pmm.Frame) (Page, *kernel.Error) {
	if protectReservedZeroedPage && frame == ReservedZeroedFrame {
		return 0, errAttemptToRWMapReservedFrame
	}

	if err := Map(PageFromAddress(tempMappingAddr), frame, FlagPresent|FlagRW); err != nil {
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
