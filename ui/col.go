package ui

import (
	"fmt"
	"image"
	"image/draw"
	"unicode"

	"github.com/eaburns/T/rope"
	"github.com/eaburns/T/text"
)

// A Col is a column of sheets.
type Col struct {
	win      *Win
	size     image.Point
	rows     []Row
	heights  []float64 // frac of height
	resizing int       // row index being resized or -1
	Row                // focus
}

// NewCol returns a new column.
// TODO: NewCol is just a temporary implementation.
func NewCol(w *Win) *Col {
	var (
		bodyTextStyles = [...]text.Style{
			{FG: fg, BG: colBG, Face: w.face},
			{BG: hiBG1},
			{BG: hiBG2},
			{BG: hiBG3},
		}
	)
	bg := NewTextBox(w, bodyTextStyles, image.ZP)
	bg.SetText(rope.New(colText))
	return &Col{
		win:      w,
		rows:     []Row{bg},
		heights:  []float64{1.0},
		resizing: -1,
		Row:      bg,
	}
}

// Add adds an element to the column.
func (c *Col) Add(row Row) {
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
	y1 := y1(c, 0)
	if y1 > c.win.lineHeight {
		y1 = c.win.lineHeight
	}
	return image.Rect(c.size.X-c.win.lineHeight, 0, c.size.X, y1)
}

// Draw draws the column.
func (c *Col) Draw(dirty bool, drawImage draw.Image) {
	img := drawImage.(*image.RGBA)
	if c.size != img.Bounds().Size() {
		c.Resize(img.Bounds().Size())
	}

	for i, o := range c.rows {
		r := img.Bounds()
		r.Min.Y = img.Bounds().Min.Y + y0(c, i)
		r.Max.Y = img.Bounds().Min.Y + y1(c, i)
		if i == 0 {
			r0 := r
			r0.Max.X = drawColHandle(c, img)
			o.Draw(dirty, img.SubImage(r0).(*image.RGBA))
		} else {
			o.Draw(dirty, img.SubImage(r).(*image.RGBA))
		}
		if i < len(c.rows)-1 {
			r.Min.Y = r.Max.Y
			r.Max.Y += framePx
			fillRect(img, frameBG, r)
		}
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
	dy := dy(c)
	// Preserve the height of the column background in pixels, not percent.
	h0 := c.heights[0] * dy
	c.size = size
	c.heights[0] = clampFrac(h0 / dy)

	if nr := len(c.rows); int(dy) < nr*c.win.lineHeight+nr*framePx {
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

	for i, o := range c.rows {
		a, b := y0(c, i), y1(c, i)
		if i == 0 {
			o.Resize(image.Pt(size.X-c.HandleBounds().Dx(), b-a))
			continue
		}
		if b-a < c.win.lineHeight {
			// The row got too small.
			// Slide the next up to fit.
			y := i*framePx + a + c.win.lineHeight
			c.heights[i] = clampFrac(float64(y) / dy)
			b = y1(c, i)
		}
		o.Resize(image.Pt(size.X, b-a))
	}
}

// Move handles mouse move events.
func (c *Col) Move(pt image.Point) {
	if c.resizing >= 0 {
		switch i := colIndex(c); {
		case i > 0 && pt.X < c.win.lineHeight:
			moveRow(c.win, c.resizing+1, c, c.win.cols[i-1], pt.Y)
		case i < len(c.win.cols)-1 && pt.X > c.size.X+c.win.lineHeight:
			moveRow(c.win, c.resizing+1, c, c.win.cols[i+1], pt.Y)
		default:
			resizeRow(c, pt)
		}
		return
	}
	pt.Y -= y0(c, focusedRow(c))
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
	if src.Row != e {
		panic("impossible")
	}

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
	dst.rows = append(dst.rows[:ri+1], append([]Row{e}, dst.rows[ri+1:]...)...)
	dst.heights = append(dst.heights[:ri], append([]float64{frac}, dst.heights[ri:]...)...)

	if dst != src {
		src.resizing = -1
		src.Row = src.rows[0]
		src.Resize(src.size)
		dst.Row = e
	}
	dst.resizing = ri
	dst.Resize(dst.size)
	w.Col = dst
}

func resizeRow(c *Col, pt image.Point) {
	// Try to center the mouse on line 1.
	newY := c.resizing*framePx + pt.Y - c.win.lineHeight/2

	// Clamp to a multiple of line height.
	snap := c.win.lineHeight
	clamp0 := snap * (newY / snap)
	clamp1 := clamp0 + snap
	if newY-clamp0 < clamp1-newY {
		newY = clamp0
	} else {
		newY = clamp1
	}

	// Clamp to the window.
	dy := dy(c)
	frac := clampFrac(float64(newY) / dy)

	var prev int
	if c.resizing > 0 {
		prev = y0(c, c.resizing)
	}
	next := y1(c, c.resizing+1)

	// Swap with above or below row in the same column.
	if pt.Y < prev-c.win.lineHeight-framePx ||
		pt.Y > next+c.win.lineHeight+framePx {
		moveRow(c.win, c.resizing+1, c, c, pt.Y)
		return
	}

	// Disallow the previous row from getting too small.
	if c.resizing > 0 && newY-prev < c.win.lineHeight {
		frac = float64(prev+c.win.lineHeight) / dy
	}
	// Disallow the current row from getting too small.
	if next-newY-framePx < c.win.lineHeight {
		frac = float64(next-c.win.lineHeight-framePx) / dy
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
		for i := range c.rows[:len(c.rows)] {
			r := c.rows[i]
			handler, ok := r.(interface{ HandleBounds() image.Rectangle })
			if !ok {
				continue
			}
			handle := handler.HandleBounds().Add(image.Pt(0, y0(c, i)))
			if pt.In(handle) {
				if c.Row != r {
					c.Row.Focus(false)
					r.Focus(true)
					c.Row = r
				}
				c.resizing = i - 1
				return
			}
		}
	}

	if button > 0 {
		setColFocus(c, pt, button)
	}
	pt.Y -= y0(c, focusedRow(c))
	button, addr := c.Row.Click(pt, button)
	if button == -2 || button == -3 {
		tb := getTextBox(c.Row)
		if tb == nil {
			return
		}
		var err error
		s := getSheet(c.Row)
		txt := getClickText(tb, addr)
		switch button {
		case -2:
			err = execCmd(c, s, txt)
		case -3:
			err = lookText(c, s, txt)
		}
		if err != nil {
			// TODO: print command errors to a sheet.
			fmt.Println(err.Error())
		}
	}
}

func getClickText(tb *TextBox, addr [2]int64) string {
	if addr[0] < addr[1] {
		return rope.Slice(tb.text, addr[0], addr[1]).String()
	}
	if dot := tb.dots[1].At; dot[0] <= addr[0] && addr[0] < dot[1] {
		return rope.Slice(tb.text, dot[0], dot[1]).String()
	}

	front, back := rope.Split(tb.text, addr[0])
	start := rope.LastIndexFunc(front, unicode.IsSpace)
	if start < 0 {
		start = 0
	} else {
		start++
	}
	end := rope.IndexFunc(back, unicode.IsSpace)
	if end < 0 {
		end = tb.text.Len()
	} else {
		end += addr[0]
	}
	return rope.Slice(tb.text, start, end).String()
}

func getTextBox(r Row) *TextBox {
	switch r := r.(type) {
	case *Sheet:
		return r.TextBox
	case *TextBox:
		return r
	default:
		return nil
	}
}

func getSheet(r Row) *Sheet {
	if r, ok := r.(*Sheet); ok {
		return r
	}
	return nil
}

func setColFocus(c *Col, pt image.Point, button int) bool {
	if button != 1 {
		return false
	}
	var i int
	var o Row
	for i, o = range c.rows {
		if pt.Y < y1(c, i) {
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

func y0(c *Col, i int) int {
	if i == 0 {
		return 0
	}
	return y1(c, i-1) + framePx
}

func y1(c *Col, i int) int { return int(c.heights[i] * dy(c)) }

func dy(c *Col) float64 { return float64(c.size.Y) }

func focusedRow(c *Col) int {
	for i, o := range c.rows {
		if o == c.Row {
			return i
		}
	}
	return 0
}
