// C64 Bitmap converter program. Given a 320x200 JPG, PNG, or GIF it
// will reduce it to an image displayable in a Commodore 64's
// multicolor bitmap display mode.
//
// The file format output is:
//  - the word $6000    (2 bytes)
//  - Raw bitmap data   (8000 bytes)
//  - Video Matrix data (1000 bytes)
//  - Color Memory data (1000 bytes)
//  - Background color  (1 byte)
//
// The result should be loadable by Koala Paint II.
package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"strings"
)

// A representation of the VIC-II's palette, on an NTSC HDTV. Since it
// was based on S/Video, not all colors can be exact.
var C64 = color.Palette{
	color.RGBA{0x00, 0x00, 0x00, 0xFF},
	color.RGBA{0xFF, 0xFF, 0xFF, 0xFF},
	color.RGBA{0x8A, 0x41, 0x33, 0xFF},
	color.RGBA{0x2B, 0xBC, 0xD8, 0xFF},
	color.RGBA{0x95, 0x49, 0x9E, 0xFF},
	color.RGBA{0x39, 0x9D, 0x2C, 0xFF},
	color.RGBA{0x40, 0x3A, 0x7B, 0xFF},
	color.RGBA{0xBF, 0xD1, 0x0E, 0xFF},
	color.RGBA{0x95, 0x56, 0x21, 0xFF},
	color.RGBA{0x53, 0x40, 0x09, 0xFF},
	color.RGBA{0xDC, 0x68, 0x52, 0xFF},
	color.RGBA{0x50, 0x50, 0x50, 0xFF},
	color.RGBA{0x78, 0x78, 0x78, 0xFF},
	color.RGBA{0x55, 0xEC, 0x42, 0xFF},
	color.RGBA{0x78, 0x6C, 0xE7, 0xFF},
	color.RGBA{0x9F, 0x9F, 0x9F, 0xFF}}

func loadImg(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	return img, err
}

type DitherResult struct {
	Preview *image.RGBA
	Data    []byte
}

func dumpPng(out *DitherResult, outpath string) {
	outfile, err := os.Create(outpath)
	if err != nil {
		fmt.Printf("Error opening %v: %v\n", outpath, err)
		os.Exit(1)
	}
	defer outfile.Close()
	err = png.Encode(outfile, out.Preview)
	if err != nil {
		fmt.Printf("Error writing PNG: %v", err)
		os.Exit(1)
	}
}

func dumpDat(out *DitherResult, outpath string) {
	outfile, err := os.Create(outpath)
	if err != nil {
		fmt.Printf("Error opening %v: %v\n", outpath, err)
		os.Exit(1)
	}
	defer outfile.Close()

	outfile.Write(out.Data)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage:\n    %v {fname}\n", os.Args[0])
		os.Exit(1)
	}

	img, err := loadImg(os.Args[1])
	if err != nil {
		fmt.Printf("Error reading %v: %v\n", os.Args[1], err)
		os.Exit(1)
	}

	result, err := convertMC(img)
	if err != nil {
		fmt.Printf("Error converting %v: %v\n", os.Args[1], err)
		os.Exit(1)
	}

	basepath := os.Args[1]
	ext := strings.LastIndex(basepath, ".")
	if ext != -1 {
		basepath = basepath[:ext]
	}
	dumpPng(result, basepath+"-dithered.png")
	dumpDat(result, basepath+".koa")
}
