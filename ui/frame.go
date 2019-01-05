package ui

import (
	"image"
	"image/color"
	"image/draw"
)

func newFrame(px int, e Elem) Elem {
	f := &frame{px: px, Elem: e}
	if _, ok := f.Elem.(interface{ HandleBounds() image.Rectangle }); ok {
		return &handleFrame{*f}
	}
	return f
}

type frame struct {
	px int
	Elem
}

type handleFrame struct {
	frame
}

func (f *handleFrame) HandleBounds() image.Rectangle {
	h := f.Elem.(interface{ HandleBounds() image.Rectangle })
	return h.HandleBounds().Add(image.Pt(f.px, f.px))
}

func (f *frame) Draw(dirty bool, drawImg draw.Image) {
	img := drawImg.(*image.RGBA)
	strokeRect(img, colBG, f.px, img.Bounds())
	b := img.Bounds().Inset(f.px)
	f.Elem.Draw(dirty, img.SubImage(b).(*image.RGBA))
}

func strokeRect(img draw.Image, c color.Color, w int, r image.Rectangle) {
	top := image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+w)
	left := image.Rect(r.Min.X, r.Min.Y, r.Min.X+w, r.Max.Y)
	right := image.Rect(r.Max.X-w, r.Min.Y, r.Max.X, r.Max.Y)
	bottom := image.Rect(r.Min.X, r.Max.Y-w, r.Max.X, r.Max.Y)
	for _, r := range [...]image.Rectangle{top, left, right, bottom} {
		fillRect(img, c, r)
	}
}

func (f *frame) Resize(size image.Point) {
	f.Elem.Resize(size.Sub(image.Pt(f.px, f.px)))
}

// Move handles mouse movement events.
func (f *frame) Move(pt image.Point) bool {
	return f.Elem.Move(pt.Sub(image.Pt(f.px, f.px)))
}

func (f *frame) Click(pt image.Point, button int) bool {
	return f.Elem.Click(pt.Sub(image.Pt(f.px, f.px)), button)
}
