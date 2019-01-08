package ui

import (
	"image"
	"image/draw"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

// A Win is a window of columns of sheets.
type Win struct {
	dpi        float32
	face       font.Face // default font face
	lineHeight int
	size       image.Point
	cols       []*Col
	widths     []float64 // frac of width
	resizing   int       // col index being resized or -1
	mods       [4]bool   // currently held modifier keys
	*Col                 // focus
}

// NewWin returns a new window.
// TODO: NewWin is just a temporary implementation.
func NewWin(dpi float32) *Win {
	face := truetype.NewFace(defaultFont, &truetype.Options{
		Size: float64(defaultFontSize),
		DPI:  float64(dpi * (72.0 / 96.0)),
	})
	h := (face.Metrics().Height + face.Metrics().Descent).Ceil()
	w := &Win{
		dpi:        dpi,
		face:       face,
		lineHeight: h,
		resizing:   -1,
	}
	w.cols = []*Col{NewCol(w)}
	w.widths = []float64{1.0}
	w.Col = w.cols[0]
	return w
}

// Add adds a new column to the window and returns it.
func (w *Win) Add() *Col {
	col := NewCol(w)
	f := 0.5
	if n := len(w.widths); n > 1 {
		f = (w.widths[n-2] + w.widths[n-1]) / 2.0
	}
	w.cols = append(w.cols, col)
	w.widths[len(w.widths)-1] = f
	w.widths = append(w.widths, 1.0)
	w.Resize(w.size)
	return col
}

// Del deletes a column unless it is the last column.
func (w *Win) Del(col *Col) {
	if len(w.cols) == 1 {
		return
	}
	for i := range w.cols {
		if w.cols[i] == col {
			w.cols = append(w.cols[:i], w.cols[i+1:]...)
			w.widths = append(w.widths[:i-1], w.widths[i:]...)
			if col == w.Col {
				w.Col = w.cols[0]
			}
			w.Resize(w.size)
			return
		}
	}
}

// Tick handles tick events.
func (w *Win) Tick() bool {
	var redraw bool
	for _, c := range w.cols {
		if c.Tick() {
			redraw = true
		}
	}
	return redraw
}

// Draw draws the window.
func (w *Win) Draw(dirty bool, drawImg draw.Image) {
	img := drawImg.(*image.RGBA)
	if w.size != img.Bounds().Size() {
		w.Resize(img.Bounds().Size())
	}
	for i, c := range w.cols {
		r := img.Bounds()
		r.Min.X = img.Bounds().Min.X + x0(w, i)
		r.Max.X = img.Bounds().Min.X + x1(w, i)
		c.Draw(dirty, img.SubImage(r).(*image.RGBA))
		if i < len(w.cols)-1 {
			r.Min.X = r.Max.X
			r.Max.X += framePx
			fillRect(img, frameColor, r)
		}
	}
}

// Resize handles resize events.
func (w *Win) Resize(size image.Point) {
	w.size = size

	if nc := len(w.cols); int(dx(w)) < nc*w.lineHeight+nc*framePx {
		// Too small to fit everything.
		// Space out as much as we can,
		// so if the window grows,
		// everything is in a good place.
		w.widths[0] = 0.0
		for i := 1; i < nc; i++ {
			w.widths[i] = w.widths[i-1] + 1.0/float64(nc)
		}
	}
	w.widths[len(w.widths)-1] = 1.0

	for i, c := range w.cols {
		c.Resize(image.Pt(x1(w, i)-x0(w, i), w.size.Y))
	}
}

// Move handles mouse move events.
func (w *Win) Move(pt image.Point) {
	if w.resizing >= 0 {
		// Center the pointer horizontally on the handle.
		x := pt.X + w.cols[w.resizing].HandleBounds().Dx()/2
		resizeCol(w, x)
		return
	}

	pt.X -= x0(w, focusedCol(w))
	w.Col.Move(pt)
}

func resizeCol(w *Win, x int) {
	dx := dx(w)
	newFrac := float64(x) / dx

	// Don't resize if either resized col would get too small.
	newX := int(newFrac * dx)
	var prev int
	if w.resizing > 0 {
		prev = x0(w, w.resizing)
	}
	if newX-prev-framePx < w.lineHeight {
		newFrac = float64(prev+w.lineHeight) / dx
	}
	next := x1(w, w.resizing+1)
	if next-newX-framePx < w.lineHeight {
		newFrac = float64(next-w.lineHeight-framePx) / dx
	}

	if w.widths[w.resizing] != newFrac {
		w.widths[w.resizing] = newFrac
		w.Resize(w.size)
	}
}

// Click handles click events.
func (w *Win) Click(pt image.Point, button int) {
	if w.resizing >= 0 && button == -1 {
		w.resizing = -1
		return
	}
	if button == 1 {
		for i, c := range w.cols[:len(w.cols)-1] {
			handle := c.HandleBounds().Add(image.Pt(x0(w, i), 0))
			if pt.In(handle) {
				// TODO: set focus on the resized column.
				w.resizing = i
				return
			}
		}
	}

	if button > 0 {
		setWinFocus(w, pt, button)
	}
	pt.X -= x0(w, focusedCol(w))
	w.Col.Click(pt, button)
}

func setWinFocus(w *Win, pt image.Point, button int) bool {
	if button != 1 {
		return false
	}
	var i int
	var c *Col
	for i, c = range w.cols {
		if pt.X < x1(w, i) {
			break
		}
	}
	if w.Col != c {
		w.Col.Focus(false)
		c.Focus(true)
		w.Col = c
		return true
	}
	return false
}

// Focus handles focus change events.
func (w *Win) Focus(focus bool) {
	if !focus {
		w.mods = [4]bool{}
	}
	w.Col.Focus(focus)
}

// Mod handles modifier key state change events.
func (w *Win) Mod(m int) {
	switch {
	case m > 0 && m < len(w.mods):
		w.mods[m] = true
	case m < 0 && -m < len(w.mods):
		w.mods[-m] = false
	}
	w.Col.Mod(m)
}

func x0(w *Win, i int) int {
	if i == 0 {
		return 0
	}
	return x1(w, i-1) + framePx
}

func x1(w *Win, i int) int { return int(w.widths[i] * dx(w)) }

func dx(w *Win) float64 { return float64(w.size.X) }

func focusedCol(w *Win) int {
	for i, c := range w.cols {
		if c == w.Col {
			return i
		}
	}
	return 0
}
