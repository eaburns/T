package syntax

import (
	"github.com/eaburns/T/rope"
	"github.com/eaburns/T/text"
)

// A Highlight is a style applied to an addressed string of text.
type Highlight struct {
	// At is the addressed string.
	At [2]int64
	// Style is the style to apply to the string.
	text.Style
}

// Tokenizer returns tokens from a rope.
type Tokenizer interface {
	// NextToken next token or false.
	NextToken(rope.Rope) (Highlight, bool)
}
