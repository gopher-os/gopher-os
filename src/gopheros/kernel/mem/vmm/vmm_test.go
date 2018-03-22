package vmm

import (
	"bytes"
	"fmt"
	"gopheros/kernel"
	"gopheros/kernel/cpu"
	"gopheros/kernel/hal/multiboot"
	"gopheros/kernel/irq"
	"gopheros/kernel/kfmt"
	"gopheros/kernel/mem"
	"gopheros/kernel/mem/pmm"
	"strings"
	"testing"
	"unsafe"
)

func TestRecoverablePageFault(t *testing.T) {
	var (
		frame      irq.Frame
		regs       irq.Regs
		pageEntry  pageTableEntry
		origPage   = make([]byte, mem.PageSize)
		clonedPage = make([]byte, mem.PageSize)
		err        = &kernel.Error{Module: "test", Message: "something went wrong"}
	)

	defer func(origPtePtr func(uintptr) unsafe.Pointer) {
		ptePtrFn = origPtePtr
		readCR2Fn = cpu.ReadCR2
		frameAllocator = nil
		mapTemporaryFn = MapTemporary
		unmapFn = Unmap
		flushTLBEntryFn = cpu.FlushTLBEntry
	}(ptePtrFn)

	specs := []struct {
		pteFlags   PageTableEntryFlag
		allocError *kernel.Error
		mapError   *kernel.Error
		expPanic   bool
	}{
		// Missing pge
		{0, nil, nil, true},
		// Page is present but CoW flag not set
		{FlagPresent, nil, nil, true},
		// Page is present but both CoW and RW flags set
		{FlagPresent | FlagRW | FlagCopyOnWrite, nil, nil, true},
		// Page is present with CoW flag set but allocating a page copy fails
		{FlagPresent | FlagCopyOnWrite, err, nil, true},
		// Page is present with CoW flag set but mapping the page copy fails
		{FlagPresent | FlagCopyOnWrite, nil, err, true},
		// Page is present with CoW flag set
		{FlagPresent | FlagCopyOnWrite, nil, nil, false},
	}

	ptePtrFn = func(entry uintptr) unsafe.Pointer { return unsafe.Pointer(&pageEntry) }
	readCR2Fn = func() uint64 { return uint64(uintptr(unsafe.Pointer(&origPage[0]))) }
	unmapFn = func(_ Page) *kernel.Error { return nil }
	flushTLBEntryFn = func(_ uintptr) {}

	for specIndex, spec := range specs {
		t.Run(fmt.Sprint(specIndex), func(t *testing.T) {
			defer func() {
				err := recover()
				if spec.expPanic && err == nil {
					t.Error("expected a panic")
				} else if !spec.expPanic {
					if err != nil {
						t.Error("unexpected panic")
						return
					}

					for i := 0; i < len(origPage); i++ {
						if origPage[i] != clonedPage[i] {
							t.Errorf("expected clone page to be a copy of the original page; mismatch at index %d", i)
						}
					}
				}
			}()

			mapTemporaryFn = func(f pmm.Frame) (Page, *kernel.Error) { return Page(f), spec.mapError }
			SetFrameAllocator(func() (pmm.Frame, *kernel.Error) {
				addr := uintptr(unsafe.Pointer(&clonedPage[0]))
				return pmm.Frame(addr >> mem.PageShift), spec.allocError
			})

			for i := 0; i < len(origPage); i++ {
				origPage[i] = byte(i % 256)
				clonedPage[i] = 0
			}

			pageEntry = 0
			pageEntry.SetFlags(spec.pteFlags)

			pageFaultHandler(2, &frame, &regs)
		})
	}

}

func TestNonRecoverablePageFault(t *testing.T) {
	defer func() {
		kfmt.SetOutputSink(nil)
	}()

	specs := []struct {
		errCode   uint64
		expReason string
	}{
		{
			0,
			"read from non-present page",
		},
		{
			1,
			"page protection violation (read)",
		},
		{
			2,
			"write to non-present page",
		},
		{
			3,
			"page protection violation (write)",
		},
		{
			4,
			"page-fault in user-mode",
		},
		{
			8,
			"page table has reserved bit set",
		},
		{
			16,
			"instruction fetch",
		},
		{
			0xf00,
			"unknown",
		},
	}

	var (
		regs  irq.Regs
		frame irq.Frame
		buf   bytes.Buffer
	)

	kfmt.SetOutputSink(&buf)
	for specIndex, spec := range specs {
		t.Run(fmt.Sprint(specIndex), func(t *testing.T) {
			buf.Reset()
			defer func() {
				if err := recover(); err != errUnrecoverableFault {
					t.Errorf("expected a panic with errUnrecoverableFault; got %v", err)
				}
			}()

			nonRecoverablePageFault(0xbadf00d000, spec.errCode, &frame, &regs, errUnrecoverableFault)
			if got := buf.String(); !strings.Contains(got, spec.expReason) {
				t.Errorf("expected reason %q; got output:\n%q", spec.expReason, got)
			}
		})
	}
}

func TestGPFHandler(t *testing.T) {
	defer func() {
		readCR2Fn = cpu.ReadCR2
	}()

	var (
		regs  irq.Regs
		frame irq.Frame
	)

	readCR2Fn = func() uint64 {
		return 0xbadf00d000
	}

	defer func() {
		if err := recover(); err != errUnrecoverableFault {
			t.Errorf("expected a panic with errUnrecoverableFault; got %v", err)
		}
	}()

	generalProtectionFaultHandler(0, &frame, &regs)
}

func TestInit(t *testing.T) {
	defer func() {
		frameAllocator = nil
		activePDTFn = cpu.ActivePDT
		switchPDTFn = cpu.SwitchPDT
		translateFn = Translate
		mapTemporaryFn = MapTemporary
		unmapFn = Unmap
		handleExceptionWithCodeFn = irq.HandleExceptionWithCode
	}()

	// reserve space for an allocated page
	reservedPage := make([]byte, mem.PageSize)

	multiboot.SetInfoPtr(uintptr(unsafe.Pointer(&emptyInfoData[0])))

	t.Run("success", func(t *testing.T) {
		// fill page with junk
		for i := 0; i < len(reservedPage); i++ {
			reservedPage[i] = byte(i % 256)
		}

		SetFrameAllocator(func() (pmm.Frame, *kernel.Error) {
			addr := uintptr(unsafe.Pointer(&reservedPage[0]))
			return pmm.Frame(addr >> mem.PageShift), nil
		})
		activePDTFn = func() uintptr {
			return uintptr(unsafe.Pointer(&reservedPage[0]))
		}
		switchPDTFn = func(_ uintptr) {}
		unmapFn = func(p Page) *kernel.Error { return nil }
		mapTemporaryFn = func(f pmm.Frame) (Page, *kernel.Error) { return Page(f), nil }
		handleExceptionWithCodeFn = func(_ irq.ExceptionNum, _ irq.ExceptionHandlerWithCode) {}

		if err := Init(0); err != nil {
			t.Fatal(err)
		}

		// reserved page should be zeroed
		for i := 0; i < len(reservedPage); i++ {
			if reservedPage[i] != 0 {
				t.Errorf("expected reserved page to be zeroed; got byte %d at index %d", reservedPage[i], i)
			}
		}
	})

	t.Run("setupPDT fails", func(t *testing.T) {
		expErr := &kernel.Error{Module: "test", Message: "out of memory"}

		// Allow the PDT allocation to succeed and then return an error when
		// trying to allocate the blank fram
		SetFrameAllocator(func() (pmm.Frame, *kernel.Error) {
			return pmm.InvalidFrame, expErr
		})

		if err := Init(0); err != expErr {
			t.Fatalf("expected error: %v; got %v", expErr, err)
		}
	})

	t.Run("blank page allocation error", func(t *testing.T) {
		expErr := &kernel.Error{Module: "test", Message: "out of memory"}

		// Allow the PDT allocation to succeed and then return an error when
		// trying to allocate the blank fram
		var allocCount int
		SetFrameAllocator(func() (pmm.Frame, *kernel.Error) {
			defer func() { allocCount++ }()

			if allocCount == 0 {
				addr := uintptr(unsafe.Pointer(&reservedPage[0]))
				return pmm.Frame(addr >> mem.PageShift), nil
			}

			return pmm.InvalidFrame, expErr
		})
		activePDTFn = func() uintptr {
			return uintptr(unsafe.Pointer(&reservedPage[0]))
		}
		switchPDTFn = func(_ uintptr) {}
		unmapFn = func(p Page) *kernel.Error { return nil }
		mapTemporaryFn = func(f pmm.Frame) (Page, *kernel.Error) { return Page(f), nil }
		handleExceptionWithCodeFn = func(_ irq.ExceptionNum, _ irq.ExceptionHandlerWithCode) {}

		if err := Init(0); err != expErr {
			t.Fatalf("expected error: %v; got %v", expErr, err)
		}
	})

	t.Run("blank page mapping error", func(t *testing.T) {
		expErr := &kernel.Error{Module: "test", Message: "map failed"}

		SetFrameAllocator(func() (pmm.Frame, *kernel.Error) {
			addr := uintptr(unsafe.Pointer(&reservedPage[0]))
			return pmm.Frame(addr >> mem.PageShift), nil
		})
		activePDTFn = func() uintptr {
			return uintptr(unsafe.Pointer(&reservedPage[0]))
		}
		switchPDTFn = func(_ uintptr) {}
		unmapFn = func(p Page) *kernel.Error { return nil }
		mapTemporaryFn = func(f pmm.Frame) (Page, *kernel.Error) { return Page(f), expErr }
		handleExceptionWithCodeFn = func(_ irq.ExceptionNum, _ irq.ExceptionHandlerWithCode) {}

		if err := Init(0); err != expErr {
			t.Fatalf("expected error: %v; got %v", expErr, err)
		}
	})
}

func TestSetupPDTForKernel(t *testing.T) {
	defer func() {
		frameAllocator = nil
		activePDTFn = cpu.ActivePDT
		switchPDTFn = cpu.SwitchPDT
		translateFn = Translate
		mapFn = Map
		mapTemporaryFn = MapTemporary
		unmapFn = Unmap
		earlyReserveLastUsed = tempMappingAddr
	}()

	// reserve space for an allocated page
	reservedPage := make([]byte, mem.PageSize)

	multiboot.SetInfoPtr(uintptr(unsafe.Pointer(&emptyInfoData[0])))

	t.Run("map kernel sections", func(t *testing.T) {
		defer func() { visitElfSectionsFn = multiboot.VisitElfSections }()

		SetFrameAllocator(func() (pmm.Frame, *kernel.Error) {
			addr := uintptr(unsafe.Pointer(&reservedPage[0]))
			return pmm.Frame(addr >> mem.PageShift), nil
		})
		activePDTFn = func() uintptr {
			return uintptr(unsafe.Pointer(&reservedPage[0]))
		}
		switchPDTFn = func(_ uintptr) {}
		translateFn = func(_ uintptr) (uintptr, *kernel.Error) { return 0xbadf00d000, nil }
		mapTemporaryFn = func(f pmm.Frame) (Page, *kernel.Error) { return Page(f), nil }
		visitElfSectionsFn = func(v multiboot.ElfSectionVisitor) {
			// address < VMA; should be ignored
			v(".debug", 0, 0, uint64(mem.PageSize>>1))
			// section uses 32-byte alignment instead of page alignment and has a size
			// equal to 1 page. Due to rounding, we need to actually map 2 pages.
			v(".text", multiboot.ElfSectionExecutable, 0x10032, uint64(mem.PageSize))
			v(".data", multiboot.ElfSectionWritable, 0x2000, uint64(mem.PageSize))
			// section is page-aligned and occupies exactly 2 pages
			v(".rodata", 0, 0x3000, uint64(mem.PageSize<<1))
		}
		mapCount := 0
		mapFn = func(page Page, frame pmm.Frame, flags PageTableEntryFlag) *kernel.Error {
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

		SetFrameAllocator(func() (pmm.Frame, *kernel.Error) {
			addr := uintptr(unsafe.Pointer(&reservedPage[0]))
			return pmm.Frame(addr >> mem.PageShift), nil
		})
		activePDTFn = func() uintptr {
			return uintptr(unsafe.Pointer(&reservedPage[0]))
		}
		switchPDTFn = func(_ uintptr) {}
		translateFn = func(_ uintptr) (uintptr, *kernel.Error) { return 0xbadf00d000, nil }
		mapTemporaryFn = func(f pmm.Frame) (Page, *kernel.Error) { return Page(f), nil }
		visitElfSectionsFn = func(v multiboot.ElfSectionVisitor) {
			v(".text", multiboot.ElfSectionExecutable, 0xbadc0ffee, uint64(mem.PageSize>>1))
		}
		mapFn = func(page Page, frame pmm.Frame, flags PageTableEntryFlag) *kernel.Error {
			return expErr
		}

		if err := setupPDTForKernel(0); err != expErr {
			t.Fatalf("expected error: %v; got %v", expErr, err)
		}
	})

	t.Run("copy allocator reservations to PDT", func(t *testing.T) {
		earlyReserveLastUsed = tempMappingAddr - uintptr(mem.PageSize)
		SetFrameAllocator(func() (pmm.Frame, *kernel.Error) {
			addr := uintptr(unsafe.Pointer(&reservedPage[0]))
			return pmm.Frame(addr >> mem.PageShift), nil
		})
		activePDTFn = func() uintptr {
			return uintptr(unsafe.Pointer(&reservedPage[0]))
		}
		switchPDTFn = func(_ uintptr) {}
		translateFn = func(_ uintptr) (uintptr, *kernel.Error) { return 0xbadf00d000, nil }
		unmapFn = func(p Page) *kernel.Error { return nil }
		mapTemporaryFn = func(f pmm.Frame) (Page, *kernel.Error) { return Page(f), nil }
		mapFn = func(page Page, frame pmm.Frame, flags PageTableEntryFlag) *kernel.Error {
			if exp := PageFromAddress(earlyReserveLastUsed); page != exp {
				t.Errorf("expected Map to be called with page %d; got %d", exp, page)
			}

			if exp := pmm.Frame(0xbadf00d000 >> mem.PageShift); frame != exp {
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

		SetFrameAllocator(func() (pmm.Frame, *kernel.Error) {
			addr := uintptr(unsafe.Pointer(&reservedPage[0]))
			return pmm.Frame(addr >> mem.PageShift), nil
		})
		activePDTFn = func() uintptr { return 0 }
		mapTemporaryFn = func(f pmm.Frame) (Page, *kernel.Error) { return 0, expErr }

		if err := setupPDTForKernel(0); err != expErr {
			t.Fatalf("expected error: %v; got %v", expErr, err)
		}
	})

	t.Run("translation fails for page in reserved address space", func(t *testing.T) {
		expErr := &kernel.Error{Module: "test", Message: "translate failed"}

		earlyReserveLastUsed = tempMappingAddr - uintptr(mem.PageSize)
		SetFrameAllocator(func() (pmm.Frame, *kernel.Error) {
			addr := uintptr(unsafe.Pointer(&reservedPage[0]))
			return pmm.Frame(addr >> mem.PageShift), nil
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

		earlyReserveLastUsed = tempMappingAddr - uintptr(mem.PageSize)
		SetFrameAllocator(func() (pmm.Frame, *kernel.Error) {
			addr := uintptr(unsafe.Pointer(&reservedPage[0]))
			return pmm.Frame(addr >> mem.PageShift), nil
		})
		activePDTFn = func() uintptr {
			return uintptr(unsafe.Pointer(&reservedPage[0]))
		}
		translateFn = func(_ uintptr) (uintptr, *kernel.Error) { return 0xbadf00d000, nil }
		mapTemporaryFn = func(f pmm.Frame) (Page, *kernel.Error) { return Page(f), nil }
		mapFn = func(page Page, frame pmm.Frame, flags PageTableEntryFlag) *kernel.Error { return expErr }

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
