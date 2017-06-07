package vmm

import (
	"testing"

	"github.com/achilleasa/gopher-os/kernel/mem"
)

func TestPageMethods(t *testing.T) {
	for pageIndex := uint64(0); pageIndex < 128; pageIndex++ {
		page := Page(pageIndex)

		if exp, got := uintptr(pageIndex<<mem.PageShift), page.Address(); got != exp {
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
