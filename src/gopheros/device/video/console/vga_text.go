package console

import (
	"gopheros/device"
	"gopheros/kernel"
	"gopheros/kernel/hal/multiboot"
	"gopheros/kernel/kfmt"
	"gopheros/kernel/mem"
	"gopheros/kernel/mem/pmm"
	"gopheros/kernel/mem/vmm"
	"image/color"
	"io"
	"reflect"
	"unsafe"
)

// VgaTextConsole implements an EGA-compatible 80x25 text console using VGA
// mode 0x3. The console supports the default 16 EGA colors which can be
// overridden using the SetPaletteColor method.
//
// Each character in the console framebuffer is represented using two bytes,
// a byte for the character ASCII code and a byte that encodes the foreground
// and background colors (4 bits for each).
//
// The default settings for the console are:
//  - light gray text (color 7) on black background (color 0).
//  - space as the clear character
type VgaTextConsole struct {
	width  uint32
	height uint32

	fbPhysAddr uintptr
	fb         []uint16

	palette   color.Palette
	defaultFg uint8
	defaultBg uint8
	clearChar uint16
}

// NewVgaTextConsole creates an new vga text console with its
// framebuffer mapped to fbPhysAddr.
func NewVgaTextConsole(columns, rows uint32, fbPhysAddr uintptr) *VgaTextConsole {
	return &VgaTextConsole{
		width:      columns,
		height:     rows,
		fbPhysAddr: fbPhysAddr,
		clearChar:  uint16(' '),
		palette: color.Palette{
			color.RGBA{R: 0, G: 0, B: 1},       /* black */
			color.RGBA{R: 0, G: 0, B: 128},     /* blue */
			color.RGBA{R: 0, G: 128, B: 1},     /* green */
			color.RGBA{R: 0, G: 128, B: 128},   /* cyan */
			color.RGBA{R: 128, G: 0, B: 1},     /* red */
			color.RGBA{R: 128, G: 0, B: 128},   /* magenta */
			color.RGBA{R: 64, G: 64, B: 1},     /* brown */
			color.RGBA{R: 128, G: 128, B: 128}, /* light gray */
			color.RGBA{R: 64, G: 64, B: 64},    /* dark gray */
			color.RGBA{R: 0, G: 0, B: 255},     /* light blue */
			color.RGBA{R: 0, G: 255, B: 1},     /* light green */
			color.RGBA{R: 0, G: 255, B: 255},   /* light cyan */
			color.RGBA{R: 255, G: 0, B: 1},     /* light red */
			color.RGBA{R: 255, G: 0, B: 255},   /* light magenta */
			color.RGBA{R: 255, G: 255, B: 1},   /* yellow */
			color.RGBA{R: 255, G: 255, B: 255}, /* white */
		},
		// light gray text on black background
		defaultFg: 7,
		defaultBg: 0,
	}
}

// Dimensions returns the console width and height in the specified dimension.
func (cons *VgaTextConsole) Dimensions(dim Dimension) (uint32, uint32) {
	switch dim {
	case Characters:
		return cons.width, cons.height
	default:
		return cons.width * 8, cons.height * 16
	}
}

// DefaultColors returns the default foreground and background colors
// used by this console.
func (cons *VgaTextConsole) DefaultColors() (fg uint8, bg uint8) {
	return cons.defaultFg, cons.defaultBg
}

// Fill sets the contents of the specified rectangular region to the requested
// color. Both x and y coordinates are 1-based.
func (cons *VgaTextConsole) Fill(x, y, width, height uint32, fg, bg uint8) {
	var (
		clr                  = (((uint16(bg) << 4) | uint16(fg)) << 8) | cons.clearChar
		rowOffset, colOffset uint32
	)

	// clip rectangle
	if x == 0 {
		x = 1
	} else if x >= cons.width {
		x = cons.width
	}

	if y == 0 {
		y = 1
	} else if y >= cons.height {
		y = cons.height
	}

	if x+width-1 > cons.width {
		width = cons.width - x + 1
	}

	if y+height-1 > cons.height {
		height = cons.height - y + 1
	}

	rowOffset = ((y - 1) * cons.width) + (x - 1)
	for ; height > 0; height, rowOffset = height-1, rowOffset+cons.width {
		for colOffset = rowOffset; colOffset < rowOffset+width; colOffset++ {
			cons.fb[colOffset] = clr
		}
	}
}

// Scroll the console contents to the specified direction. The caller
// is responsible for updating (e.g. clear or replace) the contents of
// the region that was scrolled.
func (cons *VgaTextConsole) Scroll(dir ScrollDir, lines uint32) {
	if lines == 0 || lines > cons.height {
		return
	}

	var i uint32
	offset := lines * cons.width

	switch dir {
	case ScrollDirUp:
		for ; i < (cons.height-lines)*cons.width; i++ {
			cons.fb[i] = cons.fb[i+offset]
		}
	case ScrollDirDown:
		for i = cons.height*cons.width - 1; i >= lines*cons.width; i-- {
			cons.fb[i] = cons.fb[i-offset]
		}
	}
}

// Write a char to the specified location. If fg or bg exceed the supported
// colors for this console, they will be set to their default value. Both x and
// y coordinates are 1-based
func (cons *VgaTextConsole) Write(ch byte, fg, bg uint8, x, y uint32) {
	if x < 1 || x > cons.width || y < 1 || y > cons.height {
		return
	}

	maxColorIndex := uint8(len(cons.palette) - 1)
	if fg > maxColorIndex {
		fg = cons.defaultFg
	}
	if bg >= maxColorIndex {
		bg = cons.defaultBg
	}

	cons.fb[((y-1)*cons.width)+(x-1)] = (((uint16(bg) << 4) | uint16(fg)) << 8) | uint16(ch)
}

// Palette returns the active color palette for this console.
func (cons *VgaTextConsole) Palette() color.Palette {
	return cons.palette
}

// SetPaletteColor updates the color definition for the specified
// palette index. Passing a color index greated than the number of
// supported colors should be a no-op.
func (cons *VgaTextConsole) SetPaletteColor(index uint8, rgba color.RGBA) {
	if index >= uint8(len(cons.palette)) {
		return
	}

	cons.palette[index] = rgba

	// Load palette entry to the DAC. In this mode, colors are specified
	// using 6-bits for each component; the RGB values need to be converted
	// to the 0-63 range.
	portWriteByteFn(0x3c8, index)
	portWriteByteFn(0x3c9, rgba.R>>2)
	portWriteByteFn(0x3c9, rgba.G>>2)
	portWriteByteFn(0x3c9, rgba.B>>2)
}

// DriverName returns the name of this driver.
func (cons *VgaTextConsole) DriverName() string {
	return "vga_text_console"
}

// DriverVersion returns the version of this driver.
func (cons *VgaTextConsole) DriverVersion() (uint16, uint16, uint16) {
	return 0, 0, 1
}

// DriverInit initializes this driver.
func (cons *VgaTextConsole) DriverInit(w io.Writer) *kernel.Error {
	// Map the framebuffer so we can write to it
	fbSize := mem.Size(cons.width * cons.height * 2)
	fbPage, err := mapRegionFn(
		pmm.Frame(cons.fbPhysAddr>>mem.PageShift),
		fbSize,
		vmm.FlagPresent|vmm.FlagRW,
	)

	if err != nil {
		return err
	}

	cons.fb = *(*[]uint16)(unsafe.Pointer(&reflect.SliceHeader{
		Len:  int(fbSize >> 1),
		Cap:  int(fbSize >> 1),
		Data: fbPage.Address(),
	}))

	kfmt.Fprintf(w, "mapped framebuffer to 0x%x\n", fbPage.Address())

	return nil
}

// probeForVgaTextConsole checks for the presence of a vga text console.
func probeForVgaTextConsole() device.Driver {
	var drv device.Driver
	fbInfo := getFramebufferInfoFn()
	if fbInfo.Type == multiboot.FramebufferTypeEGA {
		drv = NewVgaTextConsole(fbInfo.Width, fbInfo.Height, uintptr(fbInfo.PhysAddr))
	}

	return drv
}

func init() {
	ProbeFuncs = append(ProbeFuncs, probeForVgaTextConsole)
}
