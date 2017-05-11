package mem

// Size represents a memory block size in bytes.
type Size uint64

// Common memory block sizes.
const (
	Byte Size = 1
	Kb        = 1024 * Byte
	Mb        = 1024 * Kb
	Gb        = 1024 * Mb
)

// PageOrder represents a power-of-two multiple of the base page size and is
// used as an argument to page-based memory allocators.
//
// PageOrder(0) refers to a page with size PageSize << 0
// PageOrder(1) refers to a page with size PageSize << 1
// ...
type PageOrder uint8
