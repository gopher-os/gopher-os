package vmm

import (
	"runtime"
	"testing"
	"unsafe"

	"github.com/achilleasa/gopher-os/kernel"
	"github.com/achilleasa/gopher-os/kernel/mem"
	"github.com/achilleasa/gopher-os/kernel/mem/pmm"
)

func TestNextAddrFn(t *testing.T) {
	// Dummy test to keep coverage happy
	if exp, got := uintptr(123), nextAddrFn(uintptr(123)); exp != got {
		t.Fatalf("expected nextAddrFn to return %v; got %v", exp, got)
	}
}

func TestMapTemporaryAmd64(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("test requires amd64 runtime; skipping")
	}

	defer func(origPtePtr func(uintptr) unsafe.Pointer, origNextAddrFn func(uintptr) uintptr, origFlushTLBEntryFn func(uintptr)) {
		ptePtrFn = origPtePtr
		nextAddrFn = origNextAddrFn
		flushTLBEntryFn = origFlushTLBEntryFn
	}(ptePtrFn, nextAddrFn, flushTLBEntryFn)

	var physPages [pageLevels][mem.PageSize >> mem.PointerShift]pageTableEntry
	nextPhysPage := 0

	// allocFn returns pages from index 1; we keep index 0 for the P4 entry
	allocFn := func() (pmm.Frame, *kernel.Error) {
		nextPhysPage++
		pageAddr := unsafe.Pointer(&physPages[nextPhysPage][0])
		return pmm.Frame(uintptr(pageAddr) >> mem.PageShift), nil
	}

	pteCallCount := 0
	ptePtrFn = func(entry uintptr) unsafe.Pointer {
		pteCallCount++
		// The last 12 bits encode the page table offset in bytes
		// which we need to convert to a uint64 entry
		pteIndex := (entry & uintptr(mem.PageSize-1)) >> mem.PointerShift
		return unsafe.Pointer(&physPages[pteCallCount-1][pteIndex])
	}

	nextAddrFn = func(entry uintptr) uintptr {
		return uintptr(unsafe.Pointer(&physPages[nextPhysPage][0]))
	}

	flushTLBEntryCallCount := 0
	flushTLBEntryFn = func(uintptr) {
		flushTLBEntryCallCount++
	}

	// The temporary mappin address breaks down to:
	// p4 index: 510
	// p3 index: 511
	// p2 index: 511
	// p1 index: 511
	frame := pmm.Frame(123)
	levelIndices := []uint{510, 511, 511, 511}

	page, err := MapTemporary(frame, allocFn)
	if err != nil {
		t.Fatal(err)
	}

	if got := page.Address(); got != tempMappingAddr {
		t.Fatalf("expected temp mapping virtual address to be %x; got %x", tempMappingAddr, got)
	}

	for level, physPage := range physPages {
		pte := physPage[levelIndices[level]]
		if !pte.HasFlags(FlagPresent | FlagRW) {
			t.Errorf("[pte at level %d] expected entry to have FlagPresent and FlagRW set", level)
		}

		switch {
		case level < pageLevels-1:
			if exp, got := pmm.Frame(uintptr(unsafe.Pointer(&physPages[level+1][0]))>>mem.PageShift), pte.Frame(); got != exp {
				t.Errorf("[pte at level %d] expected entry frame to be %d; got %d", level, exp, got)
			}
		default:
			// The last pte entry should point to frame
			if got := pte.Frame(); got != frame {
				t.Errorf("[pte at level %d] expected entry frame to be %d; got %d", level, frame, got)
			}
		}
	}

	if exp := 1; flushTLBEntryCallCount != exp {
		t.Errorf("expected flushTLBEntry to be called %d times; got %d", exp, flushTLBEntryCallCount)
	}
}

func TestMapTemporaryErrorsAmd64(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("test requires amd64 runtime; skipping")
	}

	defer func(origPtePtr func(uintptr) unsafe.Pointer, origNextAddrFn func(uintptr) uintptr, origFlushTLBEntryFn func(uintptr)) {
		ptePtrFn = origPtePtr
		nextAddrFn = origNextAddrFn
		flushTLBEntryFn = origFlushTLBEntryFn
	}(ptePtrFn, nextAddrFn, flushTLBEntryFn)

	var physPages [pageLevels][mem.PageSize >> mem.PointerShift]pageTableEntry

	// The reserved virt address uses the following page level indices: 510, 511, 511, 511
	p4Index := 510
	frame := pmm.Frame(123)

	t.Run("encounter huge page", func(t *testing.T) {
		physPages[0][p4Index].SetFlags(FlagPresent | FlagHugePage)

		ptePtrFn = func(entry uintptr) unsafe.Pointer {
			// The last 12 bits encode the page table offset in bytes
			// which we need to convert to a uint64 entry
			pteIndex := (entry & uintptr(mem.PageSize-1)) >> mem.PointerShift
			return unsafe.Pointer(&physPages[0][pteIndex])
		}

		if _, err := MapTemporary(frame, nil); err != errNoHugePageSupport {
			t.Fatalf("expected to get errNoHugePageSupport; got %v", err)
		}
	})

	t.Run("allocFn returns an error", func(t *testing.T) {
		physPages[0][p4Index] = 0

		expErr := &kernel.Error{Module: "test", Message: "out of memory"}

		allocFn := func() (pmm.Frame, *kernel.Error) {
			return 0, expErr
		}

		if _, err := MapTemporary(frame, allocFn); err != expErr {
			t.Fatalf("got unexpected error %v", err)
		}
	})
}

func TestUnmapAmd64(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("test requires amd64 runtime; skipping")
	}

	defer func(origPtePtr func(uintptr) unsafe.Pointer, origFlushTLBEntryFn func(uintptr)) {
		ptePtrFn = origPtePtr
		flushTLBEntryFn = origFlushTLBEntryFn
	}(ptePtrFn, flushTLBEntryFn)

	var (
		physPages [pageLevels][mem.PageSize >> mem.PointerShift]pageTableEntry
		frame     = pmm.Frame(123)
	)

	// Emulate a page mapped to virtAddr 0 across all page levels
	for level := 0; level < pageLevels; level++ {
		physPages[level][0].SetFlags(FlagPresent | FlagRW)
		if level < pageLevels-1 {
			physPages[level][0].SetFrame(pmm.Frame(uintptr(unsafe.Pointer(&physPages[level+1][0])) >> mem.PageShift))
		} else {
			physPages[level][0].SetFrame(frame)

		}
	}

	pteCallCount := 0
	ptePtrFn = func(entry uintptr) unsafe.Pointer {
		pteCallCount++
		return unsafe.Pointer(&physPages[pteCallCount-1][0])
	}

	flushTLBEntryCallCount := 0
	flushTLBEntryFn = func(uintptr) {
		flushTLBEntryCallCount++
	}

	if err := Unmap(PageFromAddress(0)); err != nil {
		t.Fatal(err)
	}

	for level, physPage := range physPages {
		pte := physPage[0]

		switch {
		case level < pageLevels-1:
			if !pte.HasFlags(FlagPresent) {
				t.Errorf("[pte at level %d] expected entry to retain have FlagPresent set", level)
			}
			if exp, got := pmm.Frame(uintptr(unsafe.Pointer(&physPages[level+1][0]))>>mem.PageShift), pte.Frame(); got != exp {
				t.Errorf("[pte at level %d] expected entry frame to still be %d; got %d", level, exp, got)
			}
		default:
			if pte.HasFlags(FlagPresent) {
				t.Errorf("[pte at level %d] expected entry not to have FlagPresent set", level)
			}

			// The last pte entry should still point to frame
			if got := pte.Frame(); got != frame {
				t.Errorf("[pte at level %d] expected entry frame to be %d; got %d", level, frame, got)
			}
		}
	}

	if exp := 1; flushTLBEntryCallCount != exp {
		t.Errorf("expected flushTLBEntry to be called %d times; got %d", exp, flushTLBEntryCallCount)
	}
}

func TestUnmapErrorsAmd64(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("test requires amd64 runtime; skipping")
	}

	defer func(origPtePtr func(uintptr) unsafe.Pointer, origNextAddrFn func(uintptr) uintptr, origFlushTLBEntryFn func(uintptr)) {
		ptePtrFn = origPtePtr
		nextAddrFn = origNextAddrFn
		flushTLBEntryFn = origFlushTLBEntryFn
	}(ptePtrFn, nextAddrFn, flushTLBEntryFn)

	var physPages [pageLevels][mem.PageSize >> mem.PointerShift]pageTableEntry

	t.Run("encounter huge page", func(t *testing.T) {
		physPages[0][0].SetFlags(FlagPresent | FlagHugePage)

		ptePtrFn = func(entry uintptr) unsafe.Pointer {
			// The last 12 bits encode the page table offset in bytes
			// which we need to convert to a uint64 entry
			pteIndex := (entry & uintptr(mem.PageSize-1)) >> mem.PointerShift
			return unsafe.Pointer(&physPages[0][pteIndex])
		}

		if err := Unmap(PageFromAddress(0)); err != errNoHugePageSupport {
			t.Fatalf("expected to get errNoHugePageSupport; got %v", err)
		}
	})

	t.Run("virtual address not mapped", func(t *testing.T) {
		physPages[0][0].ClearFlags(FlagPresent)

		if err := Unmap(PageFromAddress(0)); err != ErrInvalidMapping {
			t.Fatalf("expected to get ErrInvalidMapping; got %v", err)
		}
	})
}
