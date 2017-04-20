package physical

import (
	"math"
	"reflect"
	"unsafe"

	"github.com/achilleasa/gopher-os/kernel/errors"
	"github.com/achilleasa/gopher-os/kernel/mem"
)

// Size defines the page sizes that can be allocated by the physical memory allocator.
type Size uint8

// The list of suported block sizes that can be passed to the allocator.
const (
	Size4k Size = iota
	Size8k
	Size16k
	Size32k
	Size64k
	Size128k
	Size256k
	Size512k
	Size1024k
	Size2048k
	maxPageOrder
)

type reservationMode uint8

const (
	markFree     reservationMode = 0
	markReserved                 = 1
)

var (
	Allocator buddyAllocator
)

type buddyAllocator struct {
	// freeCount stores the number of free pages for each allocation order.
	// Initially, only the last order contains free pages. Having a free
	// counter allows us to quickly detect when the lower orders have no
	// pages available so we can immediately start scanning the higher orders.
	freeCount [maxPageOrder]uint32

	// freeBitmap stores the free page bitmap data for each allocation order.
	// The bitmap for each order is stored as a []uint64. This allows us
	// to quickly traverse the bitmap when we search for a page to allocate
	// by examining 64 pages at a time (using bitwise ANDs) and only scan
	// individual bits when we are sure that one of the blocks contains a
	// free page.
	freeBitmap [maxPageOrder][]uint64

	// bitmapSlice stores the slice structures for the freeBitmap entries.
	// It allows us to perform 2 passes to allocate their content. The first
	// pass populates their Len and Cap values with the number of required bits.
	// After calculating the total required bits for all bitmaps we perform a
	// second pass where we scan the available memory blocks looking for a
	// block that can fit all bitmaps and adjust the slice Data pointers
	// accordingly.
	bitmapSlice [maxPageOrder]reflect.SliceHeader
}

// AllocatePage allocates a page with the given size (order) and returns back
// its address or an error if no free pages are available.
func (alloc *buddyAllocator) AllocatePage(order Size) (uintptr, error) {
	// Sanity checks
	if order >= maxPageOrder {
		return uintptr(0), errors.ErrInvalidParamValue
	}

	// If no pages are free at the requested order we may need to split a
	// higher order page to make some room.
	if alloc.freeCount[order] == 0 {
		err := alloc.splitHigherOrderPage(order)
		if err != nil {
			return uintptr(0), err
		}
	}

	// Since we are guaranteed to find a free page this call can never fail
	addr, _ := alloc.reserveFreePage(order)

	alloc.updateLowerOrderBitmaps(addr, order, markReserved)
	alloc.updateHigherOrderBitmaps(addr, order)
	return addr, nil
}

// splitHigherOrderPage searches for the first available page with order greater
// than the requested order. If a free page is found, it is marked as reserved and
// the free counts for the orders below it are updated accordingly.
func (alloc *buddyAllocator) splitHigherOrderPage(order Size) error {
	for order = order + 1; order < maxPageOrder; order++ {
		if alloc.freeCount[order] == 0 {
			continue
		}

		// This order has free pages. Reserve the first available and
		// make its space available to the order below it
		alloc.reserveFreePage(order)
		alloc.incFreeCountForLowerOrders(order)
		return nil
	}

	return mem.ErrOutOfMemory
}

// reserveFreePage scans the free page bitmaps for the given order, reserves the
// first available page and returns its address. If no pages at this order are
// available then this method returns ErrOutOfMemory.
func (alloc *buddyAllocator) reserveFreePage(order Size) (uintptr, error) {
	if order >= maxPageOrder {
		return uintptr(0), errors.ErrInvalidParamValue
	}

	for blockIndex, block := range alloc.freeBitmap[order] {
		// Entire block is allocated; skip it
		if block == math.MaxUint64 {
			continue

		}

		// Scan the individual bits to find the block and reserve it
		for bitIndex := uint8(0); bitIndex < 64; bitIndex++ {
			mask := uint64(1 << (63 - bitIndex))

			// Ignore used bits
			if (block & mask) != 0 {
				continue
			}

			// Mark page as allocated and decrement the free page count for this order
			alloc.freeBitmap[order][blockIndex] |= mask
			alloc.freeCount[order]--

			return uintptr(mem.PageSize) * ((uintptr(blockIndex) << 6) + uintptr(bitIndex)), nil
		}
	}

	return uintptr(0), mem.ErrOutOfMemory
}

// updateLowerOrderBitmaps hierarchically traverses the free bitmaps at the orders
// below the supplied order and depending on the requested reservation mode either
// sets or unsets the used bits that correspond to the supplied address.
func (alloc *buddyAllocator) updateLowerOrderBitmaps(addr uintptr, order Size, mode reservationMode) {
	order--

	var (
		firstBitIndex              uint32 = uint32(addr >> (mem.PageShift + order))
		totalBitCount              uint32 = 2
		bitsToChange, lastBitIndex uint32
	)

	for ; order >= 0 && order < maxPageOrder; order = order - 1 {
		lastBitIndex = firstBitIndex + totalBitCount
		for bitIndex := firstBitIndex; bitIndex < lastBitIndex; bitIndex += bitsToChange {
			block := bitIndex >> 6
			blockOffset := bitIndex & 63

			// We need to change min(64, lastBitIndex - bitIndex) bits in this block
			bitsToChange = lastBitIndex - bitIndex
			if bitsToChange > 64 {
				bitsToChange = 64
			}

			// To build the block mask we start with a value with the
			// bitsToChange LSB set and shift it right so it alignts with
			// the offset position in the block
			blockMask := uint64(((1 << (bitsToChange)) - 1) << (64 - blockOffset - bitsToChange))

			// Mark either as reserved (set to 1) or free (set to 0)
			if mode == markReserved {
				alloc.freeBitmap[order][block] |= blockMask
			} else {
				alloc.freeBitmap[order][block] &^= blockMask
			}
		}

		if mode == markReserved {
			alloc.freeCount[order] -= totalBitCount
		} else {
			alloc.freeCount[order] += totalBitCount
		}

		// Each time we descend an order the first bit index and the number
		// of bits we need to set/unset doubles
		firstBitIndex <<= 1
		totalBitCount <<= 1
	}
}

// updateHigherOrderBitmaps hierarchically traverses the free bitmaps from lower
// to higher orders and for each order, updates the page bit that corresponds
// to the supplied physical address based on the value of the 2 buddy pages of
// the order below it. The status of page at ord(N) is set to the OR-ed value
// of the 2 buddy pages at ord(N-1).
func (alloc *buddyAllocator) updateHigherOrderBitmaps(addr uintptr, order Size) {
	// sanity checks
	if order == maxPageOrder {
		return
	}

	// ord(0) has no children
	if order == 0 {
		order++
	}

	var bitIndex, block, blockMask, childBitIndex, childBlock, childBlockMask uint64
	var wasReserved bool
	for bitIndex = uint64(addr) >> (mem.PageShift + order); order < maxPageOrder; order, bitIndex = order+1, bitIndex>>1 {
		block = bitIndex >> 6
		blockMask = 1 << (63 - (bitIndex & 63))
		wasReserved = (alloc.freeBitmap[order][block] & blockMask) == blockMask

		// This bit should be marked as used any of the (ord-1) bits:
		// (2*bit)+1 or (2*bit)+2 are marked as used. The child mask
		// that includes these bits is calculated by shifting the
		// value "3" (11b) left childBitIndex positions.
		childBitIndex = (bitIndex << 1) + 1
		childBlock = childBitIndex >> 6
		childBlockMask = 3 << (63 - (childBitIndex & 63))

		switch alloc.freeBitmap[order-1][childBlock] & childBlockMask {
		case 0: // both bits are not set; we just need to clear the bit
			alloc.freeBitmap[order][block] &^= blockMask

			if wasReserved {
				alloc.freeCount[order]++
			}
		default: // one or both bits are set; we just need to set the bit
			alloc.freeBitmap[order][block] |= blockMask

			if !wasReserved {
				alloc.freeCount[order]--
			}
		}
	}
}

// incFreeCountForLowerOrders is called when a free page at ord(N) is allocated
// to update all free page counters for all orders less than or equal to N. The
// number of free pages that are added to the counters doubles for each order less than N.
func (alloc *buddyAllocator) incFreeCountForLowerOrders(order Size) {
	// sanity check
	if order >= maxPageOrder {
		return
	}

	// When ord reaches 0; ord - 1 will wrap to MaxUint32 so we need to check for that as well
	freeCount := uint32(2)
	for order = order - 1; order >= 0 && order < maxPageOrder; order, freeCount = order-1, freeCount<<1 {
		alloc.freeCount[order] += freeCount
	}
}

// setBitmapSizes updates the Len and Cap fields of the allocator's bitmap slice
// headers to the required number of bits for each allocation order.
//
// Given N pages of size mem.PageSize:
// the bitmap for order(0) uses align(N, 64) bits, one for each block with size (mem.PageSize << 0)
// the bitmap for order(M) uses ceil(N / M) bits, one for each block with size (mem.PageSize << M)
//
// Since we use []uint64 for our bitmap entries, this method will pad the required
// number of bits per order so they are multiples of 64.
func (alloc *buddyAllocator) setBitmapSizes(pageCount uint64) {
	for order := Size(0); order < maxPageOrder; order++ {
		requiredUint64 := requiredUint64(pageCount, order)
		alloc.bitmapSlice[order].Cap, alloc.bitmapSlice[order].Len = requiredUint64, requiredUint64
	}
}

// setBitmapPointers updates the Data field for the allocator's bitmap slice
// headers so that each slice's data begins at a 8-byte aligned offset after the
// provided baseAddr value.
//
// This method also patches the freeBitmap slice entries so that they point to the
// populated slice header structs.
//
// After a call to setBitmapPointers, the allocator will be able to freely access
// all freeBitmap entries.
func (alloc *buddyAllocator) setBitmapPointers(baseAddr uintptr) {
	var dataPtr = baseAddr
	for ord := Size(0); ord < maxPageOrder; ord++ {
		alloc.bitmapSlice[ord].Data = dataPtr
		alloc.freeBitmap[ord] = *(*[]uint64)(unsafe.Pointer(&alloc.bitmapSlice[ord]))

		// offset += ordLen * 8 bytes per uint64
		dataPtr += uintptr(alloc.bitmapSlice[ord].Len << 3)
	}
}

// align ensures that v is a multiple of n.
func align(v, n uint64) uint64 {
	return (v + (n - 1)) & ^(n - 1)
}

// requiredUint64 returns the number of uint64 required for storing a bitmap
// of order(ord) for pageCount pages.
func requiredUint64(pageCount uint64, order Size) int {
	// requiredBits = pageCount / (2*ord) + pageCount % (2*ord)
	requiredBits := (pageCount >> order) + (pageCount & ((1 << order) - 1))
	return int(align(requiredBits, 64) >> 6)
}
