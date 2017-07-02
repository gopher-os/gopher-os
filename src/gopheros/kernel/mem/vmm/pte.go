package vmm

import (
	"gopheros/kernel"
	"gopheros/kernel/mem"
	"gopheros/kernel/mem/pmm"
)

var (
	// ErrInvalidMapping is returned when trying to lookup a virtual memory address that is not yet mapped.
	ErrInvalidMapping = &kernel.Error{Module: "vmm", Message: "virtual address does not point to a mapped physical page"}
)

// PageTableEntryFlag describes a flag that can be applied to a page table entry.
type PageTableEntryFlag uintptr

// pageTableEntry describes a page table entry. These entries encode
// a physical frame address and a set of flags. The actual format
// of the entry and flags is architecture-dependent.
type pageTableEntry uintptr

// HasFlags returns true if this entry has all the input flags set.
func (pte pageTableEntry) HasFlags(flags PageTableEntryFlag) bool {
	return (uintptr(pte) & uintptr(flags)) == uintptr(flags)
}

// HasAnyFlag returns true if this entry has at least one of the input flags set.
func (pte pageTableEntry) HasAnyFlag(flags PageTableEntryFlag) bool {
	return (uintptr(pte) & uintptr(flags)) != 0
}

// SetFlags sets the input list of flags to the page table entry.
func (pte *pageTableEntry) SetFlags(flags PageTableEntryFlag) {
	*pte = (pageTableEntry)(uintptr(*pte) | uintptr(flags))
}

// ClearFlags unsets the input list of flags from the page table entry.
func (pte *pageTableEntry) ClearFlags(flags PageTableEntryFlag) {
	*pte = (pageTableEntry)(uintptr(*pte) &^ uintptr(flags))
}

// Frame returns the physical page frame that this page table entry points to.
func (pte pageTableEntry) Frame() pmm.Frame {
	return pmm.Frame((uintptr(pte) & ptePhysPageMask) >> mem.PageShift)
}

// SetFrame updates the page table entry to point the the given physical frame .
func (pte *pageTableEntry) SetFrame(frame pmm.Frame) {
	*pte = (pageTableEntry)((uintptr(*pte) &^ ptePhysPageMask) | frame.Address())
}

// pteForAddress returns the final page table entry that correspond to a
// particular virtual address. The function performs a page table walk till it
// reaches the final page table entry returning ErrInvalidMapping if the page
// is not present.
func pteForAddress(virtAddr uintptr) (*pageTableEntry, *kernel.Error) {
	var (
		err   *kernel.Error
		entry *pageTableEntry
	)

	walk(virtAddr, func(pteLevel uint8, pte *pageTableEntry) bool {
		if !pte.HasFlags(FlagPresent) {
			entry = nil
			err = ErrInvalidMapping
			return false
		}

		entry = pte
		return true
	})

	return entry, err
}
