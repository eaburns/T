package syntax

import (
	"image/color"
	"testing"

	"github.com/eaburns/T/rope"
	"github.com/eaburns/T/text"
)

func TestRegexpNextToken(t *testing.T) {
	style0 := text.Style{FG: color.White}
	style1 := text.Style{FG: color.Black}
	tests := []struct {
		name   string
		res    []Regexp
		text   string
		want   Highlight
		wantOK bool
	}{
		{
			name: "no match",
			res: []Regexp{
				{
					Regexp: "abc",
					Style:  style0,
				},
				{
					Regexp: "def",
					Style:  style1,
				},
			},
			text:   "xyz",
			wantOK: false,
		},
		{
			name: "first match",
			res: []Regexp{
				{
					Regexp: "abc",
					Style:  style0,
				},
				{
					Regexp: "def",
					Style:  style1,
				},
			},
			text:   "abc",
			want:   Highlight{At: [2]int64{0, 3}, Style: style0},
			wantOK: true,
		},
		{
			name: "second match",
			res: []Regexp{
				{
					Regexp: "abc",
					Style:  style0,
				},
				{
					Regexp: "def",
					Style:  style1,
				},
			},
			text:   "def",
			want:   Highlight{At: [2]int64{0, 3}, Style: style1},
			wantOK: true,
		},
		{
			name: "skip prefix match",
			res: []Regexp{
				{
					Regexp: "abc",
					Style:  style0,
				},
				{
					Regexp: "def",
					Style:  style1,
				},
			},
			text:   "XXXXdef",
			want:   Highlight{At: [2]int64{4, 7}, Style: style1},
			wantOK: true,
		},
		{
			name: "sub-group match",
			res: []Regexp{
				{
					Regexp: "a(b+)c",
					Group:  1,
					Style:  style0,
				},
				{
					Regexp: "def",
					Style:  style1,
				},
			},
			text:   "XXXXabbbc",
			want:   Highlight{At: [2]int64{5, 8}, Style: style0},
			wantOK: true,
		},
	}
	for _, test := range tests {
		test := test
		t.Logf("%s\n", test.name)
		t.Run(test.name, func(t *testing.T) {
			tok, err := NewRegexpTokenizer(test.res...)
			if err != nil {
				t.Fatalf("NewRegexpTokenizer(...)=_,%v, want nil", err)
			}
			got, ok := tok.NextToken(rope.New(test.text))
			if got != test.want || ok != test.wantOK {
				t.Errorf("NextToken(%q)=%v,%v, want %v,%v",
					test.text, got, ok, test.want, test.wantOK)
			}
		})
	}
}

func TestNewRegexpTokenizerError(t *testing.T) {
	_, err := NewRegexpTokenizer()
	if err == nil {
		t.Error("NewRegexpTokenizer()=_,nil, wanted an error")
	}
	_, err = NewRegexpTokenizer(Regexp{Regexp: "(abc)\nx"})
	if err == nil {
		t.Error("NewRegexpTokenizer(\"|\")=_,nil, wanted an error")
	}
	_, err = NewRegexpTokenizer(Regexp{Regexp: "|" /* malformed | */})
	if err == nil {
		t.Error("NewRegexpTokenizer(\"|\")=_,nil, wanted an error")
	}
}
