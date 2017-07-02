package vmm

import (
	"gopheros/kernel/mem"
	"unsafe"
)

var (
	// ptePointerFn returns a pointer to the supplied entry address. It is
	// used by tests to override the generated page table entry pointers so
	// walk() can be properly tested. When compiling the kernel this function
	// will be automatically inlined.
	ptePtrFn = func(entryAddr uintptr) unsafe.Pointer {
		return unsafe.Pointer(entryAddr)
	}
)

// pageTableWalker is a function that can be passed to the walk method. The
// function receives the current page level and page table entry as its
// arguments.  If the function returns false, then the page walk is aborted.
type pageTableWalker func(pteLevel uint8, pte *pageTableEntry) bool

// walk performs a page table walk for the given virtual address. It calls the
// suppplied walkFn with the page table entry that corresponds to each page
// table level. If walkFn returns an error then the walk is aborted and the
// error is returned to the caller.
func walk(virtAddr uintptr, walkFn pageTableWalker) {
	var (
		level                            uint8
		tableAddr, entryAddr, entryIndex uintptr
		ok                               bool
	)

	// tableAddr is initially set to the recursively mapped virtual address for the
	// last entry in the top-most page table. Dereferencing a pointer to this address
	// will allow us to access
	for level, tableAddr = uint8(0), pdtVirtualAddr; level < pageLevels; level, tableAddr = level+1, entryAddr {
		// Extract the bits from virtual address that correspond to the
		// index in this level's page table
		entryIndex = (virtAddr >> pageLevelShifts[level]) & ((1 << pageLevelBits[level]) - 1)

		// By shifting the table virtual address left by pageLevelShifts[level] we add
		// a new level of indirection to our recursive mapping allowing us to access
		// the table pointed to by the page entry
		entryAddr = tableAddr + (entryIndex << mem.PointerShift)

		if ok = walkFn(level, (*pageTableEntry)(ptePtrFn(entryAddr))); !ok {
			return
		}

		// Shift left by the number of bits for this paging level to get
		// the virtual address of the table pointed to by entryAddr
		entryAddr <<= pageLevelBits[level]
	}
}
