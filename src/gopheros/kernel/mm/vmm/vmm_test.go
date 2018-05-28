package vmm

import (
	"gopheros/kernel"
	"gopheros/kernel/cpu"
	"gopheros/kernel/irq"
	"gopheros/kernel/mm"
	"gopheros/multiboot"
	"testing"
	"unsafe"
)

func TestInit(t *testing.T) {
	defer func() {
		mm.SetFrameAllocator(nil)
		activePDTFn = cpu.ActivePDT
		switchPDTFn = cpu.SwitchPDT
		translateFn = Translate
		mapTemporaryFn = MapTemporary
		unmapFn = Unmap
		handleExceptionWithCodeFn = irq.HandleExceptionWithCode
	}()

	// reserve space for an allocated page
	reservedPage := make([]byte, mm.PageSize)

	multiboot.SetInfoPtr(uintptr(unsafe.Pointer(&emptyInfoData[0])))

	t.Run("success", func(t *testing.T) {
		// fill page with junk
		for i := 0; i < len(reservedPage); i++ {
			reservedPage[i] = byte(i % 256)
		}

		mm.SetFrameAllocator(func() (mm.Frame, *kernel.Error) {
			addr := uintptr(unsafe.Pointer(&reservedPage[0]))
			return mm.Frame(addr >> mm.PageShift), nil
		})
		activePDTFn = func() uintptr {
			return uintptr(unsafe.Pointer(&reservedPage[0]))
		}
		switchPDTFn = func(_ uintptr) {}
		unmapFn = func(p mm.Page) *kernel.Error { return nil }
		mapTemporaryFn = func(f mm.Frame) (mm.Page, *kernel.Error) { return mm.Page(f), nil }
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
		mm.SetFrameAllocator(func() (mm.Frame, *kernel.Error) {
			return mm.InvalidFrame, expErr
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
		mm.SetFrameAllocator(func() (mm.Frame, *kernel.Error) {
			defer func() { allocCount++ }()

			if allocCount == 0 {
				addr := uintptr(unsafe.Pointer(&reservedPage[0]))
				return mm.Frame(addr >> mm.PageShift), nil
			}

			return mm.InvalidFrame, expErr
		})
		activePDTFn = func() uintptr {
			return uintptr(unsafe.Pointer(&reservedPage[0]))
		}
		switchPDTFn = func(_ uintptr) {}
		unmapFn = func(p mm.Page) *kernel.Error { return nil }
		mapTemporaryFn = func(f mm.Frame) (mm.Page, *kernel.Error) { return mm.Page(f), nil }
		handleExceptionWithCodeFn = func(_ irq.ExceptionNum, _ irq.ExceptionHandlerWithCode) {}

		if err := Init(0); err != expErr {
			t.Fatalf("expected error: %v; got %v", expErr, err)
		}
	})

	t.Run("blank page mapping error", func(t *testing.T) {
		expErr := &kernel.Error{Module: "test", Message: "map failed"}

		mm.SetFrameAllocator(func() (mm.Frame, *kernel.Error) {
			addr := uintptr(unsafe.Pointer(&reservedPage[0]))
			return mm.Frame(addr >> mm.PageShift), nil
		})
		activePDTFn = func() uintptr {
			return uintptr(unsafe.Pointer(&reservedPage[0]))
		}
		switchPDTFn = func(_ uintptr) {}
		unmapFn = func(p mm.Page) *kernel.Error { return nil }
		mapTemporaryFn = func(f mm.Frame) (mm.Page, *kernel.Error) { return mm.Page(f), expErr }
		handleExceptionWithCodeFn = func(_ irq.ExceptionNum, _ irq.ExceptionHandlerWithCode) {}

		if err := Init(0); err != expErr {
			t.Fatalf("expected error: %v; got %v", expErr, err)
		}
	})
}
