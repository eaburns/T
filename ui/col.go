package ui

import (
	"image"
	"image/draw"

	"github.com/eaburns/T/rope"
)

// A Col is a column of sheets.
type Col struct {
	win        *Win
	size       image.Point
	lineHeight int
	minHeight  int
	rows       []Row
	heights    []float64 // frac of height
	resizing   int       // row index being resized or -1
	Row                  // focus
}

// NewCol returns a new column.
// TODO: NewCol is just a temporary implementation.
func NewCol(w *Win) *Col {
	var (
		bodyTextStyles = [...]TextStyle{
			{FG: fg, BG: colBG, Face: w.face},
			{BG: hiBG1},
			{BG: hiBG2},
			{BG: hiBG3},
		}
	)
	bg := NewTextBox(w, bodyTextStyles, image.ZP)
	bg.SetText(rope.New("Del Add\n"))
	return &Col{
		win:       w,
		minHeight: w.lineHeight + 2*int(padPt*w.dpi/72.0+0.5),
		rows:      []Row{bg},
		heights:   []float64{1.0},
		resizing:  -1,
		Row:       bg,
	}
}

// Add adds an element to the column.
func (c *Col) Add(row Row) {
	row = newFrame(int(c.win.dpi*padPt/72.0), row)
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
	c.rows = append(c.rows, row)
	c.Row.Focus(false)
	row.Focus(true)
	c.Row = row
	c.Resize(c.size)
}

// Del deletes a row from the column.
func (c *Col) Del(row Row) {
	for i := 1; i < len(c.rows); i++ {
		if c.rows[i] == row {
			c.rows = append(c.rows[:i], c.rows[i+1:]...)
			c.heights = append(c.heights[:i-1], c.heights[i:]...)
			if row == c.Row {
				c.Row = c.rows[0]
			}
			c.Resize(c.size)
			return
		}
	}
}

// Tick handles tick events.
func (c *Col) Tick() bool {
	var redraw bool
	for _, r := range c.rows {
		if r.Tick() {
			redraw = true
		}
	}
	return redraw
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
func (c *Col) Move(pt image.Point) {
	if c.resizing >= 0 {
		switch i := colIndex(c); {
		case i > 0 && pt.X < c.minHeight:
			moveRow(c.win, c.resizing+1, c, c.win.cols[i-1], pt.Y)
		case i < len(c.win.cols)-1 && pt.X > c.size.X+c.minHeight:
			moveRow(c.win, c.resizing+1, c, c.win.cols[i+1], pt.Y)
		default:
			resizeRow(c, pt)
		}
		return
	}
	pt.Y -= y0(c)
	c.Row.Move(pt)
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
	dst.rows = append(dst.rows[:ri+1], append([]Row{e}, dst.rows[ri+1:]...)...)
	dst.heights = append(dst.heights[:ri], append([]float64{frac}, dst.heights[ri:]...)...)
	if dst != src {
		src.resizing = -1
		src.Row = src.rows[0]
		src.Focus(false)
		src.Resize(src.size)

		dst.Focus(true)
	}
	dst.Row = e
	dst.resizing = ri
	dst.Resize(dst.size)
	w.Col = dst
}

func resizeRow(c *Col, pt image.Point) {
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
		return
	}

	if c.heights[c.resizing] != frac {
		c.heights[c.resizing] = frac
		c.Resize(c.size)
	}
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
func (c *Col) Click(pt image.Point, button int) {
	if c.resizing >= 0 && button == -1 {
		c.resizing = -1
		return
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
				return
			}
		}
	}

	if button > 0 {
		setColFocus(c, pt, button)
	}
	pt.Y -= y0(c)
	button, addr := c.Row.Click(pt, button)
	if button == -2 {
		txt := getText(c.Row)
		if txt == nil {
			return
		}
		cmd := rope.Slice(txt, addr[0], addr[1]).String()
		switch cmd {
		case "Add":
			c.Add(NewSheet(c.win, ""))
		case "AddCol":
			c.win.Add()
		case "Del":
			c.Del(c.Row)
		case "DelCol":
			c.win.Del(c)
		}
	}
}

func getText(r Row) rope.Rope {
	if f, ok := r.(*handleFrame); ok {
		r = f.Row
	}
	if f, ok := r.(*frame); ok {
		r = f.Row
	}
	switch r := r.(type) {
	case *Sheet:
		return r.TextBox.Text()
	case *TextBox:
		return r.Text()
	default:
		return nil
	}
}

func y0(c *Col) int {
	var y0 int
	for i, o := range c.rows {
		if o == c.Row {
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
	var o Row
	for i, o = range c.rows {
		y1 := int(c.heights[i] * float64(c.size.Y))
		if pt.Y < y1 {
			break
		}
	}
	if c.Row != o {
		c.Row.Focus(false)
		o.Focus(true)
		c.Row = o
		return true
	}
	return false
}
