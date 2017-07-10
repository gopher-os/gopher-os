package multiboot

import "unsafe"

type tagType uint32

// nolint
const (
	tagMbSectionEnd tagType = iota
	tagBootCmdLine
	tagBootLoaderName
	tagModules
	tagBasicMemoryInfo
	tagBiosBootDevice
	tagMemoryMap
	tagVbeInfo
	tagFramebufferInfo
	tagElfSymbols
	tagApmTable
)

// info describes the multiboot info section header.
type info struct {
	// Total size of multiboot info section.
	totalSize uint32

	// Always set to zero; reserved for future use
	reserved uint32
}

// tagHeader describes the header the preceedes each tag.
type tagHeader struct {
	// The type of the tag
	tagType tagType

	// The size of the tag including the header but *not* including any
	// padding. According to the spec, each tag starts at a 8-byte aligned
	// address.
	size uint32
}

// mmapHeader describes the header for a memory map specification.
type mmapHeader struct {
	// The size of each entry.
	entrySize uint32

	// The version of the entries that follow.
	entryVersion uint32
}

// FramebufferType defines the type of the initialized framebuffer.
type FramebufferType uint8

const (
	// FramebufferTypeIndexed specifies a 256-color palette.
	FramebufferTypeIndexed FramebufferType = iota

	// FramebufferTypeRGB specifies direct RGB mode.
	FramebufferTypeRGB

	// FramebufferTypeEGA specifies EGA text mode.
	FramebufferTypeEGA
)

// FramebufferInfo provides information about the initialized framebuffer.
type FramebufferInfo struct {
	// The framebuffer physical address.
	PhysAddr uint64

	// Row pitch in bytes.
	Pitch uint32

	// Width and height in pixels (or characters if Type = FramebufferTypeEGA)
	Width, Height uint32

	// Bits per pixel (non EGA modes only).
	Bpp uint8

	// Framebuffer type.
	Type FramebufferType

	reserved uint16

	// The colorInfo data begins after the reserved block and has different
	// contents depending on the framebuffer type. This dummy field is used
	// for obtaining a pointer to the color info block data.
	colorInfo [0]byte
}

// RGBColorInfo returns the FramebufferRGBColorInfo for a RGB framebuffer.
func (i *FramebufferInfo) RGBColorInfo() *FramebufferRGBColorInfo {
	if i.Type != FramebufferTypeRGB {
		return nil
	}

	// The color info data begins after the reserved attribute. To access
	// it, a pointer is created to the dummy colorInfo attribute which
	// points to the color info data start.
	return (*FramebufferRGBColorInfo)(unsafe.Pointer(&i.colorInfo))
}

// FramebufferRGBColorInfo describes the order and width of each color component
// for a 15-, 16-, 24- or 32-bit framebuffer.
type FramebufferRGBColorInfo struct {
	// The position and width (in bits) of the red component.
	RedPosition uint8
	RedMaskSize uint8

	// The position and width (in bits) of the green component.
	GreenPosition uint8
	GreenMaskSize uint8

	// The position and width (in bits) of the blue component.
	BluePosition uint8
	BlueMaskSize uint8
}

// MemoryEntryType defines the type of a MemoryMapEntry.
type MemoryEntryType uint32

const (
	// MemAvailable indicates that the memory region is available for use.
	MemAvailable MemoryEntryType = iota + 1

	// MemReserved indicates that the memory region is not available for use.
	MemReserved

	// MemAcpiReclaimable indicates a memory region that holds ACPI info that
	// can be reused by the OS.
	MemAcpiReclaimable

	// MemNvs indicates memory that must be preserved when hibernating.
	MemNvs

	// Any value >= memUnknown will be mapped to MemReserved.
	memUnknown
)

var (
	infoData uintptr
)

// MemRegionVisitor defies a visitor function that gets invoked by VisitMemRegions
// for each memory region provided by the boot loader. The visitor must return true
// to continue or false to abort the scan.
type MemRegionVisitor func(entry *MemoryMapEntry) bool

// MemoryMapEntry describes a memory region entry, namely its physical address,
// its length and its type.
type MemoryMapEntry struct {
	// The physical address for this memory region.
	PhysAddress uint64

	// The length of the memory region.
	Length uint64

	// The type of this entry.
	Type MemoryEntryType
}

// String implements fmt.Stringer for MemoryEntryType.
func (t MemoryEntryType) String() string {
	switch t {
	case MemAvailable:
		return "available"
	case MemReserved:
		return "reserved"
	case MemAcpiReclaimable:
		return "ACPI (reclaimable)"
	case MemNvs:
		return "NVS"
	default:
		return "unknown"
	}
}

// SetInfoPtr updates the internal multiboot information pointer to the given
// value. This function must be invoked before invoking any other function
// exported by this package.
func SetInfoPtr(ptr uintptr) {
	infoData = ptr
}

// VisitMemRegions will invoke the supplied visitor for each memory region that
// is defined by the multiboot info data that we received from the bootloader.
func VisitMemRegions(visitor MemRegionVisitor) {
	curPtr, size := findTagByType(tagMemoryMap)
	if size == 0 {
		return
	}

	// curPtr points to the memory map header (2 dwords long)
	ptrMapHeader := (*mmapHeader)(unsafe.Pointer(curPtr))
	endPtr := curPtr + uintptr(size)
	curPtr += 8

	var entry *MemoryMapEntry
	for curPtr != endPtr {
		entry = (*MemoryMapEntry)(unsafe.Pointer(curPtr))

		// Mark unknown entry types as reserved
		if entry.Type == 0 || entry.Type > memUnknown {
			entry.Type = MemReserved
		}

		if !visitor(entry) {
			return
		}

		curPtr += uintptr(ptrMapHeader.entrySize)
	}
}

// GetFramebufferInfo returns information about the framebuffer initialized by the
// bootloader. This function returns nil if no framebuffer info is available.
func GetFramebufferInfo() *FramebufferInfo {
	var info *FramebufferInfo

	curPtr, size := findTagByType(tagFramebufferInfo)
	if size != 0 {
		info = (*FramebufferInfo)(unsafe.Pointer(curPtr))
	}

	return info
}

// findTagByType scans the multiboot info data looking for the start of of the
// specified type. It returns a pointer to the tag contents start offset and
// the content length exluding the tag header.
//
// If the tag is not present in the multiboot info, findTagSection will return
// back (0,0).
func findTagByType(tagType tagType) (uintptr, uint32) {
	var ptrTagHeader *tagHeader

	curPtr := infoData + 8
	for ptrTagHeader = (*tagHeader)(unsafe.Pointer(curPtr)); ptrTagHeader.tagType != tagMbSectionEnd; ptrTagHeader = (*tagHeader)(unsafe.Pointer(curPtr)) {
		if ptrTagHeader.tagType == tagType {
			return curPtr + 8, ptrTagHeader.size - 8
		}

		// Tags are aligned at 8-byte aligned addresses
		curPtr += uintptr(int32(ptrTagHeader.size+7) & ^7)
	}

	return 0, 0
}
