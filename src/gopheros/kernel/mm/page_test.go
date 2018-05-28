package mm

import (
	"gopheros/kernel"
	"testing"
)

func TestFrameMethods(t *testing.T) {
	for frameIndex := uint64(0); frameIndex < 128; frameIndex++ {
		frame := Frame(frameIndex)

		if !frame.Valid() {
			t.Errorf("expected frame %d to be valid", frameIndex)
		}

		if exp, got := uintptr(frameIndex<<PageShift), frame.Address(); got != exp {
			t.Errorf("expected frame (%d, index: %d) call to Address() to return %x; got %x", frame, frameIndex, exp, got)
		}
	}

	invalidFrame := InvalidFrame
	if invalidFrame.Valid() {
		t.Error("expected InvalidFrame.Valid() to return false")
	}
}

func TestFrameFromAddress(t *testing.T) {
	specs := []struct {
		input    uintptr
		expFrame Frame
	}{
		{0, Frame(0)},
		{4095, Frame(0)},
		{4096, Frame(1)},
		{4123, Frame(1)},
	}

	for specIndex, spec := range specs {
		if got := FrameFromAddress(spec.input); got != spec.expFrame {
			t.Errorf("[spec %d] expected returned frame to be %v; got %v", specIndex, spec.expFrame, got)
		}
	}
}

func TestFrameAllocator(t *testing.T) {
	var allocCalled bool
	customAlloc := func() (Frame, *kernel.Error) {
		allocCalled = true
		return FrameFromAddress(0xbadf00), nil
	}

	defer SetFrameAllocator(nil)
	SetFrameAllocator(customAlloc)

	if _, err := AllocFrame(); err != nil {
		t.Fatalf(err.Error())
	}

	if !allocCalled {
		t.Fatal("expected custom allocator to be invoked after all to AllocFrame")
	}
}

func TestPageMethods(t *testing.T) {
	for pageIndex := uint64(0); pageIndex < 128; pageIndex++ {
		page := Page(pageIndex)

		if exp, got := uintptr(pageIndex<<PageShift), page.Address(); got != exp {
			t.Errorf("expected page (%d, index: %d) call to Address() to return %x; got %x", page, pageIndex, exp, got)
		}
	}
}

func TestPageFromAddress(t *testing.T) {
	specs := []struct {
		input   uintptr
		expPage Page
	}{
		{0, Page(0)},
		{4095, Page(0)},
		{4096, Page(1)},
		{4123, Page(1)},
	}

	for specIndex, spec := range specs {
		if got := PageFromAddress(spec.input); got != spec.expPage {
			t.Errorf("[spec %d] expected returned page to be %v; got %v", specIndex, spec.expPage, got)
		}
	}
}
