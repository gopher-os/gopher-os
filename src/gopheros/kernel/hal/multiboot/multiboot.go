package multiboot

import (
	"reflect"
	"strings"
	"unsafe"
)

var (
	infoData  uintptr
	cmdLineKV map[string]string
)

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

// MemRegionVisitor defies a visitor function that gets invoked by VisitMemRegions
// for each memory region provided by the boot loader. The visitor must return true
// to continue or false to abort the scan.
type MemRegionVisitor func(*MemoryMapEntry) bool

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

type elfSections struct {
	numSections        uint16
	sectionSize        uint32
	strtabSectionIndex uint32
	sectionData        [0]byte
}

/*
type elfSection32 struct {
	nameIndex   uint32
	sectionType uint32
	flags       uint32
	address     uint32
	offset      uint32
	size        uint32
	link        uint32
	info        uint32
	addrAlign   uint32
	entSize     uint32
}
*/

type elfSection64 struct {
	nameIndex   uint32
	sectionType uint32
	flags       uint64
	address     uint64
	offset      uint64
	size        uint64
	link        uint32
	info        uint32
	addrAlign   uint64
	entSize     uint64
}

// ElfSectionFlag defines an OR-able flag associated with an ElfSection.
type ElfSectionFlag uint32

const (
	// ElfSectionWritable marks the section as writable.
	ElfSectionWritable ElfSectionFlag = 1 << iota

	// ElfSectionAllocated means that the section is allocated in memory
	// when the image is loaded (e.g .bss sections)
	ElfSectionAllocated

	// ElfSectionExecutable marks the section as executable.
	ElfSectionExecutable
)

// ElfSectionVisitor defies a visitor function that gets invoked by VisitElfSections
// for rach ELF section that belongs to the loaded kernel image.
type ElfSectionVisitor func(name string, flags ElfSectionFlag, address uintptr, size uint64)

// VisitElfSections invokes visitor for each ELF entry that belongs to the
// loaded kernel image.
func VisitElfSections(visitor ElfSectionVisitor) {
	curPtr, size := findTagByType(tagElfSymbols)
	if size == 0 {
		return
	}

	var (
		sectionPayload  elfSection64
		ptrElfSections  = (*elfSections)(unsafe.Pointer(curPtr))
		secPtr          = uintptr(unsafe.Pointer(&ptrElfSections.sectionData))
		sizeofSection   = unsafe.Sizeof(sectionPayload)
		strTableSection = (*elfSection64)(unsafe.Pointer(secPtr + uintptr(ptrElfSections.strtabSectionIndex)*sizeofSection))
		secName         string
		secNameHeader   = (*reflect.StringHeader)(unsafe.Pointer(&secName))
	)

	for secIndex := uint16(0); secIndex < ptrElfSections.numSections; secIndex, secPtr = secIndex+1, secPtr+sizeofSection {
		secData := (*elfSection64)(unsafe.Pointer(secPtr))
		if secData.size == 0 {
			continue
		}

		// String table entries are C-style NULL-terminated strings
		end := uintptr(secData.nameIndex)
		for ; *(*byte)(unsafe.Pointer(uintptr(strTableSection.address) + end)) != 0; end++ {
		}

		secNameHeader.Len = int(end - uintptr(secData.nameIndex))
		secNameHeader.Data = uintptr(unsafe.Pointer(uintptr(strTableSection.address) + uintptr(secData.nameIndex)))

		visitor(secName, ElfSectionFlag(secData.flags), uintptr(secData.address), secData.size)
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

// GetBootCmdLine returns the command line key-value pairs passed to the
// kernel.  This function must only be invoked after bootstrapping the memory
// allocator.
func GetBootCmdLine() map[string]string {
	if cmdLineKV != nil {
		return cmdLineKV
	}

	cmdLineKV = make(map[string]string)

	curPtr, size := findTagByType(tagBootCmdLine)
	if size != 0 {
		// The command line is a C-style NULL-terminated string
		cmdLine := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
			Len:  int(size - 1),
			Cap:  int(size - 1),
			Data: curPtr,
		}))
		pairs := strings.Fields(string(cmdLine))
		for _, pair := range pairs {
			kv := strings.Split(pair, "=")
			switch len(kv) {
			case 2: // foo=bar
				cmdLineKV[kv[0]] = kv[1]
			case 1: // nofoo
				cmdLineKV[kv[0]] = kv[0]
			}
		}
	}

	return cmdLineKV
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
