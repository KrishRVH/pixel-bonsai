package art

import "math"

// painted marks a set pixel in a canvas or frame, above the 24 bits of its color.
const painted = 1 << 24

// A Canvas is a drawing in progress at a scale. Shapes take units and paint pixels; a pixel outside the canvas is
// never set. Later drawing paints over earlier.
type Canvas struct {
	scale float64
	w, h  int
	px    []uint32
}

// NewCanvas returns an empty canvas of [Size](scale).
func NewCanvas(scale float64) *Canvas {
	w, h := Size(scale)
	return &Canvas{scale: scale, w: w, h: h, px: make([]uint32, w*h)}
}

// Scale returns the pixels per unit.
func (c *Canvas) Scale() float64 { return c.scale }

// Width returns the canvas width in pixels.
func (c *Canvas) Width() int { return c.w }

// Height returns the canvas height in pixels.
func (c *Canvas) Height() int { return c.h }

func (c *Canvas) inside(i, j int) bool {
	return 0 <= i && i < c.w && 0 <= j && j < c.h
}

// Put sets pixel (i, j) when it is on the canvas.
func (c *Canvas) Put(i, j int, col Color) {
	if c.inside(i, j) {
		c.px[j*c.w+i] = painted | uint32(col)
	}
}

// Get returns pixel (i, j) and whether it is set.
func (c *Canvas) Get(i, j int) (Color, bool) {
	if !c.inside(i, j) || c.px[j*c.w+i] == 0 {
		return 0, false
	}
	return Color(c.px[j*c.w+i] &^ painted), true
}

// Has reports whether pixel (i, j) is set.
func (c *Canvas) Has(i, j int) bool {
	return c.inside(i, j) && c.px[j*c.w+i] != 0
}

// Erase clears pixel (i, j), letting the background show.
func (c *Canvas) Erase(i, j int) {
	if c.inside(i, j) {
		c.px[j*c.w+i] = 0
	}
}

// Clone returns a copy of the canvas, to draw on aside or to restore from.
func (c *Canvas) Clone() *Canvas {
	return &Canvas{scale: c.scale, w: c.w, h: c.h, px: append([]uint32(nil), c.px...)}
}

// Overlay paints the set pixels of src, a canvas of the same scale, over this one where keep reports true. A nil
// keep takes them all.
func (c *Canvas) Overlay(src *Canvas, keep func(i, j int) bool) {
	for k, v := range src.px {
		if v != 0 && (keep == nil || keep(k%c.w, k/c.w)) {
			c.px[k] = v
		}
	}
}

// Cell returns the pixel that holds the point (x, y) in units.
func (c *Canvas) Cell(x, y float64) (i, j int) {
	return int(math.Floor(x * c.scale)), int(math.Floor(y * c.scale))
}

// Center returns the center of pixel (i, j) in units.
func (c *Canvas) Center(i, j int) (x, y float64) {
	return (float64(i) + 0.5) / c.scale, (float64(j) + 0.5) / c.scale
}

// A Sample is a pixel and its center in units.
type Sample struct {
	I, J int
	X, Y float64
}

// Scan returns the pixels on the canvas that overlap a box in units, row by row.
func (c *Canvas) Scan(x0, y0, x1, y1 float64) []Sample {
	s := c.scale
	i0, i1 := max(0, int(math.Floor(x0*s))), min(c.w, int(math.Ceil(x1*s)))
	j0, j1 := max(0, int(math.Floor(y0*s))), min(c.h, int(math.Ceil(y1*s)))
	if i1 <= i0 || j1 <= j0 {
		return nil
	}
	out := make([]Sample, 0, (i1-i0)*(j1-j0))
	for j := j0; j < j1; j++ {
		for i := i0; i < i1; i++ {
			x, y := c.Center(i, j)
			out = append(out, Sample{i, j, x, y})
		}
	}
	return out
}

// Frame returns the finished drawing.
func (c *Canvas) Frame() *Frame {
	return &Frame{w: c.w, h: c.h, px: append([]uint32(nil), c.px...)}
}
