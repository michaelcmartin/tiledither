package main

import (
	"bytes"
	"errors"
	"github.com/michaelcmartin/tiledither/dither"
	"image"
	"image/color"
	"sort"
)

type c64bmp struct {
	src      image.Image
	pixmap   [][]int
	palettes [][]color.Palette
	bg       int
}

func newC64bmp(w, h int) *c64bmp {
	bmp := new(c64bmp)
	matrix := make([][]int, w)
	buf := make([]int, w*h)
	for i := range matrix {
		matrix[i], buf = buf[:h], buf[h:]
	}
	bmp.pixmap = matrix
	cw := (w + 3) / 4
	ch := (h + 7) / 8
	palmatrix := make([][]color.Palette, cw)
	palbuf := make([]color.Palette, cw*ch)
	for i := range palmatrix {
		palmatrix[i], palbuf = palbuf[:ch], palbuf[ch:]
	}
	bmp.palettes = palmatrix
	return bmp
}

func convertMC(src image.Image) (*DitherResult, error) {
	// Step one: Produce a half-width image since multicolor
	// images are doubled pixels.
	bounds := src.Bounds()
	w := bounds.Dx() / 2
	h := bounds.Dy()
	if w != 160 || h != 200 {
		return nil, errors.New("Image to convert must be 320x200")
	}
	dest := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r1, g1, b1, a1 := src.At(x*2+bounds.Min.X, y+bounds.Min.Y).RGBA()
			r2, g2, b2, a2 := src.At(x*2+bounds.Min.X+1, y+bounds.Min.Y).RGBA()
			r := uint16((r1 + r2) / 2)
			g := uint16((g1 + g2) / 2)
			b := uint16((b1 + b2) / 2)
			a := uint16((a1 + a2) / 2)
			dest.Set(x, y, color.RGBA64{r, g, b, a})
		}
	}
	// Step 2: Convert freely to the C64 palette. Store as 2D
	// array of ints (palette entries).
	pass1 := dither.ToPalette(dest, C64)
	bounds = pass1.Bounds()

	bmp := newC64bmp(w, h)
	bmp.src = dest
	p := bmp.pixmap

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			p[x][y] = C64.Index(pass1.At(x, y))
		}
	}

	// Step 3: Try each of the 16 colors to see which background
	// color involves the fewest compromises.
	besterr := 65000
	bmp.bg = -1
	for proposedBG := 0; proposedBG < 16; proposedBG++ {
		pixerr := 0
		for y := 0; y < h; y += 8 {
			for x := 0; x < w; x += 4 {
				var charUse [16]int
				charColorCount := 0
				for cy := 0; cy < 8; cy++ {
					if y+cy >= h {
						break
					}
					for cx := 0; cx < 4; cx++ {
						if x+cx >= w {
							break
						}
						c := p[x+cx][y+cy]
						if c == proposedBG {
							continue
						}
						if charUse[c] == 0 {
							charColorCount++
						}
						charUse[c]++
					}
				}
				if charColorCount > 3 {
					sort.Ints(charUse[:])
					for _, v := range charUse[:13] {
						pixerr += v
					}
				}
			}
		}
		if pixerr < besterr {
			besterr = pixerr
			bmp.bg = proposedBG
		}
	}
	// Now that we have a background color we can compute the
	// remaining palettes based on that. TODO: this is based on
	// the three most common non-background pixels in each 4x8
	// block. A more thorough system - especially one that gets to
	// select its own colors - would probably want to use
	// something like median cut.
	for y := 0; y < h; y += 8 {
		for x := 0; x < w; x += 4 {
			var charUse [16]int
			for cy := 0; cy < 8; cy++ {
				if y+cy >= h {
					break
				}
				for cx := 0; cx < 4; cx++ {
					if x+cx >= w {
						break
					}
					c := p[x+cx][y+cy]
					if c == bmp.bg {
						continue
					}
					charUse[c]++
				}
			}
			// Get the three most common pixels left
			i1, i2, i3 := 0, 0, 0
			c1, c2, c3 := -1, -1, -1
			for i, c := range charUse {
				if c > c3 {
					c1, c2, c3 = c2, c3, c
					i1, i2, i3 = i2, i3, i
				} else if c > c2 {
					c1, c2 = c2, c
					i1, i2 = i2, i
				} else if c > c1 {
					c1 = c
					i1 = i
				}
			}
			bmp.palettes[x/4][y/8] = color.Palette{C64[bmp.bg], C64[i1], C64[i2], C64[i3]}
		}
	}
	dither.Convert(bmp)

	// Package up our results
	result := new(DitherResult)
	// Preview image
	w, h = bmp.Width(), bmp.Height()
	result.Preview = image.NewRGBA(image.Rect(0, 0, w*2, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := bmp.PaletteAt(x, y)[bmp.pixmap[x][y]]
			result.Preview.Set(x*2, y, c)
			result.Preview.Set(x*2+1, y, c)
		}
	}
	// The actual data
	var outbuf bytes.Buffer
	outbuf.Write([]byte{0x00, 0x60})
	outbuf.Write(bitmap(bmp))
	outbuf.Write(textmap(bmp))
	outbuf.Write(colormap(bmp))
	outbuf.Write([]byte{byte(bmp.bg)})

	result.Data = outbuf.Bytes()
	return result, nil
}

// Implement dither.Context methods for c64bmp.
func (ctx *c64bmp) Width() int {
	return len(ctx.pixmap)
}

func (ctx *c64bmp) Height() int {
	return len(ctx.pixmap[0])
}

func (ctx *c64bmp) At(x, y int) color.Color {
	return ctx.src.At(x, y)
}

func (ctx *c64bmp) PaletteAt(x, y int) color.Palette {
	return ctx.palettes[x/4][y/8]
}

func (ctx *c64bmp) Set(x, y int, c color.Color) {
	p := ctx.PaletteAt(x, y)
	ctx.pixmap[x][y] = p.Index(c)
}

// Compute the various bitmap arrays from our arrays.
func colormap(bmp *c64bmp) []uint8 {
	w, h := len(bmp.palettes), len(bmp.palettes[0])
	result := make([]uint8, w*h)
	i := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			result[i] = uint8(C64.Index(bmp.palettes[x][y][3]))
			i++
		}
	}
	return result
}

func textmap(bmp *c64bmp) []uint8 {
	w, h := len(bmp.palettes), len(bmp.palettes[0])
	result := make([]uint8, w*h)
	i := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c1 := uint8(C64.Index(bmp.palettes[x][y][1]))
			c2 := uint8(C64.Index(bmp.palettes[x][y][2]))
			result[i] = c1*16 + c2
			i++
		}
	}
	return result
}

func bitmap(bmp *c64bmp) []uint8 {
	w, h := len(bmp.pixmap), len(bmp.pixmap[0])
	// We're computing output size based on whole characters, so
	// the result array's size is based on the palette array, not
	// the pixmap. In a sensible input image these should be the
	// same.
	result := make([]uint8, len(bmp.palettes)*len(bmp.palettes[0])*8)
	i := 0
	for y := 0; y < h; y += 8 {
		for x := 0; x < w; x += 4 {
			for cy := 0; cy < 8; cy++ {
				var v uint8 = 0
				if y+cy < h {
					for cx := 0; cx < 4; cx++ {
						v = v * 4
						if x+cx < w {
							v += uint8(bmp.pixmap[x+cx][y+cy])
						}
					}
				}
				result[i] = v
				i++
			}
		}
	}
	return result
}
