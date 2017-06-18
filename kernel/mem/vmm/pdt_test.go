package vmm

import (
	"runtime"
	"testing"
	"unsafe"

	"github.com/achilleasa/gopher-os/kernel"
	"github.com/achilleasa/gopher-os/kernel/mem"
	"github.com/achilleasa/gopher-os/kernel/mem/pmm"
)

func TestPageDirectoryTableInitAmd64(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("test requires amd64 runtime; skipping")
	}

	defer func(origFlushTLBEntry func(uintptr), origActivePDT func() uintptr, origMapTemporary func(pmm.Frame, FrameAllocatorFn) (Page, *kernel.Error), origUnmap func(Page) *kernel.Error) {
		flushTLBEntryFn = origFlushTLBEntry
		activePDTFn = origActivePDT
		mapTemporaryFn = origMapTemporary
		unmapFn = origUnmap
	}(flushTLBEntryFn, activePDTFn, mapTemporaryFn, unmapFn)

	t.Run("already mapped PDT", func(t *testing.T) {
		var (
			pdt      PageDirectoryTable
			pdtFrame = pmm.Frame(123)
		)

		activePDTFn = func() uintptr {
			return pdtFrame.Address()
		}

		mapTemporaryFn = func(_ pmm.Frame, _ FrameAllocatorFn) (Page, *kernel.Error) {
			t.Fatal("unexpected call to MapTemporary")
			return 0, nil
		}

		unmapFn = func(_ Page) *kernel.Error {
			t.Fatal("unexpected call to Unmap")
			return nil
		}

		if err := pdt.Init(pdtFrame, nil); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("not mapped PDT", func(t *testing.T) {
		var (
			pdt      PageDirectoryTable
			pdtFrame = pmm.Frame(123)
			physPage [mem.PageSize >> mem.PointerShift]pageTableEntry
		)

		// Fill phys page with random junk
		mem.Memset(uintptr(unsafe.Pointer(&physPage[0])), 0xf0, mem.PageSize)

		activePDTFn = func() uintptr {
			return 0
		}

		mapTemporaryFn = func(_ pmm.Frame, _ FrameAllocatorFn) (Page, *kernel.Error) {
			return PageFromAddress(uintptr(unsafe.Pointer(&physPage[0]))), nil
		}

		flushTLBEntryFn = func(_ uintptr) {}

		unmapCallCount := 0
		unmapFn = func(_ Page) *kernel.Error {
			unmapCallCount++
			return nil
		}

		if err := pdt.Init(pdtFrame, nil); err != nil {
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
			pdtFrame = pmm.Frame(123)
		)

		activePDTFn = func() uintptr {
			return 0
		}

		expErr := &kernel.Error{Module: "test", Message: "error mapping page"}

		mapTemporaryFn = func(_ pmm.Frame, _ FrameAllocatorFn) (Page, *kernel.Error) {
			return 0, expErr
		}

		unmapFn = func(_ Page) *kernel.Error {
			t.Fatal("unexpected call to Unmap")
			return nil
		}

		if err := pdt.Init(pdtFrame, nil); err != expErr {
			t.Fatalf("expected to get error: %v; got %v", *expErr, err)
		}
	})
}

func TestPageDirectoryTableMapAmd64(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("test requires amd64 runtime; skipping")
	}

	defer func(origFlushTLBEntry func(uintptr), origActivePDT func() uintptr, origMap func(Page, pmm.Frame, PageTableEntryFlag, FrameAllocatorFn) *kernel.Error) {
		flushTLBEntryFn = origFlushTLBEntry
		activePDTFn = origActivePDT
		mapFn = origMap
	}(flushTLBEntryFn, activePDTFn, mapFn)

	t.Run("already mapped PDT", func(t *testing.T) {
		var (
			pdtFrame = pmm.Frame(123)
			pdt      = PageDirectoryTable{pdtFrame: pdtFrame}
			page     = PageFromAddress(uintptr(100 * mem.Mb))
		)

		activePDTFn = func() uintptr {
			return pdtFrame.Address()
		}

		mapFn = func(_ Page, _ pmm.Frame, _ PageTableEntryFlag, _ FrameAllocatorFn) *kernel.Error {
			return nil
		}

		flushCallCount := 0
		flushTLBEntryFn = func(_ uintptr) {
			flushCallCount++
		}

		if err := pdt.Map(page, pmm.Frame(321), FlagRW, nil); err != nil {
			t.Fatal(err)
		}

		if exp := 0; flushCallCount != exp {
			t.Fatalf("expected flushTLBEntry to be called %d times; called %d", exp, flushCallCount)
		}
	})

	t.Run("not mapped PDT", func(t *testing.T) {
		var (
			pdtFrame       = pmm.Frame(123)
			pdt            = PageDirectoryTable{pdtFrame: pdtFrame}
			page           = PageFromAddress(uintptr(100 * mem.Mb))
			activePhysPage [mem.PageSize >> mem.PointerShift]pageTableEntry
			activePdtFrame = pmm.Frame(uintptr(unsafe.Pointer(&activePhysPage[0])) >> mem.PageShift)
		)

		// Initially, activePhysPage is recursively mapped to itself
		activePhysPage[len(activePhysPage)-1].SetFlags(FlagPresent | FlagRW)
		activePhysPage[len(activePhysPage)-1].SetFrame(activePdtFrame)

		activePDTFn = func() uintptr {
			return activePdtFrame.Address()
		}

		mapFn = func(_ Page, _ pmm.Frame, _ PageTableEntryFlag, _ FrameAllocatorFn) *kernel.Error {
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

		if err := pdt.Map(page, pmm.Frame(321), FlagRW, nil); err != nil {
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

	defer func(origFlushTLBEntry func(uintptr), origActivePDT func() uintptr, origUnmap func(Page) *kernel.Error) {
		flushTLBEntryFn = origFlushTLBEntry
		activePDTFn = origActivePDT
		unmapFn = origUnmap
	}(flushTLBEntryFn, activePDTFn, unmapFn)

	t.Run("already mapped PDT", func(t *testing.T) {
		var (
			pdtFrame = pmm.Frame(123)
			pdt      = PageDirectoryTable{pdtFrame: pdtFrame}
			page     = PageFromAddress(uintptr(100 * mem.Mb))
		)

		activePDTFn = func() uintptr {
			return pdtFrame.Address()
		}

		unmapFn = func(_ Page) *kernel.Error {
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
			pdtFrame       = pmm.Frame(123)
			pdt            = PageDirectoryTable{pdtFrame: pdtFrame}
			page           = PageFromAddress(uintptr(100 * mem.Mb))
			activePhysPage [mem.PageSize >> mem.PointerShift]pageTableEntry
			activePdtFrame = pmm.Frame(uintptr(unsafe.Pointer(&activePhysPage[0])) >> mem.PageShift)
		)

		// Initially, activePhysPage is recursively mapped to itself
		activePhysPage[len(activePhysPage)-1].SetFlags(FlagPresent | FlagRW)
		activePhysPage[len(activePhysPage)-1].SetFrame(activePdtFrame)

		activePDTFn = func() uintptr {
			return activePdtFrame.Address()
		}

		unmapFn = func(_ Page) *kernel.Error {
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
		pdtFrame = pmm.Frame(123)
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
