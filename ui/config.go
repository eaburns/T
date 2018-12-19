package ui

import (
	"image/color"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	// padPt is the point-width of padding between columns and rows.
	padPt = 2

	// minColPt is the minimum points in width of a column on resize.
	minColPt = 10
)

var (
	font, _ = truetype.Parse(goregular.TTF)
	fontPt  = 11
	fg      = color.RGBA{R: 0x10, G: 0x28, B: 0x34, A: 0xFF}
	colBG   = color.RGBA{R: 0xEA, G: 0xEA, B: 0xEA, A: 0xFF}
	tagBG   = color.RGBA{R: 0xCF, G: 0xE0, B: 0xF7, A: 0xFF}
	bodyBG  = color.RGBA{R: 0xFE, G: 0xF0, B: 0xE6, A: 0xFF}
	hiBG1   = color.RGBA{R: 0xCC, G: 0xCD, B: 0xAC, A: 0xFF}
	hiBG2   = color.RGBA{R: 0xEC, G: 0x90, B: 0x7F, A: 0xFF}
	hiBG3   = color.RGBA{R: 0xB7, G: 0xE5, B: 0xB2, A: 0xFF}
)
