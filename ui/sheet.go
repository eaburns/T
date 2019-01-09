package ui

import (
	"image"
	"image/draw"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/eaburns/T/edit"
	"github.com/eaburns/T/rope"
)

// A Sheet is a tag and a body.
// TODO: better document the Sheet type.
type Sheet struct {
	tag           *TextBox
	body          *TextBox
	tagH, minTagH int
	size          image.Point
	*TextBox      // the focus element: the tag or the body.
}

// NewSheet returns a new sheet.
func NewSheet(w *Win, title string) *Sheet {
	var (
		tagTextStyles = [...]TextStyle{
			{FG: fg, BG: tagBG, Face: w.face},
			{BG: hiBG1},
			{BG: hiBG2},
			{BG: hiBG3},
		}
		bodyTextStyles = [...]TextStyle{
			{FG: fg, BG: bodyBG, Face: w.face},
			{BG: hiBG1},
			{BG: hiBG2},
			{BG: hiBG3},
		}
	)
	tag := NewTextBox(w, tagTextStyles, image.ZP)
	body := NewTextBox(w, bodyTextStyles, image.ZP)
	s := &Sheet{
		tag:     tag,
		body:    body,
		minTagH: w.lineHeight,
		TextBox: body,
	}
	tag.setHighlighter(s)
	tag.SetText(rope.New(tagText))
	s.SetTitle(title)
	return s
}

// Body returns the sheet's body text box.
func (s *Sheet) Body() *TextBox { return s.body }

// Tick handles tic events.
func (s *Sheet) Tick() bool {
	redraw1 := s.body.Tick()
	redraw2 := s.tag.Tick()
	return redraw1 || redraw2
}

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

// Resize handles resize events.
func (s *Sheet) Resize(size image.Point) {
	s.size = size
	resetTagHeight(s, size)
	s.body.Resize(image.Pt(size.X, size.Y-s.tagH))
}

// Update watches for updates to the tag and resizes it to fit the text height.
func (s *Sheet) Update([]Highlight, edit.Diffs, rope.Rope) []Highlight {
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
	if s.tagH = s.tag.textHeight(); s.tagH < s.minTagH {
		s.tagH = s.minTagH
	}
	s.tag.Resize(image.Pt(size.X, s.tagH))
}

// Move handles movement events.
func (s *Sheet) Move(pt image.Point) {
	if s.TextBox == s.body {
		pt.Y -= s.tagH
	}
	s.TextBox.Move(pt)
}

// Click handles click events.
func (s *Sheet) Click(pt image.Point, button int) (int, [2]int64) {
	if button > 0 {
		setSheetFocus(s, pt, button)
	}

	if s.TextBox == s.body {
		pt.Y -= s.tagH
	}
	return s.TextBox.Click(pt, button)
}

func setSheetFocus(s *Sheet, pt image.Point, button int) bool {
	if button != 1 {
		return false
	}
	if pt.Y < s.tagH {
		if s.TextBox != s.tag {
			s.TextBox = s.tag
			s.body.Focus(false)
			s.tag.Focus(true)
			return true
		}
	} else {
		if s.TextBox != s.body {
			s.TextBox = s.body
			s.tag.Focus(false)
			s.body.Focus(true)
			return true
		}
	}
	return false
}

// Title returns the title of the sheet.
// The title is the first space-terminated string in the tag,
// or if the first rune of the tag is ' , it is the first ' terminated string
// with \' as an escaped ' and \\ as an escaped \.
func (s *Sheet) Title() string {
	_, title := s.title()
	return title
}

func (s *Sheet) title() (int64, string) {
	txt := s.tag.text
	if rope.IndexRune(txt, '\'') < 0 {
		i := rope.IndexFunc(txt, unicode.IsSpace)
		if i < 0 {
			i = txt.Len()
		}
		return i, rope.Slice(txt, 0, i).String()
	}

	var i int64
	var esc bool
	var title strings.Builder
	rr := rope.NewReader(txt)
	rr.ReadRune() // discard '
	for {
		r, w, err := rr.ReadRune()
		i += int64(w)
		switch {
		case err != nil: // must be io.EOF from rope.Reader
			fallthrough
		case !esc && r == '\'':
			return i + 1, title.String() // +1 for leading '
		case !esc && r == '\\':
			esc = true
		default:
			esc = false
			title.WriteRune(r)
		}
	}
}

// SetTitle sets the title of the sheet.
func (s *Sheet) SetTitle(title string) {
	r, _ := utf8.DecodeRuneInString(title)
	if r == '\'' || strings.IndexFunc(title, unicode.IsSpace) >= 0 {
		title = strings.Replace(title, `\`, `\\`, -1)
		title = strings.Replace(title, `'`, `\'`, -1)
		title = `'` + title + `'`
	}
	end, _ := s.title()
	s.tag.Change([]edit.Diff{{At: [2]int64{0, end}, Text: rope.Empty()}})
	r, _, err := rope.NewReader(s.tag.text).ReadRune()
	if err == nil && !unicode.IsSpace(r) {
		title += " "
	}
	s.tag.Change([]edit.Diff{{At: [2]int64{0, 0}, Text: rope.New(title)}})
}
