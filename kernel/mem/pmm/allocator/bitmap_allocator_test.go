package allocator

import (
	"math"
	"strconv"
	"testing"
	"unsafe"

	"github.com/achilleasa/gopher-os/kernel"
	"github.com/achilleasa/gopher-os/kernel/hal/multiboot"
	"github.com/achilleasa/gopher-os/kernel/mem"
	"github.com/achilleasa/gopher-os/kernel/mem/pmm"
	"github.com/achilleasa/gopher-os/kernel/mem/vmm"
)

func TestSetupPoolBitmaps(t *testing.T) {
	defer func() {
		mapFn = vmm.Map
		reserveRegionFn = vmm.EarlyReserveRegion
	}()

	multiboot.SetInfoPtr(uintptr(unsafe.Pointer(&multibootMemoryMap[0])))

	// The captured multiboot data corresponds to qemu running with 128M RAM.
	// The allocator will need to reserve 2 pages to store the bitmap data.
	var (
		alloc   BitmapAllocator
		physMem = make([]byte, 2*mem.PageSize)
	)

	// Init phys mem with junk
	for i := 0; i < len(physMem); i++ {
		physMem[i] = 0xf0
	}

	mapCallCount := 0
	mapFn = func(page vmm.Page, frame pmm.Frame, flags vmm.PageTableEntryFlag) *kernel.Error {
		mapCallCount++
		return nil
	}

	reserveCallCount := 0
	reserveRegionFn = func(_ mem.Size) (uintptr, *kernel.Error) {
		reserveCallCount++
		return uintptr(unsafe.Pointer(&physMem[0])), nil
	}

	if err := alloc.setupPoolBitmaps(); err != nil {
		t.Fatal(err)
	}

	if exp := 2; mapCallCount != exp {
		t.Fatalf("expected allocator to call vmm.Map %d times; called %d", exp, mapCallCount)
	}

	if exp := 1; reserveCallCount != exp {
		t.Fatalf("expected allocator to call vmm.EarlyReserveRegion %d times; called %d", exp, reserveCallCount)
	}

	if exp, got := 2, len(alloc.pools); got != exp {
		t.Fatalf("expected allocator to initialize %d pools; got %d", exp, got)
	}

	for poolIndex, pool := range alloc.pools {
		if expFreeCount := uint32(pool.endFrame - pool.startFrame + 1); pool.freeCount != expFreeCount {
			t.Errorf("[pool %d] expected free count to be %d; got %d", poolIndex, expFreeCount, pool.freeCount)
		}

		if exp, got := int(math.Ceil(float64(pool.freeCount)/64.0)), len(pool.freeBitmap); got != exp {
			t.Errorf("[pool %d] expected bitmap len to be %d; got %d", poolIndex, exp, got)
		}

		for blockIndex, block := range pool.freeBitmap {
			if block != 0 {
				t.Errorf("[pool %d] expected bitmap block %d to be cleared; got %d", poolIndex, blockIndex, block)
			}
		}
	}
}

func TestSetupPoolBitmapsErrors(t *testing.T) {
	defer func() {
		mapFn = vmm.Map
		reserveRegionFn = vmm.EarlyReserveRegion
	}()

	multiboot.SetInfoPtr(uintptr(unsafe.Pointer(&multibootMemoryMap[0])))
	var alloc BitmapAllocator

	t.Run("vmm.EarlyReserveRegion returns an error", func(t *testing.T) {
		expErr := &kernel.Error{Module: "test", Message: "something went wrong"}

		reserveRegionFn = func(_ mem.Size) (uintptr, *kernel.Error) {
			return 0, expErr
		}

		if err := alloc.setupPoolBitmaps(); err != expErr {
			t.Fatalf("expected to get error: %v; got %v", expErr, err)
		}
	})
	t.Run("vmm.Map returns an error", func(t *testing.T) {
		expErr := &kernel.Error{Module: "test", Message: "something went wrong"}

		reserveRegionFn = func(_ mem.Size) (uintptr, *kernel.Error) {
			return 0, nil
		}

		mapFn = func(page vmm.Page, frame pmm.Frame, flags vmm.PageTableEntryFlag) *kernel.Error {
			return expErr
		}

		if err := alloc.setupPoolBitmaps(); err != expErr {
			t.Fatalf("expected to get error: %v; got %v", expErr, err)
		}
	})

	t.Run("earlyAllocator returns an error", func(t *testing.T) {
		emptyInfoData := []byte{
			0, 0, 0, 0, // size
			0, 0, 0, 0, // reserved
			0, 0, 0, 0, // tag with type zero and length zero
			0, 0, 0, 0,
		}

		multiboot.SetInfoPtr(uintptr(unsafe.Pointer(&emptyInfoData[0])))

		if err := alloc.setupPoolBitmaps(); err != errBootAllocOutOfMemory {
			t.Fatalf("expected to get error: %v; got %v", errBootAllocOutOfMemory, err)
		}
	})
}

func TestBitmapAllocatorMarkFrame(t *testing.T) {
	var alloc = BitmapAllocator{
		pools: []framePool{
			{
				startFrame: pmm.Frame(0),
				endFrame:   pmm.Frame(127),
				freeCount:  128,
				freeBitmap: make([]uint64, 2),
			},
		},
		totalPages: 128,
	}

	lastFrame := pmm.Frame(alloc.totalPages)
	for frame := pmm.Frame(0); frame < lastFrame; frame++ {
		alloc.markFrame(0, frame, markReserved)

		block := uint64(frame / 64)
		blockOffset := uint64(frame % 64)
		bitIndex := (63 - blockOffset)
		bitMask := uint64(1 << bitIndex)

		if alloc.pools[0].freeBitmap[block]&bitMask != bitMask {
			t.Errorf("[frame %d] expected block[%d], bit %d to be set", frame, block, bitIndex)
		}

		alloc.markFrame(0, frame, markFree)

		if alloc.pools[0].freeBitmap[block]&bitMask != 0 {
			t.Errorf("[frame %d] expected block[%d], bit %d to be unset", frame, block, bitIndex)
		}
	}

	// Calling markFrame with a frame not part of the pool should be a no-op
	alloc.markFrame(0, pmm.Frame(0xbadf00d), markReserved)
	for blockIndex, block := range alloc.pools[0].freeBitmap {
		if block != 0 {
			t.Errorf("expected all blocks to be set to 0; block %d is set to %d", blockIndex, block)
		}
	}

	// Calling markFrame with a negative pool index should be a no-op
	alloc.markFrame(-1, pmm.Frame(0), markReserved)
	for blockIndex, block := range alloc.pools[0].freeBitmap {
		if block != 0 {
			t.Errorf("expected all blocks to be set to 0; block %d is set to %d", blockIndex, block)
		}
	}
}

func TestBitmapAllocatorPoolForFrame(t *testing.T) {
	var alloc = BitmapAllocator{
		pools: []framePool{
			{
				startFrame: pmm.Frame(0),
				endFrame:   pmm.Frame(63),
				freeCount:  64,
				freeBitmap: make([]uint64, 1),
			},
			{
				startFrame: pmm.Frame(128),
				endFrame:   pmm.Frame(191),
				freeCount:  64,
				freeBitmap: make([]uint64, 1),
			},
		},
		totalPages: 128,
	}

	specs := []struct {
		frame    pmm.Frame
		expIndex int
	}{
		{pmm.Frame(0), 0},
		{pmm.Frame(63), 0},
		{pmm.Frame(64), -1},
		{pmm.Frame(128), 1},
		{pmm.Frame(192), -1},
	}

	for specIndex, spec := range specs {
		if got := alloc.poolForFrame(spec.frame); got != spec.expIndex {
			t.Errorf("[spec %d] expected to get pool index %d; got %d", specIndex, spec.expIndex, got)
		}
	}
}

func TestBitmapAllocatorReserveKernelFrames(t *testing.T) {
	var alloc = BitmapAllocator{
		pools: []framePool{
			{
				startFrame: pmm.Frame(0),
				endFrame:   pmm.Frame(7),
				freeCount:  8,
				freeBitmap: make([]uint64, 1),
			},
			{
				startFrame: pmm.Frame(64),
				endFrame:   pmm.Frame(191),
				freeCount:  128,
				freeBitmap: make([]uint64, 2),
			},
		},
		totalPages: 136,
	}

	// kernel occupies 16 frames and starts at the beginning of pool 1
	earlyAllocator.kernelStartFrame = pmm.Frame(64)
	earlyAllocator.kernelEndFrame = pmm.Frame(79)
	kernelSizePages := uint32(earlyAllocator.kernelEndFrame - earlyAllocator.kernelStartFrame + 1)
	alloc.reserveKernelFrames()

	if exp, got := kernelSizePages, alloc.reservedPages; got != exp {
		t.Fatalf("expected reserved page counter to be %d; got %d", exp, got)
	}

	if exp, got := uint32(8), alloc.pools[0].freeCount; got != exp {
		t.Fatalf("expected free count for pool 0 to be %d; got %d", exp, got)
	}

	if exp, got := 128-kernelSizePages, alloc.pools[1].freeCount; got != exp {
		t.Fatalf("expected free count for pool 1 to be %d; got %d", exp, got)
	}

	// The first 16 bits of block 0 in pool 1 should all be set to 1
	if exp, got := uint64(((1<<16)-1)<<48), alloc.pools[1].freeBitmap[0]; got != exp {
		t.Fatalf("expected block 0 in pool 1 to be:\n%064s\ngot:\n%064s",
			strconv.FormatUint(exp, 2),
			strconv.FormatUint(got, 2),
		)
	}
}

func TestBitmapAllocatorReserveEarlyAllocatorFrames(t *testing.T) {
	var alloc = BitmapAllocator{
		pools: []framePool{
			{
				startFrame: pmm.Frame(0),
				endFrame:   pmm.Frame(63),
				freeCount:  64,
				freeBitmap: make([]uint64, 1),
			},
			{
				startFrame: pmm.Frame(64),
				endFrame:   pmm.Frame(191),
				freeCount:  128,
				freeBitmap: make([]uint64, 2),
			},
		},
		totalPages: 64,
	}

	multiboot.SetInfoPtr(uintptr(unsafe.Pointer(&multibootMemoryMap[0])))

	// Simulate 16 allocations made using the early allocator in region 0
	// as reported by the multiboot data and move the kernel to pool 1
	allocCount := uint32(16)
	earlyAllocator.allocCount = uint64(allocCount)
	earlyAllocator.kernelStartFrame = pmm.Frame(256)
	earlyAllocator.kernelEndFrame = pmm.Frame(256)
	alloc.reserveEarlyAllocatorFrames()

	if exp, got := allocCount, alloc.reservedPages; got != exp {
		t.Fatalf("expected reserved page counter to be %d; got %d", exp, got)
	}

	if exp, got := 64-allocCount, alloc.pools[0].freeCount; got != exp {
		t.Fatalf("expected free count for pool 0 to be %d; got %d", exp, got)
	}

	if exp, got := uint32(128), alloc.pools[1].freeCount; got != exp {
		t.Fatalf("expected free count for pool 1 to be %d; got %d", exp, got)
	}

	// The first 16 bits of block 0 in pool 0 should all be set to 1
	if exp, got := uint64(((1<<16)-1)<<48), alloc.pools[0].freeBitmap[0]; got != exp {
		t.Fatalf("expected block 0 in pool 0 to be:\n%064s\ngot:\n%064s",
			strconv.FormatUint(exp, 2),
			strconv.FormatUint(got, 2),
		)
	}
}

func TestBitmapAllocatorAllocAndFreeFrame(t *testing.T) {
	var alloc = BitmapAllocator{
		pools: []framePool{
			{
				startFrame: pmm.Frame(0),
				endFrame:   pmm.Frame(7),
				freeCount:  8,
				// only the first 8 bits of block 0 are used
				freeBitmap: make([]uint64, 1),
			},
			{
				startFrame: pmm.Frame(64),
				endFrame:   pmm.Frame(191),
				freeCount:  128,
				freeBitmap: make([]uint64, 2),
			},
		},
		totalPages: 136,
	}

	// Test Alloc
	for poolIndex, pool := range alloc.pools {
		for expFrame := pool.startFrame; expFrame <= pool.endFrame; expFrame++ {
			got, err := alloc.AllocFrame()
			if err != nil {
				t.Fatalf("[pool %d] unexpected error: %v", poolIndex, err)
			}

			if got != expFrame {
				t.Errorf("[pool %d] expected allocated frame to be %d; got %d", poolIndex, expFrame, got)
			}
		}

		if alloc.pools[poolIndex].freeCount != 0 {
			t.Errorf("[pool %d] expected free count to be 0; got %d", poolIndex, alloc.pools[poolIndex].freeCount)
		}
	}

	if alloc.reservedPages != alloc.totalPages {
		t.Errorf("expected reservedPages to match totalPages(%d); got %d", alloc.totalPages, alloc.reservedPages)
	}

	if _, err := alloc.AllocFrame(); err != errBitmapAllocOutOfMemory {
		t.Fatalf("expected error errBitmapAllocOutOfMemory; got %v", err)
	}

	// Test Free
	expFreeCount := []uint32{8, 128}
	for poolIndex, pool := range alloc.pools {
		for frame := pool.startFrame; frame <= pool.endFrame; frame++ {
			if err := alloc.FreeFrame(frame); err != nil {
				t.Fatalf("[pool %d] unexpected error: %v", poolIndex, err)
			}
		}

		if alloc.pools[poolIndex].freeCount != expFreeCount[poolIndex] {
			t.Errorf("[pool %d] expected free count to be %d; got %d", poolIndex, expFreeCount[poolIndex], alloc.pools[poolIndex].freeCount)
		}
	}

	if alloc.reservedPages != 0 {
		t.Errorf("expected reservedPages to be 0; got %d", alloc.reservedPages)
	}

	// Test Free errors
	if err := alloc.FreeFrame(pmm.Frame(0)); err != errBitmapAllocDoubleFree {
		t.Fatalf("expected error errBitmapAllocDoubleFree; got %v", err)
	}

	if err := alloc.FreeFrame(pmm.Frame(0xbadf00d)); err != errBitmapAllocFrameNotManaged {
		t.Fatalf("expected error errBitmapFrameNotManaged; got %v", err)
	}
}

func TestAllocatorPackageInit(t *testing.T) {
	defer func() {
		mapFn = vmm.Map
		reserveRegionFn = vmm.EarlyReserveRegion
	}()

	var (
		physMem = make([]byte, 2*mem.PageSize)
	)
	multiboot.SetInfoPtr(uintptr(unsafe.Pointer(&multibootMemoryMap[0])))

	t.Run("success", func(t *testing.T) {
		mapFn = func(page vmm.Page, frame pmm.Frame, flags vmm.PageTableEntryFlag) *kernel.Error {
			return nil
		}

		reserveRegionFn = func(_ mem.Size) (uintptr, *kernel.Error) {
			return uintptr(unsafe.Pointer(&physMem[0])), nil
		}

		mockTTY()
		if err := Init(0x100000, 0x1fa7c8); err != nil {
			t.Fatal(err)
		}

		// At this point sysAllocFrame should work
		if _, err := AllocFrame(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("error", func(t *testing.T) {
		expErr := &kernel.Error{Module: "test", Message: "something went wrong"}

		mapFn = func(page vmm.Page, frame pmm.Frame, flags vmm.PageTableEntryFlag) *kernel.Error {
			return expErr
		}

		if err := Init(0x100000, 0x1fa7c8); err != expErr {
			t.Fatalf("expected to get error: %v; got %v", expErr, err)
		}
	})
}
