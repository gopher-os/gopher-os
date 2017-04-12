package physical

import "reflect"

const (
	MaxPageOrder = 10
	PageSize     = 1 << 12 // 4096
)

var (
	Allocator buddyAllocator
)

type buddyAllocator struct {
	// freeCount stores the number of free pages for each allocation order.
	// Initially, only the last order contains free pages. Having a free
	// counter allows us to quickly detect when the lower orders have no
	// pages available so we can immediately start scanning the higher orders.
	freeCount [MaxPageOrder]uint32

	// freeBitmap stores the free page bitmap data for each allocation order.
	// The bitmap for each order is stored as a []uint64. This allows us
	// to quickly traverse the bitmap when we search for a page to allocate
	// by examining 64 pages at a time (using bitwise ANDs) and only scan
	// individual bits when we are sure that one of the blocks contains a
	// free page.
	freeBitmap [MaxPageOrder][]uint64

	// bitmapSlice stores the slice structures for the freeBitmap entries.
	// It allows us to perform 2 passes to allocate their content. The first
	// pass populates their Len and Cap values with the number of required bits.
	// After calculating the total required bits for all bitmaps we perform a
	// second pass where we scan the available memory blocks looking for a
	// block that can fit all bitmaps and adjust the slice Data pointers
	// accordingly
	bitmapSlice [MaxPageOrder]reflect.SliceHeader
}

// setBitmapSizes updates the Len and Cap fields of the allocator's bitmap slice
// headers to the required number of bits for each allocation order.
//
// Given N pages of size PageSize:
// the bitmap for order(0) uses align(N, 64) bits, one for each block with size (PageSize << 0)
// the bitmap for order(M) uses ceil(N / M) bits, one for each block with size (PageSize << M)
//
// Since we use []uint64 for our bitmap entries, this method will pad the required
// number of bits per order so they are multiples of 64.
func (alloc *buddyAllocator) setBitmapSizes(pageCount uint64) {
	// Divide the number of bits by 64 (1<<6) to get the number of uint64 for the slice
	requiredUint64 := align(pageCount, 64) >> 6
	alloc.bitmapSlice[0].Cap, alloc.bitmapSlice[0].Len = int(requiredUint64), int(requiredUint64)

	for ord := uint64(1); ord < MaxPageOrder; ord++ {
		// the following line is equivalent to align(ceil(pageCount / ord), 64)
		requiredUint64 = align((pageCount+(1<<ord)-1)>>ord, 64) >> 6
		alloc.bitmapSlice[ord].Cap, alloc.bitmapSlice[ord].Len = int(requiredUint64), int(requiredUint64)
	}
}

// align ensures that v is a multiple of n.
func align(v, n uint64) uint64 {
	return (v + (n - 1)) & ^(n - 1)
}
