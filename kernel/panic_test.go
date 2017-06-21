package kernel

import (
	"bytes"
	"testing"
	"unsafe"

	"github.com/achilleasa/gopher-os/kernel/cpu"
	"github.com/achilleasa/gopher-os/kernel/driver/video/console"
	"github.com/achilleasa/gopher-os/kernel/hal"
)

func TestPanic(t *testing.T) {
	defer func() {
		cpuHaltFn = cpu.Halt
	}()

	var cpuHaltCalled bool
	cpuHaltFn = func() {
		cpuHaltCalled = true
	}

	t.Run("with error", func(t *testing.T) {
		cpuHaltCalled = false
		fb := mockTTY()
		err := &Error{Module: "test", Message: "panic test"}

		Panic(err)

		exp := "\n-----------------------------------\n[test] unrecoverable error: panic test\n*** kernel panic: system halted ***\n-----------------------------------"

		if got := readTTY(fb); got != exp {
			t.Fatalf("expected to get:\n%q\ngot:\n%q", exp, got)
		}

		if !cpuHaltCalled {
			t.Fatal("expected cpu.Halt() to be called by Panic")
		}
	})

	t.Run("without error", func(t *testing.T) {
		cpuHaltCalled = false
		fb := mockTTY()

		Panic(nil)

		exp := "\n-----------------------------------\n*** kernel panic: system halted ***\n-----------------------------------"

		if got := readTTY(fb); got != exp {
			t.Fatalf("expected to get:\n%q\ngot:\n%q", exp, got)
		}

		if !cpuHaltCalled {
			t.Fatal("expected cpu.Halt() to be called by Panic")
		}
	})
}

func readTTY(fb []byte) string {
	var buf bytes.Buffer
	for i := 0; i < len(fb); i += 2 {
		ch := fb[i]
		if ch == 0 {
			if i+2 < len(fb) && fb[i+2] != 0 {
				buf.WriteByte('\n')
			}
			continue
		}

		buf.WriteByte(ch)
	}

	return buf.String()
}

func mockTTY() []byte {
	// Mock a tty to handle early.Printf output
	mockConsoleFb := make([]byte, 160*25)
	mockConsole := &console.Ega{}
	mockConsole.Init(80, 25, uintptr(unsafe.Pointer(&mockConsoleFb[0])))
	hal.ActiveTerminal.AttachTo(mockConsole)

	return mockConsoleFb
}
