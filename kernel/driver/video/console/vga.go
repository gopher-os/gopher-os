package console

import (
	"reflect"
	"unsafe"
)

const (
	clearColor = Black
	clearChar  = byte(' ')
)

// Vga implements an EGA-compatible text console. At the moment, it uses the
// ega console physical address as its outpucons. After implementing a memory
// allocator, each console will use its own framebuffer while the active console
// will periodically sync its internal buffer with the physical screen buffer.
type Vga struct {
	width  uint16
	height uint16

	fb []uint16
}

// Init sets up the console.
func (cons *Vga) Init() {
	cons.width = 80
	cons.height = 25

	// Set up our frame buffer object by creating a fake slice object pointing
	// to the physical address of the screen buffer.
	if cons.fb != nil {
		return
	}

	cons.fb = *(*[]uint16)(unsafe.Pointer(&reflect.SliceHeader{
		Len:  int(cons.width * cons.height),
		Cap:  int(cons.width * cons.height),
		Data: uintptr(0xB8000),
	}))
}

// Clear clears the specified rectangular region
func (cons *Vga) Clear(x, y, width, height uint16) {
	var (
		attr                 = uint16((clearColor << 4) | clearColor)
		clr                  = attr | uint16(clearChar)
		rowOffset, colOffset uint16
	)

	// clip rectangle
	if x >= cons.width {
		x = cons.width
	}
	if y >= cons.height {
		y = cons.height
	}

	if x+width > cons.width {
		width = cons.width - x
	}
	if y+height > cons.height {
		height = cons.height - y
	}

	rowOffset = (y * cons.width) + x
	for ; height > 0; height, rowOffset = height-1, rowOffset+cons.width {
		for colOffset = rowOffset; colOffset < rowOffset+width; colOffset++ {
			cons.fb[colOffset] = clr
		}
	}
}

// Dimensions returns the console width and height in characters.
func (cons *Vga) Dimensions() (uint16, uint16) {
	return cons.width, cons.height
}

// Scroll a particular number of lines to the specified direction.
func (cons *Vga) Scroll(dir ScrollDir, lines uint16) {
	if lines == 0 || lines > cons.height {
		return
	}

	var i uint16
	offset := lines * cons.width

	switch dir {
	case Up:
		for ; i < (cons.height-lines)*cons.width; i++ {
			cons.fb[i] = cons.fb[i+offset]
		}
	case Down:
		for i = cons.height*cons.width - 1; i >= lines*cons.width; i-- {
			cons.fb[i] = cons.fb[i-offset]
		}
	}
}

// Write a char to the specified location.
func (cons *Vga) Write(ch byte, attr Attr, x, y uint16) {
	if x >= cons.width || y >= cons.height {
		return
	}

	cons.fb[(y*cons.width)+x] = (uint16(attr) << 8) | uint16(ch)
}
