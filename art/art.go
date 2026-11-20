// Package art is the drawing kit every bonsai variety draws with, and the ladder of growth stages its drawings key
// off. A tree is a function of its age g in [0, 1] and a scale: a [Variety] paints it on a [Canvas].
//
// Geometry is in base units: the canvas is [W] by [H] units, x right, y down, with light from the upper left. The
// scale turns units into pixels, so a larger scale shows more detail rather than bigger blocks.
//
// Drawing is deterministic. The same age and scale give the same pixels on every call: texture comes from [Noise],
// never from random state, and drawing code never ranges over a Go map, whose order varies. A [Cover] keeps the
// pixels of a shape in drawing order for that reason.
//
// Rounding takes halves to even ([math.RoundToEven]), in the kit and in the varieties alike, so a coordinate that
// lands on a half pixel goes the same way everywhere.
package art

import "math"

// The base canvas in units: W wide and H tall, which is H/2 terminal rows of half blocks at scale 1.
const (
	W = 48
	H = 30
)

// MinScale and MaxScale bound the scales a pane draws a tree at.
const (
	MinScale = 0.5
	MaxScale = 3.0
)

// LightX and LightY are the unit vector toward the light, at the upper left.
const (
	LightX = -0.6
	LightY = -0.8
)

// Size returns the canvas in pixels at a scale: W·scale wide and H·scale tall, rounded, with the height rounded up to
// an even number so the canvas fills whole half-block rows.
func Size(scale float64) (w, h int) {
	h = int(math.RoundToEven(H * scale))
	return int(math.RoundToEven(W * scale)), h + h%2
}

// Fit returns the largest scale, in eighths and at most [MaxScale], at which a tree fits a pane width cells wide and
// rows tall whose last row holds the caption. It returns 0 below [MinScale], where the caption shows alone.
func Fit(width, rows int) float64 {
	scale := min(float64(width)/W, 2*float64(rows-1)/H, MaxScale)
	scale = math.Floor(scale*8+1e-9) / 8
	if scale < MinScale {
		return 0
	}
	return scale
}

// Clamp limits v to [0, 1].
func Clamp(v float64) float64 {
	return max(0, min(1, v))
}

// Lit returns the shade, 0 in shadow to 1 lit, of a point on a shape at offset (nx, ny) from its center in radii.
func Lit(nx, ny float64) float64 {
	return Clamp(0.5 + 0.5*(nx*LightX+ny*LightY))
}
