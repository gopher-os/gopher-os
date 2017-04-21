package physical

import (
	"strconv"
	"testing"
	"unsafe"

	"github.com/achilleasa/gopher-os/kernel/errors"
	"github.com/achilleasa/gopher-os/kernel/mem"
)

func TestAllocatePage(t *testing.T) {
	defer func() {
		memsetFn = memset
	}()

	var memsetCalled bool
	memsetFn = func(_ uintptr, _ byte, _ uint32) {
		memsetCalled = true
	}

	memSize := 2 * mem.Mb
	alloc, _ := testAllocator(memSize)
	alloc.freeCount[mem.MaxPageOrder] = 1

	// Test invalid param
	if _, err := alloc.AllocatePage(mem.MaxPageOrder+1, FlagKernel); err != errors.ErrInvalidParamValue {
		t.Fatalf("expected to get ErrInvalidParamValue; got %v", err)
	}

	// Allocate all ord(0) pages
	pageCount := memSize.Pages()
	for i := uint32(0); i < pageCount; i++ {
		memsetCalled = false

		// Even pages should not be cleared
		flags := FlagKernel
		if i%2 == 0 {
			flags |= FlagDoNotClear
		}

		expAddr := uintptr(i * uint32(mem.PageSize))
		addr, err := alloc.AllocatePage(mem.PageOrder(0), flags)
		if err != nil {
			t.Errorf("unexpected error while trying to allocate page %d/%d: %v", i, pageCount, err)
			continue
		}

		if addr != expAddr {
			t.Errorf("expected allocated page address for page %d/%d to be %d; got %d", i, pageCount, expAddr, addr)
		}

		switch i % 2 {
		case 0:
			if memsetCalled {
				t.Errorf("page: %d; expected memset not to be called when FlagDoNotClear is specified", i)
			}
		default:
			if !memsetCalled {
				t.Errorf("page: %d; expected memset to be called", i)
			}
		}
	}

	if got := alloc.freeCount[mem.MaxPageOrder]; got != 0 {
		t.Fatalf("expected ord(%d) free count to be 0; got %d", mem.MaxPageOrder, got)
	}

	// Allocating another ord(0) page should trigger a failing higher order split
	if _, err := alloc.AllocatePage(mem.PageOrder(0), FlagKernel); err != mem.ErrOutOfMemory {
		t.Fatalf("expected to get ErrOutOfMemory; got %v", err)
	}
}

func TestFreePage(t *testing.T) {
	defer func() {
		memsetFn = memset
	}()
	memsetFn = func(_ uintptr, _ byte, _ uint32) {}

	memSize := 2 * mem.Mb
	alloc, _ := testAllocator(memSize)
	alloc.freeCount[mem.MaxPageOrder] = 1

	// Test invalid param
	if err := alloc.FreePage(uintptr(0), mem.MaxPageOrder+1); err != errors.ErrInvalidParamValue {
		t.Fatalf("expected to get ErrInvalidParamValue; got %v", err)
	}

	// Test freeing of non-allocated page
	if err := alloc.FreePage(uintptr(0), mem.PageOrder(0)); err != ErrPageNotAllocated {
		t.Fatalf("expected to get ErrPageNotAllocated; got %v", err)
	}

	// Allocate and free a page
	addr, err := alloc.AllocatePage(mem.MaxPageOrder, FlagKernel)
	if err != nil {
		t.Fatal(err)
	}

	err = alloc.FreePage(addr, mem.MaxPageOrder)
	if err != nil {
		t.Fatal(err)
	}

	// Check free counts and bitmaps
	pageCount := memSize.Pages()
	for ord := mem.PageOrder(0); ord <= mem.MaxPageOrder; ord++ {
		expFreeCount := uint32(pageCount >> ord)
		if got := alloc.freeCount[ord]; got != expFreeCount {
			t.Errorf("expected free count for ord(%d) to be %d; got %d", ord, expFreeCount, got)
		}

		for blockIndex, block := range alloc.freeBitmap[ord] {
			if block != 0 {
				t.Errorf("expected all bits at ord(%d), block(%d) to be marked as free; got %064s", ord, blockIndex, strconv.FormatUint(block, 2))
			}
		}
	}
}

func TestSplitHigherOrderPage(t *testing.T) {
	memSize := 2 * mem.Mb
	alloc, _ := testAllocator(memSize)

	// If we try to split a page with no pages available we will get an error
	if err := alloc.splitHigherOrderPage(mem.PageOrder(0)); err != mem.ErrOutOfMemory {
		t.Fatalf("expected to get ErrOutOfMemory; got %v", err)
	}

	// Allow the highest order page to be split
	alloc.freeCount[mem.MaxPageOrder] = 1

	// Create a split
	if err := alloc.splitHigherOrderPage(mem.PageOrder(0)); err != nil {
		t.Fatalf("unexpected error while splitting higher order page: %v", err)
	}
}

func TestReserveFreePage(t *testing.T) {
	memSize := 2 * mem.Mb
	alloc, _ := testAllocator(memSize)
	alloc.incFreeCountForLowerOrders(mem.MaxPageOrder)

	// Asking for an invalid order should return an error
	if _, err := alloc.reserveFreePage(mem.MaxPageOrder + 1); err != errors.ErrInvalidParamValue {
		t.Fatalf("expected to get ErrInvalidParamValue; got %v", err)
	}

	// Allocate all available 4k pages
	ord0Pages := memSize.Pages()
	for i := uint32(0); i < ord0Pages; i++ {
		addr, err := alloc.reserveFreePage(mem.PageOrder(0))
		if err != nil {
			t.Errorf("got unexpected error: %v, while trying to allocate page %d/%d", err, i, ord0Pages)
			continue
		}

		expAddr := uintptr(i * uint32(mem.PageSize))
		if addr != expAddr {
			t.Errorf("expected allocated address for page %d/%d to be 0x%x; got 0x%x", i, ord0Pages, expAddr, addr)
		}
	}

	if got := alloc.freeCount[0]; got != 0 {
		t.Fatalf("expected free count to be set to 0 after allocating all available ord(0) pages; got %d", got)
	}

	// The next allocation should fail
	if _, err := alloc.reserveFreePage(mem.PageOrder(0)); err != mem.ErrOutOfMemory {
		t.Fatalf("expected to get ErrOutOfMemory; got %v", err)
	}
}

func TestUpdateLowerOrderBitmaps(t *testing.T) {
	type spec struct {
		page  uint32
		order mem.PageOrder
	}

	var specs []spec

	memSize := 2 * mem.Mb
	for pages, ord := uint32(1), mem.MaxPageOrder; ord >= 0 && ord <= mem.MaxPageOrder; pages, ord = pages<<1, ord-1 {
		for page := uint32(0); page < pages; page++ {
			specs = append(specs, spec{page, ord})
		}
	}

	for specIndex, spec := range specs {
		alloc, _ := testAllocator(memSize)
		alloc.freeCount[spec.order]++
		alloc.incFreeCountForLowerOrders(spec.order)

		addr := spec.page << (mem.PageShift + spec.order)

		// Test markReserved
		alloc.updateLowerOrderBitmaps(uintptr(addr), spec.order, markReserved)

		for ord := mem.PageOrder(0); ord < spec.order; ord++ {
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

		for ord := mem.PageOrder(0); ord < spec.order; ord++ {
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
	alloc, _ := testAllocator(1 * mem.Mb)

	// Sanity check; calling with an invalid order should have no effect
	alloc.incFreeCountForLowerOrders(mem.MaxPageOrder + 1)
	for ord := mem.PageOrder(0); ord <= mem.MaxPageOrder; ord++ {
		if got := alloc.freeCount[ord]; got != 0 {
			t.Fatalf("expected ord(%d) free count to be 0; got %d\n", ord, got)
		}
	}

	alloc.freeCount[mem.MaxPageOrder] = 1
	alloc.incFreeCountForLowerOrders(mem.MaxPageOrder)
	for ord := mem.PageOrder(0); ord <= mem.MaxPageOrder; ord++ {
		expCount := uint32(1 << (mem.MaxPageOrder - ord))
		if got := alloc.freeCount[ord]; got != expCount {
			t.Fatalf("expected ord(%d) free count to be %d; got %d\n", ord, expCount, got)
		}
	}

}

func TestUpdateHigherOrderBitmapsForInvalidOrder(t *testing.T) {
	alloc, _ := testAllocator(1 * mem.Mb)
	alloc.updateHigherOrderBitmaps(0, mem.MaxPageOrder+1)
}

func TestUpdateHigherOrderBitmapsFreeCounterUpdates(t *testing.T) {
	memSize := 2 * mem.Mb
	alloc, _ := testAllocator(memSize)
	alloc.freeCount[mem.MaxPageOrder] = 1
	alloc.incFreeCountForLowerOrders(mem.MaxPageOrder)

	// Flag the first page at ord(0) as used
	alloc.freeBitmap[0][0] |= (1 << 63)
	alloc.freeCount[0]--

	// This should reduce the available pages at each level up to and not
	// including mem.MaxPageOrder by 1 page.
	alloc.updateHigherOrderBitmaps(uintptr(0), mem.PageOrder(0))

	pageCount := memSize.Pages()

	for ord := mem.PageOrder(1); ord <= mem.MaxPageOrder; ord++ {
		expFreeCount := uint32((pageCount >> ord) - 1)
		if got := alloc.freeCount[ord]; got != expFreeCount {
			t.Errorf("expected free count at ord(%d) to be %d; got %d", ord, expFreeCount, got)
		}
	}

	// Flag the first page at ord(0) as free
	alloc.freeBitmap[0][0] &^= (1 << 63)
	alloc.freeCount[0]++

	// This should increment the available pages at each level up to
	// mem.MaxPageOrder by 1 page.
	alloc.updateHigherOrderBitmaps(uintptr(0), mem.PageOrder(0))
	for ord := mem.PageOrder(1); ord <= mem.MaxPageOrder; ord++ {
		expFreeCount := uint32(pageCount >> ord)
		if got := alloc.freeCount[ord]; got != expFreeCount {
			t.Errorf("expected free count at ord(%d) to be %d; got %d", ord, expFreeCount, got)
		}
	}
}

func TestUpdateHigherOrderBitmaps(t *testing.T) {
	memSize := 4 * mem.Mb
	pageCount := memSize.Pages()

	alloc, _ := testAllocator(memSize)

	for page := uint32(0); page < pageCount; page++ {
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
		for bitIndex, ord := page, mem.PageOrder(0); ord <= mem.MaxPageOrder; bitIndex, ord = bitIndex>>1, ord+1 {
			val := alloc.freeBitmap[ord][bitIndex/64]
			valMask := uint64(1 << (63 - (bitIndex % 64)))
			if (val & valMask) == 0 {
				t.Errorf("[page %04d] expected [ord %d, block %d, bit %d] to be 1; got block value %064s", page, ord, bitIndex/64, 63-(bitIndex%64), strconv.FormatUint(val, 2))
			}
		}

		// Now clear the ord(0) bit and make sure that all parents are marked as free
		alloc.freeBitmap[0][block] ^= blockMask
		alloc.updateHigherOrderBitmaps(uintptr(page<<mem.PageShift), 0)
		for bitIndex, ord := page, mem.PageOrder(0); ord <= mem.MaxPageOrder; bitIndex, ord = bitIndex>>1, ord+1 {
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
			for bitIndex, ord := page>>1, mem.PageOrder(1); ord <= mem.MaxPageOrder; bitIndex, ord = bitIndex>>1, ord+1 {
				val := alloc.freeBitmap[ord][bitIndex/64]
				valMask := uint64(1 << (63 - (bitIndex % 64)))
				if (val & valMask) == 0 {
					t.Errorf("[page %04d] expected [ord %d, block %d, bit %d] to be 1; got block value %064s", page, ord, bitIndex/64, 63-(bitIndex%64), strconv.FormatUint(val, 2))
				}
			}

			// Now clear the ord(0) bit for the buddy page and make sure that all parents are marked as free
			alloc.freeBitmap[0][block] ^= blockMask >> 1
			alloc.updateHigherOrderBitmaps(uintptr((page+1)<<mem.PageShift), 0)
			for bitIndex, ord := page, mem.PageOrder(0); ord <= mem.MaxPageOrder; bitIndex, ord = bitIndex>>1, ord+1 {
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
			for bitIndex, ord := page>>1, mem.PageOrder(1); ord <= mem.MaxPageOrder; bitIndex, ord = bitIndex>>1, ord+1 {
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
		size          mem.Size
		expBitmapSize [mem.MaxPageOrder + 1]int
	}{
		{
			4 * mem.Mb,
			[mem.MaxPageOrder + 1]int{16, 8, 4, 2, 1, 1, 1, 1, 1, 1},
		},
		{
			128 * mem.Mb,
			[mem.MaxPageOrder + 1]int{512, 256, 128, 64, 32, 16, 8, 4, 2, 1},
		},
		{
			4 * mem.Kb,
			// We need a full uint64 for ord(0) and we waste an empty
			// uint64 for each order due to rounding
			[mem.MaxPageOrder + 1]int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		},
		{
			4*mem.Mb + 4*mem.Kb,
			[mem.MaxPageOrder + 1]int{17, 9, 5, 3, 2, 1, 1, 1, 1, 1},
		},
	}

	for specIndex, spec := range specs {
		alloc := &buddyAllocator{}
		alloc.setBitmapSizes(spec.size.Pages())

		for ord := mem.PageOrder(0); ord <= mem.MaxPageOrder; ord++ {
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
	alloc, scratchBuf := testAllocator(4 * mem.Mb)

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
		in     uint32
		n      uint32
		expOut uint32
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

func TestMemset(t *testing.T) {
	// memset with a 0 size should be a no-op
	memset(uintptr(0), 0x00, 0)

	for ord := mem.PageOrder(0); ord <= mem.MaxPageOrder; ord++ {
		buf := make([]byte, mem.PageSize<<ord)
		for i := 0; i < len(buf); i++ {
			buf[i] = 0xFE
		}

		addr := uintptr(unsafe.Pointer(&buf[0]))
		memset(addr, 0x00, uint32(len(buf)))

		for i := 0; i < len(buf); i++ {
			if got := buf[i]; got != 0x00 {
				t.Errorf("expected ord(%d), byte: %d to be 0x00; got 0x%x", ord, i, got)
			}
		}
	}
}

func testAllocator(memSize mem.Size) (*buddyAllocator, []byte) {
	alloc := &buddyAllocator{}
	alloc.setBitmapSizes(memSize.Pages())

	requiredSize := 0
	for _, hdr := range alloc.bitmapSlice {
		requiredSize += hdr.Len * 8
	}

	// Setup pointers
	scratchBuf := make([]byte, requiredSize)
	alloc.setBitmapPointers(uintptr(unsafe.Pointer(&scratchBuf[0])))
	return alloc, scratchBuf
}
