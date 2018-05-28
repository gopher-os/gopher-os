package vmm

import (
	"gopheros/kernel"
	"gopheros/kernel/cpu"
	"gopheros/kernel/mm"
	"gopheros/multiboot"
	"runtime"
	"testing"
	"unsafe"
)

const (
	oneMb = 1024 * 1024
)

func TestPageDirectoryTableInitAmd64(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("test requires amd64 runtime; skipping")
	}

	defer func(origFlushTLBEntry func(uintptr), origActivePDT func() uintptr, origMapTemporary func(mm.Frame) (mm.Page, *kernel.Error), origUnmap func(mm.Page) *kernel.Error) {
		flushTLBEntryFn = origFlushTLBEntry
		activePDTFn = origActivePDT
		mapTemporaryFn = origMapTemporary
		unmapFn = origUnmap
	}(flushTLBEntryFn, activePDTFn, mapTemporaryFn, unmapFn)

	t.Run("already mapped PDT", func(t *testing.T) {
		var (
			pdt      PageDirectoryTable
			pdtFrame = mm.Frame(123)
		)

		activePDTFn = func() uintptr {
			return pdtFrame.Address()
		}

		mapTemporaryFn = func(_ mm.Frame) (mm.Page, *kernel.Error) {
			t.Fatal("unexpected call to MapTemporary")
			return 0, nil
		}

		unmapFn = func(_ mm.Page) *kernel.Error {
			t.Fatal("unexpected call to Unmap")
			return nil
		}

		if err := pdt.Init(pdtFrame); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("not mapped PDT", func(t *testing.T) {
		var (
			pdt      PageDirectoryTable
			pdtFrame = mm.Frame(123)
			physPage [mm.PageSize >> mm.PointerShift]pageTableEntry
		)

		// Fill phys page with random junk
		kernel.Memset(uintptr(unsafe.Pointer(&physPage[0])), 0xf0, mm.PageSize)

		activePDTFn = func() uintptr {
			return 0
		}

		mapTemporaryFn = func(_ mm.Frame) (mm.Page, *kernel.Error) {
			return mm.PageFromAddress(uintptr(unsafe.Pointer(&physPage[0]))), nil
		}

		flushTLBEntryFn = func(_ uintptr) {}

		unmapCallCount := 0
		unmapFn = func(_ mm.Page) *kernel.Error {
			unmapCallCount++
			return nil
		}

		if err := pdt.Init(pdtFrame); err != nil {
			t.Fatal(err)
		}

		if unmapCallCount != 1 {
			t.Fatalf("expected Unmap to be called 1 time; called %d", unmapCallCount)
		}

		for i := 0; i < len(physPage)-1; i++ {
			if physPage[i] != 0 {
				t.Errorf("expected PDT entry %d to be cleared; got %x", i, physPage[i])
			}
		}

		// The last page should be recursively mapped to the PDT
		lastPdtEntry := physPage[len(physPage)-1]
		if !lastPdtEntry.HasFlags(FlagPresent | FlagRW) {
			t.Fatal("expected last PDT entry to have FlagPresent and FlagRW set")
		}

		if lastPdtEntry.Frame() != pdtFrame {
			t.Fatalf("expected last PDT entry to be recursively mapped to physical frame %x; got %x", pdtFrame, lastPdtEntry.Frame())
		}
	})

	t.Run("temporary mapping failure", func(t *testing.T) {
		var (
			pdt      PageDirectoryTable
			pdtFrame = mm.Frame(123)
		)

		activePDTFn = func() uintptr {
			return 0
		}

		expErr := &kernel.Error{Module: "test", Message: "error mapping page"}

		mapTemporaryFn = func(_ mm.Frame) (mm.Page, *kernel.Error) {
			return 0, expErr
		}

		unmapFn = func(_ mm.Page) *kernel.Error {
			t.Fatal("unexpected call to Unmap")
			return nil
		}

		if err := pdt.Init(pdtFrame); err != expErr {
			t.Fatalf("expected to get error: %v; got %v", *expErr, err)
		}
	})
}

func TestPageDirectoryTableMapAmd64(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("test requires amd64 runtime; skipping")
	}

	defer func(origFlushTLBEntry func(uintptr), origActivePDT func() uintptr, origMap func(mm.Page, mm.Frame, PageTableEntryFlag) *kernel.Error) {
		flushTLBEntryFn = origFlushTLBEntry
		activePDTFn = origActivePDT
		mapFn = origMap
	}(flushTLBEntryFn, activePDTFn, mapFn)

	t.Run("already mapped PDT", func(t *testing.T) {
		var (
			pdtFrame = mm.Frame(123)
			pdt      = PageDirectoryTable{pdtFrame: pdtFrame}
			page     = mm.PageFromAddress(uintptr(100 * oneMb))
		)

		activePDTFn = func() uintptr {
			return pdtFrame.Address()
		}

		mapFn = func(_ mm.Page, _ mm.Frame, _ PageTableEntryFlag) *kernel.Error {
			return nil
		}

		flushCallCount := 0
		flushTLBEntryFn = func(_ uintptr) {
			flushCallCount++
		}

		if err := pdt.Map(page, mm.Frame(321), FlagRW); err != nil {
			t.Fatal(err)
		}

		if exp := 0; flushCallCount != exp {
			t.Fatalf("expected flushTLBEntry to be called %d times; called %d", exp, flushCallCount)
		}
	})

	t.Run("not mapped PDT", func(t *testing.T) {
		var (
			pdtFrame       = mm.Frame(123)
			pdt            = PageDirectoryTable{pdtFrame: pdtFrame}
			page           = mm.PageFromAddress(uintptr(100 * oneMb))
			activePhysPage [mm.PageSize >> mm.PointerShift]pageTableEntry
			activePdtFrame = mm.Frame(uintptr(unsafe.Pointer(&activePhysPage[0])) >> mm.PageShift)
		)

		// Initially, activePhysPage is recursively mapped to itself
		activePhysPage[len(activePhysPage)-1].SetFlags(FlagPresent | FlagRW)
		activePhysPage[len(activePhysPage)-1].SetFrame(activePdtFrame)

		activePDTFn = func() uintptr {
			return activePdtFrame.Address()
		}

		mapFn = func(_ mm.Page, _ mm.Frame, _ PageTableEntryFlag) *kernel.Error {
			return nil
		}

		flushCallCount := 0
		flushTLBEntryFn = func(_ uintptr) {
			switch flushCallCount {
			case 0:
				// the first time we flush the tlb entry, the last entry of
				// the active pdt should be pointing to pdtFrame
				if got := activePhysPage[len(activePhysPage)-1].Frame(); got != pdtFrame {
					t.Fatalf("expected last PDT entry of active PDT to be re-mapped to frame %x; got %x", pdtFrame, got)
				}
			case 1:
				// the second time we flush the tlb entry, the last entry of
				// the active pdt should be pointing back to activePdtFrame
				if got := activePhysPage[len(activePhysPage)-1].Frame(); got != activePdtFrame {
					t.Fatalf("expected last PDT entry of active PDT to be mapped back frame %x; got %x", activePdtFrame, got)
				}
			}
			flushCallCount++
		}

		if err := pdt.Map(page, mm.Frame(321), FlagRW); err != nil {
			t.Fatal(err)
		}

		if exp := 2; flushCallCount != exp {
			t.Fatalf("expected flushTLBEntry to be called %d times; called %d", exp, flushCallCount)
		}
	})
}

func TestPageDirectoryTableUnmapAmd64(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("test requires amd64 runtime; skipping")
	}

	defer func(origFlushTLBEntry func(uintptr), origActivePDT func() uintptr, origUnmap func(mm.Page) *kernel.Error) {
		flushTLBEntryFn = origFlushTLBEntry
		activePDTFn = origActivePDT
		unmapFn = origUnmap
	}(flushTLBEntryFn, activePDTFn, unmapFn)

	t.Run("already mapped PDT", func(t *testing.T) {
		var (
			pdtFrame = mm.Frame(123)
			pdt      = PageDirectoryTable{pdtFrame: pdtFrame}
			page     = mm.PageFromAddress(uintptr(100 * oneMb))
		)

		activePDTFn = func() uintptr {
			return pdtFrame.Address()
		}

		unmapFn = func(_ mm.Page) *kernel.Error {
			return nil
		}

		flushCallCount := 0
		flushTLBEntryFn = func(_ uintptr) {
			flushCallCount++
		}

		if err := pdt.Unmap(page); err != nil {
			t.Fatal(err)
		}

		if exp := 0; flushCallCount != exp {
			t.Fatalf("expected flushTLBEntry to be called %d times; called %d", exp, flushCallCount)
		}
	})

	t.Run("not mapped PDT", func(t *testing.T) {
		var (
			pdtFrame       = mm.Frame(123)
			pdt            = PageDirectoryTable{pdtFrame: pdtFrame}
			page           = mm.PageFromAddress(uintptr(100 * oneMb))
			activePhysPage [mm.PageSize >> mm.PointerShift]pageTableEntry
			activePdtFrame = mm.Frame(uintptr(unsafe.Pointer(&activePhysPage[0])) >> mm.PageShift)
		)

		// Initially, activePhysPage is recursively mapped to itself
		activePhysPage[len(activePhysPage)-1].SetFlags(FlagPresent | FlagRW)
		activePhysPage[len(activePhysPage)-1].SetFrame(activePdtFrame)

		activePDTFn = func() uintptr {
			return activePdtFrame.Address()
		}

		unmapFn = func(_ mm.Page) *kernel.Error {
			return nil
		}

		flushCallCount := 0
		flushTLBEntryFn = func(_ uintptr) {
			switch flushCallCount {
			case 0:
				// the first time we flush the tlb entry, the last entry of
				// the active pdt should be pointing to pdtFrame
				if got := activePhysPage[len(activePhysPage)-1].Frame(); got != pdtFrame {
					t.Fatalf("expected last PDT entry of active PDT to be re-mapped to frame %x; got %x", pdtFrame, got)
				}
			case 1:
				// the second time we flush the tlb entry, the last entry of
				// the active pdt should be pointing back to activePdtFrame
				if got := activePhysPage[len(activePhysPage)-1].Frame(); got != activePdtFrame {
					t.Fatalf("expected last PDT entry of active PDT to be mapped back frame %x; got %x", activePdtFrame, got)
				}
			}
			flushCallCount++
		}

		if err := pdt.Unmap(page); err != nil {
			t.Fatal(err)
		}

		if exp := 2; flushCallCount != exp {
			t.Fatalf("expected flushTLBEntry to be called %d times; called %d", exp, flushCallCount)
		}
	})
}

func TestPageDirectoryTableActivateAmd64(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("test requires amd64 runtime; skipping")
	}

	defer func(origSwitchPDT func(uintptr)) {
		switchPDTFn = origSwitchPDT
	}(switchPDTFn)

	var (
		pdtFrame = mm.Frame(123)
		pdt      = PageDirectoryTable{pdtFrame: pdtFrame}
	)

	switchPDTCallCount := 0
	switchPDTFn = func(_ uintptr) {
		switchPDTCallCount++
	}

	pdt.Activate()
	if exp := 1; switchPDTCallCount != exp {
		t.Fatalf("expected switchPDT to be called %d times; called %d", exp, switchPDTCallCount)
	}
}

func TestSetupPDTForKernel(t *testing.T) {
	defer func() {
		mm.SetFrameAllocator(nil)
		activePDTFn = cpu.ActivePDT
		switchPDTFn = cpu.SwitchPDT
		translateFn = Translate
		mapFn = Map
		mapTemporaryFn = MapTemporary
		unmapFn = Unmap
		earlyReserveLastUsed = tempMappingAddr
	}()

	// reserve space for an allocated page
	reservedPage := make([]byte, mm.PageSize)

	multiboot.SetInfoPtr(uintptr(unsafe.Pointer(&emptyInfoData[0])))

	t.Run("map kernel sections", func(t *testing.T) {
		defer func() { visitElfSectionsFn = multiboot.VisitElfSections }()

		mm.SetFrameAllocator(func() (mm.Frame, *kernel.Error) {
			addr := uintptr(unsafe.Pointer(&reservedPage[0]))
			return mm.Frame(addr >> mm.PageShift), nil
		})
		activePDTFn = func() uintptr {
			return uintptr(unsafe.Pointer(&reservedPage[0]))
		}
		switchPDTFn = func(_ uintptr) {}
		translateFn = func(_ uintptr) (uintptr, *kernel.Error) { return 0xbadf00d000, nil }
		mapTemporaryFn = func(f mm.Frame) (mm.Page, *kernel.Error) { return mm.Page(f), nil }
		visitElfSectionsFn = func(v multiboot.ElfSectionVisitor) {
			// address < VMA; should be ignored
			v(".debug", 0, 0, uint64(mm.PageSize>>1))
			// section uses 32-byte alignment instead of page alignment and has a size
			// equal to 1 page. Due to rounding, we need to actually map 2 pages.
			v(".text", multiboot.ElfSectionExecutable, 0x10032, uint64(mm.PageSize))
			v(".data", multiboot.ElfSectionWritable, 0x2000, uint64(mm.PageSize))
			// section is page-aligned and occupies exactly 2 pages
			v(".rodata", 0, 0x3000, uint64(mm.PageSize<<1))
		}
		mapCount := 0
		mapFn = func(page mm.Page, frame mm.Frame, flags PageTableEntryFlag) *kernel.Error {
			defer func() { mapCount++ }()

			var expFlags PageTableEntryFlag

			switch mapCount {
			case 0, 1:
				expFlags = FlagPresent
			case 2:
				expFlags = FlagPresent | FlagNoExecute | FlagRW
			case 3, 4:
				expFlags = FlagPresent | FlagNoExecute
			}

			if (flags & expFlags) != expFlags {
				t.Errorf("[map call %d] expected flags to be %d; got %d", mapCount, expFlags, flags)
			}

			return nil
		}

		if err := setupPDTForKernel(0x123); err != nil {
			t.Fatal(err)
		}

		if exp := 5; mapCount != exp {
			t.Errorf("expected Map to be called %d times; got %d", exp, mapCount)
		}
	})

	t.Run("map of kernel sections fials", func(t *testing.T) {
		defer func() { visitElfSectionsFn = multiboot.VisitElfSections }()
		expErr := &kernel.Error{Module: "test", Message: "map failed"}

		mm.SetFrameAllocator(func() (mm.Frame, *kernel.Error) {
			addr := uintptr(unsafe.Pointer(&reservedPage[0]))
			return mm.Frame(addr >> mm.PageShift), nil
		})
		activePDTFn = func() uintptr {
			return uintptr(unsafe.Pointer(&reservedPage[0]))
		}
		switchPDTFn = func(_ uintptr) {}
		translateFn = func(_ uintptr) (uintptr, *kernel.Error) { return 0xbadf00d000, nil }
		mapTemporaryFn = func(f mm.Frame) (mm.Page, *kernel.Error) { return mm.Page(f), nil }
		visitElfSectionsFn = func(v multiboot.ElfSectionVisitor) {
			v(".text", multiboot.ElfSectionExecutable, 0xbadc0ffee, uint64(mm.PageSize>>1))
		}
		mapFn = func(page mm.Page, frame mm.Frame, flags PageTableEntryFlag) *kernel.Error {
			return expErr
		}

		if err := setupPDTForKernel(0); err != expErr {
			t.Fatalf("expected error: %v; got %v", expErr, err)
		}
	})

	t.Run("copy allocator reservations to PDT", func(t *testing.T) {
		earlyReserveLastUsed = tempMappingAddr - uintptr(mm.PageSize)
		mm.SetFrameAllocator(func() (mm.Frame, *kernel.Error) {
			addr := uintptr(unsafe.Pointer(&reservedPage[0]))
			return mm.Frame(addr >> mm.PageShift), nil
		})
		activePDTFn = func() uintptr {
			return uintptr(unsafe.Pointer(&reservedPage[0]))
		}
		switchPDTFn = func(_ uintptr) {}
		translateFn = func(_ uintptr) (uintptr, *kernel.Error) { return 0xbadf00d000, nil }
		unmapFn = func(p mm.Page) *kernel.Error { return nil }
		mapTemporaryFn = func(f mm.Frame) (mm.Page, *kernel.Error) { return mm.Page(f), nil }
		mapFn = func(page mm.Page, frame mm.Frame, flags PageTableEntryFlag) *kernel.Error {
			if exp := mm.PageFromAddress(earlyReserveLastUsed); page != exp {
				t.Errorf("expected Map to be called with page %d; got %d", exp, page)
			}

			if exp := mm.Frame(0xbadf00d000 >> mm.PageShift); frame != exp {
				t.Errorf("expected Map to be called with frame %d; got %d", exp, frame)
			}

			if flags&(FlagPresent|FlagRW) != (FlagPresent | FlagRW) {
				t.Error("expected Map to be called FlagPresent | FlagRW")
			}
			return nil
		}

		if err := setupPDTForKernel(0); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("pdt init fails", func(t *testing.T) {
		expErr := &kernel.Error{Module: "test", Message: "translate failed"}

		mm.SetFrameAllocator(func() (mm.Frame, *kernel.Error) {
			addr := uintptr(unsafe.Pointer(&reservedPage[0]))
			return mm.Frame(addr >> mm.PageShift), nil
		})
		activePDTFn = func() uintptr { return 0 }
		mapTemporaryFn = func(f mm.Frame) (mm.Page, *kernel.Error) { return 0, expErr }

		if err := setupPDTForKernel(0); err != expErr {
			t.Fatalf("expected error: %v; got %v", expErr, err)
		}
	})

	t.Run("translation fails for page in reserved address space", func(t *testing.T) {
		expErr := &kernel.Error{Module: "test", Message: "translate failed"}

		earlyReserveLastUsed = tempMappingAddr - uintptr(mm.PageSize)
		mm.SetFrameAllocator(func() (mm.Frame, *kernel.Error) {
			addr := uintptr(unsafe.Pointer(&reservedPage[0]))
			return mm.Frame(addr >> mm.PageShift), nil
		})
		activePDTFn = func() uintptr {
			return uintptr(unsafe.Pointer(&reservedPage[0]))
		}
		translateFn = func(_ uintptr) (uintptr, *kernel.Error) {
			return 0, expErr
		}

		if err := setupPDTForKernel(0); err != expErr {
			t.Fatalf("expected error: %v; got %v", expErr, err)
		}
	})

	t.Run("map fails for page in reserved address space", func(t *testing.T) {
		expErr := &kernel.Error{Module: "test", Message: "map failed"}

		earlyReserveLastUsed = tempMappingAddr - uintptr(mm.PageSize)
		mm.SetFrameAllocator(func() (mm.Frame, *kernel.Error) {
			addr := uintptr(unsafe.Pointer(&reservedPage[0]))
			return mm.Frame(addr >> mm.PageShift), nil
		})
		activePDTFn = func() uintptr {
			return uintptr(unsafe.Pointer(&reservedPage[0]))
		}
		translateFn = func(_ uintptr) (uintptr, *kernel.Error) { return 0xbadf00d000, nil }
		mapTemporaryFn = func(f mm.Frame) (mm.Page, *kernel.Error) { return mm.Page(f), nil }
		mapFn = func(page mm.Page, frame mm.Frame, flags PageTableEntryFlag) *kernel.Error { return expErr }

		if err := setupPDTForKernel(0); err != expErr {
			t.Fatalf("expected error: %v; got %v", expErr, err)
		}
	})
}

var (
	emptyInfoData = []byte{
		0, 0, 0, 0, // size
		0, 0, 0, 0, // reserved
		0, 0, 0, 0, // tag with type zero and length zero
		0, 0, 0, 0,
	}
)

func TestPageTableEntryFlags(t *testing.T) {
	var (
		pte   pageTableEntry
		flag1 = PageTableEntryFlag(1 << 10)
		flag2 = PageTableEntryFlag(1 << 21)
	)

	if pte.HasAnyFlag(flag1 | flag2) {
		t.Fatalf("expected HasAnyFlags to return false")
	}

	pte.SetFlags(flag1 | flag2)

	if !pte.HasAnyFlag(flag1 | flag2) {
		t.Fatalf("expected HasAnyFlags to return true")
	}

	if !pte.HasFlags(flag1 | flag2) {
		t.Fatalf("expected HasFlags to return true")
	}

	pte.ClearFlags(flag1)

	if !pte.HasAnyFlag(flag1 | flag2) {
		t.Fatalf("expected HasAnyFlags to return true")
	}

	if pte.HasFlags(flag1 | flag2) {
		t.Fatalf("expected HasFlags to return false")
	}

	pte.ClearFlags(flag1 | flag2)

	if pte.HasAnyFlag(flag1 | flag2) {
		t.Fatalf("expected HasAnyFlags to return false")
	}

	if pte.HasFlags(flag1 | flag2) {
		t.Fatalf("expected HasFlags to return false")
	}
}

func TestPageTableEntryFrameEncoding(t *testing.T) {
	var (
		pte       pageTableEntry
		physFrame = mm.Frame(123)
	)

	pte.SetFrame(physFrame)
	if got := pte.Frame(); got != physFrame {
		t.Fatalf("expected pte.Frame() to return %v; got %v", physFrame, got)
	}
}

func TestPtePtrFn(t *testing.T) {
	// Dummy test to keep coverage happy
	if exp, got := unsafe.Pointer(uintptr(123)), ptePtrFn(uintptr(123)); exp != got {
		t.Fatalf("expected ptePtrFn to return %v; got %v", exp, got)
	}
}

func TestWalkAmd64(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("test requires amd64 runtime; skipping")
	}

	defer func(origPtePtr func(uintptr) unsafe.Pointer) {
		ptePtrFn = origPtePtr
	}(ptePtrFn)

	// This address breaks down to:
	// p4 index: 1
	// p3 index: 2
	// p2 index: 3
	// p1 index: 4
	// offset  : 1024
	targetAddr := uintptr(0x8080604400)

	sizeofPteEntry := uintptr(unsafe.Sizeof(pageTableEntry(0)))
	expEntryAddrBits := [pageLevels][pageLevels + 1]uintptr{
		{511, 511, 511, 511, 1 * sizeofPteEntry},
		{511, 511, 511, 1, 2 * sizeofPteEntry},
		{511, 511, 1, 2, 3 * sizeofPteEntry},
		{511, 1, 2, 3, 4 * sizeofPteEntry},
	}

	pteCallCount := 0
	ptePtrFn = func(entry uintptr) unsafe.Pointer {
		if pteCallCount >= pageLevels {
			t.Fatalf("unexpected call to ptePtrFn; already called %d times", pageLevels)
		}

		for i := 0; i < pageLevels; i++ {
			pteIndex := (entry >> pageLevelShifts[i]) & ((1 << pageLevelBits[i]) - 1)
			if pteIndex != expEntryAddrBits[pteCallCount][i] {
				t.Errorf("[ptePtrFn call %d] expected pte entry for level %d to use offset %d; got %d", pteCallCount, i, expEntryAddrBits[pteCallCount][i], pteIndex)
			}
		}

		// Check the page offset
		pteIndex := entry & ((1 << mm.PageShift) - 1)
		if pteIndex != expEntryAddrBits[pteCallCount][pageLevels] {
			t.Errorf("[ptePtrFn call %d] expected pte offset to be %d; got %d", pteCallCount, expEntryAddrBits[pteCallCount][pageLevels], pteIndex)
		}

		pteCallCount++

		return unsafe.Pointer(uintptr(0xf00))
	}

	walkFnCallCount := 0
	walk(targetAddr, func(level uint8, entry *pageTableEntry) bool {
		walkFnCallCount++
		return walkFnCallCount != pageLevels
	})

	if pteCallCount != pageLevels {
		t.Errorf("expected ptePtrFn to be called %d times; got %d", pageLevels, pteCallCount)
	}
}
