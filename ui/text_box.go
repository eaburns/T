package ui

import (
	"bufio"
	"image"
	"image/color"
	"image/draw"
	"io/ioutil"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/eaburns/T/edit"
	"github.com/eaburns/T/rope"
	"github.com/eaburns/T/syntax"
	"github.com/eaburns/T/text"
	"golang.org/x/image/math/fixed"
)

const (
	blinkDuration       = 500 * time.Millisecond
	dragScrollDuration  = 20 * time.Millisecond
	wheelScrollDuration = 20 * time.Millisecond
	doubleClickDuration = 500 * time.Millisecond
)

// TextBox is an editable text box UI widget.
type TextBox struct {
	win  *Win
	size image.Point
	text rope.Rope
	at   int64 // address of the first rune in the window

	focus      bool
	showCursor bool
	blinkTime  time.Time
	cursorCol  int // rune offset of the cursor in its line; -1 is recompute

	button         int         // currently held mouse button
	pt             image.Point // where's the mouse? 0 is just after textPadPx
	clickAt        int64       // address of the glyph clicked by the mouse
	clickTime      time.Time
	dragAt         int64           // address of the glyph under the dragging mouse
	dragTextBox    image.Rectangle // bounding-box of the dragAt glyph
	dragScrollTime time.Time       // time when dragging off screen scrolls
	wheelTime      time.Time       // time when we will consider the next wheel

	style       text.Style
	dots        [4]syntax.Highlight // cursor for unused, click 1, click 2, and click 3.
	highlight   []syntax.Highlight  // highlighted words
	syntax      []syntax.Highlight  // syntax highlighting
	highlighter updater             // syntax highlighter

	dirty  bool
	_lines []line
	now    func() time.Time
}

type line struct {
	dirty bool
	n     int64
	a, h  fixed.Int26_6
	spans []span
}

type span struct {
	w     fixed.Int26_6
	style text.Style
	text  string
}

// NewTextBox returns a new, empty text box.
// The styles are:
// 	0: default style
// 	1: 1-click selection style
// 	2: 2-click selection style
// 	3: 3-click selection style
func NewTextBox(w *Win, styles [4]text.Style, size image.Point) *TextBox {
	b := &TextBox{
		win:   w,
		size:  size,
		text:  rope.Empty(),
		style: styles[0],
		dots: [...]syntax.Highlight{
			{Style: styles[0]},
			{Style: styles[1]},
			{Style: styles[2]},
			{Style: styles[3]},
		},
		cursorCol: -1,
		now:       func() time.Time { return time.Now() },
	}
	return b
}

// Text returns the current text of the text box.
func (b *TextBox) Text() rope.Rope { return b.text }

// SetText sets the text of the text box.
// The text box always must be redrawn after setting the text.
func (b *TextBox) SetText(text rope.Rope) {
	b.text = text

	b.at = 0
	b.cursorCol = -1
	b.clickAt = 0
	b.dragAt = 0
	b.dragTextBox = image.ZR
	for i := range b.dots {
		b.dots[i].At = [2]int64{}
	}
	b.highlight = nil
	if b.highlighter != nil {
		b.syntax = b.highlighter.Update(nil, nil, b.text)
	}
	dirtyLines(b)
}

// textHeight returns the height of the displayed text.
func (b *TextBox) textHeight() int {
	var y fixed.Int26_6
	lines := b.lines()
	for _, l := range lines {
		y += l.h
	}
	if len(lines) > 0 {
		l := &lines[len(lines)-1]
		if len(l.spans) > 0 {
			s := &l.spans[len(l.spans)-1]
			r, _ := utf8.DecodeLastRuneInString(s.text)
			if r == '\n' {
				m := b.style.Face.Metrics()
				h := m.Height + m.Descent
				y += h
			}
		}
	}
	return y.Ceil()
}

type updater interface {
	Update([]syntax.Highlight, edit.Diffs, rope.Rope) []syntax.Highlight
}

func (b *TextBox) setHighlighter(highlighter updater) {
	b.highlighter = highlighter
	if b.highlighter != nil {
		b.syntax = b.highlighter.Update(nil, nil, b.text)
	}
	dirtyLines(b)
}

// Edit performs an edit on the text of the text box
// and returns the diffs applied to the text.
// If more than 0 diffs are returned, the text box needs to be redrawn.
func (b *TextBox) Edit(t string) (edit.Diffs, error) { return ed(b, t) }

func ed(b *TextBox, t string) (edit.Diffs, error) {
	dot := b.dots[1].At
	diffs, err := edit.Edit(dot, t, ioutil.Discard, b.text)
	if err != nil {
		return nil, err
	}
	if len(diffs) > 0 {
		dot := b.dots[1].At
		b.Change(diffs)
		dot[0] = diffs[len(diffs)-1].At[0]
		dot[1] = dot[0] + diffs[len(diffs)-1].TextLen()
		b.dots[1].At = dot
	}
	return diffs, nil
}

// Change applies a set of diffs to the text box.
func (b *TextBox) Change(diffs edit.Diffs) {
	if len(diffs) == 0 {
		return
	}
	dirtyLines(b)
	b.text, _ = diffs.Apply(b.text)

	// TODO: if something else deletes \n before b.at,
	// scroll to beginning of whatever the line becomes.
	b.at = diffs.Update([2]int64{b.at, b.at})[0]

	for i := 1; i < len(b.dots); i++ {
		b.dots[i].At = diffs.Update(b.dots[i].At)
	}
	if b.highlighter != nil {
		b.syntax = b.highlighter.Update(b.syntax, diffs, b.text)
	}
	for i := range b.highlight {
		b.highlight[i].At = diffs.Update(b.highlight[i].At)
	}
}

// Resize handles a resize event.
// The text box must always be redrawn after being resized.
func (b *TextBox) Resize(size image.Point) {
	b.size = size
	dirtyLines(b)
}

// Focus handles a focus state change.
func (b *TextBox) Focus(focus bool) {
	b.focus = focus
	b.showCursor = focus
	dirtyDot(b, b.dots[1].At)
	if focus {
		b.blinkTime = b.now().Add(blinkDuration)
	} else {
		b.button = 0
	}
}

// Tick handles periodic ticks that drive
// asynchronous events for the text box.
// It returns whether the text box image needs to be redrawn.
//
// Tick is intended to be called at regular intervals,
// fast enough to drive cursor blinking and mouse-drag scolling.
func (b *TextBox) Tick() bool {
	now := b.now()
	redraw := b.dirty
	if b.focus && b.dots[1].At[0] == b.dots[1].At[1] && !b.blinkTime.After(now) {
		b.blinkTime = now.Add(blinkDuration)
		b.showCursor = !b.showCursor
		dirtyDot(b, b.dots[1].At)
	}
	if b.button == 1 &&
		!b.dragScrollTime.After(now) {
		var ymax fixed.Int26_6
		atMax := b.at
		for _, l := range b.lines() {
			ymax += l.h
			atMax += l.n
		}
		switch {
		case b.pt.Y < 0:
			scrollUp(b, 1)
			b.Move(b.pt)
			b.dragScrollTime = now.Add(dragScrollDuration)
		case b.pt.Y >= ymax.Floor() && atMax < b.text.Len():
			scrollDown(b, 1)
			b.Move(b.pt)
			b.dragScrollTime = now.Add(dragScrollDuration)
		}
	}
	return redraw
}

// Move handles the event of the mouse cursor moving to a point
// and returns whether the text box image needs to be redrawn.
func (b *TextBox) Move(pt image.Point) {
	pt.X -= textPadPx
	b.pt = pt
	if b.button <= 0 || b.button >= len(b.dots) || pt.In(b.dragTextBox) {
		return
	}
	b.dragAt, b.dragTextBox = atPoint(b, pt)
	if b.clickAt <= b.dragAt {
		setDot(b, b.button, b.clickAt, b.dragAt)
	} else {
		setDot(b, b.button, b.dragAt, b.clickAt)
	}
}

// Wheel handles the event of the mouse wheel rolling
// and returns whether the text box image needs to be redrawn.
// 	-y is roll up.
// 	+y is roll down.
// 	-x is roll left.
// 	+x is roll right.
func (b *TextBox) Wheel(_ image.Point, x, y int) {
	now := b.now()
	if b.wheelTime.After(now) {
		return
	}
	b.wheelTime = now.Add(wheelScrollDuration)
	switch {
	case y < 0:
		scrollDown(b, 1)
	case y > 0:
		scrollUp(b, 1)
	}
}

// Click handles a mouse button press or release event.
// The first return value is the button ultimately pressed
// (this can differ from the argument button, for example,
// if modifier keys are being held).
// If the button is < 0, the second return value is the clicked address.
// The third return value is whether the text box image needs to be redrawn.
//
// The absolute value of the argument indicates the mouse button.
// A positive value indicates the button was pressed.
// A negative value indicates the button was released.
func (b *TextBox) Click(pt image.Point, button int) (int, [2]int64) {
	pt.X -= textPadPx
	b.pt = pt
	switch {
	case b.button > 0 && button > 0:
		// b.button/button mouse chord; ignore it for now.
		return button, [2]int64{}

	case b.button > 0 && button == -b.button:
		return unclick(b)

	case b.button == 0 && button == 1 && b.win.mods[2]:
		button = 2

	case b.button == 0 && button == 1 && b.win.mods[3]:
		button = 3

	case b.button != 1 && button == -1: // mod-button unclick
		return unclick(b)
	}
	if button > 0 {
		click(b, button)
	}
	return button, [2]int64{}
}

func unclick(b *TextBox) (int, [2]int64) {
	button := b.button
	b.button = 0
	dot := b.dots[button].At
	if button != 1 {
		setDot(b, button, 0, 0)
	}
	return -button, dot
}

func click(b *TextBox, button int) {
	b.button = button
	if button == 1 {
		if b.now().Sub(b.clickTime) < doubleClickDuration {
			doubleClick(b)
			return
		}
		b.clickTime = b.now()
	}
	b.clickAt, b.dragTextBox = atPoint(b, b.pt)
	setDot(b, button, b.clickAt, b.clickAt)
	if button == 1 {
		b.cursorCol = -1
	}
}

var delim = [][2]rune{
	{'(', ')'},
	{'{', '}'},
	{'[', ']'},
	{'<', '>'},
	{'«', '»'},
	{'\'', '\''},
	{'"', '"'},
	{'`', '`'},
	{'“', '”'},
}

func doubleClick(b *TextBox) {
	prev := prevRune(b)
	for _, ds := range delim {
		if ds[0] == prev {
			selectForwardDelim(b, ds[0], ds[1])
			return
		}
	}
	cur := curRune(b)
	for _, ds := range delim {
		if ds[1] == cur {
			selectReverseDelim(b, ds[1], ds[0])
			return
		}
	}
	if prev == -1 || prev == '\n' || cur == -1 || cur == '\n' {
		selectLine(b)
		return
	}
	if wordRune(cur) {
		selectWord(b)
		return
	}
}

func prevRune(b *TextBox) rune {
	front, _ := rope.Split(b.text, b.dots[1].At[0])
	rr := rope.NewReverseReader(front)
	r, _, err := rr.ReadRune()
	if err != nil {
		return -1
	}
	return r
}

func curRune(b *TextBox) rune {
	_, back := rope.Split(b.text, b.dots[1].At[0])
	rr := rope.NewReader(back)
	r, _, err := rr.ReadRune()
	if err != nil {
		return -1
	}
	return r
}

func selectForwardDelim(b *TextBox, open, close rune) {
	nest := 1
	_, back := rope.Split(b.text, b.dots[1].At[0])
	end := rope.IndexFunc(back, func(r rune) bool {
		switch r {
		case close:
			nest--
		case open:
			nest++
		}
		return nest == 0
	})
	if end < 0 {
		return
	}
	setDot(b, 1, b.dots[1].At[0], end+b.dots[1].At[0])
}

func selectReverseDelim(b *TextBox, open, close rune) {
	nest := 1
	front, _ := rope.Split(b.text, b.dots[1].At[0])
	start := rope.LastIndexFunc(front, func(r rune) bool {
		switch r {
		case close:
			nest--
		case open:
			nest++
		}
		return nest == 0
	})
	if start < 0 {
		return
	}
	setDot(b, 1, start+int64(utf8.RuneLen(open)), b.dots[1].At[0])
}

func selectLine(b *TextBox) {
	front, back := rope.Split(b.text, b.dots[1].At[0])
	start := rope.LastIndexFunc(front, func(r rune) bool { return r == '\n' })
	if start < 0 {
		start = 0
	} else {
		start++ // Don't include the \n.
	}
	end := rope.IndexFunc(back, func(r rune) bool { return r == '\n' })
	if end < 0 {
		end = b.text.Len()
	} else {
		end += b.dots[1].At[0] + 1 // Do include the \n.
	}
	setDot(b, 1, start, end)
}

func selectWord(b *TextBox) {
	front, back := rope.Split(b.text, b.dots[1].At[0])
	var delim rune
	start := rope.LastIndexFunc(front, func(r rune) bool {
		delim = r
		return !wordRune(r)
	})
	if start < 0 {
		start = 0
	} else {
		start += int64(utf8.RuneLen(delim))
	}
	end := rope.IndexFunc(back, func(r rune) bool { return !wordRune(r) })
	if end < 0 {
		end = b.text.Len()
	} else {
		end += b.dots[1].At[0]
	}
	setDot(b, 1, start, end)
}

func wordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_'
}

// Dir handles a keyboard directional event
// and returns whether the text box image needs to be redrawn.
//
// These events are generated by the arrow keys,
// page up and down keys, and the home and end keys.
// Exactly one of x or y must be non-zero.
//
// If the absolute value is 1, then it is treated as an arrow key
// in the corresponding direction (x-horizontal, y-vertical,
// negative-left/up, positive-right/down).
// If the absolute value is math.MinInt16, it is treated as a home event.
// If the absolute value is math.MathInt16, it is end.
// Otherwise, if the value for y is non-zero it is page up/down.
// Other non-zero values for x are currently ignored.
//
// Dir only handles key press events, not key releases.
func (b *TextBox) Dir(x, y int) {
	switch {
	case x == -1:
		at := leftRight(b, "-")
		b.cursorCol = -1
		setDot(b, 1, at, at)
	case x == 1:
		at := leftRight(b, "+")
		b.cursorCol = -1
		setDot(b, 1, at, at)
	case y == -1:
		at := upDown(b, "-")
		setDot(b, 1, at, at)
	case y == 1:
		at := upDown(b, "+")
		setDot(b, 1, at, at)
	case y == math.MinInt16:
		showAddr(b, 0)
	case y == math.MaxInt16:
		showAddr(b, b.text.Len())
	case y < 0:
		scrollUp(b, pageSize(b))
	case y > 0:
		scrollDown(b, pageSize(b))
	}
}

func pageSize(b *TextBox) int {
	m := b.style.Face.Metrics()
	h := (m.Height + m.Descent).Floor()
	if h == 0 {
		return 1
	}
	return b.size.Y / (4 * h)
}

func leftRight(b *TextBox, dir string) int64 {
	var at [2]int64
	var err error
	if b.dots[1].At[0] < b.dots[1].At[1] {
		at, err = edit.Addr(b.dots[1].At, dir+"#0", b.text)
	} else {
		at, err = edit.Addr(b.dots[1].At, dir+"#1", b.text)
	}
	if err != nil {
		return b.dots[1].At[0]
	}
	return at[0]
}

func upDown(b *TextBox, dir string) int64 {
	if b.cursorCol < 0 {
		b.cursorCol = cursorCol(b)
	}

	// prev/next line
	// -+ selects the entire line containing dot.
	// This handles the case where the cursor is at 0,
	// and 0+1 is the first line instead of the second.
	at, err := edit.Addr(b.dots[1].At, "-+"+dir, b.text)
	if err != nil {
		if dir == "+" {
			return b.text.Len()
		}
		return 0
	}

	// rune offset into the line
	max := at[1]
	at, err = edit.Addr([2]int64{at[0], at[0]}, "+#"+strconv.Itoa(b.cursorCol), b.text)
	if err != nil || max == 0 {
		return max
	}
	if at[0] >= max {
		return max - 1
	}
	return at[0]
}

func cursorCol(b *TextBox) int {
	var n int
	rr := rope.NewReverseReader(rope.Slice(b.text, 0, b.dots[1].At[0]))
	for {
		r, _, err := rr.ReadRune()
		if err != nil || r == '\n' {
			break
		}
		n++
	}
	return n
}

func scrollUp(b *TextBox, delta int) {
	if b.at == 0 {
		return
	}
	bol, err := edit.Addr([2]int64{b.at, b.at}, "-0", b.text)
	if err != nil {
		panic(err.Error())
	}
	if b.at != bol[0] {
		b.at = bol[0]
		delta--
	}
	for i := 0; i < delta; i++ {
		at, err := edit.Addr([2]int64{b.at, b.at}, "-1", b.text)
		if err != nil {
			panic(err.Error())
		}
		if b.at = at[0]; b.at == 0 {
			break
		}
	}
	dirtyLines(b)
}

func scrollDown(b *TextBox, delta int) {
	lines := b.lines()
	for i := 0; i < delta; i++ {
		if len(lines) > 0 {
			b.at += lines[0].n
			lines = lines[1:]
			continue
		}
		at, err := edit.Addr([2]int64{b.at, b.at}, "+1", b.text)
		if err != nil {
			// Must be EOF.
			b.at = b.text.Len()
			break
		}
		if b.at = at[0]; b.at == 0 {
			break
		}
	}
	dirtyLines(b)
}

// Mod handles a modifier key state change event.
func (b *TextBox) Mod(m int) {
	if b.button > 0 {
		b.Click(b.pt, m)
	}
}

const (
	esc = 0x1b
	del = 0x7f
)

// Rune handles the event of a rune being typed
// and returns whether the text box image needs to be redrawn.
//
// The argument is a rune indicating the glyph typed
// after interpretation by any system-dependent
// keyboard/layout mapping.
// For example, if the 'a' key is pressed
// while the shift key is held,
// the argument would be the letter 'A'.
//
// If the rune is positive, the event is a key press,
// if negative, a key release.
func (b *TextBox) Rune(r rune) {
	switch r {
	case '\b':
		if b.dots[1].At[0] == b.dots[1].At[1] {
			ed(b, ".-#1,.d")
		} else {
			ed(b, ".d")
		}
	case del, esc:
		if b.dots[1].At[0] == b.dots[1].At[1] {
			ed(b, ".,.+#1d")
		} else {
			ed(b, ".d")
		}
	case '/':
		ed(b, ".c/\\/")
	case '\n':
		ed(b, ".c/\\n")
	default:
		ed(b, ".c/"+string([]rune{r}))
	}
	setDot(b, 1, b.dots[1].At[1], b.dots[1].At[1])
}

// Draw draws the text box to the image with the upper-left of the box at 0,0.
func (b *TextBox) Draw(dirty bool, img draw.Image) {
	size := img.Bounds().Size()
	if dirty || size != b.size {
		b.size = size
		dirtyLines(b)
	}
	if !b.dirty {
		return
	}
	b.dirty = false
	at := b.at
	lines := b.lines()
	var y fixed.Int26_6
	for i := range lines {
		l := &lines[i]
		if !l.dirty {
			y += l.h
			at += l.n
			continue
		}
		at1 := at + l.n
		drawLine(b, img, at, y, *l)
		l.dirty = false
		y += l.h
		at = at1
	}
	if y.Floor() < size.Y {
		r := image.Rect(0, y.Floor(), size.X, size.Y)
		fillRect(img, b.style.BG, r.Add(img.Bounds().Min))
	}

	// Draw a cursor for empty text.
	if b.text.Len() == 0 {
		m := b.style.Face.Metrics()
		h := m.Height + m.Descent
		drawCursor(b, img, fixed.I(textPadPx), 0, h)
		return
	}
	// Draw a cursor just after the last line of text.
	if len(lines) == 0 {
		return
	}
	lastLine := &lines[len(lines)-1]
	if b.dots[1].At[0] == b.dots[1].At[1] &&
		at == b.dots[1].At[0] &&
		at == b.text.Len() &&
		lastRune(lastLine) == '\n' {
		m := b.style.Face.Metrics()
		h := m.Height + m.Descent
		drawCursor(b, img, fixed.I(textPadPx), y, y+h)
	}
}

func drawLine(b *TextBox, img draw.Image, at int64, y0 fixed.Int26_6, l line) {
	var prevRune rune
	x0 := fixed.I(textPadPx)
	yb, y1 := y0+l.a, y0+l.h

	// leading padding
	pad := image.Rect(0, y0.Floor(), textPadPx, y1.Floor())
	fillRect(img, b.style.BG, pad.Add(img.Bounds().Min))

	for i, s := range l.spans {
		x1 := x0 + s.w

		bbox := image.Rect(x0.Floor(), y0.Floor(), x1.Floor(), y1.Floor())
		fillRect(img, s.style.BG, bbox.Add(img.Bounds().Min))

		for _, r := range s.text {
			if prevRune != 0 {
				x0 += s.style.Face.Kern(prevRune, r)
			}
			prevRune = r
			var adv fixed.Int26_6
			if r == '\t' || r == '\n' {
				adv = advance(b, s.style, x0-fixed.I(textPadPx), r)
			} else {
				adv = drawGlyph(img, s.style, x0, yb, r)
			}
			if b.dots[1].At[0] == b.dots[1].At[1] && b.dots[1].At[0] == at {
				drawCursor(b, img, x0, y0, y1)
			}
			x0 += adv
			at += int64(utf8.RuneLen(r))
		}
		x0 = x1
		if i < len(l.spans)-1 && l.spans[i+1].style.Face != s.style.Face {
			prevRune = 0
		}
	}

	// trailing padding
	r := image.Rect(x0.Floor(), y0.Floor(), img.Bounds().Size().X, y1.Floor())
	fillRect(img, b.style.BG, r.Add(img.Bounds().Min))

	if b.dots[1].At[0] == b.dots[1].At[1] &&
		at == b.dots[1].At[0] &&
		at == b.text.Len() &&
		prevRune != '\n' {
		drawCursor(b, img, x0, y0, y1)
	}
}

func drawGlyph(img draw.Image, style text.Style, x0, yb fixed.Int26_6, r rune) fixed.Int26_6 {
	pt := fixed.Point26_6{X: x0, Y: yb}
	dr, m, mp, adv, ok := style.Face.Glyph(pt, r)
	if !ok {
		dr, m, mp, adv, _ = style.Face.Glyph(pt, unicode.ReplacementChar)
	}
	dr = dr.Add(img.Bounds().Min)
	fg := image.NewUniform(style.FG)
	draw.DrawMask(img, dr, fg, image.ZP, m, mp, draw.Over)
	return adv
}

func drawCursor(b *TextBox, img draw.Image, x, y0, y1 fixed.Int26_6) {
	if !b.showCursor {
		return
	}
	x0 := x.Floor()
	r := image.Rect(x0, y0.Floor(), x0+cursorWidthPx, y1.Floor())
	fillRect(img, b.style.FG, r.Add(img.Bounds().Min))
}

func fillRect(img draw.Image, c color.Color, r image.Rectangle) {
	draw.Draw(img, r, image.NewUniform(c), image.ZP, draw.Src)
}

func atPoint(b *TextBox, pt image.Point) (int64, image.Rectangle) {
	lines := b.lines()
	if len(lines) == 0 {
		m := b.style.Face.Metrics()
		h := m.Height + m.Descent
		return b.at, image.Rect(0, 0, 0, h.Floor())
	}

	at := b.at
	var l *line
	var y0, y1 fixed.Int26_6
	for i := range lines {
		l = &lines[i]
		y1 = y0 + l.h
		if i == len(lines)-1 || y1.Floor() > pt.Y {
			break
		}
		at += l.n
		y0 = y1
	}

	if y1.Floor() <= pt.Y && lastRune(l) == '\n' {
		// The cursor is at the beginning of the line following the last.
		m := b.style.Face.Metrics()
		h := m.Height + m.Descent
		return at + l.n, image.Rect(0, y1.Floor(), 0, (y1 + h).Floor())
	}

	at0 := at
	var s *span
	var prevTextStyle text.Style
	var prevRune rune
	var x0, x1 fixed.Int26_6
	for i := range l.spans {
		s = &l.spans[i]
		r, _ := utf8.DecodeRuneInString(s.text)
		if prevTextStyle == s.style {
			x0 += kern(s.style, prevRune, r)
		}
		x1 = x0 + s.w
		if i == len(l.spans)-1 || x1.Floor() > pt.X {
			break
		}
		at += int64(len(s.text))
		prevRune, _ = utf8.DecodeLastRuneInString(s.text)
		prevTextStyle = s.style
		x0 = x1
	}

	if y1.Floor() <= pt.Y {
		x := x0.Floor()
		return at0 + l.n, image.Rect(x, y1.Floor(), x, (y1 + l.h).Floor())
	}

	x1 = x0
	for _, r := range s.text {
		x0 += kern(s.style, prevRune, r)
		x1 = x0 + advance(b, s.style, x0, r)
		if x1.Floor() > pt.X {
			break
		}
		rl := utf8.RuneLen(r)
		at += int64(rl)
		x0 = x1
		prevRune = r
	}
	rect := image.Rect(x0.Floor(), y0.Floor(), x1.Floor(), y1.Floor())
	return at, rect
}

func lastRune(l *line) rune {
	if len(l.spans) == 0 {
		return utf8.RuneError
	}
	s := &l.spans[len(l.spans)-1]
	r, _ := utf8.DecodeLastRuneInString(s.text)
	return r
}

func setDot(b *TextBox, i int, start, end int64) {
	if start < 0 || start > b.text.Len() {
		panic("bad start")
	}
	if end < 0 || end > b.text.Len() {
		panic("bad end")
	}
	dirtyDot(b, b.dots[i].At)
	b.dots[i].At[0] = start
	b.dots[i].At[1] = end
	if i == 1 && start == end {
		b.showCursor = true
		b.blinkTime = b.now().Add(blinkDuration)
	}
	if dirtyDot(b, b.dots[i].At) {
		showAddr(b, b.dots[i].At[0])
	}
}

func showAddr(b *TextBox, at int64) {
	bol, err := edit.Addr([2]int64{at, at}, "-0", b.text)
	if err != nil {
		panic(err.Error())
	}
	b.at = bol[0]
	// TODO: This shows the start of the line containing the addr.
	// If it's a multi-line text line, then we may need to scroll forward
	// in order to see the address.
	scrollUp(b, pageSize(b))
	dirtyLines(b)
}

// dirtyDot returns true if the dot is a point that is off screen.
func dirtyDot(b *TextBox, dot [2]int64) bool {
	if dot[0] < dot[1] {
		dirtyLines(b)
		return false
	}
	b.dirty = true
	at0 := b.at
	lines := b.lines()
	var y0 fixed.Int26_6
	for i := range lines {
		at1 := at0 + lines[i].n
		if at0 <= dot[0] && dot[0] < at1 {
			lines[i].dirty = true
			return false
		}
		y0 += lines[i].h
		at0 = at1
	}
	if n := len(lines); n > 0 &&
		dot[0] == b.text.Len() &&
		lastRune(&lines[n-1]) != '\n' {
		lines[n-1].dirty = true
		return false
	}
	m := b.style.Face.Metrics()
	h := m.Height + m.Descent
	return dot[0] < b.at || (y0+h).Floor() >= b.size.Y
}

func dirtyLines(b *TextBox) {
	b.dirty = true
	b._lines = b._lines[:0]
}

func (b *TextBox) lines() []line {
	if len(b._lines) == 0 {
		reset(b)
	}
	return b._lines
}

func reset(b *TextBox) {
	at := b.at
	rs := bufio.NewReader(
		rope.NewReader(rope.Slice(b.text, b.at, b.text.Len())),
	)
	maxx := b.size.X - 2*textPadPx
	var y fixed.Int26_6
	var txt strings.Builder
	stack := [][]syntax.Highlight{b.syntax, b.highlight, {b.dots[1]}, {b.dots[2]}, {b.dots[3]}}
	for at < b.text.Len() && y < fixed.I(b.size.Y) {
		var prevRune rune
		var x0, x fixed.Int26_6
		m := b.style.Face.Metrics()
		line := line{dirty: true, a: m.Ascent, h: m.Height + m.Descent}
		style, stack, next := nextTextStyle(b.style, stack, at)
		for {
			r, w, err := rs.ReadRune()
			if err != nil {
				break
			}
			x += kern(style, prevRune, r)
			if r == '\n' {
				txt.WriteRune(r)
				at++
				line.n++
				x = fixed.I(maxx)
				break
			}
			adv := advance(b, style, x, r)
			if (x + adv).Ceil() >= maxx {
				x = fixed.I(maxx)
				rs.UnreadRune()
				break
			}
			txt.WriteRune(r)
			x += adv
			at += int64(w)
			line.n += int64(w)
			if at == next {
				appendSpan(&line, x0, x, style, &txt)
				x0 = x
				prevFace := style.Face
				style, stack, next = nextTextStyle(b.style, stack, at)
				if prevFace != style.Face {
					prevRune = 0
				}
			}
		}
		appendSpan(&line, x0, x, style, &txt)
		if y += line.h; y > fixed.I(b.size.Y) {
			break
		}
		b._lines = append(b._lines, line)
	}
}

func appendSpan(line *line, x0, x fixed.Int26_6, style text.Style, text *strings.Builder) {
	m := style.Face.Metrics()
	line.a = max(line.a, m.Ascent)
	line.h = max(line.h, m.Height+m.Descent)
	line.spans = append(line.spans, span{
		w:     x - x0,
		text:  text.String(),
		style: style,
	})
	text.Reset()
}

func kern(style text.Style, prev, cur rune) fixed.Int26_6 {
	if prev == 0 {
		return 0
	}
	return style.Face.Kern(prev, cur)
}

func max(a, b fixed.Int26_6) fixed.Int26_6 {
	if a > b {
		return a
	}
	return b
}

func nextTextStyle(def text.Style, stack [][]syntax.Highlight, at int64) (text.Style, [][]syntax.Highlight, int64) {
	style, next := def, int64(-1)
	for i := range stack {
		for len(stack[i]) > 0 && stack[i][0].At[1] <= at {
			stack[i] = stack[i][1:]
		}
		if len(stack[i]) == 0 {
			continue
		}
		hi := stack[i][0]
		if hi.At[0] > at {
			if hi.At[0] < next || next < 0 {
				next = hi.At[0]
			}
			continue
		}
		if hi.At[1] < next || next < 0 {
			next = hi.At[1]
		}
		style = style.Merge(hi.Style)
	}
	return style, stack, next
}

func advance(b *TextBox, style text.Style, x fixed.Int26_6, r rune) fixed.Int26_6 {
	switch r {
	case '\n':
		return fixed.I(b.size.X-2*textPadPx) - x
	case '\t':
		spaceWidth, ok := b.style.Face.GlyphAdvance(' ')
		if !ok {
			return 0
		}
		tabWidth := spaceWidth.Mul(fixed.I(8))
		adv := tabWidth - (x % tabWidth)
		if adv < spaceWidth {
			adv += tabWidth
		}
		return adv
	default:
		adv, ok := style.Face.GlyphAdvance(r)
		if !ok {
			adv, _ = style.Face.GlyphAdvance(unicode.ReplacementChar)
		}
		return adv
	}
}
