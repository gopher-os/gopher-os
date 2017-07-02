package cpu

import "testing"

func TestIsIntel(t *testing.T) {
	defer func() {
		cpuidFn = ID
	}()

	specs := []struct {
		eax, ebx, ecx, edx uint32
		exp                bool
	}{
		// CPUID output from an Intel CPU
		{0xd, 0x756e6547, 0x6c65746e, 0x49656e69, true},
		// CPUID output from an AMD Athlon CPU
		{0x1, 68747541, 0x444d4163, 0x69746e65, false},
	}

	for specIndex, spec := range specs {
		cpuidFn = func(_ uint32) (uint32, uint32, uint32, uint32) {
			return spec.eax, spec.ebx, spec.ecx, spec.edx
		}

		if got := IsIntel(); got != spec.exp {
			t.Errorf("[spec %d] expected IsIntel to return %t; got %t", specIndex, spec.exp, got)
		}
	}
}
