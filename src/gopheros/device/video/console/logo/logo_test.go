package logo

import "testing"

func TestBestFit(t *testing.T) {
	defer func(origList []*Image) {
		availableLogos = origList
	}(availableLogos)

	availableLogos = []*Image{
		{Height: 64},
		{Height: 96},
		{Height: 128},
	}

	specs := []struct {
		consW, consH uint32
		expIndex     int
	}{
		{320, 200, 0},
		{800, 600, 0},
		{1024, 768, 0},
		{1280, 1024, 1},
		{3000, 3000, 2},
		{2500, 1600, 2},
	}

	for specIndex, spec := range specs {
		got := BestFit(spec.consW, spec.consH)
		if got == nil {
			t.Errorf("[spec %d] unable to find a logo", specIndex)
			continue
		}

		if got.Height != availableLogos[spec.expIndex].Height {
			t.Errorf("[spec %d] expected to get logo with height %d; got %d", specIndex, availableLogos[spec.expIndex].Height, got.Height)
		}
	}
}
