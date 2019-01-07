package ui

import (
	"image"
	"image/color"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func TestTextStyle_merge(t *testing.T) {
	face1, face2 := testFace{1}, testFace{2}
	tests := []struct {
		a, b TextStyle
		want TextStyle
	}{
		{
			a:    TextStyle{},
			b:    TextStyle{},
			want: TextStyle{},
		},
		{
			a:    TextStyle{FG: color.White},
			b:    TextStyle{FG: color.Black},
			want: TextStyle{FG: color.Black},
		},
		{
			a:    TextStyle{FG: color.White},
			b:    TextStyle{BG: color.Black},
			want: TextStyle{FG: color.White, BG: color.Black},
		},
		{
			a:    TextStyle{FG: color.White},
			b:    TextStyle{Face: face1},
			want: TextStyle{FG: color.White, Face: face1},
		},
		{
			a:    TextStyle{BG: color.White},
			b:    TextStyle{BG: color.Black},
			want: TextStyle{BG: color.Black},
		},
		{
			a:    TextStyle{BG: color.White},
			b:    TextStyle{FG: color.Black},
			want: TextStyle{FG: color.Black, BG: color.White},
		},
		{
			a:    TextStyle{BG: color.White},
			b:    TextStyle{Face: face1},
			want: TextStyle{BG: color.White, Face: face1},
		},
		{
			a:    TextStyle{Face: face1},
			b:    TextStyle{Face: face2},
			want: TextStyle{Face: face2},
		},
		{
			a:    TextStyle{Face: face1},
			b:    TextStyle{FG: color.White},
			want: TextStyle{FG: color.White, Face: face1},
		},
		{
			a:    TextStyle{Face: face1},
			b:    TextStyle{BG: color.Black},
			want: TextStyle{BG: color.Black, Face: face1},
		},
		{
			a:    TextStyle{FG: color.White, BG: color.Black, Face: face1},
			b:    TextStyle{FG: color.Black, BG: color.White, Face: face2},
			want: TextStyle{FG: color.Black, BG: color.White, Face: face2},
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
