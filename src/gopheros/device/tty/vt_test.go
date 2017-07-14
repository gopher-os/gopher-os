package tty

import (
	"gopheros/device"
	"gopheros/device/video/console"
	"image/color"
	"io"
	"testing"
)

func TestVtPosition(t *testing.T) {
	specs := []struct {
		inX, inY   uint32
		expX, expY uint32
	}{
		{20, 20, 20, 20},
		{100, 20, 80, 20},
		{10, 200, 10, 25},
		{10, 200, 10, 25},
		{100, 100, 80, 25},
	}

	var term Device = NewVT(4, 0)

	// SetCursorPosition without an attached console is a no-op
	term.SetCursorPosition(2, 2)

	if curX, curY := term.CursorPosition(); curX != 1 || curY != 1 {
		t.Fatalf("expected terminal initial position to be (1, 1); got (%d, %d)", curX, curY)
	}

	cons := newMockConsole(80, 25)
	term.AttachTo(cons)

	for specIndex, spec := range specs {
		term.SetCursorPosition(spec.inX, spec.inY)
		if x, y := term.CursorPosition(); x != spec.expX || y != spec.expY {
			t.Errorf("[spec %d] expected setting position to (%d, %d) to update the position to (%d, %d); got (%d, %d)", specIndex, spec.inX, spec.inY, spec.expX, spec.expY, x, y)
		}
	}
}

func TestVtWrite(t *testing.T) {
	t.Run("inactive terminal", func(t *testing.T) {
		cons := newMockConsole(80, 25)

		term := NewVT(4, 0)
		if _, err := term.Write([]byte("foo")); err != io.ErrClosedPipe {
			t.Fatal("expected calling Write on a terminal without an attached console to return ErrClosedPipe")
		}

		term.AttachTo(cons)

		term.curFg = 2
		term.curBg = 3

		data := []byte("\b123\b4\t5\n67\r68")
		count, err := term.Write(data)
		if err != nil {
			t.Fatal(err)
		}

		if count != len(data) {
			t.Fatalf("expected to write %d bytes; wrote %d", len(data), count)
		}

		if cons.bytesWritten != 0 {
			t.Fatalf("expected writes not to be synced with console when terminal is inactive; %d bytes written", cons.bytesWritten)
		}

		specs := []struct {
			x, y    uint32
			expByte uint8
		}{
			{1, 1, '1'},
			{2, 1, '2'},
			{3, 1, '4'},
			{8, 1, '5'}, // 2 + tabWidth + 1
			{1, 2, '6'},
			{2, 2, '8'},
		}

		for specIndex, spec := range specs {
			offset := ((spec.y - 1) * term.viewportWidth * 3) + ((spec.x - 1) * 3)
			if term.data[offset] != spec.expByte {
				t.Errorf("[spec %d] expected char at (%d, %d) to be %q; got %q", specIndex, spec.x, spec.y, spec.expByte, term.data[offset])
			}

			if term.data[offset+1] != term.curFg {
				t.Errorf("[spec %d] expected fg attribute at (%d, %d) to be %d; got %d", specIndex, spec.x, spec.y, term.curFg, term.data[offset+1])
			}

			if term.data[offset+2] != term.curBg {
				t.Errorf("[spec %d] expected bg attribute at (%d, %d) to be %d; got %d", specIndex, spec.x, spec.y, term.curBg, term.data[offset+2])
			}
		}
	})

	t.Run("active terminal", func(t *testing.T) {
		cons := newMockConsole(80, 25)

		term := NewVT(4, 0)
		term.SetState(StateActive)
		term.SetState(StateActive) // calling SetState with the same state is a no-op

		if got := term.State(); got != StateActive {
			t.Fatalf("expected terminal state to be %d; got %d", StateActive, got)
		}

		term.AttachTo(cons)

		term.curFg = 2
		term.curBg = 3

		data := []byte("\b123\b4\t5\n67\r68")
		term.Write(data)

		if expCount := len(data); cons.bytesWritten != expCount {
			t.Fatalf("expected writes to be synced with console when terminal is active. %d bytes written; expected %d", cons.bytesWritten, expCount)
		}

		specs := []struct {
			x, y    uint32
			expByte uint8
		}{
			{1, 1, '1'},
			{2, 1, '2'},
			{3, 1, '4'},
			{8, 1, '5'}, // 2 + tabWidth + 1
			{1, 2, '6'},
			{2, 2, '8'},
		}

		for specIndex, spec := range specs {
			offset := ((spec.y - 1) * cons.width) + (spec.x - 1)
			if cons.chars[offset] != spec.expByte {
				t.Errorf("[spec %d] expected console char at (%d, %d) to be %q; got %q", specIndex, spec.x, spec.y, spec.expByte, cons.chars[offset])
			}

			if cons.fgAttrs[offset] != term.curFg {
				t.Errorf("[spec %d] expected console fg attribute at (%d, %d) to be %d; got %d", specIndex, spec.x, spec.y, term.curFg, cons.fgAttrs[offset])
			}

			if cons.bgAttrs[offset] != term.curBg {
				t.Errorf("[spec %d] expected console bg attribute at (%d, %d) to be %d; got %d", specIndex, spec.x, spec.y, term.curBg, cons.bgAttrs[offset])
			}
		}
	})
}

func TestVtLineFeedHandling(t *testing.T) {
	t.Run("viewport at end of terminal", func(t *testing.T) {
		cons := newMockConsole(80, 25)

		term := NewVT(4, 0)
		term.SetState(StateActive)
		term.AttachTo(cons)

		// Fill last line except the last column which will trigger a
		// line feed. Cursor position will be automatically clipped to
		// the viewport bounds
		term.SetCursorPosition(1, term.viewportHeight+1)
		for i := uint32(0); i < term.viewportWidth-1; i++ {
			term.WriteByte(byte('0' + (i % 10)))
		}

		// Emulate viewportHeight line feeds. The last one should cause a scroll
		term.SetCursorPosition(0, 0) // cursor is set to (1,1)
		for i := uint32(0); i < term.viewportHeight; i++ {
			term.lf(true)
		}

		if cons.scrollUpCount != 1 {
			t.Fatalf("expected console to be scrolled up 1 time; got %d", cons.scrollUpCount)
		}

		// Set cursor one line above the last; this line should now
		// contain the scrolled contents
		term.SetCursorPosition(1, term.viewportHeight-1)
		for col, offset := uint32(1), term.dataOffset; col <= term.viewportWidth; col, offset = col+1, offset+3 {
			expByte := byte('0' + ((col - 1) % 10))
			if col == term.viewportWidth {
				expByte = ' '
			}

			if term.data[offset] != expByte {
				t.Errorf("expected char at (%d, %d) to be %q; got %q", col, term.viewportHeight-1, expByte, term.data[offset])
			}
		}

		// Set cursor to the last line. This line should now be cleared
		term.SetCursorPosition(1, term.viewportHeight)
		for col, offset := uint32(1), term.dataOffset; col <= term.viewportWidth; col, offset = col+1, offset+3 {
			expByte := uint8(' ')
			if term.data[offset] != expByte {
				t.Errorf("expected char at (%d, %d) to be %q; got %q", col, term.viewportHeight, expByte, term.data[offset])
			}
		}
	})

	t.Run("viewport not at end of terminal", func(t *testing.T) {
		cons := newMockConsole(80, 25)

		term := NewVT(4, 1)
		term.SetState(StateActive)
		term.AttachTo(cons)

		// Fill last line except the last column which will trigger a
		// line feed. Cursor position will be automatically clipped to
		// the viewport bounds
		term.SetCursorPosition(1, term.viewportHeight+1)
		for i := uint32(0); i < term.viewportWidth-1; i++ {
			term.WriteByte(byte('0' + (i % 10)))
		}

		// Fill first line including the last column
		term.SetCursorPosition(1, 1)
		for i := uint32(0); i < term.viewportWidth; i++ {
			term.WriteByte(byte('0' + (i % 10)))
		}

		// Emulate viewportHeight line feeds. The last one should cause a scroll
		// in the console but only a viewport adjustment in the terminal
		term.SetCursorPosition(0, 0) // cursor is set to (1,1)
		for i := uint32(0); i < term.viewportHeight; i++ {
			term.lf(true)
		}

		if cons.scrollUpCount != 1 {
			t.Fatalf("expected console to be scrolled up 1 time; got %d", cons.scrollUpCount)
		}

		if expViewportY := uint32(1); term.viewportY != expViewportY {
			t.Fatalf("expected terminal viewportY to be adjusted to %d; got %d", expViewportY, term.viewportY)
		}

		// Check that first line is still available in the terminal buffer
		// that is not currently visible
		term.SetCursorPosition(1, 1)
		offset := term.dataOffset - uint(term.viewportWidth*3)

		for col := uint32(1); col <= term.viewportWidth; col, offset = col+1, offset+3 {
			expByte := byte('0' + ((col - 1) % 10))

			if term.data[offset] != expByte {
				t.Errorf("expected char at hidden region (%d, -1) to be %q; got %q", col, expByte, term.data[offset])
			}
		}
	})
}

func TestVtAttach(t *testing.T) {
	cons := newMockConsole(80, 25)

	term := NewVT(4, 1)

	// AttachTo with a nil console should be a no-op
	term.AttachTo(nil)
	if term.termWidth != 0 || term.termHeight != 0 || term.viewportWidth != 0 || term.viewportHeight != 0 {
		t.Fatal("expected attaching a nil console to be a no-op")
	}

	term.AttachTo(cons)
	if term.termWidth != cons.width ||
		term.termHeight != cons.height+term.scrollback ||
		term.viewportWidth != cons.width ||
		term.viewportHeight != cons.height ||
		term.data == nil {
		t.Fatal("expected the terminal to initialize using the attached console info")
	}
}

func TestVtSetState(t *testing.T) {
	cons := newMockConsole(80, 25)
	term := NewVT(4, 1)
	term.AttachTo(cons)

	// Fill terminal viewport using a rotating pattern. Writing the last
	// character will cause a scroll operation moving the viewport down
	row := 0
	for index := 0; index < int(term.viewportWidth*term.viewportHeight); index++ {
		if index != 0 && index%int(term.viewportWidth) == 0 {
			row++
		}
		term.curFg = uint8((row + index + 1) % 10)
		term.curBg = uint8((row + index + 2) % 10)
		term.WriteByte(byte('0' + (row+index)%10))
	}

	// Activating this terminal should trigger a copy of the terminal viewport
	// contents to the console.
	term.SetState(StateActive)
	row = 1
	for index := 0; index < len(cons.chars); index++ {
		if index != 0 && index%int(cons.width) == 0 {
			row++
		}

		expCh := uint8('0' + (row+index)%10)
		expFg := uint8((row + index + 1) % 10)
		expBg := uint8((row + index + 2) % 10)

		// last line should be cleared due to the scroll operation
		if row == int(cons.height) {
			expCh = ' '
			expFg = 7
			expBg = 0
		}

		if cons.chars[index] != expCh {
			t.Errorf("expected console char at index %d to be %q; got %q", index, expCh, cons.chars[index])
		}

		if cons.fgAttrs[index] != expFg {
			t.Errorf("expected console fg attr at index %d to be %d; got %d", index, expFg, cons.fgAttrs[index])
		}

		if cons.bgAttrs[index] != expBg {
			t.Errorf("expected console bg attr at index %d to be %d; got %d", index, expBg, cons.bgAttrs[index])
		}
	}
}

func TestVTDriverInterface(t *testing.T) {
	var dev device.Driver = NewVT(0, 0)

	if err := dev.DriverInit(nil); err != nil {
		t.Fatal(err)
	}

	if dev.DriverName() == "" {
		t.Fatal("DriverName() returned an empty string")
	}

	if major, minor, patch := dev.DriverVersion(); major+minor+patch == 0 {
		t.Fatal("DriverVersion() returned an invalid version number")
	}
}

func TestVTProbe(t *testing.T) {
	if drv := probeForVT(); drv == nil {
		t.Fatal("expected probeForVT to return a driver")
	}
}

type mockConsole struct {
	width, height   uint32
	fg, bg          uint8
	chars           []uint8
	fgAttrs         []uint8
	bgAttrs         []uint8
	bytesWritten    int
	scrollUpCount   int
	scrollDownCount int
}

func newMockConsole(w, h uint32) *mockConsole {
	return &mockConsole{
		width:   w,
		height:  h,
		fg:      7,
		bg:      0,
		chars:   make([]uint8, w*h),
		fgAttrs: make([]uint8, w*h),
		bgAttrs: make([]uint8, w*h),
	}
}

func (cons *mockConsole) Dimensions(_ console.Dimension) (uint32, uint32) {
	return cons.width, cons.height
}

func (cons *mockConsole) DefaultColors() (uint8, uint8) {
	return cons.fg, cons.bg
}

func (cons *mockConsole) Fill(x, y, width, height uint32, fg, bg uint8) {
	yEnd := y + height - 1
	xEnd := x + width - 1

	for fy := y; fy <= yEnd; fy++ {
		offset := ((fy - 1) * cons.width)
		for fx := x; fx <= xEnd; fx, offset = fx+1, offset+1 {
			cons.chars[offset] = ' '
			cons.fgAttrs[offset] = fg
			cons.bgAttrs[offset] = bg
		}
	}
}

func (cons *mockConsole) Scroll(dir console.ScrollDir, lines uint32) {
	switch dir {
	case console.ScrollDirUp:
		cons.scrollUpCount++
	case console.ScrollDirDown:
		cons.scrollDownCount++
	}
}

func (cons *mockConsole) Palette() color.Palette {
	return nil
}

func (cons *mockConsole) SetPaletteColor(index uint8, color color.RGBA) {
}

func (cons *mockConsole) Write(b byte, fg, bg uint8, x, y uint32) {
	offset := ((y - 1) * cons.width) + (x - 1)
	cons.chars[offset] = b
	cons.fgAttrs[offset] = fg
	cons.bgAttrs[offset] = bg
	cons.bytesWritten++
}
