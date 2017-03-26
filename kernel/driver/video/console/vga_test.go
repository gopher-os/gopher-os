package console

import "testing"

func TestVgaInit(t *testing.T) {
	var cons Vga
	cons.Init()

	var expWidth uint16 = 80
	var expHeight uint16 = 25

	if w, h := cons.Dimensions(); w != expWidth || h != expHeight {
		t.Fatalf("expected console dimensions after Init() to be (%d, %d); got (%d, %d)", expWidth, expHeight, w, h)
	}
}

func TestVgaClear(t *testing.T) {
	specs := []struct {
		// Input rect
		x, y, w, h uint16

		// Expected area to be cleared
		expX, expY, expW, expH uint16
	}{
		{
			0, 0, 500, 500,
			0, 0, 80, 25,
		},
		{
			10, 10, 11, 50,
			10, 10, 11, 15,
		},
		{
			10, 10, 110, 1,
			10, 10, 70, 1,
		},
		{
			70, 20, 20, 20,
			70, 20, 10, 5,
		},
		{
			90, 25, 20, 20,
			0, 0, 0, 0,
		},
		{
			12, 12, 5, 6,
			12, 12, 5, 6,
		},
	}

	var cons = Vga{fb: make([]uint16, 80*25)}
	cons.Init()

	testPat := uint16(0xDEAD)
	clearPat := (uint16(clearColor) << 8) | uint16(clearChar)

nextSpec:
	for specIndex, spec := range specs {
		// Fill FB with test pattern
		for i := 0; i < len(cons.fb); i++ {
			cons.fb[i] = testPat
		}

		cons.Clear(spec.x, spec.y, spec.w, spec.h)

		var x, y uint16
		for y = 0; y < cons.height; y++ {
			for x = 0; x < cons.width; x++ {
				fbVal := cons.fb[(y*cons.width)+x]

				if x < spec.expX || y < spec.expY || x >= spec.expX+spec.expW || y >= spec.expY+spec.expH {
					if fbVal != testPat {
						t.Errorf("[spec %d] expected char at (%d, %d) not to be cleared", specIndex, x, y)
						continue nextSpec
					}
				} else {
					if fbVal != clearPat {
						t.Errorf("[spec %d] expected char at (%d, %d) to be cleared", specIndex, x, y)
						continue nextSpec
					}
				}
			}
		}
	}
}

func TestVgaScrollUp(t *testing.T) {
	specs := []uint16{
		0,
		1,
		2,
	}

	var cons = Vga{fb: make([]uint16, 80*25)}
	cons.Init()

nextSpec:
	for specIndex, lines := range specs {
		// Fill buffer with test pattern
		var x, y, index uint16
		for y = 0; y < cons.height; y++ {
			for x = 0; x < cons.width; x++ {
				cons.fb[index] = (y << 8) | x
				index++
			}
		}

		cons.Scroll(Up, lines)

		// Check that rows 1 to (height - lines) have been scrolled up
		index = 0
		for y = 0; y < cons.height-lines; y++ {
			for x = 0; x < cons.width; x++ {
				expVal := ((y + lines) << 8) | x
				if cons.fb[index] != expVal {
					t.Errorf("[spec %d] expected value at (%d, %d) to be %d; got %d", specIndex, x, y, expVal, cons.fb[index])
					continue nextSpec
				}
				index++
			}
		}
	}
}

func TestVgaScrollDown(t *testing.T) {
	specs := []uint16{
		0,
		1,
		2,
	}

	var cons = Vga{fb: make([]uint16, 80*25)}
	cons.Init()

nextSpec:
	for specIndex, lines := range specs {
		// Fill buffer with test pattern
		var x, y, index uint16
		for y = 0; y < cons.height; y++ {
			for x = 0; x < cons.width; x++ {
				cons.fb[index] = (y << 8) | x
				index++
			}
		}

		cons.Scroll(Down, lines)

		// Check that rows lines to height have been scrolled down
		index = lines * cons.width
		for y = lines; y < cons.height-lines; y++ {
			for x = 0; x < cons.width; x++ {
				expVal := ((y - lines) << 8) | x
				if cons.fb[index] != expVal {
					t.Errorf("[spec %d] expected value at (%d, %d) to be %d; got %d", specIndex, x, y, expVal, cons.fb[index])
					continue nextSpec
				}
				index++
			}
		}
	}
}

func TestVgaWriteWithOffScreenCoords(t *testing.T) {
	var cons = Vga{fb: make([]uint16, 80*25)}
	cons.Init()

	specs := []struct {
		x, y uint16
	}{
		{80, 25},
		{90, 24},
		{79, 30},
		{100, 100},
	}

nextSpec:
	for specIndex, spec := range specs {
		for i := 0; i < len(cons.fb); i++ {
			cons.fb[i] = 0
		}

		cons.Write('!', Red, spec.x, spec.y)

		for i := 0; i < len(cons.fb); i++ {
			if got := cons.fb[i]; got != 0 {
				t.Errorf("[spec %d] expected Write() with off-screen coords to be a no-op", specIndex)
				continue nextSpec
			}
		}
	}
}

func TestVgaWrite(t *testing.T) {
	var cons = Vga{fb: make([]uint16, 80*25)}
	cons.Init()

	attr := (Black << 4) | Red
	cons.Write('!', attr, 0, 0)

	expVal := uint16(attr<<8) | uint16('!')
	if got := cons.fb[0]; got != expVal {
		t.Errorf("expected call to Write() to set fb[0] to %d; got %d", expVal, got)
	}
}
