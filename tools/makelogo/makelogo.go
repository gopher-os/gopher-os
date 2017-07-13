package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
	"image"
	"image/color"
	"os"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

// The max number of colors that are allowed in a logo.
const maxColors = 16

func exit(err error) {
	fmt.Fprintf(os.Stderr, "[makelogo] error: %s\n", err.Error())
	os.Exit(1)
}

func buildPalette(img image.Image, transColor color.RGBA) ([]color.RGBA, map[color.RGBA]int, error) {
	var (
		palette         []color.RGBA
		colorToPalIndex = make(map[color.RGBA]int)
	)

	// Transparent color is always first
	palette = append(palette, transColor)
	colorToPalIndex[palette[0]] = 0

	bounds := img.Bounds()
	for y := 0; y < bounds.Size().Y; y++ {
		for x := 0; x < bounds.Size().X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			c := color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b)}
			if _, exists := colorToPalIndex[c]; exists {
				continue
			}

			colorToPalIndex[c] = len(colorToPalIndex)
			palette = append(palette, c)
		}
	}

	if got := len(palette); got > maxColors {
		return nil, nil, fmt.Errorf("logo should not contain more than %d colors; got %d", maxColors, got)
	}

	return palette, colorToPalIndex, nil
}

func genLogoFile(img image.Image, transColor color.RGBA, logoVar, align string) (string, error) {
	var (
		buf         bytes.Buffer
		bounds      = img.Bounds()
		logoVarName = fmt.Sprintf("%s%dx%d", logoVar, bounds.Size().X, bounds.Size().Y)
	)

	// Generate palette
	palette, colorToPalIndex, err := buildPalette(img, transColor)
	if err != nil {
		return "", err
	}

	// Output header
	fmt.Fprintf(&buf, `
package logo

import "image/color"

var (
%s = Image{
Width: %d,
Height: %d,
Align: %s,
TransparentIndex: 0,
`, logoVarName, bounds.Size().X, bounds.Size().Y, align)

	// Output palette
	fmt.Fprint(&buf, "Palette: []color.RGBA{\n")
	for _, c := range palette {
		fmt.Fprintf(&buf, "\t{R:%d, G:%d, B:%d},\n", c.R, c.G, c.B)
	}
	fmt.Fprint(&buf, "},\n")

	// Output image data
	fmt.Fprint(&buf, "Data: []uint8{\n")

	pixelIndex := 0
	for y := 0; y < bounds.Size().Y; y++ {
		for x := 0; x < bounds.Size().X; x, pixelIndex = x+1, pixelIndex+1 {
			if pixelIndex != 0 && pixelIndex%16 == 0 {
				buf.WriteByte('\n')
			}

			r, g, b, _ := img.At(x, y).RGBA()
			colorIndex := colorToPalIndex[color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b)}]

			fmt.Fprintf(&buf, "0x%x, ", colorIndex)
		}
	}
	fmt.Fprint(&buf, "\n},\n")

	// Footer
	fmt.Fprint(&buf, "}\n)\n")
	fmt.Fprintf(&buf, "func init(){\navailableLogos = append(availableLogos, &%s)\n}\n", logoVarName)

	return buf.String(), nil
}

func runTool() error {
	transR := flag.Uint("trans-r", 255, "the red component value for the transparent color")
	transG := flag.Uint("trans-g", 0, "the green component value for the transparent color")
	transB := flag.Uint("trans-b", 255, "the blue component value for the transparent color")
	logoVar := flag.String("var-name", "logo", "the name of the variable containing the logo data")
	align := flag.String("align", "center", "the horizontal alignment for the logo (left, center or right)")
	output := flag.String("out", "-", "a file to write the generated logo or - to output to STDOUT")
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, "makelogo: convert a png/jpg or gif image to a 8bpp console logo\n\n")
		fmt.Fprint(os.Stderr, "Usage: makelogo [options] image\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		exit(errors.New("missing image file argument"))
	}

	switch *align {
	case "left":
		*align = "AlignLeft"
	case "center":
		*align = "AlignCenter"
	case "right":
		*align = "AlignRight"
	default:
		exit(errors.New("invalid alignment specification; supported values are: left, center or right"))
	}

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		return err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return err
	}

	logoData, err := genLogoFile(
		img,
		color.RGBA{R: uint8(*transR), G: uint8(*transG), B: uint8(*transB)},
		*logoVar,
		*align,
	)
	if err != nil {
		return err
	}

	// Pretty-print generated file using go/printer
	fSet := token.NewFileSet()
	astFile, err := parser.ParseFile(fSet, "", logoData, parser.ParseComments)
	if err != nil {
		return err
	}

	switch *output {
	case "-":
		printer.Fprint(os.Stdout, fSet, astFile)
	default:
		fOut, err := os.Create(*output)
		if err != nil {
			return err
		}
		defer fOut.Close()

		printer.Fprint(fOut, fSet, astFile)
	}

	return nil
}

func main() {
	if err := runTool(); err != nil {
		exit(err)
	}
}
