package allocator

import (
	"github.com/achilleasa/gopher-os/kernel"
	"github.com/achilleasa/gopher-os/kernel/hal/multiboot"
	"github.com/achilleasa/gopher-os/kernel/kfmt/early"
	"github.com/achilleasa/gopher-os/kernel/mem"
	"github.com/achilleasa/gopher-os/kernel/mem/pmm"
)

var (
	// earlyAllocator is a boot mem allocator instance used for page
	// allocations before switching to a more advanced allocator.
	earlyAllocator bootMemAllocator

	errBootAllocOutOfMemory = &kernel.Error{Module: "boot_mem_alloc", Message: "out of memory"}
)

// bootMemAllocator implements a rudimentary physical memory allocator which is
// used to bootstrap the kernel.
//
// The allocator implementation uses the memory region information provided by
// the bootloader to detect free memory blocks and return the next available
// free frame.  Allocations are tracked via an internal counter that contains
// the last allocated frame.
//
// Due to the way that the allocator works, it is not possible to free
// allocated pages. Once the kernel is properly initialized, the allocated
// blocks will be handed over to a more advanced memory allocator that does
// support freeing.
type bootMemAllocator struct {
	// allocCount tracks the total number of allocated frames.
	allocCount uint64

	// lastAllocFrame tracks the last allocated frame number.
	lastAllocFrame pmm.Frame
}

// init sets up the boot memory allocator internal state.
func (alloc *bootMemAllocator) init() {
	// TODO
}

// AllocFrame scans the system memory regions reported by the bootloader and
// reserves the next available free frame.
//
// AllocFrame returns an error if no more memory can be allocated.
func (alloc *bootMemAllocator) AllocFrame() (pmm.Frame, *kernel.Error) {
	var err = errBootAllocOutOfMemory

	multiboot.VisitMemRegions(func(region *multiboot.MemoryMapEntry) bool {
		// Ignore reserved regions and regions smaller than a single page
		if region.Type != multiboot.MemAvailable || region.Length < uint64(mem.PageSize) {
			return true
		}

		// Reported addresses may not be page-aligned; round up to get
		// the start frame and round down to get the end frame
		pageSizeMinus1 := uint64(mem.PageSize - 1)
		regionStartFrame := pmm.Frame(((region.PhysAddress + pageSizeMinus1) & ^pageSizeMinus1) >> mem.PageShift)
		regionEndFrame := pmm.Frame(((region.PhysAddress+region.Length) & ^pageSizeMinus1)>>mem.PageShift) - 1

		// Ignore already allocated regions
		if alloc.allocCount != 0 && alloc.lastAllocFrame >= regionEndFrame {
			return true
		}

		// The last allocated frame will be either pointing to a
		// previous region or will point inside this region. In the
		// first case (or if this is the first allocation) we select
		// the start frame for this region. In the latter case we
		// select the next available frame.
		if alloc.allocCount == 0 || alloc.lastAllocFrame < regionStartFrame {
			alloc.lastAllocFrame = regionStartFrame
		} else {
			alloc.lastAllocFrame++
		}
		err = nil
		return false
	})

	if err != nil {
		return pmm.InvalidFrame, errBootAllocOutOfMemory
	}

	alloc.allocCount++
	return alloc.lastAllocFrame, nil
}

// printMemoryMap scans the memory region information provided by the
// bootloader and prints out the system's memory map.
func (alloc *bootMemAllocator) printMemoryMap() {
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

// Init sets up the kernel physical memory allocation sub-system.
func Init() *kernel.Error {
	earlyAllocator.init()
	earlyAllocator.printMemoryMap()
	return nil
}
