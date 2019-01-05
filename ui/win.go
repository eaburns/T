package ui

import (
	"image"
	"image/draw"

	"github.com/eaburns/T/rope"
	"github.com/eaburns/T/text"
)

// A Win is a window of columns of sheets.
type Win struct {
	dpi      float32
	size     image.Point
	minWidth int
	cols     []*Col
	widths   []float64 // frac of width
	resizing int       // col index being resized or -1
	Elem               // focus
}

// NewWin returns a new window.
// TODO: NewWin is just a temporary implementation.
func NewWin(dpi float32, sheet *Sheet) *Win {
	w := &Win{
		dpi:      dpi,
		minWidth: int(dpi * minColPt / 72.0),
		widths:   []float64{0.33, 0.66, 1.0},
		resizing: -1,
	}
	w.cols = []*Col{
		NewCol(w),
		NewCol(w),
		NewCol(w),
	}
	w.Elem = w.cols[0]
	w.cols[0].Add(sheet)
	w.cols[0].rows[0].(*text.Box).SetText(rope.New("Exit Del New"))
	w.cols[1].Add(NewSheet(dpi, "sheet0"))
	w.cols[1].Add(NewSheet(dpi, "sheet1"))
	w.cols[2].Add(NewSheet(dpi, "sheet2"))
	return w
}

// Draw draws the window.
func (w *Win) Draw(dirty bool, img draw.Image) {
	if w.size != img.Bounds().Size() {
		w.Resize(img.Bounds().Size())
	}

	r := img.Bounds()
	x0 := r.Min.X
	r.Max.X = x0
	fillRect(img, colBG, r)

	r.Min.X = r.Max.X
	for i, c := range w.cols {
		r.Max.X = x0 + int(w.widths[i]*float64(w.size.X))
		c.Draw(dirty, img.(*image.RGBA).SubImage(r).(*image.RGBA))
		r.Min.X = r.Max.X
	}
}

// Resize handles resize events.
func (w *Win) Resize(size image.Point) {
	w.size = size
	var x0 int
	for i, c := range w.cols {
		x1 := int(w.widths[i] * float64(w.size.X))
		c.Resize(image.Pt(x1-x0, w.size.Y))
		x0 = x1
	}
}

// Move handles mouse move events.
func (w *Win) Move(pt image.Point) bool {
	if w.resizing >= 0 {
		// Center the pointer horizontally on the handle.
		x := pt.X + w.cols[w.resizing].HandleBounds().Dx()/2
		return resizeCol(w, x)
	}

	pt.X -= x0(w)
	return w.Elem.Move(pt)
}

func resizeCol(w *Win, x int) bool {
	newFrac := float64(x) / float64(w.size.X)

	// Don't resize if either resized col would get too small.
	var x0 int
	if w.resizing > 0 {
		x0 = int(w.widths[w.resizing-1] * float64(w.size.X))
	}
	newX1 := int(newFrac * float64(w.size.X))
	if newX1-x0 < w.minWidth {
		newFrac = float64(x0+w.minWidth) / float64(w.size.X)
	}
	x2 := int(w.widths[w.resizing+1] * float64(w.size.X))
	if x2-newX1 < w.minWidth {
		newFrac = float64(x2-w.minWidth) / float64(w.size.X)
	}

	if w.widths[w.resizing] == newFrac {
		return false
	}
	w.widths[w.resizing] = newFrac
	w.Resize(w.size)
	return true
}

// Click handles click events.
func (w *Win) Click(pt image.Point, button int) bool {
	if w.resizing >= 0 && button == -1 {
		w.resizing = -1
		return false
	}
	if button == 1 {
		var x0 int
		for i, c := range w.cols[:len(w.cols)-1] {
			handle := c.HandleBounds().Add(image.Pt(x0, 0))
			if pt.In(handle) {
				// TODO: set focus on the resized column.
				w.resizing = i
				return false
			}
			x0 = int(w.widths[i] * float64(w.size.X))
		}
	}

	var redraw bool
	if button > 0 {
		redraw = setWinFocus(w, pt, button)
	}
	pt.X -= x0(w)
	return w.Elem.Click(pt, button) || redraw
}

func x0(w *Win) int {
	var x0 int
	for i, c := range w.cols {
		if c == w.Elem {
			break
		}
		x0 = int(w.widths[i] * float64(w.size.X))
	}
	return x0
}

func setWinFocus(w *Win, pt image.Point, button int) bool {
	if button != 1 {
		return false
	}
	var i int
	var c *Col
	for i, c = range w.cols {
		x1 := int(w.widths[i] * float64(w.size.X))
		if pt.X < x1 {
			break
		}
	}
	if w.Elem != c {
		w.Elem.Focus(false)
		c.Focus(true)
		w.Elem = c
		return true
	}
	return false
}
