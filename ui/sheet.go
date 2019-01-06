package ui

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/eaburns/T/edit"
	"github.com/eaburns/T/rope"
	"github.com/eaburns/T/text"
)

// A Sheet is a tag and a body.
// TODO: better document the Sheet type.
type Sheet struct {
	tag           *text.Box
	body          *text.Box
	tagH, minTagH int
	size          image.Point
	Elem          // the focus element: the tag or the body.
}

// NewSheet returns a new sheet.
func NewSheet(c *Col, title string) *Sheet {
	var (
		tagStyles = [...]text.Style{
			{FG: fg, BG: tagBG, Face: c.win.face},
			{BG: hiBG1},
			{BG: hiBG2},
			{BG: hiBG3},
		}
		bodyStyles = [...]text.Style{
			{FG: fg, BG: bodyBG, Face: c.win.face},
			{BG: hiBG1},
			{BG: hiBG2},
			{BG: hiBG3},
		}
	)
	tag := text.NewBox(tagStyles, image.ZP)
	tag.SetText(rope.New(title + " | Del Undo Put"))
	body := text.NewBox(bodyStyles, image.ZP)
	s := &Sheet{
		tag:     tag,
		body:    body,
		minTagH: c.win.lineHeight,
		Elem:    body,
	}
	tag.SetSyntax(s)
	return s
}

// Body returns the sheet's body text box.
func (s *Sheet) Body() *text.Box { return s.body }

// Draw draws the sheet.
func (s *Sheet) Draw(dirty bool, drawImg draw.Image) {
	img := drawImg.(*image.RGBA)

	tagRect := img.Bounds()
	tagRect.Max.X = drawSheetHandle(s, img)
	tagRect.Max.Y = tagRect.Min.Y + s.tagH
	s.tag.Draw(dirty, img.SubImage(tagRect).(*image.RGBA))

	bodyRect := img.Bounds()
	bodyRect.Min.Y = tagRect.Max.Y
	s.body.Draw(dirty, img.SubImage(bodyRect).(*image.RGBA))
}

func drawSheetHandle(s *Sheet, img *image.RGBA) int {
	const pad = 6
	handle := s.HandleBounds().Add(img.Bounds().Min)
	r := handle
	r.Max.Y = r.Min.Y + s.tagH
	fillRect(img, tagBG, r)
	fillRect(img, colBG, handle.Inset(pad))
	return r.Min.X
}

// HandleBounds returns the bounding box of the handle.
func (s *Sheet) HandleBounds() image.Rectangle {
	return image.Rect(s.size.X-s.minTagH, 0, s.size.X, s.minTagH)
}

func fillRect(img draw.Image, c color.Color, r image.Rectangle) {
	draw.Draw(img, r, image.NewUniform(c), image.ZP, draw.Src)
}

// Resize handles resize events.
func (s *Sheet) Resize(size image.Point) {
	s.size = size
	resetTagHeight(s, size)
	s.body.Resize(image.Pt(size.X, size.Y-s.tagH))
}

// Update watches for updates to the tag and resizes it to fit the text height.
func (s *Sheet) Update([]text.Highlight, edit.Diffs, rope.Rope) []text.Highlight {
	oldTagH := s.tagH
	resetTagHeight(s, s.size)
	if s.tagH != oldTagH {
		s.body.Resize(image.Pt(s.size.X, s.size.Y-s.tagH))
	}
	return nil
}

func resetTagHeight(s *Sheet, size image.Point) {
	size.X -= s.minTagH // handle
	s.tag.Resize(size)
	s.tag.Dir(0, math.MinInt16)
	if s.tagH = s.tag.TextHeight(); s.tagH < s.minTagH {
		s.tagH = s.minTagH
	}
	s.tag.Resize(image.Pt(size.X, s.tagH))
}

// Move handles movement events.
func (s *Sheet) Move(pt image.Point) bool {
	if s.Elem == s.body {
		pt.Y -= s.tagH
	}
	return s.Elem.Move(pt)
}

// Click handles click events.
func (s *Sheet) Click(pt image.Point, button int) bool {
	var redraw bool
	if button > 0 {
		redraw = setSheetFocus(s, pt, button)
	}

	if s.Elem == s.body {
		pt.Y -= s.tagH
	}
	return s.Elem.Click(pt, button) || redraw
}

func setSheetFocus(s *Sheet, pt image.Point, button int) bool {
	if button != 1 {
		return false
	}
	if pt.Y < s.tagH {
		if s.Elem != s.tag {
			s.Elem = s.tag
			s.body.Focus(false)
			s.tag.Focus(true)
			return true
		}
	} else {
		if s.Elem != s.body {
			s.Elem = s.body
			s.tag.Focus(false)
			s.body.Focus(true)
			return true
		}
	}
	return false
}
