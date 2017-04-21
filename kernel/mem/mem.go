package mem

import (
	"reflect"
	"unsafe"

	"github.com/achilleasa/gopher-os/kernel/hal/multiboot"
)

const (
	// PageShift is equal to log2(PageSize). This constant is used when
	// we need to convert a physical address to a page number (shift right by PageShift)
	// and vice-versa.
	PageShift = 12

	// PageSize defines the system's page size in bytes. By default it is
	// set to 4096 bytes.
	PageSize = Size(1 << PageShift)

	// MaxPageOrder defines the maximum page order that can be requested by
	// a page-based allocator.
	MaxPageOrder = PageOrder(9)
)

// Size represents a memory block size in bytes.
type Size uint64

// Common memory block sizes
const (
	Byte Size = 1
	Kb        = 1024 * Byte
	Mb        = 1024 * Kb
	Gb        = 1024 * Mb
)

// Order returns the smallest PageOrder that is suitable for storing a block of this size.
// Depending on the size, Order() may return a page order that is greater than MaxPageOrder.
func (s Size) Order() PageOrder {
	var order = PageOrder(0)
	for ; ; order++ {
		if PageSize<<order >= s {
			break
		}
	}

	return order
}

// Pages returns the number of pages that are required for storing this size.
func (s Size) Pages() uint32 {
	pageSizeMinus1 := PageSize - 1
	return uint32((s+pageSizeMinus1)&^pageSizeMinus1) >> PageShift
}

// PageOrder represents a power-of-two multiple of the base page size (PageSize)
// and is used as an argument to page-based memory allocators.
//
// PageOrder(0) refers to a page with size PageSize
// PageOrder(1) refers to a page with size PageSize * 2
// ...
// PageOrder(MaxPageOrder) refers to a page with size PageSize * 2^(MaxPageOrder)
type PageOrder uint8

var (
	// Overriden by tests
	visitMemRegionFn = multiboot.VisitMemRegions
)

// TotalSystemMemory returns the total amount of free or reserved memory on this
// system in bytes. The information about the available memory is retrieved using the
// multiboot package.
func TotalSystemMemory() Size {
	var total Size
	visitMemRegionFn(func(entry *multiboot.MemoryMapEntry) {
		total += Size(entry.Length)
	})

	return total
}

// Memset sets size bytes at the given address to the supplied value. The implementation
// is based on bytes.Repeat; instead of using a for loop, this function uses
// log2(size) copy calls which should give us a speed boost as page addresses
// are always aligned.
func Memset(addr uintptr, value byte, size uint32) {
	if size == 0 {
		return
	}

	// overlay a slice on top of this address region
	target := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Len:  int(size),
		Cap:  int(size),
		Data: addr,
	}))

	// Set first element and make log2(size) optimized copies
	target[0] = value
	for index := uint32(1); index < size; index *= 2 {
		copy(target[index:], target[:index])
	}
}

// Align ensures that v is a multiple of n.
func Align(v uint64, n Size) uint64 {
	return (v + uint64(n-1)) & ^uint64(n-1)
}
