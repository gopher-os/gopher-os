package pfn

import (
	"github.com/achilleasa/gopher-os/kernel/hal/multiboot"
	"github.com/achilleasa/gopher-os/kernel/kfmt/early"
	"github.com/achilleasa/gopher-os/kernel/mem"
)

var (
	// EarlyAllocator points to a static instance of the boot memory allocator
	// which is used to bootstrap the kernel before initializing a more
	// advanced memory allocator.
	EarlyAllocator BootMemAllocator
)

// BootMemAllocator implements a rudimentary physical memory allocator which is used
// to bootstrap the kernel.
//
// The allocator implementation uses the memory region information provided by
// the bootloader to detect free memory blocks and return the next available
// free frame.
//
// Allocations are tracked via an internal counter that contains the last
// allocated frame index.  The system memory regions are mapped into a linear
// page index by aligning the region start address to the system's page size
// and then dividing by the page size.
//
// Due to the way that the allocator works, it is not possible to free
// allocated pages. Once the kernel is properly initialized, the allocated
// blocks will be handed over to a more advanced memory allocator that does
// support freeing.
type BootMemAllocator struct {
	initialized bool

	// allocCount tracks the total number of allocated frames.
	allocCount uint64

	// lastAllocIndex tracks the last allocated frame index.
	lastAllocIndex int64
}

// init sets up the boot memory allocator internal state and prints out the
// system memory map.
func (alloc *BootMemAllocator) init() {
	alloc.lastAllocIndex = -1
	alloc.initialized = true

	early.Printf("[boot_mem_alloc] system memory map:\n")
	var totalFree mem.Size
	multiboot.VisitMemRegions(func(region *multiboot.MemoryMapEntry) bool {
		early.Printf("\t[0x%10x - 0x%10x], size: %10d, type: %s\n", region.PhysAddress, region.PhysAddress+region.Length, region.Length, region.Type.String())

		if region.Type == multiboot.MemAvailable {
			totalFree += mem.Size(region.Length)
		}
		return true
	})
	early.Printf("[boot_mem_alloc] free memory: %dKb\n", uint64(totalFree/mem.Kb))
}

// AllocFrame scans the system memory regions reported by the bootloader and
// reseves the next available free frame. AllocFrame returns false if no more
// memory can be allocated.
//
// The allocator only supports allocating blocks equal to the page size.
// Requests for a page order > 0 will cause the allocator to return false.
//
// The use of a bool return value is intentional; if this method returned an
// error then the compiler would call runtime.convT2I which in turn invokes the
// yet uninitialized Go allocator.
func (alloc *BootMemAllocator) AllocFrame(order mem.PageOrder) (Frame, bool) {
	if !alloc.initialized {
		alloc.init()
	}

	if order > 0 {
		return InvalidFrame, false
	}

	var (
		foundPageIndex                           int64 = -1
		regionStartPageIndex, regionEndPageIndex int64
	)
	multiboot.VisitMemRegions(func(region *multiboot.MemoryMapEntry) bool {
		if region.Type != multiboot.MemAvailable {
			return true
		}

		// Align region start address to a page boundary and find the start
		// and end page indices for the region
		regionStartPageIndex = int64(((mem.Size(region.PhysAddress) + (mem.PageSize - 1)) & ^(mem.PageSize - 1)) >> mem.PageShift)
		regionEndPageIndex = int64(((mem.Size(region.PhysAddress+region.Length) - (mem.PageSize - 1)) & ^(mem.PageSize - 1)) >> mem.PageShift)

		// Ignore already allocated regions
		if alloc.lastAllocIndex >= regionEndPageIndex {
			return true
		}

		// We found a block that can be allocated. The last allocated
		// index will be either pointing to a previous region or will
		// point inside this region. In the first case we just need to
		// select the regionStartPageIndex. In the latter case we can
		// simply select the next available page in the current region.
		if alloc.lastAllocIndex < regionStartPageIndex {
			foundPageIndex = regionStartPageIndex
		} else {
			foundPageIndex = alloc.lastAllocIndex + 1
		}
		return false
	})

	if foundPageIndex == -1 {
		return InvalidFrame, false
	}

	alloc.allocCount++
	alloc.lastAllocIndex = foundPageIndex

	return Frame(foundPageIndex), true
}
