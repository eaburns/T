package ui

import (
	"image"
	"image/draw"

	"github.com/eaburns/T/rope"
	"github.com/eaburns/T/text"
	"github.com/golang/freetype/truetype"
)

// A Col is a column of sheets.
type Col struct {
	win        *Win
	size       image.Point
	lineHeight int
	minHeight  int
	rows       []Elem
	heights    []float64 // frac of height
	resizing   int       // row index being resized or -1
	Elem                 // focus
}

// NewCol returns a new column.
// TODO: NewCol is just a temporary implementation.
func NewCol(w *Win) *Col {
	var (
		face = truetype.NewFace(font, &truetype.Options{
			Size: float64(fontPt),
			DPI:  float64(w.dpi * (72.0 / 96.0)),
		})
		bodyStyles = [...]text.Style{
			{FG: fg, BG: colBG, Face: face},
			{BG: hiBG1},
			{BG: hiBG2},
			{BG: hiBG3},
		}
	)
	bg := text.NewBox(bodyStyles, image.ZP)
	bg.SetText(rope.New("Del Add\n"))
	h := (face.Metrics().Height + face.Metrics().Descent).Ceil()
	return &Col{
		win:        w,
		lineHeight: h,
		minHeight:  h + 2*int(padPt*w.dpi/72.0+0.5),
		rows:       []Elem{bg},
		heights:    []float64{1.0},
		resizing:   -1,
		Elem:       bg,
	}
}

// Add adds an element to the column.
func (c *Col) Add(e Elem) {
	e = newFrame(int(c.win.dpi*padPt/72.0), e)
	switch n := len(c.rows); n {
	case 0:
		panic("impossible")
	case 1:
		c.heights = []float64{0.05, 1.0}
	default:
		h0 := c.heights[n-2]
		c.heights[n-1] = h0 + (1.0-h0)*0.5
		c.heights = append(c.heights, 1.0)
	}
	c.rows = append(c.rows, e)
	c.Elem.Focus(false)
	e.Focus(true)
	c.Elem = e
}

// HandleBounds returns the bounding box of the handle.
func (c *Col) HandleBounds() image.Rectangle {
	y1 := int(c.heights[0] * float64(c.size.Y))
	if y1 > c.lineHeight {
		y1 = c.lineHeight
	}
	return image.Rect(c.size.X-c.lineHeight, 0, c.size.X, y1)
}

// Draw draws the column.
func (c *Col) Draw(dirty bool, drawImage draw.Image) {
	img := drawImage.(*image.RGBA)
	if c.size != img.Bounds().Size() {
		c.Resize(img.Bounds().Size())
	}

	r := img.Bounds()
	y0 := r.Min.Y
	for i, o := range c.rows {
		r.Max.Y = y0 + int(c.heights[i]*float64(c.size.Y))
		if i == 0 {
			r0 := r
			r0.Max.X = drawColHandle(c, img)
			o.Draw(dirty, img.SubImage(r0).(*image.RGBA))
		} else {
			o.Draw(dirty, img.SubImage(r).(*image.RGBA))
		}
		r.Min.Y = r.Max.Y
	}
}

func drawColHandle(c *Col, img *image.RGBA) int {
	const pad = 6
	handle := c.HandleBounds().Add(img.Bounds().Min)
	r := handle
	r.Max.Y = r.Min.Y + int(c.heights[0]*float64(c.size.Y))
	fillRect(img, colBG, r)
	fillRect(img, tagBG, handle.Inset(pad))
	return r.Min.X
}

// Resize handles resize events.
func (c *Col) Resize(size image.Point) {
	// Preserve the height of the column background in pixels, not percent.
	h0 := float64(c.heights[0] * float64(c.size.Y))
	c.size = size
	c.heights[0] = clampFrac(h0 / float64(c.size.Y))

	if nr := len(c.rows); size.Y < nr*c.minHeight {
		// Too small to fit everything.
		// Space out as much as we can,
		// so if the window grows,
		// everything is in a good place.
		c.heights[0] = 0.0
		for i := 1; i < nr; i++ {
			c.heights[i] = c.heights[i-1] + 1.0/float64(nr)
		}
	}

	c.heights[len(c.heights)-1] = 1.0

	var y0 int
	for i, o := range c.rows {
		y1 := int(c.heights[i] * float64(c.size.Y))
		if i == 0 {
			o.Resize(image.Pt(size.X-c.HandleBounds().Dx(), y1-y0))
			y0 = y1
			continue
		}
		if y1-y0 <= c.minHeight {
			// The row got too small.
			// Slide the next up to fit.
			c.heights[i] = clampFrac(float64(y0+c.minHeight) / float64(c.size.Y))
			y1 = int(c.heights[i] * float64(c.size.Y))
		}
		o.Resize(image.Pt(size.X, y1-y0))
		y0 = y1
	}
}

// Move handles mouse move events.
func (c *Col) Move(pt image.Point) bool {
	if c.resizing >= 0 {
		switch i := colIndex(c); {
		case i > 0 && pt.X < c.minHeight:
			moveRow(c.win, c.resizing+1, c, c.win.cols[i-1], pt.Y)
			return true
		case i < len(c.win.cols)-1 && pt.X > c.size.X+c.minHeight:
			moveRow(c.win, c.resizing+1, c, c.win.cols[i+1], pt.Y)
			return true
		default:
			return resizeRow(c, pt)
		}
	}
	pt.Y -= y0(c)
	return c.Elem.Move(pt)
}

// colIndex returns the index of the Col its Win's cols array.
func colIndex(c *Col) int {
	for i := range c.win.cols {
		if c.win.cols[i] == c {
			return i
		}
	}
	return -1
}

func moveRow(w *Win, ri int, src, dst *Col, y int) {
	e := src.rows[ri]

	// Remove the row.
	src.rows = append(src.rows[:ri], src.rows[ri+1:]...)
	src.heights = append(src.heights[:ri-1], src.heights[ri:]...)

	// Focus the dest column and add the row.
	frac := clampFrac(float64(y) / float64(dst.size.Y))
	for ri = range dst.heights {
		if frac <= dst.heights[ri] {
			break
		}
	}
	// TODO: only move a sheet if the dst can fit it.
	// TODO: when moving a sheet, squish columns to fit.
	dst.rows = append(dst.rows[:ri+1], append([]Elem{e}, dst.rows[ri+1:]...)...)
	dst.heights = append(dst.heights[:ri], append([]float64{frac}, dst.heights[ri:]...)...)
	if dst != src {
		src.resizing = -1
		src.Elem = src.rows[0]
		src.Focus(false)
		src.Resize(src.size)

		dst.Focus(true)
	}
	dst.Elem = e
	dst.resizing = ri
	dst.Resize(dst.size)
	w.Elem = dst
}

func resizeRow(c *Col, pt image.Point) bool {
	// Try to center the mouse on line 1.
	newY1 := pt.Y - c.minHeight/2

	// Clamp to a multiple of line height.
	clamp0 := c.minHeight * (newY1 / c.minHeight)
	clamp1 := clamp0 + c.minHeight
	if newY1-clamp0 < clamp1-newY1 {
		newY1 = clamp0
	} else {
		newY1 = clamp1
	}

	// Clamp to the window.
	dy := float64(c.size.Y)
	frac := clampFrac(float64(newY1) / dy)

	// Disallow the above row from getting too small.
	var y0 int
	if c.resizing > 0 {
		y0 = int(c.heights[c.resizing-1] * dy)
	}
	if c.resizing > 0 && newY1-y0 < c.minHeight {
		frac = float64(y0+c.minHeight) / dy
	}

	// Disallow the below row from getting too small.
	y2 := int(c.heights[c.resizing+1] * dy)
	if y2-newY1 < c.minHeight {
		frac = float64(y2-c.minHeight) / dy
	}

	if newY1 < y0-c.minHeight || newY1 > y2+c.minHeight {
		moveRow(c.win, c.resizing+1, c, c, pt.Y)
		return true
	}

	if c.heights[c.resizing] == frac {
		return false
	}
	c.heights[c.resizing] = frac
	c.Resize(c.size)
	return true
}

func clampFrac(f float64) float64 {
	switch {
	case f < 0.0:
		return 0.0
	case f > 1.0:
		return 1.0
	default:
		return f
	}
}

// Click handles click events.
func (c *Col) Click(pt image.Point, button int) bool {
	if c.resizing >= 0 && button == -1 {
		c.resizing = -1
		return false
	}
	if button == 1 {
		for i := range c.rows[:len(c.rows)-1] {
			r := c.rows[i+1]
			y0 := int(c.heights[i] * float64(c.size.Y))
			handler, ok := r.(interface{ HandleBounds() image.Rectangle })
			if !ok {
				continue
			}
			handle := handler.HandleBounds().Add(image.Pt(0, y0))
			if pt.In(handle) {
				// TODO: set focus on the resized row.
				c.resizing = i
				return false
			}
		}
	}

	var redraw bool
	if button > 0 {
		redraw = setColFocus(c, pt, button)
	}
	pt.Y -= y0(c)
	return c.Elem.Click(pt, button) || redraw
}

func y0(c *Col) int {
	var y0 int
	for i, o := range c.rows {
		if o == c.Elem {
			break
		}
		y0 = int(c.heights[i] * float64(c.size.Y))
	}
	return y0
}

func setColFocus(c *Col, pt image.Point, button int) bool {
	if button != 1 {
		return false
	}
	var i int
	var o Elem
	for i, o = range c.rows {
		y1 := int(c.heights[i] * float64(c.size.Y))
		if pt.Y < y1 {
			break
		}
	}
	if c.Elem != o {
		c.Elem.Focus(false)
		o.Focus(true)
		c.Elem = o
		return true
	}
	return false
}
