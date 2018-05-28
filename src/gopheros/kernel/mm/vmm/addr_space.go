package vmm

import (
	"gopheros/kernel"
	"gopheros/kernel/mm"
)

var (
	// earlyReserveLastUsed tracks the last reserved page address and is
	// decreased after each allocation request. Initially, it points to
	// tempMappingAddr which coincides with the end of the kernel address
	// space.
	earlyReserveLastUsed = tempMappingAddr

	errEarlyReserveNoSpace = &kernel.Error{Module: "early_reserve", Message: "remaining virtual address space not large enough to satisfy reservation request"}
)

// EarlyReserveRegion reserves a page-aligned contiguous virtual memory region
// with the requested size in the kernel address space and returns its virtual
// address. If size is not a multiple of mm.PageSize it will be automatically
// rounded up.
//
// This function allocates regions starting at the end of the kernel address
// space. It should only be used during the early stages of kernel initialization.
func EarlyReserveRegion(size uintptr) (uintptr, *kernel.Error) {
	size = (size + (mm.PageSize - 1)) & ^(mm.PageSize - 1)

	// reserving a region of the requested size will cause an underflow
	if size > earlyReserveLastUsed {
		return 0, errEarlyReserveNoSpace
	}

	earlyReserveLastUsed -= size
	return earlyReserveLastUsed, nil
}
