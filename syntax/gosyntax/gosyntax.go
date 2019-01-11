// Package gosyntax implements a syntax.Highlighter for Go syntax.
package gosyntax

import (
	"image/color"

	"github.com/eaburns/T/syntax"
	"github.com/eaburns/T/text"
	"golang.org/x/image/font/gofont/gomedium"
)

// NewTokenizer returns a new syntax highlighter tokenizer for Go.
func NewTokenizer(dpi float32) syntax.Tokenizer {
	const (
		blockComment = `/[*]([^*]|[*][^/])*[*]/`
		lineComment  = `//.*`
		interpString = `("([^"\n]|\\["\n])*([^\\\n]|\\\n)")|""`
		rawString    = "`[^`]*`"
	)
	tok, err := syntax.NewRegexpTokenizer(
		syntax.Regexp{
			// TODO: Go highlighting for keywords doesn't handle non-ASCII word boundaries.
			// Ideally, instead of suing [^a-ZA-Z0-9_] for word bourdaries, re1 would have a unicode-aware \b.
			Regexp: `(^|[^a-zA-Z0-9_])(break|default|func|interface|select|case|defer|go|map|struct|chan|else|goto|package|switch|const|fallthrough|if|range|type|continue|for|import|return|var)([^a-zA-Z0-9_]|$)`,
			Group:  2,
			Style: text.Style{
				Face: text.Face(gomedium.TTF, dpi, 11 /* pt */),
			},
		},
		syntax.Regexp{
			Regexp: "(" + blockComment + ")|(" + lineComment + ")",
			Style: text.Style{
				FG: color.RGBA{R: 0x70, G: 0x70, B: 0x70, A: 0xFF},
			},
		},
		syntax.Regexp{
			Regexp: "(" + interpString + ")|(" + rawString + ")",
			Style: text.Style{
				FG: color.RGBA{R: 0x2F, G: 0x6F, B: 0x89, A: 0xFF},
			},
		},
		syntax.Regexp{
			// TODO: Go highlighting for runes doesn't handle the \000, \xFF, \uFFFF, or \UFFFFFFFF forms.
			Regexp: `'[^']'|'\\t'|'\\n'|'\\\\'|'\\''`,
			Style: text.Style{
				FG: color.RGBA{R: 0x2F, G: 0x6F, B: 0x89, A: 0xFF},
			},
		},
	)
	if err != nil {
		panic(err.Error())
	}
	return tok
}
