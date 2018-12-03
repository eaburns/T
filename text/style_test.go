package text

import (
	"image"
	"image/color"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func TestStyle_merge(t *testing.T) {
	face1, face2 := testFace{1}, testFace{2}
	tests := []struct {
		a, b Style
		want Style
	}{
		{
			a:    Style{},
			b:    Style{},
			want: Style{},
		},
		{
			a:    Style{FG: color.White},
			b:    Style{FG: color.Black},
			want: Style{FG: color.Black},
		},
		{
			a:    Style{FG: color.White},
			b:    Style{BG: color.Black},
			want: Style{FG: color.White, BG: color.Black},
		},
		{
			a:    Style{FG: color.White},
			b:    Style{Face: face1},
			want: Style{FG: color.White, Face: face1},
		},
		{
			a:    Style{BG: color.White},
			b:    Style{BG: color.Black},
			want: Style{BG: color.Black},
		},
		{
			a:    Style{BG: color.White},
			b:    Style{FG: color.Black},
			want: Style{FG: color.Black, BG: color.White},
		},
		{
			a:    Style{BG: color.White},
			b:    Style{Face: face1},
			want: Style{BG: color.White, Face: face1},
		},
		{
			a:    Style{Face: face1},
			b:    Style{Face: face2},
			want: Style{Face: face2},
		},
		{
			a:    Style{Face: face1},
			b:    Style{FG: color.White},
			want: Style{FG: color.White, Face: face1},
		},
		{
			a:    Style{Face: face1},
			b:    Style{BG: color.Black},
			want: Style{BG: color.Black, Face: face1},
		},
		{
			a:    Style{FG: color.White, BG: color.Black, Face: face1},
			b:    Style{FG: color.Black, BG: color.White, Face: face2},
			want: Style{FG: color.Black, BG: color.White, Face: face2},
		},
	}
	for _, test := range tests {
		got := test.a.merge(test.b)
		if got != test.want {
			t.Errorf("(%v).merge(%v)=%v, want %v",
				test.a, test.b, got, test.want)
		}
	}
}

type testFace struct{ int }

func (testFace) Close() error { panic("unimplemented") }
func (testFace) Glyph(fixed.Point26_6, rune) (image.Rectangle, image.Image, image.Point, fixed.Int26_6, bool) {
	panic("unimplemented")
}
func (testFace) GlyphBounds(rune) (fixed.Rectangle26_6, fixed.Int26_6, bool) { panic("unimplemented") }
func (testFace) GlyphAdvance(rune) (fixed.Int26_6, bool)                     { panic("unimplemented") }
func (testFace) Kern(rune, rune) fixed.Int26_6                               { panic("unimplemented") }
func (testFace) Metrics() font.Metrics                                       { panic("unimplemented") }
