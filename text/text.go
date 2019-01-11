// Package text has text styles.
package text

import (
	"image/color"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomedium"
)

// A Style describes the color, font, and size of text.
type Style struct {
	// FG and BG are the foreground and background colors of the text.
	FG, BG color.Color
	// Face is the font face, describing the font and size.
	font.Face
}

// Merge returns other with any nil fields
// replaced by the corresponding field of sty.
func (sty Style) Merge(other Style) Style {
	if other.FG == nil {
		other.FG = sty.FG
	}
	if other.BG == nil {
		other.BG = sty.BG
	}
	if other.Face == nil {
		other.Face = sty.Face
	}
	return other
}

// Face returns a font.Face for a TTF of a given size at a given DPI.
func Face(ttf []byte, dpi float32, sizePt int) font.Face {
	f, err := truetype.Parse(gomedium.TTF)
	if err != nil {
		panic(err.Error())
	}
	return truetype.NewFace(f, &truetype.Options{
		Size: float64(sizePt),
		DPI:  float64(dpi * (72.0 / 96.0)),
	})
}
