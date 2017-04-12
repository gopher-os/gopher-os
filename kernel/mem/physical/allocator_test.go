package physical

import (
	"testing"
	"unsafe"
)

func TestSetBitmapSizes(t *testing.T) {
	specs := []struct {
		pages         uint64
		expBitmapSize [MaxPageOrder]int
	}{
		{
			1024, // 4mb
			[MaxPageOrder]int{16, 8, 4, 2, 1, 1, 1, 1, 1, 1},
		},
		{
			32 * 1024, // 128MB
			[MaxPageOrder]int{512, 256, 128, 64, 32, 16, 8, 4, 2, 1},
		},
		{
			1, // 4K
			// We need a full uint64 for ord(0) and we waste an empty
			// uint64 for each order due to rounding
			[MaxPageOrder]int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		},
		{
			1025, // 4mb + 4k
			[MaxPageOrder]int{17, 9, 5, 3, 2, 1, 1, 1, 1, 1},
		},
	}

	for specIndex, spec := range specs {
		alloc := &buddyAllocator{}
		alloc.setBitmapSizes(spec.pages)

		for ord := 0; ord < MaxPageOrder; ord++ {
			if alloc.bitmapSlice[ord].Len != alloc.bitmapSlice[ord].Cap {
				t.Errorf("[spec %d] ord(%d): expected slice Len to be equal to the slice Cap; got %d, %d", specIndex, ord, alloc.bitmapSlice[ord].Len, alloc.bitmapSlice[ord].Cap)
			}

			if alloc.bitmapSlice[ord].Len != spec.expBitmapSize[ord] {
				t.Errorf("[spec %d] expected bitmap size for ord(%d) to be %d; got %d", specIndex, ord, spec.expBitmapSize[ord], alloc.bitmapSlice[ord].Len)
			}
		}
	}
}

func TestSetBitmapPointers(t *testing.T) {
	var availMemInMB uint64 = 4

	alloc := &buddyAllocator{}
	alloc.setBitmapSizes(availMemInMB * 1024 * 1024 / PageSize)

	requiredSize := 0
	for _, hdr := range alloc.bitmapSlice {
		requiredSize += hdr.Len * 8
	}

	// Allocate scratch buffer and set it to a known pattern
	scratchBuf := make([]byte, requiredSize)
	for i := 0; i < len(scratchBuf); i++ {
		scratchBuf[i] = 0xFF
	}

	// Setup pointers and iterate each freeBitmap setting its contents to 0
	alloc.setBitmapPointers(uintptr(unsafe.Pointer(&scratchBuf[0])))
	for _, bitmap := range alloc.freeBitmap {
		for i := 0; i < len(bitmap); i++ {
			bitmap[i] = 0
		}
	}

	// Check that the entire scratchBuf has been erased
	for i := 0; i < len(scratchBuf); i++ {
		if got := scratchBuf[i]; got != 0 {
			t.Errorf("expected scratchBuf[%d] to be set to 0; got 0x%x", i, got)
		}
	}
}

func TestAlign(t *testing.T) {
	specs := []struct {
		in     uint64
		n      uint64
		expOut uint64
	}{
		{0, 64, 0},
		{1, 64, 64},
		{63, 64, 64},
		{64, 64, 64},
		{65, 64, 128},
	}

	for specIndex, spec := range specs {
		out := align(spec.in, spec.n)
		if out != spec.expOut {
			t.Errorf("[spec %d] expected align(%d, %d) to return %d; got %d", specIndex, spec.in, spec.n, spec.expOut, out)
		}
	}
}
