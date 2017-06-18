package mem

import (
	"testing"
	"unsafe"

	"github.com/achilleasa/gopher-os/kernel/hal/multiboot"
)

func TestSizeToOrder(t *testing.T) {
	specs := []struct {
		size     Size
		expOrder PageOrder
	}{
		{1 * Kb, PageOrder(0)},
		{PageSize, PageOrder(0)},
		{8 * Kb, PageOrder(1)},
		{2 * Mb, PageOrder(9)},
	}

	for specIndex, spec := range specs {
		if got := spec.size.Order(); got != spec.expOrder {
			t.Errorf("[spec %d] expected to get page order %d; got %d", specIndex, spec.expOrder, got)
		}
	}
}

func TestSizeToPages(t *testing.T) {
	specs := []struct {
		size     Size
		expPages uint32
	}{
		{1023 * Kb, 256},
		{1024 * Kb, 256},
		{1 * Byte, 1},
	}

	for specIndex, spec := range specs {
		if got := spec.size.Pages(); got != spec.expPages {
			t.Errorf("[spec %d] expected Pages(%d bytes) to equal %d; got %d", specIndex, spec.size, spec.expPages, got)
		}
	}
}

func TestAlign(t *testing.T) {
	specs := []struct {
		in     uint64
		n      Size
		expOut uint64
	}{
		{0, 64 * Byte, 0},
		{1, 64 * Byte, 64},
		{63, 64 * Byte, 64},
		{64, 64 * Byte, 64},
		{65, 64 * Byte, 128},
	}

	for specIndex, spec := range specs {
		out := Align(spec.in, spec.n)
		if out != spec.expOut {
			t.Errorf("[spec %d] expected align(%d, %d) to return %d; got %d", specIndex, spec.in, spec.n, spec.expOut, out)
		}
	}
}

func TestMemset(t *testing.T) {
	// memset with a 0 size should be a no-op
	Memset(uintptr(0), 0x00, 0)

	for ord := PageOrder(0); ord <= MaxPageOrder; ord++ {
		buf := make([]byte, PageSize<<ord)
		for i := 0; i < len(buf); i++ {
			buf[i] = 0xFE
		}

		addr := uintptr(unsafe.Pointer(&buf[0]))
		Memset(addr, 0x00, uint32(len(buf)))

		for i := 0; i < len(buf); i++ {
			if got := buf[i]; got != 0x00 {
				t.Errorf("expected ord(%d), byte: %d to be 0x00; got 0x%x", ord, i, got)
			}
		}
	}
}

func TestTotalSystemMemory(t *testing.T) {
	pageList := []multiboot.MemoryMapEntry{
		{Length: 1024},
		{Length: 2049},
		{Length: 31234},
		{Length: 4096},
	}

	var expTotal Size
	for _, p := range pageList {
		expTotal += Size(p.Length)
	}

	orig := visitMemRegionFn
	defer func() {
		visitMemRegionFn = orig
	}()

	visitMemRegionFn = func(visitor multiboot.MemRegionVisitor) {
		for _, p := range pageList {
			visitor(&p)
		}
	}

	if total := TotalSystemMemory(); total != expTotal {
		t.Fatalf("expected returned total memory to be %d; got %d", expTotal, total)
	}
}
