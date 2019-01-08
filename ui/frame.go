package ui

import (
	"image"
	"image/color"
	"image/draw"
)

func newFrame(px int, row Row) Row {
	f := &frame{px: px, Row: row}
	if _, ok := f.Row.(interface{ HandleBounds() image.Rectangle }); ok {
		return &handleFrame{*f}
	}
	return f
}

type frame struct {
	px int
	Row
}

type handleFrame struct {
	frame
}

func (f *handleFrame) HandleBounds() image.Rectangle {
	h := f.Row.(interface{ HandleBounds() image.Rectangle })
	return h.HandleBounds().Add(image.Pt(f.px, f.px))
}

func (f *frame) Draw(dirty bool, drawImg draw.Image) {
	img := drawImg.(*image.RGBA)
	strokeRect(img, padColor, f.px, img.Bounds())
	b := img.Bounds().Inset(f.px)
	f.Row.Draw(dirty, img.SubImage(b).(*image.RGBA))
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
	f.Row.Resize(size.Sub(image.Pt(f.px, f.px)))
}

// Move handles mouse movement events.
func (f *frame) Move(pt image.Point) {
	f.Row.Move(pt.Sub(image.Pt(f.px, f.px)))
}

func (f *frame) Click(pt image.Point, button int) (int, [2]int64) {
	return f.Row.Click(pt.Sub(image.Pt(f.px, f.px)), button)
}
