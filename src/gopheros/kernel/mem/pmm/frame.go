// Package pmm contains code that manages physical memory frame allocations.
package pmm

import (
	"gopheros/kernel/mem"
	"math"
)

// Frame describes a physical memory page index.
type Frame uintptr

const (
	// InvalidFrame is returned by page allocators when
	// they fail to reserve the requested frame.
	InvalidFrame = Frame(math.MaxUint64)
)

// Valid returns true if this is a valid frame.
func (f Frame) Valid() bool {
	return f != InvalidFrame
}

// Address returns a pointer to the physical memory address pointed to by this Frame.
func (f Frame) Address() uintptr {
	return uintptr(f << mem.PageShift)
}
