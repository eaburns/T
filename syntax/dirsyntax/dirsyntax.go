// Package dirsyntax implements a syntax.Highlighter for directory entries.
package dirsyntax

import (
	"image/color"

	"github.com/eaburns/T/syntax"
	"github.com/eaburns/T/text"
)

// NewTokenizer returns a new syntax highlighter tokenizer for directory entires.
func NewTokenizer(dpi float32) syntax.Tokenizer {
	tok, err := syntax.NewRegexpTokenizer(
		syntax.Regexp{
			Regexp: `.*/$`,
			Style: text.Style{
				FG: color.RGBA{R: 0x2F, G: 0x6F, B: 0x89, A: 0xFF},
			},
		},
		syntax.Regexp{
			Regexp: `(^|.+/)\..*`,
			Style: text.Style{
				FG: color.RGBA{R: 0x70, G: 0x70, B: 0x70, A: 0xFF},
			},
		},
	)
	if err != nil {
		panic(err.Error())
	}
	return tok
}
