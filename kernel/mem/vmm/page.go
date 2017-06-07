package vmm

import "github.com/achilleasa/gopher-os/kernel/mem"

// Page describes a virtual memory page index.
type Page uintptr

// Address returns a pointer to the virtual memory address pointed to by this Page.
func (f Page) Address() uintptr {
	return uintptr(f << mem.PageShift)
}

// PageFromAddress returns a Page that corresponds to the given virtual
// address. This function can handle both page-aligned and not aligned virtual
// addresses. in the latter case, the input address will be rounded down to the
// page that contains it.
func PageFromAddress(virtAddr uintptr) Page {
	return Page((virtAddr & ^(uintptr(mem.PageSize - 1))) >> mem.PageShift)
}
