// T is a text editor.
package main

import (
	"context"
	"image"
	"image/color"
	"image/draw"
	"math"
	"time"

	"github.com/eaburns/T/edit"
	"github.com/eaburns/T/re1"
	"github.com/eaburns/T/rope"
	"github.com/eaburns/T/text"
	"github.com/eaburns/T/ui"
	"golang.org/x/exp/shiny/driver/gldriver"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/image/font"
	"golang.org/x/image/math/f64"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
)

const tickRate = 20 * time.Millisecond

func main() {
	gldriver.Main(func(scr screen.Screen) {
		<-newWindow(context.Background(), scr).done
	})
}

type win struct {
	ctx    context.Context
	cancel func()
	done   chan struct{}

	dpi  float32
	size image.Point
	screen.Window

	win *ui.Win
}

func newWindow(ctx context.Context, scr screen.Screen) *win {
	window, err := scr.NewWindow(nil)
	if err != nil {
		panic(err)
	}
	var e size.Event
	for {
		var ok bool
		if e, ok = window.NextEvent().(size.Event); ok {
			break
		}
	}

	ctx, cancel := context.WithCancel(ctx)
	w := &win{
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
		dpi:    float32(e.PixelsPerPt) * 72.0,
		size:   e.Size(),
		Window: window,
	}
	w.win = ui.NewWin(w.dpi)
	w.win.Resize(w.size)

	go tick(w)
	go poll(scr, w)
	return w
}

func (w *win) Release() { w.cancel() }

type done struct{}

func tick(w *win) {
	ticker := time.NewTicker(tickRate)
	for {
		select {
		case <-ticker.C:
			w.Send(time.Now())
		case <-w.ctx.Done():
			ticker.Stop()
			w.Send(done{})
			return
		}
	}
}

func poll(scr screen.Screen, w *win) {
	var mods [4]bool
	dirty := true
	buf, tex := bufTex(scr, w.size)

	for {
		switch e := w.NextEvent().(type) {
		case done:
			buf.Release()
			tex.Release()
			w.Window.Release()
			close(w.done)
			return

		case time.Time:
			if w.win.Tick() {
				w.Send(paint.Event{})
			}

		case lifecycle.Event:
			if e.To == lifecycle.StageDead {
				w.cancel()
				continue
			}
			w.win.Focus(e.To == lifecycle.StageFocused)

		case size.Event:
			if e.Size() == image.ZP {
				w.cancel()
				continue
			}
			w.size = e.Size()
			w.win.Resize(w.size)
			dirty = true
			if b := tex.Bounds(); b.Dx() < w.size.X || b.Dy() < w.size.Y {
				tex.Release()
				buf.Release()
				buf, tex = bufTex(scr, w.size.Mul(2))
			}

		case paint.Event:
			rect := image.Rectangle{Max: w.size}
			img := buf.RGBA().SubImage(rect).(*image.RGBA)
			w.win.Draw(dirty, img)
			dirty = false
			tex.Upload(image.ZP, buf, buf.Bounds())
			w.Draw(f64.Aff3{
				1, 0, 0,
				0, 1, 0,
			}, tex, tex.Bounds(), draw.Src, nil)
			w.Publish()

		case mouse.Event:
			mouseEvent(w, e)

		case key.Event:
			mods = keyEvent(w, mods, e)
		}
	}
}

func mouseEvent(w *win, e mouse.Event) {
	var redraw bool
	switch pt := image.Pt(int(e.X), int(e.Y)); {
	case e.Button == mouse.ButtonWheelUp:
		redraw = w.win.Wheel(0, 1)

	case e.Button == mouse.ButtonWheelDown:
		redraw = w.win.Wheel(0, -1)

	case e.Button == mouse.ButtonWheelLeft:
		redraw = w.win.Wheel(-1, 0)

	case e.Button == mouse.ButtonWheelRight:
		redraw = w.win.Wheel(1, 0)

	case e.Direction == mouse.DirNone:
		redraw = w.win.Move(pt)

	case e.Direction == mouse.DirPress:
		redraw = w.win.Click(pt, int(e.Button))

	case e.Direction == mouse.DirRelease:
		redraw = w.win.Click(pt, -int(e.Button))

	case e.Direction == mouse.DirStep:
		redraw = w.win.Click(pt, int(e.Button))
		redraw = w.win.Click(pt, -int(e.Button)) || redraw
	}
	if redraw {
		w.Send(paint.Event{})
	}
}

func keyEvent(w *win, mods [4]bool, e key.Event) [4]bool {
	if e.Direction == key.DirNone {
		e.Direction = key.DirPress
	}
	if e.Direction == key.DirPress && dirKeyCode[e.Code] {
		dirKey(w, e)
		return mods
	}

	switch {
	case e.Code == key.CodeDeleteBackspace:
		e.Rune = '\b'
	case e.Code == key.CodeDeleteForward:
		e.Rune = 0x7f
	case e.Rune == '\r':
		e.Rune = '\n'
	}
	if e.Rune > 0 {
		if e.Direction == key.DirPress {
			if w.win.Rune(e.Rune) {
				w.Send(paint.Event{})
			}
		}
		return mods
	}

	return modKey(w, mods, e)
}

var dirKeyCode = map[key.Code]bool{
	key.CodeUpArrow:    true,
	key.CodeDownArrow:  true,
	key.CodeLeftArrow:  true,
	key.CodeRightArrow: true,
	key.CodePageUp:     true,
	key.CodePageDown:   true,
	key.CodeHome:       true,
	key.CodeEnd:        true,
}

func dirKey(w *win, e key.Event) {
	var redraw bool
	switch e.Code {
	case key.CodeUpArrow:
		redraw = w.win.Dir(0, -1)

	case key.CodeDownArrow:
		redraw = w.win.Dir(0, 1)

	case key.CodeLeftArrow:
		redraw = w.win.Dir(-1, 0)

	case key.CodeRightArrow:
		redraw = w.win.Dir(1, 0)

	case key.CodePageUp:
		redraw = w.win.Dir(0, -2)

	case key.CodePageDown:
		redraw = w.win.Dir(0, 2)

	case key.CodeHome:
		redraw = w.win.Dir(0, math.MinInt16)

	case key.CodeEnd:
		redraw = w.win.Dir(0, math.MaxInt16)

	default:
		panic("impossible")
	}
	if redraw {
		w.Send(paint.Event{})
	}
}

func modKey(w *win, mods [4]bool, e key.Event) [4]bool {
	var newMods [4]bool
	if e.Modifiers&key.ModShift != 0 {
		newMods[1] = true
	}
	if e.Modifiers&key.ModAlt != 0 {
		newMods[2] = true
	}
	if e.Modifiers&key.ModMeta != 0 ||
		e.Modifiers&key.ModControl != 0 {
		newMods[3] = true
	}
	for i := 0; i < len(newMods); i++ {
		if newMods[i] != mods[i] {
			m := i
			if !newMods[i] {
				m = -m
			}
			if w.win.Mod(m) {
				w.Send(paint.Event{})
			}
			mods = newMods
			break
		}
	}
	return mods
}

func bufTex(scr screen.Screen, sz image.Point) (screen.Buffer, screen.Texture) {
	buf, err := scr.NewBuffer(sz)
	if err != nil {
		panic(err)
	}
	tex, err := scr.NewTexture(sz)
	if err != nil {
		panic(err)
	}
	return buf, tex
}

type syntax struct {
	Regexp *re1.Regexp
	Group  int
	Style  text.Style
}

var (
	stringColor  = color.RGBA{R: 0x2F, G: 0x6F, B: 0x89, A: 0xFF}
	commentColor = color.RGBA{R: 0x70, G: 0x70, B: 0x70, A: 0xFF}

	operator = mustCompile(`([ \t]|^)(\+|&|\+=|&=|&&|==|!=|\(|\)|-|\||-=|\|=|\|\||<|<=|\[|\]|\*|\^|\*=|\^=|<-|>|>=|{|}|/|<<|/=|<<=|\+\+|=|:=|,|;|%|>>|%=|>>=|--|!|\.\.\.|\.|:|&\^|&\^=)([ \t]|$)`)

	keyword = mustCompile(`(^|[^a-zA-Z0-9_])(break|default|func|interface|select|case|defer|go|map|struct|chan|else|goto|package|switch|const|fallthrough|if|range|type|continue|for|import|return|var)([^a-zA-Z0-9_]|$)`)

	blockComment = `/[*]([^*]|[*][^/])*[*]/`
	lineComment  = `//.*`
	comment      = mustCompile("(" + blockComment + ")|(" + lineComment + ")")

	interpString  = `("([^"\n]|\\["\n])*([^\\\n]|\\\n)")|""`
	rawString     = "`[^`]*`"
	stringLiteral = mustCompile("(" + interpString + ")|(" + rawString + ")")
	runeLiteral   = mustCompile(`'[^']'|'\\t'|'\\n'|'\\\\'|'\\''`)
)

type highlighter struct {
	face font.Face
}

func (h highlighter) Update(_ []text.Highlight, _ edit.Diffs, txt rope.Rope) []text.Highlight {
	syntax := []syntax{
		{Regexp: comment, Style: text.Style{FG: commentColor}},
		{Regexp: stringLiteral, Style: text.Style{FG: stringColor}},
		{Regexp: runeLiteral, Style: text.Style{FG: stringColor}},
		{Regexp: keyword, Group: 2, Style: text.Style{Face: h.face}},
	}

	var at int64
	var hi []text.Highlight
	for {
		matches := make([][]int64, len(syntax))
		for i, s := range syntax {
			matches[i] = s.Regexp.FindInRope(txt, at, txt.Len())
		}
		next, index := int64(-1), -1
		for i := range matches {
			if matches[i] == nil {
				continue
			}
			if next < 0 || matches[i][2*syntax[i].Group] < next {
				next = matches[i][2*syntax[i].Group]
				index = i
			}
		}
		if index < 0 {
			return hi
		}
		at0 := at
		at = matches[index][2*syntax[index].Group+1]
		if at0 == at {
			panic("bad syntax regexp")
		}
		hi = append(hi, text.Highlight{
			At:    [2]int64{matches[index][0], matches[index][1]},
			Style: syntax[index].Style,
		})
	}
}

func mustCompile(str string) *re1.Regexp {
	re, residual, err := re1.New(str, re1.Opts{})
	if err != nil || residual != "" {
		panic("err: " + err.Error() + ", residual: " + residual)
	}
	return re
}
