// Package ui is the user interface of the editor.
package ui

import (
	"image"
	"image/draw"
	"strings"
	"sync"

	"github.com/eaburns/T/clipboard"
	"github.com/eaburns/T/edit"
	"github.com/eaburns/T/rope"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

// A Win is a window of columns of sheets.
type Win struct {
	size     image.Point
	*Col     // focus
	cols     []*Col
	widths   []float64 // frac of width
	resizing int       // col index being resized or -1

	dpi        float32
	lineHeight int
	mods       [4]bool // currently held modifier keys
	clipboard  clipboard.Clipboard
	face       font.Face // default font face
	output     *Sheet

	mu           sync.Mutex
	outputBuffer strings.Builder
}

// NewWin returns a new window.
func NewWin(dpi float32) *Win {
	face := truetype.NewFace(defaultFont, &truetype.Options{
		Size: float64(defaultFontSize),
		DPI:  float64(dpi * (72.0 / 96.0)),
	})
	h := (face.Metrics().Height + face.Metrics().Descent).Ceil()
	w := &Win{
		resizing:   -1,
		dpi:        dpi,
		face:       face,
		lineHeight: h,
		clipboard:  clipboard.New(),
	}
	w.cols = []*Col{NewCol(w)}
	w.widths = []float64{1.0}
	w.Col = w.cols[0]
	w.output = NewSheet(w, "Output")
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
	setWinFocus(w, col)
	return col
}

// Del deletes a column unless it is the last column.
func (w *Win) Del(c *Col) {
	if len(w.cols) == 1 {
		return
	}
	i := colIndex(c)
	if i < 0 {
		return
	}
	w.cols = append(w.cols[:i], w.cols[i+1:]...)
	w.widths = append(w.widths[:i], w.widths[i+1:]...)
	w.widths[len(w.cols)-1] = 1.0
	if w.Col == c {
		if i == 0 {
			setWinFocus(w, w.cols[0])
		} else {
			setWinFocus(w, w.cols[i-1])
		}
	}
	w.Resize(w.size)
}

// Tick handles tick events.
func (w *Win) Tick() bool {
	var redraw bool
	if showOutput(w) {
		redraw = true
	}
	for _, c := range w.cols {
		if c.Tick() {
			redraw = true
		}
	}
	return redraw
}

func showOutput(w *Win) bool {
	w.mu.Lock()
	output := w.outputBuffer.String()
	w.outputBuffer.Reset()
	w.mu.Unlock()

	if len(output) == 0 {
		return false
	}

	b := w.output.body
	b.Change(edit.Diffs{{
		At:   [2]int64{b.text.Len(), b.text.Len()},
		Text: rope.New(output),
	}})
	setDot(b, 1, b.text.Len(), b.text.Len())
	// TODO: only showAddr on Output if the cursor was visible to begin with.
	// If the user scrolls up, for example, we shouldn't scroll them back down.
	// This should probably just be the behavior of b.Change by default.
	showAddr(b, b.dots[1].At[1])

	w.outputBuffer.Reset()
	for _, c := range w.cols {
		for _, r := range c.rows {
			if r == w.output {
				return true
			}
		}
	}
	w.cols[len(w.cols)-1].Add(w.output)
	return true
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
			fillRect(img, frameBG, r)
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

// Wheel handles mouse wheel events.
func (w *Win) Wheel(pt image.Point, x, y int) {
	for i, c := range w.cols {
		if pt.X < x1(w, i) {
			pt.X -= x0(w, i)
			c.Wheel(pt, x, y)
			return
		}
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
				if w.Col != c {
					w.Col.Focus(false)
					c.Focus(true)
					w.Col = c
				}
				w.resizing = i
				return
			}
		}
	}

	if button > 0 {
		setWinFocusPt(w, pt)
	}
	pt.X -= x0(w, focusedCol(w))
	w.Col.Click(pt, button)
}

func setWinFocusPt(w *Win, pt image.Point) {
	for i, c := range w.cols {
		if pt.X < x1(w, i) {
			setWinFocus(w, c)
			break
		}
	}
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

// OutputString appends a string to the Output sheet
// and ensures that the Output sheet is visible.
// It is safe for concurrent calls.
func (w *Win) OutputString(str string) {
	w.mu.Lock()
	w.outputBuffer.WriteString(str)
	w.mu.Unlock()
}

// OutputBytes appends bytes to the Output sheet
// and ensures that the Output sheet is visible.
// It is safe for concurrent calls.
func (w *Win) OutputBytes(data []byte) {
	w.mu.Lock()
	w.outputBuffer.Write(data)
	w.mu.Unlock()
}

func x0(w *Win, i int) int {
	if i == 0 {
		return 0
	}
	return x1(w, i-1) + framePx
}

func x1(w *Win, i int) int { return int(w.widths[i] * dx(w)) }

func dx(w *Win) float64 { return float64(w.size.X) }

func setWinFocus(w *Win, c *Col) {
	if w.Col == c {
		return
	}
	w.Col.Focus(false)
	c.Focus(true)
	w.Col = c
}

func focusedCol(w *Win) int {
	i := colIndex(w.Col)
	if i < 0 {
		return 0
	}
	return i
}
