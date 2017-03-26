package tty

import (
	"testing"

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

	var cons console.Vga
	cons.Init()

	var vt Vt
	vt.Init(&cons)

	for specIndex, spec := range specs {
		vt.SetPosition(spec.inX, spec.inY)
		if x, y := vt.Position(); x != spec.expX || y != spec.expY {
			t.Errorf("[spec %d] expected setting position to (%d, %d) to update the position to (%d, %d); got (%d, %d)", specIndex, spec.inX, spec.inY, spec.expX, spec.expY, x, y)
		}
	}
}

func TestWrite(t *testing.T) {
	fb := make([]uint16, 80*25)
	cons := &console.Vga{}
	cons.OverrideFb(fb)
	cons.Init()

	var vt Vt
	vt.Init(cons)

	vt.Clear()
	vt.SetPosition(0, 1)
	vt.Write([]byte("12\n3\n4\r56"))

	// Trigger scroll
	vt.SetPosition(79, 24)
	vt.Write([]byte{'!'})

	specs := []struct {
		x, y    uint16
		expChar byte
	}{
		{0, 0, '1'},
		{1, 0, '2'},
		{0, 1, '3'},
		{0, 2, '5'},
		{1, 2, '6'},
		{79, 23, '!'},
	}

	for specIndex, spec := range specs {
		ch := (byte)(fb[(spec.y*vt.width)+spec.x] & 0xFF)
		if ch != spec.expChar {
			t.Errorf("[spec %d] expected char at (%d, %d) to be %c; got %c", specIndex, spec.x, spec.y, spec.expChar, ch)
		}
	}
}
