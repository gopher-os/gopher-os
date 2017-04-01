package multiboot

import (
	"testing"
	"unsafe"
)

func TestFindTagByType(t *testing.T) {
	specs := []struct {
		tagType tagType
		expSize uint32
	}{
		{tagBootCmdLine, 1},
		{tagBootLoaderName, 27},
		{tagBasicMemoryInfo, 8},
		{tagBiosBootDevice, 12},
		{tagMemoryMap, 152},
		{tagFramebufferInfo, 24},
		{tagElfSymbols, 972},
		{tagApmTable, 20},
	}

	SetInfoPtr(uintptr(unsafe.Pointer(&multibootInfoTestData[0])))

	for specIndex, spec := range specs {
		_, size := findTagByType(spec.tagType)

		if size != spec.expSize {
			t.Errorf("[spec %d] expected tag size for tag type %d to be %d; got %d", specIndex, spec.tagType, spec.expSize, size)

		}
	}
}

func TestFindTagByTypeWithMissingTag(t *testing.T) {
	SetInfoPtr(uintptr(unsafe.Pointer(&multibootInfoTestData[0])))

	if offset, size := findTagByType(tagModules); offset != 0 || size != 0 {
		t.Fatalf("expected findTagByType to return (0,0) for missing tag; got (%d, %d)", offset, size)
	}
}

func TestVisitMemRegion(t *testing.T) {
	specs := []struct {
		expPhys uint64
		expLen  uint64
		expType MemoryEntryType
	}{
		// This region type is actually MemAvailable but we patch it to
		// a bogus value to test whether it gets flagged as reserved
		{0, 654336, MemReserved},
		{654336, 1024, MemReserved},
		{983040, 65536, MemReserved},
		{1048576, 133038080, MemAvailable},
		{134086656, 131072, MemReserved},
		{4294705152, 262144, MemReserved},
	}

	var visitCount int

	SetInfoPtr(uintptr(unsafe.Pointer(&emptyInfoData[0])))
	VisitMemRegions(func(_ *MemoryMapEntry) {
		visitCount++
	})

	if visitCount != 0 {
		t.Fatal("expected visitor not to be invoked when no memory map tag is present")
	}

	// Set a bogus type for the first entry in the map
	SetInfoPtr(uintptr(unsafe.Pointer(&multibootInfoTestData[0])))
	multibootInfoTestData[128] = 0xFF

	VisitMemRegions(func(entry *MemoryMapEntry) {
		if entry.PhysAddress != specs[visitCount].expPhys {
			t.Errorf("[visit %d] expected physical address to be %x; got %x", visitCount, specs[visitCount].expPhys, entry.PhysAddress)
		}
		if entry.Length != specs[visitCount].expLen {
			t.Errorf("[visit %d] expected region len to be %x; got %x", visitCount, specs[visitCount].expLen, entry.Length)
		}
		if entry.Type != specs[visitCount].expType {
			t.Errorf("[visit %d] expected region type to be %d; got %d", visitCount, specs[visitCount].expType, entry.Type)
		}
		visitCount++
	})

	if visitCount != len(specs) {
		t.Errorf("expected the visitor func to be invoked %d times; got %d", len(specs), visitCount)
	}
}

func TestGetFramebufferInfo(t *testing.T) {
	SetInfoPtr(uintptr(unsafe.Pointer(&emptyInfoData[0])))

	if GetFramebufferInfo() != nil {
		t.Fatalf("expected GetFramebufferInfo() to return nil when no framebuffer tag is present")
	}

	SetInfoPtr(uintptr(unsafe.Pointer(&multibootInfoTestData[0])))
	fbInfo := GetFramebufferInfo()

	if fbInfo.Type != FramebufferTypeEGA {
		t.Errorf("expected framebuffer type to be %d; got %d", FramebufferTypeEGA, fbInfo.Type)
	}

	if fbInfo.PhysAddr != 0xB8000 {
		t.Errorf("expected physical address for EGA text mode to be 0xB8000; got %x", fbInfo.PhysAddr)
	}

	if fbInfo.Width != 80 || fbInfo.Height != 25 {
		t.Errorf("expected framebuffer dimensions to be 80x25; got %dx%d", fbInfo.Width, fbInfo.Height)
	}

	if fbInfo.Pitch != 160 {
		t.Errorf("expected pitch to be 160; got %x", fbInfo.Pitch)
	}
}

var (
	emptyInfoData = []byte{
		0, 0, 0, 0, // size
		0, 0, 0, 0, // reserved
		0, 0, 0, 0, // tag with type zero and length zero
		0, 0, 0, 0,
	}

	// A dump of multiboot data when running under qemu.
	multibootInfoTestData = []byte{
		72, 5, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 9, 0, 0, 0,
		0, 171, 253, 7, 118, 119, 123, 0, 2, 0, 0, 0, 35, 0, 0, 0,
		71, 82, 85, 66, 32, 50, 46, 48, 50, 126, 98, 101, 116, 97, 50, 45,
		57, 117, 98, 117, 110, 116, 117, 49, 46, 54, 0, 0, 0, 0, 0, 0,
		10, 0, 0, 0, 28, 0, 0, 0, 2, 1, 0, 240, 4, 213, 0, 0,
		0, 240, 0, 240, 3, 0, 240, 255, 240, 255, 240, 255, 0, 0, 0, 0,
		6, 0, 0, 0, 160, 0, 0, 0, 24, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 252, 9, 0, 0, 0, 0, 0,
		1, 0, 0, 0, 0, 0, 0, 0, 0, 252, 9, 0, 0, 0, 0, 0,
		0, 4, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 15, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0,
		2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 16, 0, 0, 0, 0, 0,
		0, 0, 238, 7, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 254, 7, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0,
		2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 252, 255, 0, 0, 0, 0,
		0, 0, 4, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0,
		9, 0, 0, 0, 212, 3, 0, 0, 24, 0, 0, 0, 40, 0, 0, 0,
		21, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 27, 0, 0, 0,
		1, 0, 0, 0, 2, 0, 0, 0, 0, 0, 16, 0, 0, 16, 0, 0,
		24, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 8, 0, 0, 0,
		0, 0, 0, 0, 38, 0, 0, 0, 1, 0, 0, 0, 6, 0, 0, 0,
		0, 16, 16, 0, 0, 32, 0, 0, 135, 26, 4, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 16, 0, 0, 0, 0, 0, 0, 44, 0, 0, 0,
		1, 0, 0, 0, 2, 0, 0, 0, 0, 48, 20, 0, 0, 64, 4, 0,
		194, 167, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 16, 0, 0,
		0, 0, 0, 0, 52, 0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0,
		224, 215, 21, 0, 224, 231, 5, 0, 176, 6, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 32, 0, 0, 0, 0, 0, 0, 0, 62, 0, 0, 0,
		1, 0, 0, 0, 2, 0, 0, 0, 144, 222, 21, 0, 144, 238, 5, 0,
		4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0,
		0, 0, 0, 0, 72, 0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0,
		160, 222, 21, 0, 160, 238, 5, 0, 119, 23, 2, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 32, 0, 0, 0, 0, 0, 0, 0, 83, 0, 0, 0,
		7, 0, 0, 0, 2, 0, 0, 0, 32, 246, 23, 0, 32, 6, 8, 0,
		56, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 32, 0, 0, 0,
		0, 0, 0, 0, 100, 0, 0, 0, 1, 0, 0, 0, 3, 0, 0, 0,
		0, 0, 24, 0, 0, 16, 8, 0, 204, 5, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 16, 0, 0, 0, 0, 0, 0, 106, 0, 0, 0,
		1, 0, 0, 0, 3, 0, 0, 0, 224, 5, 24, 0, 224, 21, 8, 0,
		178, 9, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 32, 0, 0, 0,
		0, 0, 0, 0, 117, 0, 0, 0, 8, 0, 0, 0, 3, 4, 0, 0,
		148, 15, 24, 0, 146, 31, 8, 0, 4, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0, 123, 0, 0, 0,
		8, 0, 0, 0, 3, 0, 0, 0, 0, 16, 24, 0, 146, 31, 8, 0,
		176, 61, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 16, 0, 0,
		0, 0, 0, 0, 128, 0, 0, 0, 8, 0, 0, 0, 3, 0, 0, 0,
		192, 77, 25, 0, 146, 31, 8, 0, 32, 56, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 32, 0, 0, 0, 0, 0, 0, 0, 138, 0, 0, 0,
		1, 0, 0, 0, 0, 0, 0, 0, 224, 133, 25, 0, 146, 31, 8, 0,
		64, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0,
		0, 0, 0, 0, 153, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0,
		32, 134, 25, 0, 210, 31, 8, 0, 129, 26, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 169, 0, 0, 0,
		1, 0, 0, 0, 0, 0, 0, 0, 161, 160, 25, 0, 83, 58, 8, 0,
		2, 201, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0,
		0, 0, 0, 0, 181, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0,
		163, 105, 27, 0, 85, 3, 10, 0, 25, 1, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 195, 0, 0, 0,
		1, 0, 0, 0, 0, 0, 0, 0, 188, 106, 27, 0, 110, 4, 10, 0,
		67, 153, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0,
		0, 0, 0, 0, 207, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0,
		0, 4, 28, 0, 184, 157, 10, 0, 252, 112, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 0, 220, 0, 0, 0,
		1, 0, 0, 0, 0, 0, 0, 0, 252, 116, 28, 0, 180, 14, 11, 0,
		16, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0,
		0, 0, 0, 0, 231, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0,
		12, 117, 28, 0, 196, 14, 11, 0, 239, 79, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 17, 0, 0, 0,
		3, 0, 0, 0, 0, 0, 0, 0, 251, 196, 28, 0, 179, 94, 11, 0,
		247, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0,
		0, 0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0,
		244, 197, 28, 0, 108, 99, 11, 0, 80, 77, 0, 0, 23, 0, 0, 0,
		210, 4, 0, 0, 4, 0, 0, 0, 16, 0, 0, 0, 9, 0, 0, 0,
		3, 0, 0, 0, 0, 0, 0, 0, 68, 19, 29, 0, 188, 176, 11, 0,
		107, 104, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 16, 0, 0, 0,
		127, 2, 0, 0, 128, 251, 1, 0, 5, 0, 0, 0, 20, 0, 0, 0,
		224, 0, 0, 0, 255, 255, 255, 255, 255, 255, 255, 255, 0, 0, 0, 0,
		8, 0, 0, 0, 32, 0, 0, 0, 0, 128, 11, 0, 0, 0, 0, 0,
		160, 0, 0, 0, 80, 0, 0, 0, 25, 0, 0, 0, 16, 2, 0, 0,
		14, 0, 0, 0, 28, 0, 0, 0, 82, 83, 68, 32, 80, 84, 82, 32,
		89, 66, 79, 67, 72, 83, 32, 0, 220, 24, 254, 7, 0, 0, 0, 0,
		0, 0, 0, 0, 8, 0, 0, 0,
	}
)
