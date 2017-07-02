package allocator

import (
	"gopheros/kernel/driver/video/console"
	"gopheros/kernel/hal"
	"gopheros/kernel/hal/multiboot"
	"testing"
	"unsafe"
)

func TestBootMemoryAllocator(t *testing.T) {
	multiboot.SetInfoPtr(uintptr(unsafe.Pointer(&multibootMemoryMap[0])))

	specs := []struct {
		kernelStart, kernelEnd uintptr
		expAllocCount          uint64
	}{
		{
			// the kernel is loaded in a reserved memory region
			0xa0000,
			0xa0000,
			// region 1 extents get rounded to [0, 9f000] and provides 159 frames [0 to 158]
			// region 1 uses the original extents [100000 - 7fe0000] and provides 32480 frames [256-32735]
			159 + 32480,
		},
		{
			// the kernel is loaded at the beginning of region 1 taking 2.5 pages
			0x0,
			0x2800,
			// region 1 extents get rounded to [0, 9f000] and provides 159 frames [0 to 158]; out of these
			// frames 0,1 and 2 (round up kernel end) are used by the kernel
			// region 1 uses the original extents [100000 - 7fe0000] and provides 32480 frames [256-32735]
			159 - 3 + 32480,
		},
		{
			// the kernel is loaded at the end of region 1 taking 2.5 pages
			0x9c800,
			0x9f000,
			// region 1 extents get rounded to [0, 9f000] and provides 159 frames [0 to 158]; out of these
			// frames 156,157 and 158 (round down kernel start) are used by the kernel
			// region 1 uses the original extents [100000 - 7fe0000] and provides 32480 frames [256-32735]
			159 - 3 + 32480,
		},
		{
			// the kernel (after rounding) uses the entire region 1
			0x123,
			0x9fc00,
			// region 1 extents get rounded to [0, 9f000] and provides 159 frames [0 to 158]; all are used
			// by the kernel
			// region 1 uses the original extents [100000 - 7fe0000] and provides 32480 frames [256-32735]
			32480,
		},
		{
			// the kernel is loaded at region 2 start + 2K taking 1.5 pages
			0x100800,
			0x102000,
			// region 1 extents get rounded to [0, 9f000] and provides 159 frames [0 to 158]
			// region 1 uses the original extents [100000 - 7fe0000] and provides 32480 frames [256-32735];
			// out of these frames 256 (kernel start rounded down) and 257 is used by the kernel
			159 + 32480 - 2,
		},
	}

	var alloc bootMemAllocator
	for specIndex, spec := range specs {
		alloc.allocCount = 0
		alloc.lastAllocFrame = 0
		alloc.init(spec.kernelStart, spec.kernelEnd)

		for {
			frame, err := alloc.AllocFrame()
			if err != nil {
				if err == errBootAllocOutOfMemory {
					break
				}
				t.Errorf("[spec %d] [frame %d] unexpected allocator error: %v", specIndex, alloc.allocCount, err)
				break
			}

			if frame != alloc.lastAllocFrame {
				t.Errorf("[spec %d] [frame %d] expected allocated frame to be %d; got %d", specIndex, alloc.allocCount, alloc.lastAllocFrame, frame)
			}

			if !frame.Valid() {
				t.Errorf("[spec %d] [frame %d] expected IsValid() to return true", specIndex, alloc.allocCount)
			}
		}

		if alloc.allocCount != spec.expAllocCount {
			t.Errorf("[spec %d] expected allocator to allocate %d frames; allocated %d", specIndex, spec.expAllocCount, alloc.allocCount)
		}
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
