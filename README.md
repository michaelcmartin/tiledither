# TileDither

TileDither is an implementation of Floyd-Steinberg dithering specialized for 8-bit tiled-palette systems. At the moment it only supports turning things into Commodore 64 multicolor bitmaps.

## Installing and compiling

You should be able to get this installed and built with the usual command:

    go get github.com/michaelcmartin/tiledither

And that should clone the repo and build an executable.

## Using it

Start with a 320x200 image file. JPEG, PNG, and GIF are all supported. Then run tiledither with that as its argument. If you convert filename.jpg, you will produce filename-dithered.png and filename.koa. The former is a PNG showing what the dithered image will look like. The latter is a data file in the popular Koala Painter format, suitable for use in Commodore 64 systems.

If you do not have access to Koala Painter, you may also prepend showpic.bin to it with a command like (on Linux):

    cat showpic.bin filename.koa > filename.prg

or (on Windows):

    copy /B showpic.bin+filename.koa filename.prg

and this will result in an excutable PRG file that should run directly on a C64 emulator such as VICE, or which can be loaded onto a diskette for use on real hardware.
