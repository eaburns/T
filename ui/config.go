package ui

import (
	"image/color"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	// padPt is the point-width of padding around rows.
	padPt = 0.5

	// minColPt is the minimum points in width of a column on resize.
	minColPt = 10

	// colText is the default column background text.
	colText = "AddCol DelCol Add\n"

	// tagText is the default tag text.
	tagText = " | Del"
)

var (
	// defaultFont is the default font.
	defaultFont, _ = truetype.Parse(goregular.TTF)

	// defaultFontSize is the default font size in points.
	defaultFontSize = 11

	// fg is the text foreground color.
	fg = color.RGBA{R: 0x10, G: 0x28, B: 0x34, A: 0xFF}

	// colBG is the column background color.
	colBG = color.White

	// padColor is the padding color.
	padColor = color.RGBA{R: 0x10, G: 0x28, B: 0x34, A: 0xFF}

	// tagBG is the tag background color.
	tagBG = color.RGBA{R: 0xCF, G: 0xE0, B: 0xF7, A: 0xFF}

	// bodyBG is a body background color.
	bodyBG = color.RGBA{R: 0xFA, G: 0xF0, B: 0xE6, A: 0xFF}

	// hiBG1, hiBG2, and hiBG2 are the background colors
	// of 1-, 2-, and 3-click highlighted text.
	hiBG1 = color.RGBA{R: 0xCC, G: 0xCD, B: 0xAC, A: 0xFF}
	hiBG2 = color.RGBA{R: 0xEC, G: 0x90, B: 0x7F, A: 0xFF}
	hiBG3 = color.RGBA{R: 0xB7, G: 0xE5, B: 0xB2, A: 0xFF}
)
