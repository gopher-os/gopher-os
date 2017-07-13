// Package logo contains logos that can be used with a framebuffer console.
package logo

import "image/color"

// ConsoleLogo defines the logo used by framebuffer consoles. If set to nil
// then no logo will be displayed.
var ConsoleLogo *Image

// Alignment defines the supported horizontal alignments for a console logo.
type Alignment uint8

const (
	// AlignLeft aligns the logo to the left side of the console.
	AlignLeft Alignment = iota

	// AlignCenter aligns the logo to the center of the console.
	AlignCenter

	// AlignRight aligns the logo to the right side of the console.
	AlignRight
)

// Image describes an 8bpp image with
type Image struct {
	// The width and height of the logo in pixels.
	Width  uint32
	Height uint32

	// Align specifies the horizontal alignment for the logo.
	Align Alignment

	// TransparentIndex defines a color index that will be treated as
	// transparent when drawing the logo.
	TransparentIndex uint8

	// The palette for the logo. The console remaps the palette
	// entries to the end of its own palette.
	Palette []color.RGBA

	// The logo data comprises of Width*Height bytes where each byte
	// represents an index in the logo palette.
	Data []uint8
}
