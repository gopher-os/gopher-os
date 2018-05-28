package vmm

import (
	"gopheros/kernel"
	"gopheros/kernel/mm"
	"runtime"
	"testing"
	"unsafe"
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
		mm.SetFrameAllocator(nil)
	}(ptePtrFn, nextAddrFn, flushTLBEntryFn)

	var physPages [pageLevels][mm.PageSize >> mm.PointerShift]pageTableEntry
	nextPhysPage := 0

	// allocFn returns pages from index 1; we keep index 0 for the P4 entry
	mm.SetFrameAllocator(func() (mm.Frame, *kernel.Error) {
		nextPhysPage++
		pageAddr := unsafe.Pointer(&physPages[nextPhysPage][0])
		return mm.Frame(uintptr(pageAddr) >> mm.PageShift), nil
	})

	pteCallCount := 0
	ptePtrFn = func(entry uintptr) unsafe.Pointer {
		pteCallCount++
		// The last 12 bits encode the page table offset in bytes
		// which we need to convert to a uint64 entry
		pteIndex := (entry & uintptr(mm.PageSize-1)) >> mm.PointerShift
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
	frame := mm.Frame(123)
	levelIndices := []uint{510, 511, 511, 511}

	page, err := MapTemporary(frame)
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
			if exp, got := mm.Frame(uintptr(unsafe.Pointer(&physPages[level+1][0]))>>mm.PageShift), pte.Frame(); got != exp {
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

func TestMapRegion(t *testing.T) {
	defer func() {
		mapFn = Map
		earlyReserveRegionFn = EarlyReserveRegion
	}()

	t.Run("success", func(t *testing.T) {
		mapCallCount := 0
		mapFn = func(_ mm.Page, _ mm.Frame, flags PageTableEntryFlag) *kernel.Error {
			mapCallCount++
			return nil
		}

		earlyReserveRegionCallCount := 0
		earlyReserveRegionFn = func(_ uintptr) (uintptr, *kernel.Error) {
			earlyReserveRegionCallCount++
			return 0xf00, nil
		}

		if _, err := MapRegion(mm.Frame(0xdf0000), 4097, FlagPresent|FlagRW); err != nil {
			t.Fatal(err)
		}

		if exp := 2; mapCallCount != exp {
			t.Errorf("expected Map to be called %d time(s); got %d", exp, mapCallCount)
		}

		if exp := 1; earlyReserveRegionCallCount != exp {
			t.Errorf("expected EarlyReserveRegion to be called %d time(s); got %d", exp, earlyReserveRegionCallCount)
		}
	})

	t.Run("EarlyReserveRegion fails", func(t *testing.T) {
		expErr := &kernel.Error{Module: "test", Message: "out of address space"}

		earlyReserveRegionFn = func(_ uintptr) (uintptr, *kernel.Error) {
			return 0, expErr
		}

		if _, err := MapRegion(mm.Frame(0xdf0000), 128000, FlagPresent|FlagRW); err != expErr {
			t.Fatalf("expected error: %v; got %v", expErr, err)
		}
	})

	t.Run("Map fails", func(t *testing.T) {
		expErr := &kernel.Error{Module: "test", Message: "map failed"}

		earlyReserveRegionCallCount := 0
		earlyReserveRegionFn = func(_ uintptr) (uintptr, *kernel.Error) {
			earlyReserveRegionCallCount++
			return 0xf00, nil
		}

		mapFn = func(_ mm.Page, _ mm.Frame, flags PageTableEntryFlag) *kernel.Error {
			return expErr
		}

		if _, err := MapRegion(mm.Frame(0xdf0000), 128000, FlagPresent|FlagRW); err != expErr {
			t.Fatalf("expected error: %v; got %v", expErr, err)
		}

		if exp := 1; earlyReserveRegionCallCount != exp {
			t.Errorf("expected EarlyReserveRegion to be called %d time(s); got %d", exp, earlyReserveRegionCallCount)
		}
	})
}

func TestIdentityMapRegion(t *testing.T) {
	defer func() {
		mapFn = Map
	}()

	t.Run("success", func(t *testing.T) {
		mapCallCount := 0
		mapFn = func(_ mm.Page, _ mm.Frame, flags PageTableEntryFlag) *kernel.Error {
			mapCallCount++
			return nil
		}

		if _, err := IdentityMapRegion(mm.Frame(0xdf0000), 4097, FlagPresent|FlagRW); err != nil {
			t.Fatal(err)
		}

		if exp := 2; mapCallCount != exp {
			t.Errorf("expected Map to be called %d time(s); got %d", exp, mapCallCount)
		}
	})

	t.Run("Map fails", func(t *testing.T) {
		expErr := &kernel.Error{Module: "test", Message: "map failed"}

		mapFn = func(_ mm.Page, _ mm.Frame, flags PageTableEntryFlag) *kernel.Error {
			return expErr
		}

		if _, err := IdentityMapRegion(mm.Frame(0xdf0000), 128000, FlagPresent|FlagRW); err != expErr {
			t.Fatalf("expected error: %v; got %v", expErr, err)
		}
	})
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

	var physPages [pageLevels][mm.PageSize >> mm.PointerShift]pageTableEntry

	// The reserved virt address uses the following page level indices: 510, 511, 511, 511
	p4Index := 510
	frame := mm.Frame(123)

	t.Run("encounter huge page", func(t *testing.T) {
		physPages[0][p4Index].SetFlags(FlagPresent | FlagHugePage)

		ptePtrFn = func(entry uintptr) unsafe.Pointer {
			// The last 12 bits encode the page table offset in bytes
			// which we need to convert to a uint64 entry
			pteIndex := (entry & uintptr(mm.PageSize-1)) >> mm.PointerShift
			return unsafe.Pointer(&physPages[0][pteIndex])
		}

		if _, err := MapTemporary(frame); err != errNoHugePageSupport {
			t.Fatalf("expected to get errNoHugePageSupport; got %v", err)
		}
	})

	t.Run("allocFn returns an error", func(t *testing.T) {
		defer func() { mm.SetFrameAllocator(nil) }()
		physPages[0][p4Index] = 0

		expErr := &kernel.Error{Module: "test", Message: "out of mmory"}

		mm.SetFrameAllocator(func() (mm.Frame, *kernel.Error) {
			return 0, expErr
		})

		if _, err := MapTemporary(frame); err != expErr {
			t.Fatalf("got unexpected error %v", err)
		}
	})

	t.Run("map BlankReservedFrame RW", func(t *testing.T) {
		defer func() { protectReservedZeroedPage = false }()

		protectReservedZeroedPage = true
		if err := Map(mm.Page(0), ReservedZeroedFrame, FlagRW); err != errAttemptToRWMapReservedFrame {
			t.Fatalf("expected errAttemptToRWMapReservedFrame; got: %v", err)
		}
	})

	t.Run("temp-map BlankReservedFrame RW", func(t *testing.T) {
		defer func() { protectReservedZeroedPage = false }()

		protectReservedZeroedPage = true
		if _, err := MapTemporary(ReservedZeroedFrame); err != errAttemptToRWMapReservedFrame {
			t.Fatalf("expected errAttemptToRWMapReservedFrame; got: %v", err)
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
		physPages [pageLevels][mm.PageSize >> mm.PointerShift]pageTableEntry
		frame     = mm.Frame(123)
	)

	// Emulate a page mapped to virtAddr 0 across all page levels
	for level := 0; level < pageLevels; level++ {
		physPages[level][0].SetFlags(FlagPresent | FlagRW)
		if level < pageLevels-1 {
			physPages[level][0].SetFrame(mm.Frame(uintptr(unsafe.Pointer(&physPages[level+1][0])) >> mm.PageShift))
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

	if err := Unmap(mm.PageFromAddress(0)); err != nil {
		t.Fatal(err)
	}

	for level, physPage := range physPages {
		pte := physPage[0]

		switch {
		case level < pageLevels-1:
			if !pte.HasFlags(FlagPresent) {
				t.Errorf("[pte at level %d] expected entry to retain have FlagPresent set", level)
			}
			if exp, got := mm.Frame(uintptr(unsafe.Pointer(&physPages[level+1][0]))>>mm.PageShift), pte.Frame(); got != exp {
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

	var physPages [pageLevels][mm.PageSize >> mm.PointerShift]pageTableEntry

	t.Run("encounter huge page", func(t *testing.T) {
		physPages[0][0].SetFlags(FlagPresent | FlagHugePage)

		ptePtrFn = func(entry uintptr) unsafe.Pointer {
			// The last 12 bits encode the page table offset in bytes
			// which we need to convert to a uint64 entry
			pteIndex := (entry & uintptr(mm.PageSize-1)) >> mm.PointerShift
			return unsafe.Pointer(&physPages[0][pteIndex])
		}

		if err := Unmap(mm.PageFromAddress(0)); err != errNoHugePageSupport {
			t.Fatalf("expected to get errNoHugePageSupport; got %v", err)
		}
	})

	t.Run("virtual address not mapped", func(t *testing.T) {
		physPages[0][0].ClearFlags(FlagPresent)

		if err := Unmap(mm.PageFromAddress(0)); err != ErrInvalidMapping {
			t.Fatalf("expected to get ErrInvalidMapping; got %v", err)
		}
	})
}

func TestTranslateAmd64(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("test requires amd64 runtime; skipping")
	}

	defer func(origPtePtr func(uintptr) unsafe.Pointer) {
		ptePtrFn = origPtePtr
	}(ptePtrFn)

	// the virtual address just contains the page offset
	virtAddr := uintptr(1234)
	expFrame := mm.Frame(42)
	expPhysAddr := expFrame.Address() + virtAddr
	specs := [][pageLevels]bool{
		{true, true, true, true},
		{false, true, true, true},
		{true, false, true, true},
		{true, true, false, true},
		{true, true, true, false},
	}

	for specIndex, spec := range specs {
		pteCallCount := 0
		ptePtrFn = func(entry uintptr) unsafe.Pointer {
			var pte pageTableEntry
			pte.SetFrame(expFrame)
			if specs[specIndex][pteCallCount] {
				pte.SetFlags(FlagPresent)
			}
			pteCallCount++

			return unsafe.Pointer(&pte)
		}

		// An error is expected if any page level contains a non-present page
		expError := false
		for _, hasMapping := range spec {
			if !hasMapping {
				expError = true
				break
			}
		}

		physAddr, err := Translate(virtAddr)
		switch {
		case expError && err != ErrInvalidMapping:
			t.Errorf("[spec %d] expected to get ErrInvalidMapping; got %v", specIndex, err)
		case !expError && err != nil:
			t.Errorf("[spec %d] unexpected error %v", specIndex, err)
		case !expError && physAddr != expPhysAddr:
			t.Errorf("[spec %d] expected phys addr to be 0x%x; got 0x%x", specIndex, expPhysAddr, physAddr)
		}
	}
}
