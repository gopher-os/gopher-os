package vmm

import "math"

const (
	// pageLevels indicates the number of page levels supported by the amd64 architecture.
	pageLevels = 4

	// ptePhysPageMask is a mask that allows us to extract the physical memory
	// address pointed to by a page table entry. For this particular architecture,
	// bits 12-51 contain the physical memory address.
	ptePhysPageMask = uintptr(0x000ffffffffff000)

	// tempMappingAddr is a reserved virtual page address used for
	// temporary physical page mappings (e.g. when mapping inactive PDT
	// pages). For amd64 this address uses the following table indices:
	// 510, 511, 511, 511.
	tempMappingAddr = uintptr(0Xffffff7ffffff000)
)

var (
	// pdtVirtualAddr is a special virtual address that exploits the
	// recursive mapping used in the last PDT entry for each page directory
	// to allow accessing the PDT (P4) table using the system's MMU address
	// translation mechanism.  By setting all page level bits to 1 the MMU
	// keeps following the last P4 entry for all page levels landing on the
	// P4.
	pdtVirtualAddr = uintptr(math.MaxUint64 &^ ((1 << 12) - 1))

	// pageLevelBits defines the number of virtual address bits that correspond to each
	// page level. For the amd64 architecture each PageLevel uses 9 bits which amounts to
	// 512 entries for each page level.
	pageLevelBits = [pageLevels]uint8{
		9,
		9,
		9,
		9,
	}

	// pageLevelShifts defines the shift required to access each page table component
	// of a virtual address.
	pageLevelShifts = [pageLevels]uint8{
		39,
		30,
		21,
		12,
	}
)

const (
	// FlagPresent is set when the page is available in memory and not swapped out.
	FlagPresent PageTableEntryFlag = 1 << iota

	// FlagRW is set if the page can be written to.
	FlagRW

	// FlagUserAccessible is set if user-mode processes can access this page. If
	// not set only kernel code can access this page.
	FlagUserAccessible

	// FlagWriteThroughCaching implies write-through caching when set and write-back
	// caching if cleared.
	FlagWriteThroughCaching

	// FlagDoNotCache prevents this page from being cached if set.
	FlagDoNotCache

	// FlagAccessed is set by the CPU when this page is accessed.
	FlagAccessed

	// FlagDirty is set by the CPU when this page is modified.
	FlagDirty

	// FlagHugePage is set if when using 2Mb pages instead of 4K pages.
	FlagHugePage

	// FlagGlobal if set, prevents the TLB from flushing the cached memory address
	// for this page when the swapping page tables by updating the CR3 register.
	FlagGlobal

	// FlagCopyOnWrite is used to implement copy-on-write functionality. This
	// flag and FlagRW are mutually exclusive.
	FlagCopyOnWrite = 1 << 9

	// FlagNoExecute if set, indicates that a page contains non-executable code.
	FlagNoExecute = 1 << 63
)
