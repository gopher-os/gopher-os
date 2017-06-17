package allocator

import (
	"github.com/achilleasa/gopher-os/kernel"
	"github.com/achilleasa/gopher-os/kernel/hal/multiboot"
	"github.com/achilleasa/gopher-os/kernel/kfmt/early"
	"github.com/achilleasa/gopher-os/kernel/mem"
	"github.com/achilleasa/gopher-os/kernel/mem/pmm"
)

var (
	// EarlyAllocator points to a static instance of the boot memory allocator
	// which is used to bootstrap the kernel before initializing a more
	// advanced memory allocator.
	EarlyAllocator BootMemAllocator

	errBootAllocUnsupportedPageSize = &kernel.Error{Module: "boot_mem_alloc", Message: "allocator only support allocation requests of order(0)"}
	errBootAllocOutOfMemory         = &kernel.Error{Module: "boot_mem_alloc", Message: "out of memory"}
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
	// allocCount tracks the total number of allocated frames.
	allocCount uint64

	// lastAllocIndex tracks the last allocated frame index.
	lastAllocIndex int64
}

// Init sets up the boot memory allocator internal state and prints out the
// system memory map.
func (alloc *BootMemAllocator) Init() {
	alloc.lastAllocIndex = -1

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
// reserves the next available free frame.
//
// AllocFrame returns an error if no more memory can be allocated or when the
// requested page order is > 0.
func (alloc *BootMemAllocator) AllocFrame(order mem.PageOrder) (pmm.Frame, *kernel.Error) {
	if order > 0 {
		return pmm.InvalidFrame, errBootAllocUnsupportedPageSize
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
		return pmm.InvalidFrame, errBootAllocOutOfMemory
	}

	alloc.allocCount++
	alloc.lastAllocIndex = foundPageIndex

	return pmm.Frame(foundPageIndex), nil
}
