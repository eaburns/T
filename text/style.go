package text

import (
	"image/color"

	"golang.org/x/image/font"
)

// A Style describes the color, font, and size of text.
type Style struct {
	// FG and BG are the foreground and background colors of the text.
	FG, BG color.Color
	// Face is the font face, describing the font and size.
	font.Face
}

// mergeStyles returns other with any nil fields
// replaced by the corresponding field of sty.
func (sty Style) merge(other Style) Style {
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

// A Highlight is a style applied to an addressed string of text.
type Highlight struct {
	// At is the addressed string.
	At [2]int64
	// Style is the style to apply to the string.
	Style
}
