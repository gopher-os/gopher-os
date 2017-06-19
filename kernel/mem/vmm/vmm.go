package vmm

import (
	"github.com/achilleasa/gopher-os/kernel"
	"github.com/achilleasa/gopher-os/kernel/mem/pmm"
)

var frameAllocator FrameAllocatorFn

// FrameAllocatorFn is a function that can allocate physical frames.
type FrameAllocatorFn func() (pmm.Frame, *kernel.Error)

// SetFrameAllocator registers a frame allocator function that will be used by
// the vmm code when new physical frames need to be allocated.
func SetFrameAllocator(allocFn FrameAllocatorFn) {
	frameAllocator = allocFn
}
