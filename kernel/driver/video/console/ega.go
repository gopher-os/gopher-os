package console

import (
	"reflect"
	"sync"
	"unsafe"
)

const (
	clearColor = Black
	clearChar  = byte(' ')
)

// Ega implements an EGA-compatible text console. At the moment, it uses the
// ega console physical address as its outpucons. After implementing a memory
// allocator, each console will use its own framebuffer while the active console
// will periodically sync its internal buffer with the physical screen buffer.
type Ega struct {
	sync.Mutex

	width  uint16
	height uint16

	fb []uint16
}

// Init sets up the console.
func (cons *Ega) Init(width, height uint16, fbPhysAddr uintptr) {
	cons.width = width
	cons.height = height

	cons.fb = *(*[]uint16)(unsafe.Pointer(&reflect.SliceHeader{
		Len:  int(cons.width * cons.height),
		Cap:  int(cons.width * cons.height),
		Data: fbPhysAddr,
	}))
}

// Clear clears the specified rectangular region
func (cons *Ega) Clear(x, y, width, height uint16) {
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
func (cons *Ega) Dimensions() (uint16, uint16) {
	return cons.width, cons.height
}

// Scroll a particular number of lines to the specified direction.
func (cons *Ega) Scroll(dir ScrollDir, lines uint16) {
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
func (cons *Ega) Write(ch byte, attr Attr, x, y uint16) {
	if x >= cons.width || y >= cons.height {
		return
	}

	cons.fb[(y*cons.width)+x] = (uint16(attr) << 8) | uint16(ch)
}
