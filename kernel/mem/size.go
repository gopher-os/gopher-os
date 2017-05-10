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
