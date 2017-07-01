package irq

import (
	"bytes"
	"gopheros/kernel/driver/video/console"
	"gopheros/kernel/hal"
	"testing"
	"unsafe"
)

func TestRegsPrint(t *testing.T) {
	fb := mockTTY()
	regs := Regs{
		RAX: 1,
		RBX: 2,
		RCX: 3,
		RDX: 4,
		RSI: 5,
		RDI: 6,
		RBP: 7,
		R8:  8,
		R9:  9,
		R10: 10,
		R11: 11,
		R12: 12,
		R13: 13,
		R14: 14,
		R15: 15,
	}
	regs.Print()

	exp := "RAX = 0000000000000001 RBX = 0000000000000002\nRCX = 0000000000000003 RDX = 0000000000000004\nRSI = 0000000000000005 RDI = 0000000000000006\nRBP = 0000000000000007\nR8  = 0000000000000008 R9  = 0000000000000009\nR10 = 000000000000000a R11 = 000000000000000b\nR12 = 000000000000000c R13 = 000000000000000d\nR14 = 000000000000000e R15 = 000000000000000f"

	if got := readTTY(fb); got != exp {
		t.Fatalf("expected to get:\n%q\ngot:\n%q", exp, got)
	}
}

func TestFramePrint(t *testing.T) {
	fb := mockTTY()
	frame := Frame{
		RIP:    1,
		CS:     2,
		RFlags: 3,
		RSP:    4,
		SS:     5,
	}
	frame.Print()

	exp := "RIP = 0000000000000001 CS  = 0000000000000002\nRSP = 0000000000000004 SS  = 0000000000000005\nRFL = 0000000000000003"

	if got := readTTY(fb); got != exp {
		t.Fatalf("expected to get:\n%q\ngot:\n%q", exp, got)
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
