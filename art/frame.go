package art

import (
	"image"
	"image/color"
)

// A Frame is a finished drawing: a grid of pixels, each set to a color or transparent. It is an [image.Image], so the
// standard image encoders write it.
type Frame struct {
	w, h int
	px   []uint32
}

// A Pixel is one set pixel of a frame, with the origin at the top left.
type Pixel struct {
	X, Y  int
	Color Color
}

// Width returns the frame width in pixels.
func (f *Frame) Width() int { return f.w }

// Height returns the frame height in pixels.
func (f *Frame) Height() int { return f.h }

// Pixel returns pixel (x, y) and whether it is set.
func (f *Frame) Pixel(x, y int) (Color, bool) {
	if x < 0 || x >= f.w || y < 0 || y >= f.h || f.px[y*f.w+x] == 0 {
		return 0, false
	}
	return Color(f.px[y*f.w+x] &^ painted), true
}

// Pixels returns the set pixels in row-major order.
func (f *Frame) Pixels() []Pixel {
	out := make([]Pixel, 0, f.Len())
	for k, v := range f.px {
		if v != 0 {
			out = append(out, Pixel{k % f.w, k / f.w, Color(v &^ painted)})
		}
	}
	return out
}

// Len returns how many pixels are set.
func (f *Frame) Len() int {
	n := 0
	for _, v := range f.px {
		if v != 0 {
			n++
		}
	}
	return n
}

// ColorModel implements [image.Image].
func (f *Frame) ColorModel() color.Model { return color.NRGBAModel }

// Bounds implements [image.Image].
func (f *Frame) Bounds() image.Rectangle { return image.Rect(0, 0, f.w, f.h) }

// At implements [image.Image]; an unset pixel is transparent.
func (f *Frame) At(x, y int) color.Color {
	c, ok := f.Pixel(x, y)
	if !ok {
		return color.NRGBA{}
	}
	r, g, b := c.RGB()
	return color.NRGBA{R: r, G: g, B: b, A: 0xff}
}
