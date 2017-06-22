package vmm

import (
	"bytes"
	"strings"
	"testing"
	"unsafe"

	"github.com/achilleasa/gopher-os/kernel"
	"github.com/achilleasa/gopher-os/kernel/cpu"
	"github.com/achilleasa/gopher-os/kernel/driver/video/console"
	"github.com/achilleasa/gopher-os/kernel/hal"
	"github.com/achilleasa/gopher-os/kernel/irq"
)

func TestPageFaultHandler(t *testing.T) {
	defer func() {
		panicFn = kernel.Panic
		readCR2Fn = cpu.ReadCR2
	}()

	specs := []struct {
		errCode   uint64
		expReason string
		expPanic  bool
	}{
		{
			0,
			"read from non-present page",
			true,
		},
		{
			1,
			"page protection violation (read)",
			true,
		},
		{
			2,
			"write to non-present page",
			true,
		},
		{
			3,
			"page protection violation (write)",
			true,
		},
		{
			4,
			"page-fault in user-mode",
			true,
		},
		{
			8,
			"page table has reserved bit set",
			true,
		},
		{
			16,
			"instruction fetch",
			true,
		},
		{
			0xf00,
			"unknown",
			true,
		},
	}

	var (
		regs  irq.Regs
		frame irq.Frame
	)

	readCR2Fn = func() uint64 {
		return 0xbadf00d000
	}

	panicCalled := false
	panicFn = func(_ *kernel.Error) {
		panicCalled = true
	}

	for specIndex, spec := range specs {
		fb := mockTTY()
		panicCalled = false

		pageFaultHandler(spec.errCode, &frame, &regs)
		if got := readTTY(fb); !strings.Contains(got, spec.expReason) {
			t.Errorf("[spec %d] expected reason %q; got output:\n%q", specIndex, spec.expReason, got)
			continue
		}

		if spec.expPanic != panicCalled {
			t.Errorf("[spec %d] expected panic %t; got %t", specIndex, spec.expPanic, panicCalled)
		}
	}
}

func TestGPtHandler(t *testing.T) {
	defer func() {
		panicFn = kernel.Panic
		readCR2Fn = cpu.ReadCR2
	}()

	var (
		regs  irq.Regs
		frame irq.Frame
		fb    = mockTTY()
	)

	readCR2Fn = func() uint64 {
		return 0xbadf00d000
	}

	panicCalled := false
	panicFn = func(_ *kernel.Error) {
		panicCalled = true
	}

	generalProtectionFaultHandler(0, &frame, &regs)

	exp := "\nGeneral protection fault while accessing address: 0xbadf00d000\nRegisters:\nRAX = 0000000000000000 RBX = 0000000000000000\nRCX = 0000000000000000 RDX = 0000000000000000\nRSI = 0000000000000000 RDI = 0000000000000000\nRBP = 0000000000000000\nR8  = 0000000000000000 R9  = 0000000000000000\nR10 = 0000000000000000 R11 = 0000000000000000\nR12 = 0000000000000000 R13 = 0000000000000000\nR14 = 0000000000000000 R15 = 0000000000000000\nRIP = 0000000000000000 CS  = 0000000000000000\nRSP = 0000000000000000 SS  = 0000000000000000\nRFL = 0000000000000000"
	if got := readTTY(fb); got != exp {
		t.Errorf("expected output:\n%q\ngot:\n%q", exp, got)
	}

	if !panicCalled {
		t.Error("expected kernel.Panic to be called")
	}
}

func TestInit(t *testing.T) {
	defer func() {
		handleExceptionWithCodeFn = irq.HandleExceptionWithCode
	}()

	handleExceptionWithCodeFn = func(_ irq.ExceptionNum, _ irq.ExceptionHandlerWithCode) {}

	if err := Init(); err != nil {
		t.Fatal(err)
	}
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
