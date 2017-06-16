package tty

import (
	"testing"
	"unsafe"

	"github.com/achilleasa/gopher-os/kernel/driver/video/console"
)

func TestVtPosition(t *testing.T) {
	specs := []struct {
		inX, inY   uint16
		expX, expY uint16
	}{
		{20, 20, 20, 20},
		{100, 20, 79, 20},
		{10, 200, 10, 24},
		{10, 200, 10, 24},
		{100, 100, 79, 24},
	}

	fb := make([]uint16, 80*25)
	var cons console.Ega
	cons.Init(80, 25, uintptr(unsafe.Pointer(&fb[0])))

	var vt Vt
	vt.AttachTo(&cons)

	w, h := vt.Dimensions()
	if w != 80 || h != 25 {
		t.Fatalf("Dimensions wrong: got %v x %v", w, h)
	}

	for specIndex, spec := range specs {
		vt.SetPosition(spec.inX, spec.inY)
		if x, y := vt.Position(); x != spec.expX || y != spec.expY {
			t.Errorf("[spec %d] expected setting position to (%d, %d) to update the position to (%d, %d); got (%d, %d)", specIndex, spec.inX, spec.inY, spec.expX, spec.expY, x, y)
		}
	}
}

func TestWrite(t *testing.T) {
	fb := make([]uint16, 80*25)
	var cons console.Ega
	cons.Init(80, 25, uintptr(unsafe.Pointer(&fb[0])))

	var vt Vt
	vt.AttachTo(&cons)

	vt.Clear()
	vt.SetPosition(0, 1)
	vt.Write([]byte("12\n\t3\n4\r567\b8"))

	// Tab spanning rows
	vt.SetPosition(78, 4)
	vt.WriteByte('\t')
	vt.WriteByte('9')

	// Trigger scroll and WriteAtPosition into the new blank line.
	vt.SetPosition(79, 24)
	vt.Write([]byte{'!'})
	vt.WriteAtPosition(79, 24, console.White, '!')

	specs := []struct {
		x, y    uint16
		expChar byte
	}{
		{0, 0, '1'},
		{1, 0, '2'},
		// tabs
		{0, 1, ' '},
		{1, 1, ' '},
		{2, 1, ' '},
		{3, 1, ' '},
		{4, 1, '3'},
		// tab spanning 2 rows
		{78, 3, ' '},
		{79, 3, ' '},
		{0, 4, ' '},
		{1, 4, ' '},
		{2, 4, '9'},
		//
		{0, 2, '5'},
		{1, 2, '6'},
		{2, 2, '8'}, // overwritten by BS
		{79, 23, '!'},
		{79, 24, '!'},
	}

	for specIndex, spec := range specs {
		ch := (byte)(fb[(spec.y*vt.width)+spec.x] & 0xFF)
		if ch != spec.expChar {
			t.Errorf("[spec %d] expected char at (%d, %d) to be %c; got %c", specIndex, spec.x, spec.y, spec.expChar, ch)
		}
	}
}
