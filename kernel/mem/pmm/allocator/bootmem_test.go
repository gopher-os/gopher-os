package allocator

import (
	"bytes"
	"testing"
	"unsafe"

	"github.com/achilleasa/gopher-os/kernel/driver/video/console"
	"github.com/achilleasa/gopher-os/kernel/hal"
	"github.com/achilleasa/gopher-os/kernel/hal/multiboot"
)

func TestBootMemoryAllocator(t *testing.T) {
	multiboot.SetInfoPtr(uintptr(unsafe.Pointer(&multibootMemoryMap[0])))

	// region 1 extents get rounded to [0, 9f000] and provides 159 frames [0 to 158]
	// region 1 uses the original extents [100000 - 7fe0000] and provides 32480 frames [256-32735]
	var totalFreeFrames uint64 = 159 + 32480

	var (
		alloc           bootMemAllocator
		allocFrameCount uint64
	)
	for {
		frame, err := alloc.AllocFrame()
		if err != nil {
			if err == errBootAllocOutOfMemory {
				break
			}
			t.Fatalf("[frame %d] unexpected allocator error: %v", allocFrameCount, err)
		}
		allocFrameCount++
		if frame != alloc.lastAllocFrame {
			t.Errorf("[frame %d] expected allocated frame to be %d; got %d", allocFrameCount, alloc.lastAllocFrame, frame)
		}

		if !frame.Valid() {
			t.Errorf("[frame %d] expected IsValid() to return true", allocFrameCount)
		}
	}

	if allocFrameCount != totalFreeFrames {
		t.Fatalf("expected allocator to allocate %d frames; allocated %d", totalFreeFrames, allocFrameCount)
	}
}

func TestAllocatorPackageInit(t *testing.T) {
	fb := mockTTY()
	multiboot.SetInfoPtr(uintptr(unsafe.Pointer(&multibootMemoryMap[0])))

	Init()

	var buf bytes.Buffer
	for i := 0; i < len(fb); i += 2 {
		if fb[i] == 0x0 {
			continue
		}
		buf.WriteByte(fb[i])
	}

	exp := "[boot_mem_alloc] system memory map:    [0x0000000000 - 0x000009fc00], size:     654336, type: available    [0x000009fc00 - 0x00000a0000], size:       1024, type: reserved    [0x00000f0000 - 0x0000100000], size:      65536, type: reserved    [0x0000100000 - 0x0007fe0000], size:  133038080, type: available    [0x0007fe0000 - 0x0008000000], size:     131072, type: reserved    [0x00fffc0000 - 0x0100000000], size:     262144, type: reserved[boot_mem_alloc] free memory: 130559Kb"
	if got := buf.String(); got != exp {
		t.Fatalf("expected printMemoryMap to generate the following output:\n%q\ngot:\n%q", exp, got)
	}
}

var (
	// A dump of multiboot data when running under qemu containing only the
	// memory region tag.  The dump encodes the following available memory
	// regions:
	// [     0 -   9fc00] length:    654336
	// [100000 - 7fe0000] length: 133038080
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

func mockTTY() []byte {
	// Mock a tty to handle early.Printf output
	mockConsoleFb := make([]byte, 160*25)
	mockConsole := &console.Ega{}
	mockConsole.Init(80, 25, uintptr(unsafe.Pointer(&mockConsoleFb[0])))
	hal.ActiveTerminal.AttachTo(mockConsole)

	return mockConsoleFb
}
