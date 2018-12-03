// +build ignore

package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"runtime"
	"time"

	"github.com/eaburns/T/edit"
	"github.com/eaburns/T/re1"
	"github.com/eaburns/T/rope"
	"github.com/eaburns/T/text"
	"github.com/golang/freetype/truetype"
	"golang.org/x/exp/shiny/driver/gldriver"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/math/f64"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
)

var (
	fg           = color.RGBA{R: 0x10, G: 0x28, B: 0x34, A: 0xFF}
	bg           = color.RGBA{R: 0xFE, G: 0xF0, B: 0xE6, A: 0xFF}
	stringColor  = color.RGBA{R: 0x2F, G: 0x6F, B: 0x89, A: 0xFF}
	commentColor = color.RGBA{R: 0x70, G: 0x70, B: 0x70, A: 0xFF}
	hiBG1        = color.RGBA{R: 0xB6, G: 0xDA, B: 0xFD, A: 0xFF}
	hiBG2        = color.RGBA{R: 0xEC, G: 0x90, B: 0x7F, A: 0xFF}
	hiBG3        = color.RGBA{R: 0xB7, G: 0xE5, B: 0xB2, A: 0xFF}
	tagBG        = color.RGBA{R: 0xF0, G: 0xF0, B: 0xF0, A: 0xFF}

	regular, _                             = truetype.Parse(goregular.TTF)
	medium, _                              = truetype.Parse(gomedium.TTF)
	bold, _                                = truetype.Parse(gobold.TTF)
	italic, _                              = truetype.Parse(goitalic.TTF)
	face, faceMedium, faceBold, faceItalic font.Face
)

func init() { runtime.LockOSThread() }

func main() { gldriver.Main(Main) }

func Main(scr screen.Screen) {
	win, err := scr.NewWindow(nil)
	if err != nil {
		panic(err)
	}
	defer win.Release()

	var box *text.Box
	var buf screen.Buffer
	var tex screen.Texture
	var winSize image.Point
	for {
		e := win.NextEvent()
		sz, ok := e.(size.Event)
		if !ok {
			fmt.Printf("ignoring %#v\n", e)
			continue
		}
		winSize = sz.Size()
		if tex, err = scr.NewTexture(winSize); err != nil {
			panic(err)
		}
		if buf, err = scr.NewBuffer(winSize); err != nil {
			panic(err)
		}
		dpi := sz.PixelsPerPt * 72.0
		face = truetype.NewFace(regular, &truetype.Options{
			Size: 11, // pt
			// This seems to be off by the ration of 72/96.
			DPI: float64(dpi * (72.0 / 96.0)),
		})
		faceMedium = truetype.NewFace(medium, &truetype.Options{
			Size: 11, // pt
			// This seems to be off by the ration of 72/96.
			DPI: float64(dpi * (72.0 / 96.0)),
		})
		faceBold = truetype.NewFace(bold, &truetype.Options{
			Size: 11, // pt
			// This seems to be off by the ration of 72/96.
			DPI: float64(dpi * (72.0 / 96.0)),
		})
		faceItalic = truetype.NewFace(italic, &truetype.Options{
			Size: 11, // pt
			// This seems to be off by the ration of 72/96.
			DPI: float64(dpi * (72.0 / 96.0)),
		})
		styles := [...]text.Style{
			{FG: fg, BG: bg, Face: face},
			{BG: hiBG1},
			{BG: hiBG2},
			{BG: hiBG3},
		}
		box = text.NewBox(styles, buf.Bounds().Size())
		box.SetText(rope.New(Text))
		box.SetSyntax(highlighter{})
		break
	}

	go func() {
		for range time.Tick(20 * time.Millisecond) {
			win.Send(time.Now())
		}
	}()

	var dirty bool
	var mods [4]bool
	for {
		e := win.NextEvent()
		switch e := e.(type) {
		case mouse.Event:
			var redraw bool
			switch pt := image.Pt(int(e.X), int(e.Y)); {
			case e.Button == mouse.ButtonWheelUp:
				redraw = box.HandleWheel(0, 1)
			case e.Button == mouse.ButtonWheelDown:
				redraw = box.HandleWheel(0, -1)
			case e.Button == mouse.ButtonWheelLeft:
				redraw = box.HandleWheel(-1, 0)
			case e.Button == mouse.ButtonWheelRight:
				redraw = box.HandleWheel(1, 0)
			case e.Direction == mouse.DirNone:
				redraw = box.HandleMove(pt)
			case e.Direction == mouse.DirPress:
				redraw = box.HandleClick(pt, int(e.Button))
			case e.Direction == mouse.DirRelease:
				redraw = box.HandleClick(pt, -int(e.Button))
			case e.Direction == mouse.DirStep:
				redraw = box.HandleClick(pt, int(e.Button))
				redraw = box.HandleClick(pt, -int(e.Button)) || redraw
			}
			if redraw {
				win.Send(paint.Event{})
			}

		case key.Event:
			if e.Direction == key.DirNone {
				e.Direction = key.DirPress
			}
			if e.Direction == key.DirPress &&
				(e.Code == key.CodeUpArrow ||
					e.Code == key.CodeDownArrow ||
					e.Code == key.CodeLeftArrow ||
					e.Code == key.CodeRightArrow ||
					e.Code == key.CodePageUp ||
					e.Code == key.CodePageDown ||
					e.Code == key.CodeHome ||
					e.Code == key.CodeEnd) {
				var redraw bool
				switch e.Code {
				case key.CodeUpArrow:
					redraw = box.HandleDir(0, -1)
				case key.CodeDownArrow:
					redraw = box.HandleDir(0, 1)
				case key.CodeLeftArrow:
					redraw = box.HandleDir(-1, 0)
				case key.CodeRightArrow:
					redraw = box.HandleDir(1, 0)
				case key.CodePageUp:
					redraw = box.HandleDir(0, -2)
				case key.CodePageDown:
					redraw = box.HandleDir(0, 2)
				case key.CodeHome:
					redraw = box.HandleDir(0, math.MinInt16)
				case key.CodeEnd:
					redraw = box.HandleDir(0, math.MaxInt16)
				}
				if redraw {
					win.Send(paint.Event{})
				}
				break
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
					if box.HandleRune(e.Rune) {
						win.Send(paint.Event{})
					}
				}
				break
			}

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
					if box.HandleMod(m) {
						win.Send(paint.Event{})
					}
					mods = newMods
					break
				}
			}

		case lifecycle.Event:
			if e.To == lifecycle.StageDead {
				return
			}
			box.HandleFocus(e.To == lifecycle.StageFocused)

		case size.Event:
			if e.Size().X == 0 && e.Size().Y == 0 {
				return
			}
			winSize = e.Size()
			if b := tex.Bounds(); b.Dx() < winSize.X || b.Dy() < winSize.Y {
				tex.Release()
				sz := winSize
				sz.X *= 2
				sz.Y *= 2
				if tex, err = scr.NewTexture(sz); err != nil {
					panic(err)
				}
			}
			if b := buf.Bounds(); b.Dx() < winSize.X || b.Dy() < winSize.Y {
				buf.Release()
				sz := winSize
				sz.X *= 2
				sz.Y *= 2
				if buf, err = scr.NewBuffer(sz); err != nil {
					panic(err)
				}
			}
			dirty = true
			box.HandleResize(e.Size())

		case time.Time:
			if box.HandleTick() {
				win.Send(paint.Event{})
			}

		case paint.Event:
			rect := image.Rectangle{Max: winSize}
			img := buf.RGBA().SubImage(rect).(*image.RGBA)
			box.Draw(dirty, img)
			dirty = false
			tex.Upload(image.ZP, buf, buf.Bounds())
			win.Draw(f64.Aff3{
				1, 0, 0,
				0, 1, 0,
			}, tex, tex.Bounds(), draw.Src, nil)
			win.Publish()
		}
	}
}

type Syntax struct {
	Regexp *re1.Regexp
	Group  int
	Style  text.Style
}

var (
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

type highlighter struct{}

func (highlighter) Update(_ []text.Highlight, _ edit.Diffs, txt rope.Rope) []text.Highlight {
	syntax := []Syntax{
		{Regexp: comment, Style: text.Style{FG: commentColor}},
		{Regexp: stringLiteral, Style: text.Style{FG: stringColor}},
		{Regexp: runeLiteral, Style: text.Style{FG: stringColor}},
		{Regexp: keyword, Group: 2, Style: text.Style{Face: faceMedium}},
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

var Text = `"Jabberwocky"

’Twas brillig, and the slithy toves
Did gyre and gimble in the wabe;
All mimsy were the borogoves,
And the mome raths outgrabe.

“Beware the Jabberwock, my son!
The jaws that bite, the claws that catch!
Beware the Jubjub bird, and shun
The frumious Bandersnatch!”

He took his vorpal sword in hand:
Long time the manxome foe he sought—
So rested he by the Tumtum tree,
And stood awhile in thought.

And as in uffish thought he stood,
The Jabberwock, with eyes of flame,
Came whiffling through the tulgey wood,
And burbled as it came!

One, two! One, two! And through and through
The vorpal blade went snicker-snack!
He left it dead, and with its head
He went galumphing back.

“And hast thou slain the Jabberwock?
Come to my arms, my beamish boy!
O frabjous day! Callooh! Callay!”
He chortled in his joy.

’Twas brillig, and the slithy toves
Did gyre and gimble in the wabe;
All mimsy were the borogoves,
And the mome raths outgrabe.`
