package pmm

import (
	"gopheros/kernel/mem"
	"testing"
)

func TestFrameMethods(t *testing.T) {
	for frameIndex := uint64(0); frameIndex < 128; frameIndex++ {
		frame := Frame(frameIndex)

		if !frame.Valid() {
			t.Errorf("expected frame %d to be valid", frameIndex)
		}

		if exp, got := uintptr(frameIndex<<mem.PageShift), frame.Address(); got != exp {
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
