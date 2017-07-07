package vmm

import (
	"bytes"
	"fmt"
	"gopheros/kernel"
	"gopheros/kernel/cpu"
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

func TestGPtHandler(t *testing.T) {
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
		mapTemporaryFn = MapTemporary
		unmapFn = Unmap
		handleExceptionWithCodeFn = irq.HandleExceptionWithCode
	}()

	// reserve space for an allocated page
	reservedPage := make([]byte, mem.PageSize)

	t.Run("success", func(t *testing.T) {
		// fill page with junk
		for i := 0; i < len(reservedPage); i++ {
			reservedPage[i] = byte(i % 256)
		}

		SetFrameAllocator(func() (pmm.Frame, *kernel.Error) {
			addr := uintptr(unsafe.Pointer(&reservedPage[0]))
			return pmm.Frame(addr >> mem.PageShift), nil
		})
		unmapFn = func(p Page) *kernel.Error { return nil }
		mapTemporaryFn = func(f pmm.Frame) (Page, *kernel.Error) { return Page(f), nil }
		handleExceptionWithCodeFn = func(_ irq.ExceptionNum, _ irq.ExceptionHandlerWithCode) {}

		if err := Init(); err != nil {
			t.Fatal(err)
		}

		// reserved page should be zeroed
		for i := 0; i < len(reservedPage); i++ {
			if reservedPage[i] != 0 {
				t.Errorf("expected reserved page to be zeroed; got byte %d at index %d", reservedPage[i], i)
			}
		}
	})

	t.Run("blank page allocation error", func(t *testing.T) {
		expErr := &kernel.Error{Module: "test", Message: "out of memory"}

		SetFrameAllocator(func() (pmm.Frame, *kernel.Error) { return pmm.InvalidFrame, expErr })
		unmapFn = func(p Page) *kernel.Error { return nil }
		mapTemporaryFn = func(f pmm.Frame) (Page, *kernel.Error) { return Page(f), nil }
		handleExceptionWithCodeFn = func(_ irq.ExceptionNum, _ irq.ExceptionHandlerWithCode) {}

		if err := Init(); err != expErr {
			t.Fatalf("expected error: %v; got %v", expErr, err)
		}
	})

	t.Run("blank page mapping error", func(t *testing.T) {
		expErr := &kernel.Error{Module: "test", Message: "map failed"}

		SetFrameAllocator(func() (pmm.Frame, *kernel.Error) {
			addr := uintptr(unsafe.Pointer(&reservedPage[0]))
			return pmm.Frame(addr >> mem.PageShift), nil
		})
		unmapFn = func(p Page) *kernel.Error { return nil }
		mapTemporaryFn = func(f pmm.Frame) (Page, *kernel.Error) { return Page(f), expErr }
		handleExceptionWithCodeFn = func(_ irq.ExceptionNum, _ irq.ExceptionHandlerWithCode) {}

		if err := Init(); err != expErr {
			t.Fatalf("expected error: %v; got %v", expErr, err)
		}
	})
}
