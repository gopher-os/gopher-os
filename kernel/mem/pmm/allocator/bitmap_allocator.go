package allocator

import (
	"reflect"
	"unsafe"

	"github.com/achilleasa/gopher-os/kernel"
	"github.com/achilleasa/gopher-os/kernel/hal/multiboot"
	"github.com/achilleasa/gopher-os/kernel/mem"
	"github.com/achilleasa/gopher-os/kernel/mem/pmm"
	"github.com/achilleasa/gopher-os/kernel/mem/vmm"
)

var (
	// FrameAllocator is a BitmapAllocator instance that serves as the
	// primary allocator for reserving pages.
	FrameAllocator BitmapAllocator

	// The followning functions are used by tests to mock calls to the vmm package
	// and are automatically inlined by the compiler.
	reserveRegionFn = vmm.EarlyReserveRegion
	mapFn           = vmm.Map
)

type framePool struct {
	// startFrame is the frame number for the first page in this pool.
	// each free bitmap entry i corresponds to frame (startFrame + i).
	startFrame pmm.Frame

	// endFrame tracks the last frame in the pool. The total number of
	// frames is given by: (endFrame - startFrame) - 1
	endFrame pmm.Frame

	// freeCount tracks the available pages in this pool. The allocator
	// can use this field to skip fully allocated pools without the need
	// to scan the free bitmap.
	freeCount uint32

	// freeBitmap tracks used/free pages in the pool.
	freeBitmap    []uint64
	freeBitmapHdr reflect.SliceHeader
}

// BitmapAllocator implements a physical frame allocator that tracks frame
// reservations across the available memory pools using bitmaps.
type BitmapAllocator struct {
	// totalPages tracks the total number of pages across all pools.
	totalPages uint32

	// reservedPages tracks the number of reserved pages across all pools.
	reservedPages uint32

	pools    []framePool
	poolsHdr reflect.SliceHeader
}

// init allocates space for the allocator structures using the early bootmem
// allocator and flags any allocated pages as reserved.
func (alloc *BitmapAllocator) init() *kernel.Error {
	return alloc.setupPoolBitmaps()
}

// setupPoolBitmaps uses the early allocator and vmm region reservation helper
// to initialize the list of available pools and their free bitmap slices.
func (alloc *BitmapAllocator) setupPoolBitmaps() *kernel.Error {
	var (
		err                 *kernel.Error
		sizeofPool          = unsafe.Sizeof(framePool{})
		pageSizeMinus1      = uint64(mem.PageSize - 1)
		requiredBitmapBytes mem.Size
	)

	// Detect available memory regions and calculate their pool bitmap
	// requirements.
	multiboot.VisitMemRegions(func(region *multiboot.MemoryMapEntry) bool {
		if region.Type != multiboot.MemAvailable {
			return true
		}

		alloc.poolsHdr.Len++
		alloc.poolsHdr.Cap++

		// Reported addresses may not be page-aligned; round up to get
		// the start frame and round down to get the end frame
		regionStartFrame := pmm.Frame(((region.PhysAddress + pageSizeMinus1) & ^pageSizeMinus1) >> mem.PageShift)
		regionEndFrame := pmm.Frame(((region.PhysAddress+region.Length) & ^pageSizeMinus1)>>mem.PageShift) - 1
		pageCount := uint32(regionEndFrame - regionStartFrame)
		alloc.totalPages += pageCount

		// To represent the free page bitmap we need pageCount bits. Since our
		// slice uses uint64 for storing the bitmap we need to round up the
		// required bits so they are a multiple of 64 bits
		requiredBitmapBytes += mem.Size(((pageCount + 63) &^ 63) >> 3)
		return true
	})

	// Reserve enough pages to hold the allocator state
	requiredBytes := mem.Size(((uint64(uintptr(alloc.poolsHdr.Len)*sizeofPool) + uint64(requiredBitmapBytes)) + pageSizeMinus1) & ^pageSizeMinus1)
	requiredPages := requiredBytes >> mem.PageShift
	alloc.poolsHdr.Data, err = reserveRegionFn(requiredBytes)
	if err != nil {
		return err
	}

	for page, index := vmm.PageFromAddress(alloc.poolsHdr.Data), mem.Size(0); index < requiredPages; page, index = page+1, index+1 {
		nextFrame, err := earlyAllocFrame()
		if err != nil {
			return err
		}

		if err = mapFn(page, nextFrame, vmm.FlagPresent|vmm.FlagRW|vmm.FlagNoExecute); err != nil {
			return err
		}

		mem.Memset(page.Address(), 0, mem.PageSize)
	}

	alloc.pools = *(*[]framePool)(unsafe.Pointer(&alloc.poolsHdr))

	// Run a second pass to initialize the free bitmap slices for all pools
	bitmapStartAddr := alloc.poolsHdr.Data + uintptr(alloc.poolsHdr.Len)*sizeofPool
	poolIndex := 0
	multiboot.VisitMemRegions(func(region *multiboot.MemoryMapEntry) bool {
		if region.Type != multiboot.MemAvailable {
			return true
		}

		regionStartFrame := pmm.Frame(((region.PhysAddress + pageSizeMinus1) & ^pageSizeMinus1) >> mem.PageShift)
		regionEndFrame := pmm.Frame(((region.PhysAddress+region.Length) & ^pageSizeMinus1)>>mem.PageShift) - 1
		bitmapBytes := uintptr((((regionEndFrame - regionStartFrame) + 63) &^ 63) >> 3)

		alloc.pools[poolIndex].startFrame = regionStartFrame
		alloc.pools[poolIndex].endFrame = regionEndFrame
		alloc.pools[poolIndex].freeCount = uint32(regionEndFrame - regionStartFrame + 1)
		alloc.pools[poolIndex].freeBitmapHdr.Len = int(bitmapBytes >> 3)
		alloc.pools[poolIndex].freeBitmapHdr.Cap = alloc.pools[poolIndex].freeBitmapHdr.Len
		alloc.pools[poolIndex].freeBitmapHdr.Data = bitmapStartAddr
		alloc.pools[poolIndex].freeBitmap = *(*[]uint64)(unsafe.Pointer(&alloc.pools[poolIndex].freeBitmapHdr))

		bitmapStartAddr += bitmapBytes
		poolIndex++
		return true
	})

	return nil
}

// earlyAllocFrame is a helper that delegates a frame allocation request to the
// early allocator instance. This function is passed as an argument to
// vmm.SetFrameAllocator instead of earlyAllocator.AllocFrame. The latter
// confuses the compiler's escape analysis into thinking that
// earlyAllocator.Frame escapes to heap.
func earlyAllocFrame() (pmm.Frame, *kernel.Error) {
	return earlyAllocator.AllocFrame()
}

// Init sets up the kernel physical memory allocation sub-system.
func Init(kernelStart, kernelEnd uintptr) *kernel.Error {
	earlyAllocator.init(kernelStart, kernelEnd)
	earlyAllocator.printMemoryMap()

	vmm.SetFrameAllocator(earlyAllocFrame)
	return FrameAllocator.init()
}
