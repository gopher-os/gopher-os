package mm

import (
	"gopheros/kernel"
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
	return uintptr(f << PageShift)
}

// FrameFromAddress returns a Frame that corresponds to
// the given physical address. This function can handle
// both page-aligned and not aligned addresses. in the
// latter case, the input address will be rounded down
// to the frame that contains it.
func FrameFromAddress(physAddr uintptr) Frame {
	return Frame((physAddr & ^(uintptr(PageSize - 1))) >> PageShift)
}

var (
	// frameAllocator points to a frame allocator function registered using
	// SetFrameAllocator.
	frameAllocator FrameAllocatorFn
)

// FrameAllocatorFn is a function that can allocate physical frames.
type FrameAllocatorFn func() (Frame, *kernel.Error)

// SetFrameAllocator registers a frame allocator function that will be used by
// the vmm code when new physical frames need to be allocated.
func SetFrameAllocator(allocFn FrameAllocatorFn) { frameAllocator = allocFn }

// AllocFrame allocates a new physical frame using the currently active
// physical frame allocator.
func AllocFrame() (Frame, *kernel.Error) { return frameAllocator() }

// Page describes a virtual memory page index.
type Page uintptr

// Address returns a pointer to the virtual memory address pointed to by this Page.
func (f Page) Address() uintptr {
	return uintptr(f << PageShift)
}

// PageFromAddress returns a Page that corresponds to the given virtual
// address. This function can handle both page-aligned and not aligned virtual
// addresses. in the latter case, the input address will be rounded down to the
// page that contains it.
func PageFromAddress(virtAddr uintptr) Page {
	return Page((virtAddr & ^(uintptr(PageSize - 1))) >> PageShift)
}
