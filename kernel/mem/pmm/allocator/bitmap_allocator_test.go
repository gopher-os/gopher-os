package allocator

import (
	"bytes"
	"math"
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

func TestAllocatorPackageInit(t *testing.T) {
	defer func() {
		mapFn = vmm.Map
		reserveRegionFn = vmm.EarlyReserveRegion
	}()

	var (
		physMem = make([]byte, 2*mem.PageSize)
		fb      = mockTTY()
		buf     bytes.Buffer
	)
	multiboot.SetInfoPtr(uintptr(unsafe.Pointer(&multibootMemoryMap[0])))

	t.Run("success", func(t *testing.T) {
		mapFn = func(page vmm.Page, frame pmm.Frame, flags vmm.PageTableEntryFlag) *kernel.Error {
			return nil
		}

		reserveRegionFn = func(_ mem.Size) (uintptr, *kernel.Error) {
			return uintptr(unsafe.Pointer(&physMem[0])), nil
		}

		if err := Init(0x100000, 0x1fa7c8); err != nil {
			t.Fatal(err)
		}

		for i := 0; i < len(fb); i += 2 {
			if fb[i] == 0x0 {
				continue
			}
			buf.WriteByte(fb[i])
		}

		exp := "[boot_mem_alloc] system memory map:    [0x0000000000 - 0x000009fc00], size:     654336, type: available    [0x000009fc00 - 0x00000a0000], size:       1024, type: reserved    [0x00000f0000 - 0x0000100000], size:      65536, type: reserved    [0x0000100000 - 0x0007fe0000], size:  133038080, type: available    [0x0007fe0000 - 0x0008000000], size:     131072, type: reserved    [0x00fffc0000 - 0x0100000000], size:     262144, type: reserved[boot_mem_alloc] available memory: 130559Kb[boot_mem_alloc] kernel loaded at 0x100000 - 0x1fa7c8[boot_mem_alloc] size: 1025992 bytes, reserved pages: 251"
		if got := buf.String(); got != exp {
			t.Fatalf("expected printMemoryMap to generate the following output:\n%q\ngot:\n%q", exp, got)
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
