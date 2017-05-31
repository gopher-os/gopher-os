package pmm

import (
	"testing"

	"github.com/achilleasa/gopher-os/kernel/mem"
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
