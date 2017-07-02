package vmm

import (
	"runtime"
	"testing"
)

func TestEarlyReserveAmd64(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("test requires amd64 runtime; skipping")
	}

	defer func(origLastUsed uintptr) {
		earlyReserveLastUsed = origLastUsed
	}(earlyReserveLastUsed)

	earlyReserveLastUsed = 4096
	next, err := EarlyReserveRegion(42)
	if err != nil {
		t.Fatal(err)
	}
	if exp := uintptr(0); next != exp {
		t.Fatal("expected reservation request to be rounded to nearest page")
	}

	if _, err = EarlyReserveRegion(1); err != errEarlyReserveNoSpace {
		t.Fatalf("expected to get errEarlyReserveNoSpace; got %v", err)
	}
}
