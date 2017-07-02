package console

// Attr defines a color attribute.
type Attr uint16

// The set of attributes that can be passed to Write().
const (
	Black Attr = iota
	Blue
	Green
	Cyan
	Red
	Magenta
	Brown
	LightGrey
	Grey
	LightBlue
	LightGreen
	LightCyan
	LightRed
	LightMagenta
	LightBrown
	White
)

// ScrollDir defines a scroll direction.
type ScrollDir uint8

// The supported list of scroll directions for the console Scroll() calls.
const (
	Up ScrollDir = iota
	Down
)

// The Console interface is implemented by objects that can function as physical consoles.
type Console interface {
	// Dimensions returns the width and height of the console in characters.
	Dimensions() (uint16, uint16)

	// Clear clears the specified rectangular region
	Clear(x, y, width, height uint16)

	// Scroll a particular number of lines to the specified direction.
	Scroll(dir ScrollDir, lines uint16)

	// Write a char to the specified location.
	Write(ch byte, attr Attr, x, y uint16)
}
