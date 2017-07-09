package console

import (
	"gopheros/device"
	"gopheros/device/video/console/font"
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

type VesaFbConsole struct {
	bpp        uint32
	fbPhysAddr uintptr
	fb         []uint8

	// Console dimensions in pixels
	width  uint32
	height uint32

	// offsetY specifies a the pixel offset for the beginning for text.
	// The rows of the framebuffer between 0 and offsetY are reserved and
	// cannot be used for displaying text.
	offsetY uint32

	// Size of a row in bytes
	pitch uint32

	// Console dimensions in characters
	font          *font.Font
	widthInChars  uint32
	heightInChars uint32

	palette   color.Palette
	defaultFg uint8
	defaultBg uint8
	clearChar uint16
}

func NewVesaFbConsole(width, height uint32, bpp uint8, pitch uint32, fbPhysAddr uintptr) *VesaFbConsole {
	return &VesaFbConsole{
		bpp:        uint32(bpp),
		fbPhysAddr: fbPhysAddr,
		width:      width,
		height:     height,
		pitch:      pitch,
		// light gray text on black background
		defaultFg: 7,
		defaultBg: 0,
		clearChar: uint16(' '),
	}
}

// SetFont selects a bitmap font to be used by the console.
func (cons *VesaFbConsole) SetFont(f *font.Font) {
	if f == nil {
		return
	}

	cons.font = f
	cons.widthInChars = cons.width / uint32(f.GlyphWidth)
	cons.heightInChars = (cons.height - cons.offsetY) / uint32(f.GlyphHeight)
}

// Dimensions returns the console width and height in the specified dimension.
func (cons *VesaFbConsole) Dimensions(dim Dimension) (uint32, uint32) {
	switch dim {
	case Characters:
		return cons.widthInChars, cons.heightInChars
	default:
		return cons.width, cons.height
	}
}

// DefaultColors returns the default foreground and background colors
// used by this console.
func (cons *VesaFbConsole) DefaultColors() (fg uint8, bg uint8) {
	return cons.defaultFg, cons.defaultBg
}

// Fill sets the contents of the specified rectangular region to the requested
// color. Both x and y coordinates are 1-based.
func (cons *VesaFbConsole) Fill(x, y, width, height uint32, _, bg uint8) {
	if cons.font == nil {
		return
	}

	// clip rectangle
	if x == 0 {
		x = 1
	} else if x >= cons.widthInChars {
		x = cons.widthInChars
	}

	if y == 0 {
		y = 1
	} else if y >= cons.heightInChars {
		y = cons.heightInChars
	}

	if x+width-1 > cons.widthInChars {
		width = cons.widthInChars - x + 1
	}

	if y+height-1 > cons.heightInChars {
		height = cons.heightInChars - y + 1
	}

	pX := (x - 1) * cons.font.GlyphWidth
	pY := (y - 1) * cons.font.GlyphHeight
	pW := width * cons.font.GlyphWidth
	pH := height * cons.font.GlyphHeight
	switch cons.bpp {
	case 8:
		cons.fill8(pX, pY, pW, pH, bg)
	}
}

// fill8 implements a fill operation using an 8bpp framebuffer.
func (cons *VesaFbConsole) fill8(pX, pY, pW, pH uint32, bg uint8) {
	fbRowOffset := cons.fbOffset(pX, pY)
	for ; pH > 0; pH, fbRowOffset = pH-1, fbRowOffset+cons.pitch {
		for fbOffset := fbRowOffset; fbOffset < fbRowOffset+pW; fbOffset++ {
			cons.fb[fbOffset] = bg
		}
	}
}

// Scroll the console contents to the specified direction. The caller
// is responsible for updating (e.g. clear or replace) the contents of
// the region that was scrolled.
func (cons *VesaFbConsole) Scroll(dir ScrollDir, lines uint32) {
	if cons.font == nil || lines == 0 || lines > cons.heightInChars {
		return
	}

	offset := cons.fbOffset(0, lines*cons.font.GlyphHeight-cons.offsetY)

	switch dir {
	case ScrollDirUp:
		startOffset := cons.fbOffset(0, 0)
		endOffset := cons.fbOffset(0, cons.height-lines*cons.font.GlyphHeight-cons.offsetY)
		for i := startOffset; i < endOffset; i++ {
			cons.fb[i] = cons.fb[i+offset]
		}
	case ScrollDirDown:
		startOffset := cons.fbOffset(0, lines*cons.font.GlyphHeight)
		for i := uint32(len(cons.fb) - 1); i >= startOffset; i-- {
			cons.fb[i] = cons.fb[i-offset]
		}
	}
}

// Write a char to the specified location. If fg or bg exceed the supported
// colors for this console, they will be set to their default value. Both x and
// y coordinates are 1-based
func (cons *VesaFbConsole) Write(ch byte, fg, bg uint8, x, y uint32) {
	if x < 1 || x > cons.widthInChars || y < 1 || y > cons.heightInChars || cons.font == nil {
		return
	}

	pX := (x - 1) * cons.font.GlyphWidth
	pY := (y - 1) * cons.font.GlyphHeight

	switch cons.bpp {
	case 8:
		cons.write8(ch, fg, bg, pX, pY)
	}
}

// write8 writes a charactero using an 8bpp framebuffer.
func (cons *VesaFbConsole) write8(glyphIndex, fg, bg uint8, pX, pY uint32) {
	var (
		fontOffset  = uint32(glyphIndex) * cons.font.BytesPerRow * cons.font.GlyphHeight
		fbRowOffset = cons.fbOffset(pX, pY)
		fbOffset    uint32
		x, y        uint32
		mask        uint8
	)

	for y = 0; y < cons.font.GlyphHeight; y, fbRowOffset, fontOffset = y+1, fbRowOffset+cons.pitch, fontOffset+1 {
		fbOffset = fbRowOffset
		fontRowData := cons.font.Data[fontOffset]
		mask = 1 << 7
		for x = 0; x < cons.font.GlyphWidth; x, fbOffset, mask = x+1, fbOffset+1, mask>>1 {
			// If mask becomes zero while we are still in this loop
			// then the font uses > 1 byte per row. We need to
			// fetch the next byte and reset the mask.
			if mask == 0 {
				fontOffset++
				fontRowData = cons.font.Data[fontOffset]
				mask = 1 << 7
			}

			if (fontRowData & mask) != 0 {
				cons.fb[fbOffset] = fg
			} else {
				cons.fb[fbOffset] = bg
			}
		}
	}
}

// fbOffset returns the linear offset into the framebuffer that corresponds to
// the pixel at (x,y).
func (cons *VesaFbConsole) fbOffset(x, y uint32) uint32 {
	return ((y + cons.offsetY) * cons.pitch) + (x * cons.bpp >> 3)
}

// Palette returns the active color palette for this console.
func (cons *VesaFbConsole) Palette() color.Palette {
	return cons.palette
}

// SetPaletteColor updates the color definition for the specified
// palette index. Passing a color index greated than the number of
// supported colors should be a no-op.
func (cons *VesaFbConsole) SetPaletteColor(index uint8, rgba color.RGBA) {
	cons.palette[index] = rgba

	// Only program the DAC when we are in indexed (8bpp) mode
	if cons.bpp > 8 {
		return
	}

	// Load palette entry to the DAC. Each DAC entry is a 6-bit value so
	// we need to scale the RGB values in the [0-63] range.
	portWriteByteFn(0x3c8, index)
	portWriteByteFn(0x3c9, rgba.R>>2)
	portWriteByteFn(0x3c9, rgba.G>>2)
	portWriteByteFn(0x3c9, rgba.B>>2)
}

// loadDefaultPalette is called during driver initialization to setup the
// console palette. Regardless of the framebuffer depth, the console always
// uses a 256-color palette.
func (cons *VesaFbConsole) loadDefaultPalette() {
	cons.palette = make(color.Palette, 256)

	egaPalette := []color.RGBA{
		color.RGBA{R: 0, G: 0, B: 0},       /* black */
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
	}

	// Load default EFA palette for colors 0-16
	var index int
	for ; index < len(egaPalette); index++ {
		cons.SetPaletteColor(uint8(index), egaPalette[index])
	}

	// Set all other colors to black
	for ; index < len(cons.palette); index++ {
		cons.SetPaletteColor(uint8(index), egaPalette[0])
	}
}

// DriverName returns the name of this driver.
func (cons *VesaFbConsole) DriverName() string {
	return "vesa_fb_console"
}

// DriverVersion returns the version of this driver.
func (cons *VesaFbConsole) DriverVersion() (uint16, uint16, uint16) {
	return 0, 0, 1
}

// DriverInit initializes this driver.
func (cons *VesaFbConsole) DriverInit(w io.Writer) *kernel.Error {
	// Map the framebuffer so we can write to it
	fbSize := mem.Size(cons.height * cons.pitch)
	fbPage, err := mapRegionFn(
		pmm.Frame(cons.fbPhysAddr>>mem.PageShift),
		fbSize,
		vmm.FlagPresent|vmm.FlagRW,
	)

	if err != nil {
		return err
	}

	cons.fb = *(*[]uint8)(unsafe.Pointer(&reflect.SliceHeader{
		Len:  int(fbSize),
		Cap:  int(fbSize),
		Data: fbPage.Address(),
	}))

	kfmt.Fprintf(w, "mapped framebuffer to 0x%x\n", fbPage.Address())

	cons.loadDefaultPalette()

	return nil
}

// probeForVesaFbConsole checks for the presence of a vga text console.
func probeForVesaFbConsole() device.Driver {
	var drv device.Driver

	fbInfo := getFramebufferInfoFn()
	if fbInfo.Type == multiboot.FramebufferTypeIndexed {
		drv = NewVesaFbConsole(fbInfo.Width, fbInfo.Height, fbInfo.Bpp, fbInfo.Pitch, uintptr(fbInfo.PhysAddr))
	}

	return drv
}

func init() {
	ProbeFuncs = append(ProbeFuncs, probeForVesaFbConsole)
}
