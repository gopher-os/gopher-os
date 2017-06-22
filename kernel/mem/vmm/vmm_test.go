package vmm

import (
	"bytes"
	"strings"
	"testing"
	"unsafe"

	"github.com/achilleasa/gopher-os/kernel"
	"github.com/achilleasa/gopher-os/kernel/cpu"
	"github.com/achilleasa/gopher-os/kernel/driver/video/console"
	"github.com/achilleasa/gopher-os/kernel/hal"
	"github.com/achilleasa/gopher-os/kernel/irq"
	"github.com/achilleasa/gopher-os/kernel/mem"
	"github.com/achilleasa/gopher-os/kernel/mem/pmm"
)

func TestRecoverablePageFault(t *testing.T) {
	var (
		frame       irq.Frame
		regs        irq.Regs
		panicCalled bool
		pageEntry   pageTableEntry
		origPage    = make([]byte, mem.PageSize)
		clonedPage  = make([]byte, mem.PageSize)
		err         = &kernel.Error{Module: "test", Message: "something went wrong"}
	)

	defer func(origPtePtr func(uintptr) unsafe.Pointer) {
		ptePtrFn = origPtePtr
		panicFn = kernel.Panic
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

	mockTTY()

	panicFn = func(_ *kernel.Error) {
		panicCalled = true
	}

	ptePtrFn = func(entry uintptr) unsafe.Pointer { return unsafe.Pointer(&pageEntry) }
	readCR2Fn = func() uint64 { return uint64(uintptr(unsafe.Pointer(&origPage[0]))) }
	unmapFn = func(_ Page) *kernel.Error { return nil }
	flushTLBEntryFn = func(_ uintptr) {}

	for specIndex, spec := range specs {
		mapTemporaryFn = func(f pmm.Frame) (Page, *kernel.Error) { return Page(f), spec.mapError }
		SetFrameAllocator(func() (pmm.Frame, *kernel.Error) {
			addr := uintptr(unsafe.Pointer(&clonedPage[0]))
			return pmm.Frame(addr >> mem.PageShift), spec.allocError
		})

		for i := 0; i < len(origPage); i++ {
			origPage[i] = byte(i % 256)
			clonedPage[i] = 0
		}

		panicCalled = false
		pageEntry = 0
		pageEntry.SetFlags(spec.pteFlags)

		pageFaultHandler(2, &frame, &regs)

		if spec.expPanic != panicCalled {
			t.Errorf("[spec %d] expected panic %t; got %t", specIndex, spec.expPanic, panicCalled)
		}

		if !spec.expPanic {
			for i := 0; i < len(origPage); i++ {
				if origPage[i] != clonedPage[i] {
					t.Errorf("[spec %d] expected clone page to be a copy of the original page; mismatch at index %d", specIndex, i)
				}
			}
		}
	}

}

func TestNonRecoverablePageFault(t *testing.T) {
	defer func() {
		panicFn = kernel.Panic
	}()

	specs := []struct {
		errCode   uint64
		expReason string
		expPanic  bool
	}{
		{
			0,
			"read from non-present page",
			true,
		},
		{
			1,
			"page protection violation (read)",
			true,
		},
		{
			2,
			"write to non-present page",
			true,
		},
		{
			3,
			"page protection violation (write)",
			true,
		},
		{
			4,
			"page-fault in user-mode",
			true,
		},
		{
			8,
			"page table has reserved bit set",
			true,
		},
		{
			16,
			"instruction fetch",
			true,
		},
		{
			0xf00,
			"unknown",
			true,
		},
	}

	var (
		regs  irq.Regs
		frame irq.Frame
	)

	panicCalled := false
	panicFn = func(_ *kernel.Error) {
		panicCalled = true
	}

	for specIndex, spec := range specs {
		fb := mockTTY()
		panicCalled = false

		nonRecoverablePageFault(0xbadf00d000, spec.errCode, &frame, &regs, nil)
		if got := readTTY(fb); !strings.Contains(got, spec.expReason) {
			t.Errorf("[spec %d] expected reason %q; got output:\n%q", specIndex, spec.expReason, got)
			continue
		}

		if spec.expPanic != panicCalled {
			t.Errorf("[spec %d] expected panic %t; got %t", specIndex, spec.expPanic, panicCalled)
		}
	}
}

func TestGPtHandler(t *testing.T) {
	defer func() {
		panicFn = kernel.Panic
		readCR2Fn = cpu.ReadCR2
	}()

	var (
		regs  irq.Regs
		frame irq.Frame
		fb    = mockTTY()
	)

	readCR2Fn = func() uint64 {
		return 0xbadf00d000
	}

	panicCalled := false
	panicFn = func(_ *kernel.Error) {
		panicCalled = true
	}

	generalProtectionFaultHandler(0, &frame, &regs)

	exp := "\nGeneral protection fault while accessing address: 0xbadf00d000\nRegisters:\nRAX = 0000000000000000 RBX = 0000000000000000\nRCX = 0000000000000000 RDX = 0000000000000000\nRSI = 0000000000000000 RDI = 0000000000000000\nRBP = 0000000000000000\nR8  = 0000000000000000 R9  = 0000000000000000\nR10 = 0000000000000000 R11 = 0000000000000000\nR12 = 0000000000000000 R13 = 0000000000000000\nR14 = 0000000000000000 R15 = 0000000000000000\nRIP = 0000000000000000 CS  = 0000000000000000\nRSP = 0000000000000000 SS  = 0000000000000000\nRFL = 0000000000000000"
	if got := readTTY(fb); got != exp {
		t.Errorf("expected output:\n%q\ngot:\n%q", exp, got)
	}

	if !panicCalled {
		t.Error("expected kernel.Panic to be called")
	}
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

func readTTY(fb []byte) string {
	var buf bytes.Buffer
	for i := 0; i < len(fb); i += 2 {
		ch := fb[i]
		if ch == 0 {
			if i+2 < len(fb) && fb[i+2] != 0 {
				buf.WriteByte('\n')
			}
			continue
		}

		buf.WriteByte(ch)
	}

	return buf.String()
}

func mockTTY() []byte {
	// Mock a tty to handle early.Printf output
	mockConsoleFb := make([]byte, 160*25)
	mockConsole := &console.Ega{}
	mockConsole.Init(80, 25, uintptr(unsafe.Pointer(&mockConsoleFb[0])))
	hal.ActiveTerminal.AttachTo(mockConsole)

	return mockConsoleFb
}
