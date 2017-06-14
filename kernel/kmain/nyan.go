// Original from https://github.com/NARKOZ/go-nyancat
// Hacked up to work in gopher-os.

package kmain

import (
	"github.com/achilleasa/gopher-os/kernel/driver/video/console"
	"github.com/achilleasa/gopher-os/kernel/hal"
)

// We do not have timing working yet, so just spin to get a sleep.
func sleep() {
	for i := 0; i < 1e8; i++ {
		// do nothing, just spin
	}
}

// This is a function, not a map as in the original because maps allocate
// RAM, which we cannot do yet.
func color(in rune) (a console.Attr) {
	switch in {
	case '+', '@':
		a = console.LightBrown
	case ',':
		a = console.Black
	case '-':
		a = console.Magenta
	case '#':
		a = console.Green
	case '.':
		a = console.White
	case '$', '%':
		a = console.LightRed
	case ';':
		a = console.LightMagenta
	case '&':
		a = console.Brown
	case '=':
		a = console.LightBlue
	case '>':
		a = console.Red
	case '*':
		a = console.Grey
	case '\'':
		a = console.Black
	default:
		a = console.White
	}
	return
}

func Nyan() {
	// Get TTY size
	vt := hal.ActiveTerminal
	w, h := vt.Dimensions()
	termWidth, termHeight := int(w), int(h)

	minRow := 0
	maxRow := len(frames[0])

	minCol := 0
	maxCol := len(frames[0][0])

	if maxRow > termHeight {
		minRow = (maxRow - termHeight) / 2
		maxRow = minRow + termHeight
	}

	if maxCol > termWidth {
		minCol = (maxCol - termWidth) / 2
		maxCol = minCol + termWidth
	}

	// Clear screen
	for y := 0; y < termHeight; y++ {
		for x := 0; x < termWidth; x++ {
			vt.WriteAtPosition(uint16(x), uint16(y), console.Black, byte(' '))
		}
	}

	for {
		for _, frame := range frames {
			// Print the next frame
			for x, line := range frame[minRow:maxRow] {
				for y, char := range line[minCol:maxCol] {
					vt.WriteAtPosition(uint16(y), uint16(x), color(char), byte(char))
				}
			}
			sleep()
		}
	}
}
