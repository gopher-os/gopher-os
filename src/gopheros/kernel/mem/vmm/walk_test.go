package vmm

import (
	"gopheros/kernel/mem"
	"runtime"
	"testing"
	"unsafe"
)

func TestPtePtrFn(t *testing.T) {
	// Dummy test to keep coverage happy
	if exp, got := unsafe.Pointer(uintptr(123)), ptePtrFn(uintptr(123)); exp != got {
		t.Fatalf("expected ptePtrFn to return %v; got %v", exp, got)
	}
}

func TestWalkAmd64(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("test requires amd64 runtime; skipping")
	}

	defer func(origPtePtr func(uintptr) unsafe.Pointer) {
		ptePtrFn = origPtePtr
	}(ptePtrFn)

	// This address breaks down to:
	// p4 index: 1
	// p3 index: 2
	// p2 index: 3
	// p1 index: 4
	// offset  : 1024
	targetAddr := uintptr(0x8080604400)

	sizeofPteEntry := uintptr(unsafe.Sizeof(pageTableEntry(0)))
	expEntryAddrBits := [pageLevels][pageLevels + 1]uintptr{
		{511, 511, 511, 511, 1 * sizeofPteEntry},
		{511, 511, 511, 1, 2 * sizeofPteEntry},
		{511, 511, 1, 2, 3 * sizeofPteEntry},
		{511, 1, 2, 3, 4 * sizeofPteEntry},
	}

	pteCallCount := 0
	ptePtrFn = func(entry uintptr) unsafe.Pointer {
		if pteCallCount >= pageLevels {
			t.Fatalf("unexpected call to ptePtrFn; already called %d times", pageLevels)
		}

		for i := 0; i < pageLevels; i++ {
			pteIndex := (entry >> pageLevelShifts[i]) & ((1 << pageLevelBits[i]) - 1)
			if pteIndex != expEntryAddrBits[pteCallCount][i] {
				t.Errorf("[ptePtrFn call %d] expected pte entry for level %d to use offset %d; got %d", pteCallCount, i, expEntryAddrBits[pteCallCount][i], pteIndex)
			}
		}

		// Check the page offset
		pteIndex := entry & ((1 << mem.PageShift) - 1)
		if pteIndex != expEntryAddrBits[pteCallCount][pageLevels] {
			t.Errorf("[ptePtrFn call %d] expected pte offset to be %d; got %d", pteCallCount, expEntryAddrBits[pteCallCount][pageLevels], pteIndex)
		}

		pteCallCount++

		return unsafe.Pointer(uintptr(0xf00))
	}

	walkFnCallCount := 0
	walk(targetAddr, func(level uint8, entry *pageTableEntry) bool {
		walkFnCallCount++
		return walkFnCallCount != pageLevels
	})

	if pteCallCount != pageLevels {
		t.Errorf("expected ptePtrFn to be called %d times; got %d", pageLevels, pteCallCount)
	}
}
