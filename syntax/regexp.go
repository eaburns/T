package syntax

import (
	"errors"

	"github.com/eaburns/T/re1"
	"github.com/eaburns/T/rope"
	"github.com/eaburns/T/text"
)

type reTokenizer struct {
	regexps []Regexp
	re      *re1.Regexp
}

// A Regexp describes a syntactic element using a regular expression.
type Regexp struct {
	// Regexp is the re1 regular expression to match the element.
	Regexp string
	// Group is the numbered capture group of the element text.
	Group int
	Style text.Style
}

// NewRegexpTokenizer returns a new Tokenizer
// defined by a set of re1 regular expressions.
func NewRegexpTokenizer(regexps ...Regexp) (Tokenizer, error) {
	var rs []*re1.Regexp
	for i, regexp := range regexps {
		switch r, residual, err := re1.New(regexp.Regexp, re1.Opts{ID: i}); {
		case err != nil:
			return nil, err
		case residual != "":
			return nil, errors.New("expected end-of-input, got " + residual)
		default:
			rs = append(rs, r)
		}
	}
	re := re1.Union(rs...)
	if re == nil {
		return nil, errors.New("no regexps")
	}
	return &reTokenizer{regexps: regexps, re: re}, nil
}

func (t *reTokenizer) NextToken(txt rope.Rope) (Highlight, bool) {
	ms := t.re.FindInRope(txt, 0, txt.Len())
	if ms == nil {
		return Highlight{}, false
	}
	i := int(ms[len(ms)-1])
	j := 2 * t.regexps[i].Group
	h := Highlight{
		At:    [2]int64{ms[j], ms[j+1]},
		Style: t.regexps[i].Style,
	}
	return h, true
}
