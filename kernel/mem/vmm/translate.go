package vmm

import "github.com/achilleasa/gopher-os/kernel"

// Translate returns the physical address that corresponds to the supplied
// virtual address or ErrInvalidMapping if the virtual address does not
// correspond to a mapped physical address.
func Translate(virtAddr uintptr) (uintptr, *kernel.Error) {
	pte, err := pteForAddress(virtAddr)
	if err != nil {
		return 0, err
	}

	// Calculate the physical address by taking the physical frame address and
	// appending the offset from the virtual address
	physAddr := pte.Frame().Address() + (virtAddr & ((1 << pageLevelShifts[pageLevels-1]) - 1))

	return physAddr, nil
}
