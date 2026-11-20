package pine

import (
	"math"

	"github.com/KrishRVH/pixel-bonsai/art"
)

// A bark plate's width and length in pixels, so a larger scale shows more plates.
const (
	plateW = 2.8
	plateL = 3.2
)

// nebariRoots are the roots spread over the soil: each one's side, stage, and reach over the soil.
var nebariRoots = []struct {
	side  float64
	born  art.Stage
	reach float64
}{
	{-1, art.Established, 3.6},
	{1, art.Established, 4.2},
	{-1, art.Established, 5.8},
	{1, art.Established, 6.8},
	{1, art.Established, 9.2},
	{-1, art.Mature, 7.6},
}

// jins are jin on the ends of pads: the pad they grow from (its index in plans, the crown being last), their angle
// (radians, y down), length, and stage.
var jins = []struct {
	pad           int
	angle, length float64
	born          art.Stage
}{{5, -2.2, 4.6, art.Old}, {4, -0.9, 4.4, art.Old}, {len(plans), -2.3, 3.8, art.Old}, {7, -0.6, 3.8, art.Venerable}}

// stubs are jin on the trunk: their height, angle, length, and stage.
var stubs = []struct {
	h, angle, length float64
	born             art.Stage
}{{7.5, -0.45, 4.2, art.Old}, {13.0, -2.7, 3.8, art.Old}}

// trunkX returns the trunk's x at a share of its full height: an informal S that leans back over its foot, bend of
// the way into it.
func trunkX(share, bend float64) float64 {
	points := make([]art.Point, len(sway))
	for k, s := range sway {
		points[k] = art.Point{X: s.share, Y: s.dx}
	}
	// The spline runs through (share, shift) pairs.
	path := art.Spline(points, 6)
	for k := 1; k < len(path); k++ {
		a, b := path[k-1], path[k]
		if a.X <= share && share <= b.X && b.X > a.X {
			return axisX + bend*(a.Y+(b.Y-a.Y)*(share-a.X)/(b.X-a.X))
		}
	}
	return axisX + bend*path[len(path)-1].Y
}

// trunkWidth returns the trunk's width at a share of its full height.
func trunkWidth(lk *look, share float64) float64 {
	return lk.base * (1 - (1-lk.taper)*math.Pow(share, 0.8))
}

// trunkAt returns the trunk's point h units above its foot.
func trunkAt(h, bend float64) art.Point {
	return art.Point{X: trunkX(h/fullHeight, bend), Y: baseY - h}
}

// A barkStyle is how a stroke of bark looks: how rough its plates are, its salt, where deadwood bares it, how much its
// plates stand out, and the band of y it paints between.
type barkStyle struct {
	rough          float64
	salt           int
	bared          func(u, across float64) bool // when set, where deadwood bares the bark, by distance along and across
	lift           float64
	floor, ceiling float64 // nothing is painted below floor (the trunk's foot in the soil) or above ceiling
}

// barkStroke draws a stroke of bark along a path, its widths spaced evenly along it, and returns its cover.
func barkStroke(c *art.Canvas, path art.Path, widths []float64, b barkStyle) *art.Cover {
	cover, start := art.NewCover(), 0.0
	parts := len(widths) - 1
	for k := range parts {
		w0, w1 := widths[k], widths[k+1]
		part := path[k*(len(path)-1)/parts : (k+1)*(len(path)-1)/parts+1]
		cover.Merge(plates(c, c.Trace(part, w0, w1), w0, w1, start, b))
		length := 0.0
		for n := 1; n < len(part); n++ {
			length += math.Hypot(part[n].X-part[n-1].X, part[n].Y-part[n-1].Y)
		}
		start += length
	}
	return cover
}

// plates paints a stroke's trace, w0 wide at its start and w1 at its end and start units along the tree, as plated
// bark: lit plates split by fissures, rust in the cracks. It returns the cover of what it painted.
func plates(c *art.Canvas, cells []art.StrokeCell, w0, w1, start float64, b barkStyle) *art.Cover {
	cover := art.NewCover()
	for _, cell := range cells {
		if _, y := c.Center(cell.I, cell.J); y < b.ceiling || y > b.floor {
			continue
		}
		c.Put(cell.I, cell.J, b.color(c, cell, (w0+(w1-w0)*cell.F)/2, start))
		cover.Set(cell.I, cell.J, cell.Shade)
	}
	return cover
}

// color returns the color of a bark pixel on a stroke half units wide on each side: deadwood where it is bared, a
// fissure between plates or rust in one, or a plate, lit along its top and left edges.
func (b barkStyle) color(c *art.Canvas, cell art.StrokeCell, half, start float64) art.Color {
	u := cell.U + start
	col := cell.Across * half * c.Scale() / plateW
	ci := int(math.Floor(col))
	row := u*c.Scale()/plateL + art.Noise(ci, 3, b.salt)
	ri := int(math.Floor(row))
	fc, fr := col-float64(ci), row-float64(ri)
	// Fissures run down the left edge of some columns of plates and across the top of some plates, more of them as
	// the bark roughens.
	down := fc < 1/plateW && art.Noise(ci, 5, b.salt) < 0.6*b.rough
	across := fr < 1/plateL && art.Noise(ci, ri, b.salt+6) < 0.4*b.rough
	switch {
	case b.bared != nil && b.bared(u, cell.Across):
		if art.Noise(cell.I, cell.J, b.salt+8) < 0.12 {
			return dead[0]
		}
		return dead[:3].Shade(cell.Shade, 0.3, art.Noise(cell.I, cell.J, b.salt+9))
	case down || across:
		if b.rough > 0.8 && art.Noise(ci, ri, b.salt+4) < 0.4*b.rough && cell.Shade > 0.3 {
			return rust[int(art.Noise(ci, ri, b.salt+7)*float64(len(rust)))]
		}
		return bark[0]
	}
	s := 0.2 + b.lift + 0.65*math.Pow(cell.Shade, 1.2) + (art.Noise(ci, ri, b.salt+1)-0.5)*0.3*b.rough
	if fc < 2/plateW && fr < 2/plateL {
		s += 0.15 * b.rough
	}
	return bark.Shade(art.Clamp(s), 0.2, art.Noise(cell.I, cell.J, b.salt+3))
}

// shari reports where the trunk's vein of deadwood runs: a strip up its lit side that narrows to a point at both ends.
func shari(u, across float64) bool {
	narrow := art.Clamp(min(u-1.0, 9.5-u) / 2.5)
	return math.Abs(across+0.35+0.15*art.Wobble(u*0.5, 0)) < 0.4*narrow
}

// wire draws copper wire wound round a stroke: thin diagonal stripes, lit on the upper left, none below the soil.
func wire(c *art.Canvas, path art.Path, w0, w1 float64) {
	paint := func(_, shade float64) art.Color {
		switch {
		case shade > 0.6:
			return wireTone[2]
		case shade > 0.3:
			return wireTone[1]
		}
		return wireTone[0]
	}
	aboveSoil := func(i, j int) bool {
		_, y := c.Center(i, j)
		return y < baseY
	}
	c.Coil(art.Coil{Path: path, W0: w0, W1: w1, Paint: paint, Pitch: 2.6, Twist: 0.4, Duty: 0.28, Grow: 0.6, Test: aboveSoil})
}

// rootFlare draws the trunk's foot flaring over the soil from ThickeningTrunk, wider from Established.
func rootFlare(c *art.Canvas, g float64, lk *look) {
	if !art.From(g, art.ThickeningTrunk) {
		return
	}
	overSoil := func(_, y float64) bool { return y < baseY }
	c.Blob(art.Blob{X: axisX, Y: baseY - 0.4, RX: lk.base * 0.75, RY: lk.flare, Tones: bark[1:], Jitter: 0.3, Where: overSoil})
}

// nebari draws the roots spread over the soil from Established, in front of the stone.
func nebari(c *art.Canvas, g float64, lk *look) {
	for k, root := range nebariRoots {
		if !art.From(g, root.born) {
			continue
		}
		reach := min(root.reach*lk.spread, 9.4)
		path := art.Spline([]art.Point{
			{X: axisX + root.side*lk.base*0.45, Y: soilY - 0.9},
			{X: axisX + root.side*(lk.base*0.4+reach*0.45), Y: soilY - 0.2},
			{X: axisX + root.side*reach, Y: soilY + 1.3},
		}, 4)
		style := barkStyle{rough: min(lk.rough, 0.7), salt: 9 + k, lift: 0.3, floor: soilY + 1.5}
		barkStroke(c, path, []float64{2.6 + 0.3*lk.base/5, 0.8}, style)
	}
}

// deadwood draws bleached jin, stubs of dead branches, from Old: on the outer ends of pads and on the trunk.
func deadwood(c *art.Canvas, g float64, placed []pad, lk *look) {
	for k, j := range jins {
		if art.From(g, j.born) {
			p := placed[j.pad]
			x := p.x + sign(p.x > p.ax)*(p.hw+0.4)
			jin(c, x, p.base-p.th*0.4, j.angle, j.length, 1.5, k)
		}
	}
	for k, s := range stubs {
		if art.From(g, s.born) {
			at := trunkAt(s.h, lk.bend)
			x := at.X + sign(math.Cos(s.angle) > 0)*trunkWidth(lk, s.h/fullHeight)*0.3
			jin(c, x, at.Y, s.angle, s.length, 2.2, 9+k)
		}
	}
}

// jin draws a bleached stub of a dead branch from (x, y), width units wide there and tapering to a point.
func jin(c *art.Canvas, x, y, angle, length, width float64, salt int) {
	end := art.Point{X: x + math.Cos(angle)*length, Y: y + math.Sin(angle)*length}
	mid := art.Point{X: x + math.Cos(angle)*length*0.5 + 0.3, Y: y + math.Sin(angle)*length*0.5 - 0.3}
	path := art.Spline([]art.Point{{X: x, Y: y}, mid, end}, 3)
	cover := c.Stroke(art.Stroke{Path: path, W0: width, W1: 0.3, Tones: dead, Jitter: 0.3, Salt: salt})
	c.ScatterCover(cover, art.Scatter{Colors: dead[:1], Density: 0.12, Salt: 3})
}
