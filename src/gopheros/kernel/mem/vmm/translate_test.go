package vmm

import (
	"gopheros/kernel/mem/pmm"
	"runtime"
	"testing"
	"unsafe"
)

func TestTranslateAmd64(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("test requires amd64 runtime; skipping")
	}

	defer func(origPtePtr func(uintptr) unsafe.Pointer) {
		ptePtrFn = origPtePtr
	}(ptePtrFn)

	// the virtual address just contains the page offset
	virtAddr := uintptr(1234)
	expFrame := pmm.Frame(42)
	expPhysAddr := expFrame.Address() + virtAddr
	specs := [][pageLevels]bool{
		{true, true, true, true},
		{false, true, true, true},
		{true, false, true, true},
		{true, true, false, true},
		{true, true, true, false},
	}

	for specIndex, spec := range specs {
		pteCallCount := 0
		ptePtrFn = func(entry uintptr) unsafe.Pointer {
			var pte pageTableEntry
			pte.SetFrame(expFrame)
			if specs[specIndex][pteCallCount] {
				pte.SetFlags(FlagPresent)
			}
			pteCallCount++

			return unsafe.Pointer(&pte)
		}

		// An error is expected if any page level contains a non-present page
		expError := false
		for _, hasMapping := range spec {
			if !hasMapping {
				expError = true
				break
			}
		}

		physAddr, err := Translate(virtAddr)
		switch {
		case expError && err != ErrInvalidMapping:
			t.Errorf("[spec %d] expected to get ErrInvalidMapping; got %v", specIndex, err)
		case !expError && err != nil:
			t.Errorf("[spec %d] unexpected error %v", specIndex, err)
		case !expError && physAddr != expPhysAddr:
			t.Errorf("[spec %d] expected phys addr to be 0x%x; got 0x%x", specIndex, expPhysAddr, physAddr)
		}
	}
}

/*
	phys, err := vmm.Translate(uintptr(100 * mem.Mb))
	if err != nil {
		early.Printf("err: %s\n", err.Error())
	} else {
		early.Printf("phys: 0x%x\n", phys)
	}
*/
