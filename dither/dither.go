/*
Package dither implements Floyd-Steinberg dithering on images or
generalizations thereof.

It specifically expands on the basic algorithm by allowing each point
to have its own palette to commit to. This lets it use colors more
thoroughly on tile-based systems like the C64.
*/
package dither

import (
	"image"
	"image/color"
)

// When determining which color is nearest to a pixel,
// color.Palette.Convert() almost solves the problem, but doesn't
// quite manage it. It does not contemplate colors with negative
// values (as can happen after applying the error term in the
// dithering algorithm). We adapt their implementation here to make
// the color values signed.
type colError struct {
	R, G, B int32
}

func closest(r, g, b int32, p color.Palette) color.Color {
	best_index := 0
	best_diff := uint32(1<<32 - 1)
	for i, v := range p {
		vr, vg, vb, _ := v.RGBA()
		component := (r - int32(vr)) >> 1
		ssd := uint32(component * component)
		component = (g - int32(vg)) >> 1
		ssd += uint32(component * component)
		component = (b - int32(vb)) >> 1
		ssd += uint32(component * component)
		if ssd < best_diff {
			best_index = i
			best_diff = ssd
		}
	}
	return p[best_index]
}

// Context represents an interface that can be dithered. All its
// methods but PaletteAt carry the same meanings as they would in
// image.Image.
type Context interface {
	Width() int
	Height() int
	At(x, y int) color.Color
	PaletteAt(x, y int) color.Palette
	Set(x, y int, c color.Color)
}

// Convert executes the dithering algorithm upon a context. It will
// call At and PaletteAt, and Set on every pixel left to right, top to
// bottom.
func Convert(ctx Context) {
	h, w := ctx.Height(), ctx.Width()
	gerror := make([][]colError, h)
	for y, _ := range gerror {
		gerror[y] = make([]colError, w)
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, _ := ctx.At(x, y).RGBA()
			cr := int32(r) + gerror[y][x].R
			cg := int32(g) + gerror[y][x].G
			cb := int32(b) + gerror[y][x].B

			target := closest(cr, cg, cb, ctx.PaletteAt(x, y))
			tr, tg, tb, _ := target.RGBA()
			cr = int32(r) - int32(tr)
			cg = int32(g) - int32(tg)
			cb = int32(b) - int32(tb)
			if x+1 < len(gerror[y]) {
				gerror[y][x+1].R += cr * 7 / 16
				gerror[y][x+1].G += cg * 7 / 16
				gerror[y][x+1].B += cb * 7 / 16
			}
			if y+1 < len(gerror) {
				if x-1 > 0 {
					gerror[y+1][x-1].R += cr * 3 / 16
					gerror[y+1][x-1].G += cg * 3 / 16
					gerror[y+1][x-1].B += cb * 3 / 16
				}
				if x+2 < len(gerror[y]) {
					gerror[y+1][x+1].R += cr / 16
					gerror[y+1][x+1].G += cg / 16
					gerror[y+1][x+1].B += cb / 16
				}
				gerror[y+1][x].R += cr * 5 / 16
				gerror[y+1][x].G += cg * 5 / 16
				gerror[y+1][x].B += cb * 5 / 16
			}
			ctx.Set(x, y, target)
		}
	}
}

// Basic context used to map from one image to another
type imageCtx struct {
	src     image.Image
	dest    *image.RGBA
	palette color.Palette
}

func (ctx *imageCtx) Width() int {
	return ctx.src.Bounds().Dx()
}

func (ctx *imageCtx) Height() int {
	return ctx.src.Bounds().Dy()
}

func (ctx *imageCtx) At(x, y int) color.Color {
	bounds := ctx.src.Bounds()
	return ctx.src.At(x+bounds.Min.X, y+bounds.Min.Y)
}

func (ctx *imageCtx) PaletteAt(x, y int) color.Palette {
	return ctx.palette
}

func (ctx *imageCtx) Set(x, y int, c color.Color) {
	bounds := ctx.dest.Bounds()
	ctx.dest.Set(x+bounds.Min.X, y+bounds.Min.Y, c)
}

// ToPalette takes an image and a palette and returns a new image that
// uses only colors in that palette and is a dithered representation
// of its image argument.
func ToPalette(img image.Image, palette color.Palette) *image.RGBA {
	ctx := new(imageCtx)
	ctx.src = img
	bounds := img.Bounds()
	ctx.dest = image.NewRGBA(bounds)
	ctx.palette = palette
	Convert(ctx)
	return ctx.dest
}
