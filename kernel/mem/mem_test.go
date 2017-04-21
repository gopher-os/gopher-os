package mem

import "testing"

func TestSizeToOrder(t *testing.T) {
	specs := []struct {
		size     Size
		expOrder PageOrder
	}{
		{1 * Kb, PageOrder(0)},
		{PageSize, PageOrder(0)},
		{8 * Kb, PageOrder(1)},
		{2 * Mb, PageOrder(9)},
	}

	for specIndex, spec := range specs {
		if got := spec.size.Order(); got != spec.expOrder {
			t.Errorf("[spec %d] expected to get page order %d; got %d", specIndex, spec.expOrder, got)
		}
	}
}

func TestSizeToPages(t *testing.T) {
	specs := []struct {
		size     Size
		expPages uint32
	}{
		{1023 * Kb, 256},
		{1024 * Kb, 256},
		{1 * Byte, 1},
	}

	for specIndex, spec := range specs {
		if got := spec.size.Pages(); got != spec.expPages {
			t.Errorf("[spec %d] expected Pages(%d bytes) to equal %d; got %d", specIndex, spec.size, spec.expPages, got)
		}
	}
}
