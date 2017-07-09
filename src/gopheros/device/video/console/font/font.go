package font

var (
	// The list of available fonts.
	availableFonts []*Font
)

// Font describes a bitmap font that can be used by a console device.
type Font struct {
	// The name of the font
	Name string

	// The width of each glyph in pixels.
	GlyphWidth uint32

	// The height of each glyph in pixels.
	GlyphHeight uint32

	// The recommended console resolution for this font.
	RecommendedWidth  uint32
	RecommendedHeight uint32

	// Font priority (lower is better). When auto-detecting a font to use, the font with
	// the lowest priority will be preferred
	Priority uint32

	// The number of bytes describing a row in a glyph.
	BytesPerRow uint32

	// The font bitmap. Each character consists of BytesPerRow * Height
	// bytes where each bit indicates whether a pixel should be set to the
	// foreground or the background color.
	Data []byte
}

// FindByName looks up a font instance by name. If the font is not found then
// the function returns nil.
func FindByName(name string) *Font {
	for _, f := range availableFonts {
		if f.Name == name {
			return f
		}
	}

	return nil
}

// BestFit returns the best font from the available font list given the
// specified console dimensions. If multiple fonts match the dimension criteria
// then their priority attribute is used to select one.
//
// The algorithm for selecting the best font is the following:
//  For each font:
//    - calculate the sum of abs differences between the font recommended dimension
//      and the console dimensions.
//    - if the font score is lower than the current best font's score then the
//      font becomes the new best font.
//    - if the font score is equal to the current best font's score then the
//      font with the lowest priority becomes the new best font.
func BestFit(consoleWidth, consoleHeight uint32) *Font {
	var (
		best                           *Font
		bestDelta                      uint32
		absDeltaW, absDeltaH, absDelta uint32
	)

	for _, f := range availableFonts {
		if f.RecommendedWidth > consoleWidth {
			absDeltaW = f.RecommendedWidth - consoleWidth
		} else {
			absDeltaW = consoleWidth - f.RecommendedWidth
		}

		if f.RecommendedHeight > consoleHeight {
			absDeltaH = f.RecommendedHeight - consoleHeight
		} else {
			absDeltaH = consoleHeight - f.RecommendedHeight
		}

		absDelta = absDeltaW + absDeltaH

		if best == nil {
			best = f
			bestDelta = absDelta
			continue
		}

		if best.Priority < f.Priority || absDelta > bestDelta {
			continue
		}

		best = f
		bestDelta = absDelta
	}

	return best
}
