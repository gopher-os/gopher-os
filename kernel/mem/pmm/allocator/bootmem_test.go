package allocator

import (
	"testing"
	"unsafe"

	"github.com/achilleasa/gopher-os/kernel/driver/video/console"
	"github.com/achilleasa/gopher-os/kernel/hal"
	"github.com/achilleasa/gopher-os/kernel/hal/multiboot"
	"github.com/achilleasa/gopher-os/kernel/mem"
)

func TestBootMemoryAllocator(t *testing.T) {
	// Mock a tty to handle early.Printf output
	mockConsoleFb := make([]byte, 160*25)
	mockConsole := &console.Ega{}
	mockConsole.Init(80, 25, uintptr(unsafe.Pointer(&mockConsoleFb[0])))
	hal.ActiveTerminal.AttachTo(mockConsole)

	multiboot.SetInfoPtr(uintptr(unsafe.Pointer(&multibootMemoryMap[0])))

	var totalFreeFrames uint64
	multiboot.VisitMemRegions(func(region *multiboot.MemoryMapEntry) bool {
		if region.Type == multiboot.MemAvailable {
			regionStartFrameIndex := uint64(((mem.Size(region.PhysAddress) + (mem.PageSize - 1)) & ^(mem.PageSize - 1)) >> mem.PageShift)
			regionEndFrameIndex := uint64(((mem.Size(region.PhysAddress+region.Length) - (mem.PageSize - 1)) & ^(mem.PageSize - 1)) >> mem.PageShift)

			totalFreeFrames += regionEndFrameIndex - regionStartFrameIndex + 1
		}

		return true
	})

	var (
		alloc           BootMemAllocator
		allocFrameCount uint64
	)
	for alloc.Init(); ; allocFrameCount++ {
		frame, err := alloc.AllocFrame()
		if err != nil {
			if err == errBootAllocOutOfMemory {
				break
			}
			t.Fatalf("[frame %d] unexpected allocator error: %v", allocFrameCount, err)
		}

		expAddress := uintptr(uint64(alloc.lastAllocIndex) * uint64(mem.PageSize))
		if got := frame.Address(); got != expAddress {
			t.Errorf("[frame %d] expected frame address to be 0x%x; got 0x%x", allocFrameCount, expAddress, got)
		}

		if !frame.Valid() {
			t.Errorf("[frame %d] expected IsValid() to return true", allocFrameCount)
		}
	}

	if allocFrameCount != totalFreeFrames {
		t.Fatalf("expected allocator to allocate %d frames; allocated %d", totalFreeFrames, allocFrameCount)
	}
}

var (
	// A dump of multiboot data when running under qemu containing only the memory region tag.
	multibootMemoryMap = []byte{
		72, 5, 0, 0, 0, 0, 0, 0,
		6, 0, 0, 0, 160, 0, 0, 0, 24, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 252, 9, 0, 0, 0, 0, 0,
		1, 0, 0, 0, 0, 0, 0, 0, 0, 252, 9, 0, 0, 0, 0, 0,
		0, 4, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 15, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0,
		2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 16, 0, 0, 0, 0, 0,
		0, 0, 238, 7, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 254, 7, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0,
		2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 252, 255, 0, 0, 0, 0,
		0, 0, 4, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0,
		9, 0, 0, 0, 212, 3, 0, 0, 24, 0, 0, 0, 40, 0, 0, 0,
		21, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 27, 0, 0, 0,
		1, 0, 0, 0, 2, 0, 0, 0, 0, 0, 16, 0, 0, 16, 0, 0,
		24, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}
)
