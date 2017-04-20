package physical

import (
	"strconv"
	"testing"
	"unsafe"

	"github.com/achilleasa/gopher-os/kernel/errors"
	"github.com/achilleasa/gopher-os/kernel/mem"
)

func TestSplitHigherOrderPage(t *testing.T) {
	memSizeMB := 2
	alloc, _ := testAllocator(uint64(memSizeMB))

	// If we try to split a page with no pages available we will get an error
	if err := alloc.splitHigherOrderPage(Size4k); err != mem.ErrOutOfMemory {
		t.Fatalf("expected to get ErrOutOfMemory; got %v", err)
	}

	// Allow the highest order page to be split
	alloc.freeCount[maxPageOrder-1] = 1

	// Create a split
	if err := alloc.splitHigherOrderPage(Size4k); err != nil {
		t.Fatalf("unexpected error while splitting higher order page: %v", err)
	}
}

func TestReserveFreePage(t *testing.T) {
	memSizeMB := 2
	alloc, _ := testAllocator(uint64(memSizeMB))
	alloc.incFreeCountForLowerOrders(Size2048k)

	// Asking for an invalid order should return an error
	if _, err := alloc.reserveFreePage(maxPageOrder); err != errors.ErrInvalidParamValue {
		t.Fatalf("expected to get ErrInvalidParamValue; got %v", err)
	}

	// Allocate all available 4k pages
	ord0Pages := memSizeMB * 1024 * 1024 / mem.PageSize
	for i := 0; i < ord0Pages; i++ {
		addr, err := alloc.reserveFreePage(Size4k)
		if err != nil {
			t.Errorf("got unexpected error: %v, while trying to allocate page %d/%d", err, i, ord0Pages)
			continue
		}

		expAddr := uintptr(i * mem.PageSize)
		if addr != expAddr {
			t.Errorf("expected allocated address for page %d/%d to be 0x%x; got 0x%x", i, ord0Pages, expAddr, addr)
		}
	}

	if got := alloc.freeCount[0]; got != 0 {
		t.Fatalf("expected free count to be set to 0 after allocating all available ord(0) pages; got %d", got)
	}

	// The next allocation should fail
	if _, err := alloc.reserveFreePage(Size4k); err != mem.ErrOutOfMemory {
		t.Fatalf("expected to get ErrOutOfMemory; got %v", err)
	}
}

func TestUpdateLowerOrderBitmaps(t *testing.T) {
	type spec struct {
		page  uint32
		order Size
	}

	var specs []spec

	memSizeMB := 2
	for pages, ord := uint32(1), maxPageOrder-1; ord >= 0 && ord < maxPageOrder; pages, ord = pages<<1, ord-1 {
		for page := uint32(0); page < pages; page++ {
			specs = append(specs, spec{page, ord})
		}
	}

	for specIndex, spec := range specs {
		alloc, _ := testAllocator(uint64(memSizeMB))
		alloc.freeCount[spec.order]++
		alloc.incFreeCountForLowerOrders(spec.order)

		addr := spec.page << (mem.PageShift + spec.order)

		// Test markReserved
		alloc.updateLowerOrderBitmaps(uintptr(addr), spec.order, markReserved)

		for ord := Size(0); ord < spec.order; ord++ {
			if gotFree := alloc.freeCount[ord]; gotFree > 0 {
				t.Errorf("[spec %d] expected ord(%d) free page count to be 0; got %d", specIndex, ord, gotFree)
			}

			firstBit := uint32(addr >> uint32(mem.PageShift+ord))
			totalBits := uint32(1 << (spec.order - ord))

			for bit := firstBit; bit < firstBit+totalBits; bit++ {
				block := bit >> 6
				mask := uint64(1 << (63 - (bit & 63)))

				if (alloc.freeBitmap[ord][block] & mask) != mask {
					t.Errorf("[spec %d] expected ord(%d), block(%d) to have MSB bit %d set", specIndex, ord, block, bit&63)
				}
			}
		}

		// Test markFree
		alloc.updateLowerOrderBitmaps(uintptr(addr), spec.order, markFree)

		for ord := Size(0); ord < spec.order; ord++ {
			expFreeCount := uint32(1 << (spec.order - ord))
			if gotFree := alloc.freeCount[ord]; gotFree != expFreeCount {
				t.Errorf("[spec %d] expected ord(%d) free page count to be %d; got %d", specIndex, ord, expFreeCount, gotFree)
			}

			firstBit := uint32(addr >> uint32(mem.PageShift+ord))
			totalBits := uint32(1 << (spec.order - ord))

			for bit := firstBit; bit < firstBit+totalBits; bit++ {
				block := bit >> 6
				mask := uint64(1 << (63 - (bit & 63)))

				if (alloc.freeBitmap[ord][block] & mask) != 0 {
					t.Errorf("[spec %d] expected ord(%d), block(%d) to have MSB bit %d unset", specIndex, ord, block, bit&63)
				}
			}
		}
	}
}

func TestIncFreeCount(t *testing.T) {
	alloc, _ := testAllocator(1)

	// Sanity check; calling with an invalid order should have no effect
	alloc.incFreeCountForLowerOrders(maxPageOrder)
	for ord := Size(0); ord < maxPageOrder; ord++ {
		if got := alloc.freeCount[ord]; got != 0 {
			t.Fatalf("expected ord(%d) free count to be 0; got %d\n", ord, got)
		}
	}

	alloc.incFreeCountForLowerOrders(maxPageOrder - 1)
	for ord := Size(0); ord < maxPageOrder-2; ord++ {
		expCount := uint32(1 << (maxPageOrder - ord - 1))
		if got := alloc.freeCount[ord]; got != expCount {
			t.Fatalf("expected ord(%d) free count to be %d; got %d\n", ord, expCount, got)
		}
	}

}

func TestUpdateHigherOrderFlagsForInvalidOrder(t *testing.T) {
	alloc, _ := testAllocator(1)
	alloc.updateHigherOrderBitmaps(0, maxPageOrder)
	alloc.updateHigherOrderBitmaps(0, maxPageOrder+1)
}

func TestUpdateHigherOrderFlags(t *testing.T) {
	memSizeMB := uint64(4)
	pageCount := memSizeMB * 1024 * 1024 >> mem.PageShift

	alloc, _ := testAllocator(memSizeMB)

	for page := uint64(0); page < pageCount; page++ {
		for _, bitmap := range alloc.freeBitmap {
			for i := 0; i < len(bitmap); i++ {
				bitmap[i] = 0
			}
		}

		// Set the ord(0) bit that corresponds to that page to 1 and check that all parents are marked as reserved
		block := page / 64
		blockMask := uint64(1 << (63 - (page % 64)))
		alloc.freeBitmap[0][block] |= blockMask
		alloc.updateHigherOrderBitmaps(uintptr(page<<mem.PageShift), 0)
		for bitIndex, ord := page, Size(0); ord < maxPageOrder; bitIndex, ord = bitIndex>>1, ord+1 {
			val := alloc.freeBitmap[ord][bitIndex/64]
			valMask := uint64(1 << (63 - (bitIndex % 64)))
			if (val & valMask) == 0 {
				t.Errorf("[page %04d] expected [ord %d, block %d, bit %d] to be 1; got block value %064s", page, ord, bitIndex/64, 63-(bitIndex%64), strconv.FormatUint(val, 2))
			}
		}

		// Now clear the ord(0) bit and make sure that all parents are marked as free
		alloc.freeBitmap[0][block] ^= blockMask
		alloc.updateHigherOrderBitmaps(uintptr(page<<mem.PageShift), 0)
		for bitIndex, ord := page, Size(0); ord < maxPageOrder; bitIndex, ord = bitIndex>>1, ord+1 {
			val := alloc.freeBitmap[ord][bitIndex/64]
			if val != 0 {
				t.Errorf("[page %04d] expected [ord %d, block %d, bit %d] to be 0; got block value %064s", page, ord, bitIndex/64, 63-(bitIndex%64), strconv.FormatUint(val, 2))
			}
		}

		// Check buddy pages for even pages
		if page%2 == 0 {
			// Set the ord(0) bit for the buddy page and check that all parents (starting at ord 1) are marked as reserved
			// same bits to be set for ord(1 to maxPageOrder)
			alloc.freeBitmap[0][block] |= blockMask >> 1
			alloc.updateHigherOrderBitmaps(uintptr((page+1)<<mem.PageShift), 0)
			for bitIndex, ord := page>>1, Size(1); ord < maxPageOrder; bitIndex, ord = bitIndex>>1, ord+1 {
				val := alloc.freeBitmap[ord][bitIndex/64]
				valMask := uint64(1 << (63 - (bitIndex % 64)))
				if (val & valMask) == 0 {
					t.Errorf("[page %04d] expected [ord %d, block %d, bit %d] to be 1; got block value %064s", page, ord, bitIndex/64, 63-(bitIndex%64), strconv.FormatUint(val, 2))
				}
			}

			// Now clear the ord(0) bit for the buddy page and make sure that all parents are marked as free
			alloc.freeBitmap[0][block] ^= blockMask >> 1
			alloc.updateHigherOrderBitmaps(uintptr((page+1)<<mem.PageShift), 0)
			for bitIndex, ord := page, Size(0); ord < maxPageOrder; bitIndex, ord = bitIndex>>1, ord+1 {
				val := alloc.freeBitmap[ord][bitIndex/64]
				if val != 0 {
					t.Errorf("[page %04d] expected [ord %d, block %d, bit %d] to be 0; got block value %064s", page, ord, bitIndex/64, 63-(bitIndex%64), strconv.FormatUint(val, 2))
				}
			}

			// Finally mark both buddy pages at ord(0) as used and check that all parents (starting at ord 1) are marked as reserved
			alloc.freeBitmap[0][block] |= blockMask
			alloc.freeBitmap[0][block] |= blockMask >> 1
			alloc.updateHigherOrderBitmaps(uintptr(page<<mem.PageShift), 0)
			alloc.updateHigherOrderBitmaps(uintptr((page+1)<<mem.PageShift), 0)
			for bitIndex, ord := page>>1, Size(1); ord < maxPageOrder; bitIndex, ord = bitIndex>>1, ord+1 {
				val := alloc.freeBitmap[ord][bitIndex/64]
				valMask := uint64(1 << (63 - (bitIndex % 64)))
				if (val & valMask) == 0 {
					t.Errorf("[page %04d] expected [ord %d, block %d, bit %d] to be 1; got block value %064s", page, ord, bitIndex/64, 63-(bitIndex%64), strconv.FormatUint(val, 2))
				}
			}
		}
	}
}

func TestSetBitmapSizes(t *testing.T) {
	specs := []struct {
		pages         uint64
		expBitmapSize [maxPageOrder]int
	}{
		{
			1024, // 4mb
			[maxPageOrder]int{16, 8, 4, 2, 1, 1, 1, 1, 1, 1},
		},
		{
			32 * 1024, // 128MB
			[maxPageOrder]int{512, 256, 128, 64, 32, 16, 8, 4, 2, 1},
		},
		{
			1, // 4K
			// We need a full uint64 for ord(0) and we waste an empty
			// uint64 for each order due to rounding
			[maxPageOrder]int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		},
		{
			1025, // 4mb + 4k
			[maxPageOrder]int{17, 9, 5, 3, 2, 1, 1, 1, 1, 1},
		},
	}

	for specIndex, spec := range specs {
		alloc := &buddyAllocator{}
		alloc.setBitmapSizes(spec.pages)

		for ord := Size(0); ord < maxPageOrder; ord++ {
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
	alloc, scratchBuf := testAllocator(4)

	// Fill each bitmap entry with a special pattern
	for _, bitmap := range alloc.freeBitmap {
		for i := 0; i < len(bitmap); i++ {
			bitmap[i] = 0xFEFEFEFEFEFEFEFE
		}
	}

	// Check that the entire scratchBuf has been erased
	for i := 0; i < len(scratchBuf); i++ {
		if got := scratchBuf[i]; got != 0xFE {
			t.Errorf("expected scratchBuf[%d] to be set to 0xFE; got 0x%x", i, got)
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

func testAllocator(memInMB uint64) (*buddyAllocator, []byte) {
	alloc := &buddyAllocator{}
	alloc.setBitmapSizes(memInMB * 1024 * 1024 / mem.PageSize)

	requiredSize := 0
	for _, hdr := range alloc.bitmapSlice {
		requiredSize += hdr.Len * 8
	}

	// Setup pointers
	scratchBuf := make([]byte, requiredSize)
	alloc.setBitmapPointers(uintptr(unsafe.Pointer(&scratchBuf[0])))
	return alloc, scratchBuf
}
