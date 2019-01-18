package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"math"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/eaburns/T/rope"
	"github.com/eaburns/T/syntax"
	"github.com/eaburns/T/text"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/gofont/goregular"
)

var (
	A              = basicfont.Face7x13.Advance
	H              = basicfont.Face7x13.Height + basicfont.Face7x13.Descent
	testTextStyle1 = text.Style{FG: color.Black, BG: color.White, Face: basicfont.Face7x13}
	testTextStyle2 = text.Style{FG: color.White, BG: color.Black, Face: basicfont.Face7x13}
	testTextStyle3 = text.Style{FG: color.Black, BG: color.Black, Face: basicfont.Face7x13}
	testTextStyle4 = text.Style{FG: color.White, BG: color.White, Face: basicfont.Face7x13}
	testTextStyles = [4]text.Style{testTextStyle1, testTextStyle2, testTextStyle3, testTextStyle4}
	testSize       = image.Pt(200, 200)
	zp             = image.Pt(textPadPx, 0)
	testWin        = newTestWin()
)

func newTestWin() *Win {
	w := &Win{
		face:       basicfont.Face7x13,
		lineHeight: H,
	}
	c := NewCol(w)
	w.cols = []*Col{c}
	w.Col = c
	return w
}

func TestEdit(t *testing.T) {
	tests := []struct {
		in      string
		inDot   [2]int64
		ed      string
		want    string
		wantDot [2]int64
		err     string // regexp
	}{
		{
			in:      "abcdefgh",
			inDot:   [2]int64{5, 5},
			ed:      "100d",
			err:     "address out of range",
			wantDot: [2]int64{5, 5},
		},
		{
			in:      "line1\nline2\nline3",
			inDot:   [2]int64{6, 12},
			ed:      ".d",
			want:    "line1\nline3",
			wantDot: [2]int64{6, 6},
		},
		{
			in:      "line1\nline2\nline3",
			inDot:   [2]int64{0, 0},
			ed:      "2d",
			want:    "line1\nline3",
			wantDot: [2]int64{6, 6},
		},
	}
	for _, test := range tests {
		b := NewTextBox(testWin, testTextStyles, testSize)
		b.SetText(rope.New(test.in))
		b.dots[1].At = test.inDot

		switch _, err := b.Edit(test.ed); {
		case test.err != "" && err != nil:
			if !match(test.err, err.Error()) {
				t.Errorf("(%q).Edit(%q)=%v, wanted matching %q",
					test.in, test.ed, err, test.err)
			}

		case test.err != "" && err == nil:
			t.Errorf("(%q).Edit(%q)=nil, wanted matching %q",
				test.in, test.ed, test.err)

		case test.err == "" && err != nil:
			t.Errorf("(%q).Edit(%q)=%v, wanted nil", test.in, test.ed, err)

		case test.err == "" && err == nil:
			if got := b.text.String(); got != test.want {
				t.Errorf("(%q).Edit(%q), text=%q, want %q",
					test.in, test.ed, got, test.want)
			}
		}
		if b.dots[1].At != test.wantDot {
			t.Errorf("(%q).Edit(%q), dot=%v, want %v",
				test.in, test.ed, b.dots[1].At, test.wantDot)
		}
	}
}

func match(re, str string) bool {
	return regexp.MustCompile(re).MatchString(str)
}

func TestTextHeight(t *testing.T) {
	const str = "Hello,\nWorld!"
	b := NewTextBox(testWin, testTextStyles, testSize)
	b.SetText(rope.New(str))
	if h := b.textHeight(); h != 2*H {
		t.Errorf("(%q).TextHeight()=%d, want %d", str, h, 2*H)
	}
}

func TestTextHeightTrailingNewline(t *testing.T) {
	const str = "Hello,\nWorld!\n"
	b := NewTextBox(testWin, testTextStyles, testSize)
	b.SetText(rope.New(str))
	if h := b.textHeight(); h != 3*H {
		t.Errorf("(%q).TextHeight()=%d, want %d", str, h, 3*H)
	}
}

func TestClick1(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		hi      []syntax.Highlight
		pt      image.Point
		wantDot [2]int64
	}{
		{
			name:    "empty",
			in:      "",
			pt:      image.Pt(-1, 5),
			wantDot: [2]int64{0, 0},
		},
		{
			name:    "negative x",
			in:      "line1\nline2\n",
			pt:      image.Pt(-1, 5),
			wantDot: [2]int64{0, 0},
		},
		{
			name:    "negative y",
			in:      "line1\nline2\n",
			pt:      image.Pt(5, -1),
			wantDot: [2]int64{0, 0},
		},
		{
			name:    "negative xy",
			in:      "line1\nline2\n",
			pt:      image.Pt(-1, -1),
			wantDot: [2]int64{0, 0},
		},
		{
			name:    "zero",
			in:      "line1\nline2\n",
			pt:      image.ZP,
			wantDot: [2]int64{0, 0},
		},
		{
			name:    "middle of rune 0",
			in:      "line1\nline2\n",
			pt:      image.Pt(A/2, H/2),
			wantDot: [2]int64{0, 0},
		},
		{
			name:    "middle of rune 2",
			in:      "line1\nline2\n",
			pt:      image.Pt(A+A/2, H/2),
			wantDot: [2]int64{1, 1},
		},
		{
			name:    "middle of rune 2",
			in:      "line1\nline2\n",
			pt:      image.Pt(2*A+A/2, H/2),
			wantDot: [2]int64{2, 2},
		},
		{
			name:    "end of non-final line",
			in:      "line1\nline2\n",
			pt:      image.Pt(5*A, H/2),
			wantDot: [2]int64{5, 5},
		},
		{
			name:    "beyond end of non-final line",
			in:      "line1\nline2\n",
			pt:      image.Pt(5*A+20, H/2),
			wantDot: [2]int64{5, 5},
		},
		{
			name:    "beginning of second line",
			in:      "line1\nline2\n",
			pt:      image.Pt(A/2, H+H/2),
			wantDot: [2]int64{6, 6},
		},
		{
			name:    "second line second rune",
			in:      "line1\nline2\n",
			pt:      image.Pt(A+A/2, H+H/2),
			wantDot: [2]int64{7, 7},
		},
		{
			name:    "end within final line",
			in:      "line1\nline2\n",
			pt:      image.Pt(5*A, H+H/2),
			wantDot: [2]int64{11, 11},
		},
		{
			name:    "beyond end within final line",
			in:      "line1\nline2\n",
			pt:      image.Pt(5*A+20, H+H/2),
			wantDot: [2]int64{11, 11},
		},
		{
			name:    "beyond final line with trailing newline",
			in:      "line1\nline2\n",
			pt:      image.Pt(A/2, 2*H),
			wantDot: [2]int64{12, 12},
		},
		{
			name:    "beyond final line without trailing newline",
			in:      "line1\nline2",
			pt:      image.Pt(A/2, 3*H),
			wantDot: [2]int64{11, 11},
		},
		{
			name: "with highlight",
			in:   "line1\nline2\nline3",
			hi: []syntax.Highlight{
				{
					At:    [2]int64{0, 4}, // "line" in line1
					Style: testTextStyle2,
				},
				{
					At:    [2]int64{6, 7}, // "l" in line2
					Style: testTextStyle3,
				},
				{
					At:    [2]int64{12, 16}, // "line" in line3
					Style: testTextStyle4,
				},
			},
			pt:      image.Pt(A+A/2, H+H/2), // line 1, col 1
			wantDot: [2]int64{7, 7},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			b := NewTextBox(testWin, testTextStyles, testSize)
			b.SetText(rope.New(test.in))
			b.highlight = test.hi
			b.Click(test.pt.Add(zp), 1)
			if b.dots[1].At != test.wantDot {
				t.Errorf("got dot=%v, want dot=%v", b.dots[1].At, test.wantDot)
			}
		})
	}
}

var dragTests = []struct {
	name    string
	in      string
	hi      []syntax.Highlight
	pt      [2]image.Point
	wantDot [2]int64
}{
	{
		name: "empty",
		in:   "",
		pt: [2]image.Point{
			{A / 2, H / 2},
			{A + A/2, H + H/2},
		},
		wantDot: [2]int64{0, 0},
	},
	{
		name: "negative x",
		in:   "line1\nline2\n",
		pt: [2]image.Point{
			{-1, 5},
			{-2, 5},
		},
		wantDot: [2]int64{0, 0},
	},
	{
		name: "negative y",
		in:   "line1\nline2\n",
		pt: [2]image.Point{
			{5, -1},
			{5, -2},
		},
		wantDot: [2]int64{0, 0},
	},
	{
		name: "negative xy",
		in:   "line1\nline2\n",
		pt: [2]image.Point{
			{-5, -1},
			{-4, -2},
		},
		wantDot: [2]int64{0, 0},
	},
	{
		name: "no move",
		in:   "line1\nline2\n",
		pt: [2]image.Point{
			{A + A/2, H / 2},
			{A + A/2, H / 2},
		},
		wantDot: [2]int64{1, 1},
	},
	{
		name: "move within rune",
		in:   "line1\nline2\n",
		pt: [2]image.Point{
			{A + A/2, H / 2},
			{A + A/2 + A/4, H/2 + H/4},
		},
		wantDot: [2]int64{1, 1},
	},
	{
		name: "select rune 0",
		in:   "line1\nline2\n",
		pt: [2]image.Point{
			{A / 2, H / 2},
			{A + A/2, H / 2},
		},
		wantDot: [2]int64{0, 1},
	},
	{
		name: "select rune 1",
		in:   "line1\nline2\n",
		pt: [2]image.Point{
			{A + A/2, H / 2},
			{2*A + A/2, H / 2},
		},
		wantDot: [2]int64{1, 2},
	},
	{
		name: "select last rune",
		in:   "line1\nline2",
		pt: [2]image.Point{
			{5*A + A/2, H + H/2},
			{6*A + A/2, H + H/2},
		},
		wantDot: [2]int64{11, 11},
	},
	{
		// If we select beyond the end of the line,
		// we don't select the newline, but  the rune before it.
		name: "select beyond end of line",
		in:   "line1\nline2\n",
		pt: [2]image.Point{
			{A / 2, H / 2},
			{12*A + A/2, H / 2}, // only 6 runes on this line.
		},
		wantDot: [2]int64{0, 5},
	},
	{
		name: "select non-last newline",
		in:   "line1\nline2\n",
		pt: [2]image.Point{
			{A / 2, H / 2},
			{0, H + H/2},
		},
		wantDot: [2]int64{0, 6},
	},
	{
		name: "select last rune newline",
		in:   "line1\nline2\n",
		pt: [2]image.Point{
			{A / 2, H + H/2},
			{0, 2*H + H/2},
		},
		wantDot: [2]int64{6, 12},
	},
	{
		name: "select end of file",
		in:   "line1\nline2\n",
		pt: [2]image.Point{
			{0, 2*H + H/2},
			{A, 2*H + H/2},
		},
		wantDot: [2]int64{12, 12},
	},
	{
		name: "whole file",
		in:   "line1\nline2\n",
		pt: [2]image.Point{
			{0, 0},
			{A, 2*H + H/2},
		},
		wantDot: [2]int64{0, 12},
	},
	{
		name: "backwards",
		in:   "line1\nline2\n",
		pt: [2]image.Point{
			{A, 2*H + H/2},
			{0, 0},
		},
		wantDot: [2]int64{0, 12},
	},
	{
		name: "with highlight",
		in:   "line1\nline2\nline3",
		hi: []syntax.Highlight{
			{
				At:    [2]int64{0, 4}, // "line" in line1
				Style: testTextStyle2,
			},
			{
				At:    [2]int64{6, 7}, // "l" in line2
				Style: testTextStyle3,
			},
			{
				At:    [2]int64{12, 16}, // "line" in line3
				Style: testTextStyle4,
			},
		},
		pt: [2]image.Point{
			{A + A/2, H + H/2},     // line 1, col 1
			{2*A + A/2, 2*H + H/2}, // line 2, col 3
		},
		wantDot: [2]int64{7, 14},
	},
}

func TestDrag1(t *testing.T) {
	for _, test := range dragTests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			b := NewTextBox(testWin, testTextStyles, testSize)
			b.SetText(rope.New(test.in))
			b.highlight = test.hi
			b.Click(test.pt[0].Add(zp), 1)
			b.Move(test.pt[1].Add(zp))
			if b.dots[1].At != test.wantDot {
				t.Errorf("got dot=%v, want dot=%v", b.dots[1].At, test.wantDot)
			}
		})
	}
}

func TestDrag2(t *testing.T) {
	for _, test := range dragTests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			b := NewTextBox(testWin, testTextStyles, testSize)
			b.SetText(rope.New(test.in))
			b.highlight = test.hi
			b.Click(test.pt[0].Add(zp), 2)
			b.Move(test.pt[1].Add(zp))
			if b.dots[2].At != test.wantDot {
				t.Errorf("drag: got dot=%v, want dot=%v", b.dots[2].At, test.wantDot)
			}

			// Unclicking button > 1resets the selection back to 0,0.
			b.Click(test.pt[1].Add(zp), -2)
			if b.dots[2].At != [2]int64{} {
				t.Errorf("unclick: got dot=%v, want dot={}", b.dots[2].At)
			}
		})
	}
}

var fixedTime = func() time.Time {
	return time.Date(2018, 12, 15, 10, 35, 36, 0, time.UTC)
}

func TestDoubleClick1(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		pt      image.Point
		wantDot [2]int64
	}{
		{
			name:    "empty",
			in:      "",
			pt:      image.Point{5, 5},
			wantDot: [2]int64{0, 0},
		},
		{
			name:    "select word bof to eof",
			in:      "012345",
			pt:      image.Point{A, H / 2},
			wantDot: [2]int64{0, 6},
		},
		{
			name:    "select word bof to punct",
			in:      "012345...",
			pt:      image.Point{A, H / 2},
			wantDot: [2]int64{0, 6},
		},
		{
			name:    "select word bof to space",
			in:      "012345 ",
			pt:      image.Point{A, H / 2},
			wantDot: [2]int64{0, 6},
		},
		{
			name:    "select word punct to eof",
			in:      "...012345",
			pt:      image.Point{5 * A, H / 2},
			wantDot: [2]int64{3, 9},
		},
		{
			name:    "select word space to eof",
			in:      "   012345",
			pt:      image.Point{5 * A, H / 2},
			wantDot: [2]int64{3, 9},
		},
		{
			name:    "select word punct to punct",
			in:      "...012345---",
			pt:      image.Point{5 * A, H / 2},
			wantDot: [2]int64{3, 9},
		},
		{
			name:    "select word space to space",
			in:      "   012345\n",
			pt:      image.Point{5 * A, H / 2},
			wantDot: [2]int64{3, 9},
		},
		{
			name:    "just before eol selects word",
			in:      "line1\nline2\nline3",
			pt:      image.Point{5*A - 1, H / 2},
			wantDot: [2]int64{0, 5},
		},
		{
			name:    "select line start bof to newline",
			in:      "line1\nline2\nline3",
			pt:      image.Point{0, H / 2},
			wantDot: [2]int64{0, 6},
		},
		{
			name:    "select line end bof to newline",
			in:      "line1\nline2\nline3",
			pt:      image.Point{5 * A, H / 2},
			wantDot: [2]int64{0, 6},
		},
		{
			name:    "select line start newline to newline",
			in:      "line1\nline2\nline3",
			pt:      image.Point{0, H + H/2},
			wantDot: [2]int64{6, 12},
		},
		{
			name:    "select line end newline to newline",
			in:      "line1\nline2\nline3",
			pt:      image.Point{5 * A, H + H/2},
			wantDot: [2]int64{6, 12},
		},
		{
			name:    "select line start newline to eof",
			in:      "line1\nline2\nline3",
			pt:      image.Point{0, H + 2*H/2},
			wantDot: [2]int64{12, 17},
		},
		{
			name:    "select line end newline to eof",
			in:      "line1\nline2\nline3",
			pt:      image.Point{5 * A, 2*H + H/2},
			wantDot: [2]int64{12, 17},
		},
		{
			name:    "select forward delim",
			in:      "012{0123456789}012",
			pt:      image.Point{4 * A, H / 2},
			wantDot: [2]int64{4, 14},
		},
		{
			name:    "select forward delim start==end",
			in:      "012`0123456789`012",
			pt:      image.Point{4 * A, H / 2},
			wantDot: [2]int64{4, 14},
		},
		{
			name:    "select forward delim with nesting",
			in:      "012{{{{{{}}}}}}012",
			pt:      image.Point{4 * A, H / 2},
			wantDot: [2]int64{4, 14},
		},
		{
			name:    "select forward delim not terminated",
			in:      "012{0123456789",
			pt:      image.Point{4 * A, H / 2},
			wantDot: [2]int64{4, 4},
		},
		{
			name:    "select reverse delim",
			in:      "012{0123456789}012",
			pt:      image.Point{14 * A, H / 2},
			wantDot: [2]int64{4, 14},
		},
		{
			name:    "select reverse delim start==end",
			in:      "012`0123456789`012",
			pt:      image.Point{14 * A, H / 2},
			wantDot: [2]int64{4, 14},
		},
		{
			name:    "select reverse delim with nesting",
			in:      "012{{{{{{}}}}}}012",
			pt:      image.Point{14 * A, H / 2},
			wantDot: [2]int64{4, 14},
		},
		{
			name:    "select reverse delim not terminated",
			in:      "0123456789}0123",
			pt:      image.Point{10 * A, H / 2},
			wantDot: [2]int64{10, 10},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			b := NewTextBox(testWin, testTextStyles, testSize)
			b.SetText(rope.New(test.in))
			b.now = fixedTime
			b.Click(test.pt.Add(zp), 1)
			b.Click(test.pt.Add(zp), -1)
			b.Click(test.pt.Add(zp), 1)
			if b.dots[1].At != test.wantDot {
				t.Errorf("got dot=%v, want dot=%v", b.dots[1].At, test.wantDot)
			}
		})
	}
}

func TestDir1(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		dot     [2]int64
		x, y    int
		wantDot [2]int64
	}{
		{
			name:    "empty up",
			in:      "",
			dot:     [2]int64{0, 0},
			y:       -1,
			wantDot: [2]int64{0, 0},
		},
		{
			name:    "empty down",
			in:      "",
			dot:     [2]int64{0, 0},
			y:       +1,
			wantDot: [2]int64{0, 0},
		},
		{
			name:    "empty left",
			in:      "",
			dot:     [2]int64{0, 0},
			x:       -1,
			wantDot: [2]int64{0, 0},
		},
		{
			name:    "empty right",
			in:      "",
			dot:     [2]int64{0, 0},
			x:       +1,
			wantDot: [2]int64{0, 0},
		},
		{
			name:    "up",
			in:      "0123\n5678",
			dot:     [2]int64{6, 6},
			y:       -1,
			wantDot: [2]int64{1, 1},
		},
		{
			name:    "up from selection",
			in:      "0123\n5678",
			dot:     [2]int64{6, 9},
			y:       -1,
			wantDot: [2]int64{1, 1},
		},
		{
			name:    "up stop at bof",
			in:      "0123\n5678",
			dot:     [2]int64{1, 1},
			y:       -1,
			wantDot: [2]int64{0, 0},
		},
		{
			name:    "down",
			in:      "0123\n5678",
			dot:     [2]int64{1, 1},
			y:       +1,
			wantDot: [2]int64{6, 6},
		},
		{
			name:    "down from selection",
			in:      "0123\n5678",
			dot:     [2]int64{1, 3},
			y:       +1,
			wantDot: [2]int64{6, 6},
		},
		{
			name:    "down stop at eof",
			in:      "0123\n5678",
			dot:     [2]int64{6, 6},
			y:       +1,
			wantDot: [2]int64{9, 9},
		},
		{
			name:    "left",
			in:      "01234",
			dot:     [2]int64{3, 3},
			x:       -1,
			wantDot: [2]int64{2, 2},
		},
		{
			name:    "left from selection",
			in:      "01234",
			dot:     [2]int64{2, 4},
			x:       -1,
			wantDot: [2]int64{2, 2},
		},
		{
			name:    "left stops at bof",
			in:      "01234",
			dot:     [2]int64{0, 0},
			x:       -1,
			wantDot: [2]int64{0, 0},
		},
		{
			name:    "right",
			in:      "01234",
			dot:     [2]int64{3, 3},
			x:       +1,
			wantDot: [2]int64{4, 4},
		},
		{
			name:    "right from selection",
			in:      "01234",
			dot:     [2]int64{1, 3},
			x:       +1,
			wantDot: [2]int64{3, 3},
		},
		{
			name:    "right stops at eof",
			in:      "01234",
			dot:     [2]int64{5, 5},
			x:       +1,
			wantDot: [2]int64{5, 5},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			b := NewTextBox(testWin, testTextStyles, testSize)
			b.SetText(rope.New(test.in))
			b.dots[1].At = test.dot
			b.Dir(test.x, test.y)
			if b.dots[1].At != test.wantDot {
				t.Errorf("Dir(%d, %d) dot=%v, want %v\n",
					test.x, test.y, b.dots[1], test.wantDot)
			}
		})
	}
}

func TestUpPreservesColumn(t *testing.T) {
	b := NewTextBox(testWin, testTextStyles, testSize)
	b.SetText(rope.New("012\n4\n678"))
	b.dots[1].At = [2]int64{8, 8}

	// Column 3, up to a 1-column line.
	b.Dir(0, -1)
	if b.dots[1].At != [2]int64{5, 5} {
		t.Fatalf("got %v, want 5,5", b.dots[1].At)
	}

	// Then up to another 3-column line.
	b.Dir(0, -1)
	if b.dots[1].At != [2]int64{2, 2} {
		t.Errorf("got %v, want 2,2", b.dots[1].At)
	}
}

func TestDownPreservesColumn(t *testing.T) {
	b := NewTextBox(testWin, testTextStyles, testSize)
	b.SetText(rope.New("012\n4\n678"))
	b.dots[1].At = [2]int64{3, 3}

	// Column 3, down to a 1-column line.
	b.Dir(0, +1)
	if b.dots[1].At != [2]int64{5, 5} {
		t.Fatalf("got %v, want 5,5", b.dots[1].At)
	}

	// Then down to another 3-column line.
	b.Dir(0, +1)
	if b.dots[1].At != [2]int64{8, 8} {
		t.Errorf("got %v, want 8,8", b.dots[1].At)
	}
}

var lines500 = strings.Repeat("\n", 500)

func TestWheelUp(t *testing.T) {
	text := rope.New(lines500)
	b := NewTextBox(testWin, testTextStyles, testSize)
	b.SetText(text)
	var now time.Time
	b.now = func() time.Time {
		n := now
		now = now.Add(wheelScrollDuration)
		return n
	}
	b.at = text.Len() - 2

	b.Wheel(image.ZP, 0, -1)
	if want := text.Len() - 1; b.at != want {
		t.Fatalf("WheelUp, at=%d, wanted %d", b.at, want)
	}

	b.Wheel(image.ZP, 0, -1)
	if want := text.Len(); b.at != want {
		t.Fatalf("WheelUp WheelUp, at=%d, wanted %d", b.at, want)
	}

	b.Wheel(image.ZP, 0, -1)
	if want := text.Len(); b.at != want {
		t.Errorf("WheelUp WheelUp WheelUp, at=%d, wanted %d", b.at, want)
	}
}

func TestWheelDown(t *testing.T) {
	text := rope.New(lines500)
	b := NewTextBox(testWin, testTextStyles, testSize)
	b.SetText(text)
	var now time.Time
	b.now = func() time.Time {
		n := now
		now = now.Add(wheelScrollDuration)
		return n
	}
	b.at = 2

	b.Wheel(image.ZP, 0, +1)
	if b.at != 1 {
		t.Fatalf("WheelDown, at=%d, wanted 1", b.at)
	}

	b.Wheel(image.ZP, 0, +1)
	if b.at != 0 {
		t.Fatalf("WheelDown WheelDown, at=%d, wanted 0", b.at)
	}

	b.Wheel(image.ZP, 0, +1)
	if b.at != 0 {
		t.Errorf("WheelDown WheelDown WheelDown, at=%d, wanted 0", b.at)
	}
}

func TestDragScrollUp(t *testing.T) {
	text := rope.New(lines500)
	b := NewTextBox(testWin, testTextStyles, testSize)
	b.SetText(text)
	var now time.Time
	b.now = func() time.Time {
		n := now
		now = now.Add(dragScrollDuration)
		return n
	}
	b.at = 2
	b.Focus(true)
	b.Click(image.Pt(A/2, H/2), 1)
	b.Move(image.Pt(A/2, -H))

	b.Tick()
	if want := 1; b.at != int64(want) {
		t.Fatalf("Tick 1, at=%d, wanted %d", b.at, want)
	}

	b.Tick()
	if want := 0; b.at != int64(want) {
		t.Fatalf("Tick 2, at=%d, wanted %d", b.at, want)
	}

	b.Tick()
	if want := 0; b.at != int64(want) {
		t.Fatalf("Tick 3, at=%d, wanted %d", b.at, want)
	}
}

func TestDragScrollDown(t *testing.T) {
	text := rope.New(lines500)
	b := NewTextBox(testWin, testTextStyles, testSize)
	b.SetText(text)
	var now time.Time
	b.now = func() time.Time {
		n := now
		now = now.Add(dragScrollDuration)
		return n
	}
	at := text.Len() - int64(b.size.Y/H)
	b.at = at - 2
	b.Focus(true)
	b.Click(image.Pt(A/2, b.size.Y-H/2), 1)
	b.Move(image.Pt(A/2, b.size.Y+H))

	b.Tick()
	if want := at - 1; b.at != int64(want) {
		t.Fatalf("Tick 1, at=%d, wanted %d", b.at, want)
	}

	b.Tick()
	if want := at; b.at != int64(want) {
		t.Fatalf("Tick 2, at=%d, wanted %d", b.at, want)
	}

	b.Tick()
	if want := at; b.at != int64(want) {
		t.Fatalf("Tick 3, at=%d, wanted %d", b.at, want)
	}
}

func TestPageUp(t *testing.T) {
	text := rope.New(lines500)
	b := NewTextBox(testWin, testTextStyles, testSize)
	b.SetText(text)
	b.at = int64(2.5 * float64(pageSize(b)))

	b.Dir(0, -5)
	if want := int64(1.5 * float64(pageSize(b))); b.at != want {
		t.Fatalf("PageUp, at=%d, wanted %d", b.at, want)
	}

	b.Dir(0, -5)
	if want := int64(0.5 * float64(pageSize(b))); b.at != want {
		t.Fatalf("PageUp PageUp, at=%d, wanted %d", b.at, want)
	}

	b.Dir(0, -5)
	if b.at != 0 {
		t.Errorf("PageUp  PageUp PageUp, at=%d, wanted 0", b.at)
	}
}

func TestPageDown(t *testing.T) {
	text := rope.New(lines500)
	b := NewTextBox(testWin, testTextStyles, testSize)
	b.SetText(text)
	b.at = text.Len() - int64(2.5*float64(pageSize(b)))

	b.Dir(0, +5)
	if want := text.Len() - int64(1.5*float64(pageSize(b))); b.at != want {
		t.Fatalf("PageDown, at=%d, wanted %d", b.at, want)
	}

	b.Dir(0, +5)
	if want := text.Len() - int64(0.5*float64(pageSize(b))); b.at != want {
		t.Fatalf("PageDown PageDown, at=%d, wanted %d", b.at, want)
	}

	b.Dir(0, +5)
	if b.at != text.Len() {
		t.Errorf("PageDown PageDown PageDown, at=%d, wanted %d",
			b.at, text.Len())
	}
}

func TestHome(t *testing.T) {
	text := rope.New(lines500)
	b := NewTextBox(testWin, testTextStyles, testSize)
	b.SetText(text)
	b.at = text.Len()

	b.Dir(0, math.MinInt16)
	if b.at != 0 {
		t.Fatalf("Home, at=%d, wanted 0", b.at)
	}

	b.Dir(0, math.MinInt16)
	if b.at != 0 {
		t.Errorf("Home Home, at=%d, wanted 0", b.at)
	}
}

func TestEnd(t *testing.T) {
	text := rope.New(lines500)
	b := NewTextBox(testWin, testTextStyles, testSize)
	b.SetText(text)
	b.at = text.Len()

	end := text.Len() - int64(pageSize(b))
	b.Dir(0, math.MaxInt16)
	if b.at != text.Len()-int64(pageSize(b)) {
		t.Fatalf("End, at=%d, wanted %d", b.at, end)
	}

	b.Dir(0, math.MaxInt16)
	if b.at != text.Len()-int64(pageSize(b)) {
		t.Errorf("End End, at=%d, wanted %d", b.at, end)
	}
}

func TestType(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		dot     [2]int64
		r       rune
		want    string
		wantDot [2]int64
	}{
		{
			name:    "empty",
			in:      "",
			dot:     [2]int64{0, 0},
			r:       'a',
			want:    "a",
			wantDot: [2]int64{1, 1},
		},
		{
			name:    "non-ascii",
			in:      "",
			dot:     [2]int64{0, 0},
			r:       '☺',
			want:    "☺",
			wantDot: [2]int64{3, 3}, // ☺ is 3 bytes in utf8.
		},
		{
			name:    "edit delimiter",
			in:      "",
			dot:     [2]int64{0, 0},
			r:       '/',
			want:    "/",
			wantDot: [2]int64{1, 1},
		},
		{
			name:    "newline",
			in:      "",
			dot:     [2]int64{0, 0},
			r:       '\n',
			want:    "\n",
			wantDot: [2]int64{1, 1},
		},
		{
			name:    "append",
			in:      "012",
			dot:     [2]int64{3, 3},
			r:       '3',
			want:    "0123",
			wantDot: [2]int64{4, 4},
		},
		{
			name:    "insert",
			in:      "012",
			dot:     [2]int64{1, 1},
			r:       '3',
			want:    "0312",
			wantDot: [2]int64{2, 2},
		},
		{
			name:    "change",
			in:      "0123",
			dot:     [2]int64{1, 3},
			r:       'a',
			want:    "0a3",
			wantDot: [2]int64{2, 2},
		},
		{
			name:    "backspace",
			in:      "012",
			dot:     [2]int64{3, 3},
			r:       '\b',
			want:    "01",
			wantDot: [2]int64{2, 2},
		},
		{
			name:    "backspace from selection",
			in:      "0123",
			dot:     [2]int64{1, 3},
			r:       '\b',
			want:    "03",
			wantDot: [2]int64{1, 1},
		},
		{
			name:    "backspace stops at bof",
			in:      "",
			dot:     [2]int64{0, 0},
			r:       '\b',
			want:    "",
			wantDot: [2]int64{0, 0},
		},
		{
			name:    "delete",
			in:      "012",
			dot:     [2]int64{0, 0},
			r:       del,
			want:    "12",
			wantDot: [2]int64{0, 0},
		},
		{
			// same as delete
			name:    "escape",
			in:      "012",
			dot:     [2]int64{0, 0},
			r:       esc,
			want:    "12",
			wantDot: [2]int64{0, 0},
		},
		{
			name:    "delete from selection",
			in:      "0123",
			dot:     [2]int64{1, 3},
			r:       del,
			want:    "03",
			wantDot: [2]int64{1, 1},
		},
		{
			name:    "delete stops at eof",
			in:      "",
			dot:     [2]int64{0, 0},
			r:       del,
			want:    "",
			wantDot: [2]int64{0, 0},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			b := NewTextBox(testWin, testTextStyles, testSize)
			b.SetText(rope.New(test.in))
			b.dots[1].At = test.dot
			b.Rune(test.r)
			if got := b.text.String(); got != test.want {
				t.Errorf("(%q @ %v).Type(%q) text=%q, want %q",
					test.in, test.dot, test.r, got, test.want)
			}
			if b.dots[1].At != test.wantDot {
				t.Errorf("(%q @ %v).Type(%q) dot=%v, want %v",
					test.in, test.dot, test.r, b.dots[1].At, test.wantDot)
			}
		})
	}
}

// This is testing a bug where clicking below and to the right
// of the last line of text  with >1 spans would cause dot to be set
// out-of-bounds of the text.
func TestClickAfterLastLineSelected(t *testing.T) {
	const str = "Hello"
	b := NewTextBox(testWin, testTextStyles, testSize)
	b.SetText(rope.New(str))
	b.syntax = []syntax.Highlight{
		{At: [2]int64{0, 3}, Style: testTextStyle1},
	}
	b.Click(image.Pt(10*A+A/2, 4*H+H/2), 1)

	n := int64(len(str))
	want := [2]int64{n, n}
	if b.dots[1].At != want {
		t.Errorf("dot=%v, wanted=%v\n", b.dots[1].At, want)
	}
}

const testText = `	"Jabberwocky"

’Twas brillig, and the slithy toves
Did gyre and gimble in the wabe;
All mimsy were the borogoves,
And the mome raths outgrabe.

“Beware the Jabberwock, my son!
The jaws that bite, the claws that catch!
Beware the Jubjub bird, and shun
The frumious Bandersnatch!”

He took his vorpal sword in hand:
Long time the manxome foe he sought—
So rested he by the Tumtum tree,
And stood awhile in thought.

And as in uffish thought he stood,
The Jabberwock, with eyes of flame,
Came whiffling through the tulgey wood,
And burbled as it came!

One, two! One, two! And through and through
The vorpal blade went snicker-snack!
He left it dead, and with its head
He went galumphing back.

“And hast thou slain the Jabberwock?
Come to my arms, my beamish boy!
O frabjous day! Callooh! Callay!”
He chortled in his joy.

’Twas brillig, and the slithy toves
Did gyre and gimble in the wabe;
All mimsy were the borogoves,
And the mome raths outgrabe.`

func TestDraw(t *testing.T) {
	goregular, err := truetype.Parse(goregular.TTF)
	if err != nil {
		t.Fatalf(err.Error())
	}
	size := image.Pt(300, 750)
	style := text.Style{
		FG:   color.RGBA{R: 0x10, G: 0x28, B: 0x34, A: 0xFF},
		BG:   color.RGBA{R: 0xFE, G: 0xF0, B: 0xE6, A: 0xFF},
		Face: truetype.NewFace(goregular, &truetype.Options{Size: 16}),
	}
	style1 := text.Style{
		FG:   color.RGBA{R: 0x10, G: 0x28, B: 0x34, A: 0xFF},
		BG:   color.RGBA{R: 0xB6, G: 0xDA, B: 0xFD, A: 0xFF},
		Face: truetype.NewFace(goregular, &truetype.Options{Size: 16}),
	}
	styles := [4]text.Style{style, style1, style, style}
	b := NewTextBox(testWin, styles, size)
	b.SetText(rope.New(testText))
	b.dots[1].At = [2]int64{7, 12}
	b.dots[1].Style = style1
	img := image.NewRGBA(image.Rectangle{Max: size})
	b.Draw(true, img)
	goldenImageTest(img, t)
}

func TestDrawEmptyText(t *testing.T) {
	size := image.Pt(100, 100)
	b := NewTextBox(testWin, testTextStyles, size)
	b.Focus(true)
	img := image.NewRGBA(image.Rectangle{Max: size})
	b.Draw(true, img)
	goldenImageTest(img, t)
}

func TestDrawCursorMidLine(t *testing.T) {
	size := image.Pt(100, 100)
	b := NewTextBox(testWin, testTextStyles, size)
	b.SetText(rope.New("Hello"))
	b.Focus(true)
	img := image.NewRGBA(image.Rectangle{Max: size})
	b.dots[1].At = [2]int64{1, 1}
	b.Draw(true, img)
	goldenImageTest(img, t)
}

func TestDrawCursorAtEndOfLastLine(t *testing.T) {
	size := image.Pt(100, 100)
	b := NewTextBox(testWin, testTextStyles, size)
	b.SetText(rope.New("Hello"))
	b.Focus(true)
	img := image.NewRGBA(image.Rectangle{Max: size})
	b.dots[1].At = [2]int64{5, 5}
	b.Draw(true, img)
	goldenImageTest(img, t)
}

func TestDrawCursorOnLineAfterLastLine(t *testing.T) {
	size := image.Pt(100, 100)
	b := NewTextBox(testWin, testTextStyles, size)
	b.SetText(rope.New("Hello\n"))
	b.Focus(true)
	img := image.NewRGBA(image.Rectangle{Max: size})
	b.dots[1].At = [2]int64{6, 6}
	b.Draw(true, img)
	goldenImageTest(img, t)
}

func TestBlank(t *testing.T) {
	size := image.Pt(100, 100)
	b := NewTextBox(testWin, testTextStyles, size)
	b.SetText(rope.New(""))
	img := image.NewRGBA(image.Rectangle{Max: size})
	b.Draw(true, img)
	goldenImageTest(img, t)
}

func TestCursorBlink(t *testing.T) {
	size := image.Pt(100, 100)
	b := NewTextBox(testWin, testTextStyles, size)
	b.SetText(rope.New(""))
	img := image.NewRGBA(image.Rectangle{Max: size})
	var now time.Time
	b.now = func() time.Time {
		n := now
		now = now.Add(blinkDuration)
		return n
	}
	b.Focus(true)
	b.Tick()
	b.Draw(true, img)
	goldenImageTest(img, t)
}

func goldenImageTest(img image.Image, t *testing.T) {
	var (
		goldenFile = "testdata/" + t.Name() + "_golden.png"
		newFile    = "testdata/" + t.Name() + "_new.png"
	)

	t.Helper()

	buf := bytes.NewBuffer(nil)
	png.Encode(buf, img)
	got := buf.Bytes()

	want, err := ioutil.ReadFile(goldenFile)
	if err != nil {
		ioutil.WriteFile(newFile, got, 0666)
		t.Fatalf(err.Error())
	}
	if !bytes.Equal(got, want) {
		ioutil.WriteFile(newFile, got, 0666)
		t.Errorf("%s does not match %s\n", newFile, goldenFile)
	}
}
