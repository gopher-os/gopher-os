// Package pmm contains code that manages physical memory frame allocations.
package pmm

import (
	"math"

	"github.com/achilleasa/gopher-os/kernel/mem"
)

// Frame describes a physical memory page index.
type Frame uint64

const (
	// InvalidFrame is returned by page allocators when
	// they fail to reserve the requested frame.
	InvalidFrame = Frame(math.MaxUint64)
)

// IsValid returns true if this is a valid frame.
func (f Frame) IsValid() bool {
	return f != InvalidFrame
}

// Address returns a pointer to the physical memory address pointed to by this Frame.
func (f Frame) Address() uintptr {
	return uintptr(f << mem.PageShift)
}

// PageOrder returns the page order of this frame. The page order is encoded in the
// 8 MSB of the frame number.
func (f Frame) PageOrder() mem.PageOrder {
	return mem.PageOrder((f >> 56) & 0xFF)
}

// Size returns the size of this frame.
func (f Frame) Size() mem.Size {
	return mem.PageSize << ((f >> 56) & 0xFF)
}
