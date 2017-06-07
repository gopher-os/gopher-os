package vmm

import (
	"testing"

	"github.com/achilleasa/gopher-os/kernel/mem/pmm"
)

func TestPageTableEntryFlags(t *testing.T) {
	var (
		pte   pageTableEntry
		flag1 = PageTableEntryFlag(1 << 10)
		flag2 = PageTableEntryFlag(1 << 21)
	)

	if pte.HasAnyFlag(flag1 | flag2) {
		t.Fatalf("expected HasAnyFlags to return false")
	}

	pte.SetFlags(flag1 | flag2)

	if !pte.HasAnyFlag(flag1 | flag2) {
		t.Fatalf("expected HasAnyFlags to return true")
	}

	if !pte.HasFlags(flag1 | flag2) {
		t.Fatalf("expected HasFlags to return true")
	}

	pte.ClearFlags(flag1)

	if !pte.HasAnyFlag(flag1 | flag2) {
		t.Fatalf("expected HasAnyFlags to return true")
	}

	if pte.HasFlags(flag1 | flag2) {
		t.Fatalf("expected HasFlags to return false")
	}

	pte.ClearFlags(flag1 | flag2)

	if pte.HasAnyFlag(flag1 | flag2) {
		t.Fatalf("expected HasAnyFlags to return false")
	}

	if pte.HasFlags(flag1 | flag2) {
		t.Fatalf("expected HasFlags to return false")
	}
}

func TestPageTableEntryFrameEncoding(t *testing.T) {
	var (
		pte       pageTableEntry
		physFrame = pmm.Frame(123)
	)

	pte.SetFrame(physFrame)
	if got := pte.Frame(); got != physFrame {
		t.Fatalf("expected pte.Frame() to return %v; got %v", physFrame, got)
	}
}
