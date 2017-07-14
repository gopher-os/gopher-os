package font

import "testing"

func TestFindByName(t *testing.T) {
	defer func(origList []*Font) {
		availableFonts = origList
	}(availableFonts)

	availableFonts = []*Font{
		&Font{Name: "foo"},
		&Font{Name: "bar"},
	}

	exp := availableFonts[1]
	if got := FindByName("bar"); got != exp {
		t.Fatalf("expected to get font: %v; got %v", exp, got)
	}

	if got := FindByName("not-existing-font"); got != nil {
		t.Fatalf("expected to get nil for a font that does not exist; got %v", got)
	}
}

func TestBestFit(t *testing.T) {
	defer func(origList []*Font) {
		availableFonts = origList
	}(availableFonts)

	availableFonts = []*Font{
		&Font{Name: "retina1", RecommendedWidth: 2560, RecommendedHeight: 1600, Priority: 2},
		&Font{Name: "retina2", RecommendedWidth: 2560, RecommendedHeight: 1600, Priority: 1},
		&Font{Name: "default", RecommendedWidth: 800, RecommendedHeight: 600, Priority: 0},
		&Font{Name: "standard", RecommendedWidth: 1024, RecommendedHeight: 768, Priority: 0},
	}

	specs := []struct {
		consW, consH uint32
		expName      string
	}{
		{320, 200, "default"},
		{800, 600, "default"},
		{1024, 768, "standard"},
		{3000, 3000, "retina2"},
		{2500, 600, "retina2"},
	}

	for specIndex, spec := range specs {
		got := BestFit(spec.consW, spec.consH)
		if got == nil {
			t.Errorf("[spec %d] unable to find a font", specIndex)
			continue
		}

		if got.Name != spec.expName {
			t.Errorf("[spec %d] expected to get font %q; got %q", specIndex, spec.expName, got.Name)
		}
	}
}
