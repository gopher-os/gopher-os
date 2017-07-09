package console

import (
	"gopheros/device/video/console/font"
	"image/color"
)

// ScrollDir defines a scroll direction.
type ScrollDir uint8

// The supported list of scroll directions for the console Scroll() calls.
const (
	ScrollDirUp ScrollDir = iota
	ScrollDirDown
)

// Dimension defines the types of dimensions that can be queried off a device.
type Dimension uint8

const (
	// Characters describes the number of characters in
	// the console depending on the currently active
	// font.
	Characters Dimension = iota

	// Pixels describes the number of pixels in the console framebuffer.
	Pixels
)

// The Device interface is implemented by objects that can function as system
// consoles.
type Device interface {
	// Pixel returns the width and height of the console
	// using a particual dimension.
	Dimensions(Dimension) (uint32, uint32)

	// DefaultColors returns the default foreground and background colors
	// used by this console.
	DefaultColors() (fg, bg uint8)

	// Fill sets the contents of the specified rectangular region to the
	// requested color. Both x and y coordinates are 1-based (top-left
	// corner has coordinates 1,1).
	Fill(x, y, width, height uint32, fg, bg uint8)

	// Scroll the console contents to the specified direction. The caller
	// is responsible for updating (e.g. clear or replace) the contents of
	// the region that was scrolled.
	Scroll(dir ScrollDir, lines uint32)

	// Write a char to the specified location. Both x and y coordinates are
	// 1-based (top-left corner has coordinates 1,1).
	Write(ch byte, fg, bg uint8, x, y uint32)

	// Palette returns the active color palette for this console.
	Palette() color.Palette

	// SetPaletteColor updates the color definition for the specified
	// palette index. Passing a color index greated than the number of
	// supported colors should be a no-op.
	SetPaletteColor(uint8, color.RGBA)
}

// FontSetter is an interface implemented by console devices that
// support loadable bitmap fonts.
//
// SetFont selects a bitmap font to be used by the console.
type FontSetter interface {
	SetFont(*font.Font)
}
