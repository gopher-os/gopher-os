package pfn

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
	for ; ; allocFrameCount++ {
		frame, ok := alloc.AllocFrame(mem.PageOrder(0))
		if !ok {
			break
		}

		expAddress := uintptr(uint64(alloc.lastAllocIndex) * uint64(mem.PageSize))
		if got := frame.Address(); got != expAddress {
			t.Errorf("[frame %d] expected frame address to be 0x%x; got 0x%x", allocFrameCount, expAddress, got)
		}

		if !frame.IsValid() {
			t.Errorf("[frame %d] expected IsValid() to return true", allocFrameCount)
		}

		if got := frame.PageOrder(); got != mem.PageOrder(0) {
			t.Errorf("[frame %d] expected allocated frame page order to be 0; got %d", allocFrameCount, got)
		}

		if got := frame.Size(); got != mem.PageSize {
			t.Errorf("[frame %d] expected allocated frame size to be %d; got %d", allocFrameCount, mem.PageSize, got)
		}
	}

	if allocFrameCount != totalFreeFrames {
		t.Fatalf("expected allocator to allocate %d frames; allocated %d", totalFreeFrames, allocFrameCount)
	}

	// This allocator only works with order(0) blocks
	if frame, ok := alloc.AllocFrame(mem.PageOrder(1)); ok || frame.IsValid() {
		t.Fatalf("expected allocator to return false and an invalid frame when requested to allocate a block with order > 0; got %t, %v", ok, frame)
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
