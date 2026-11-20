package pine

import (
	"math"

	"github.com/KrishRVH/pixel-bonsai/art"
)

// aboveRim clips the moss and the stone on the soil to above the pot's rim.
func aboveRim(_, y float64) bool {
	return y < 22.6
}

// ground draws the pot, the soil, and everything on the ground.
func ground(c *art.Canvas, g float64) {
	if art.During(g, art.Seed, art.ThickeningTrunk) {
		nursery(c)
	} else {
		bonsaiPot(c)
	}
	if art.During(g, art.Seed, art.Sprout) {
		c.Blob(art.Blob{X: 23.4, Y: 21.4, RX: 1.5, RY: 0.9, Tones: seed})
		c.Line(24.4, 20.9, 27.0, 20.1, wing, nil)
		c.Line(20.5, 22.0, 22.4, 21.6, soil[0], nil)
		c.Line(25.5, 22.0, 27.5, 21.7, soil[0], nil)
	}
	mossOnSoil(c, g)
	if art.From(g, art.Venerable) {
		c.ScatterBox(14.3, soilY, 33.7, 22.4, art.Scatter{Colors: oldNeedle, Density: 0.12, Salt: 5, Run: 2})
	}
	if art.From(g, art.ShowTree) {
		c.Block(art.Block{X0: 7.5, Y0: 28.4, X1: 44.5, Y1: 29.2, Tones: stand})
		c.Block(art.Block{X0: 9, Y0: 29.2, X1: 12, Y1: 30, Tones: stand[:2]})
		c.Block(art.Block{X0: 40, Y0: 29.2, X1: 43, Y1: 30, Tones: stand[:2]})
		accent(c)
	}
}

// nursery draws the plastic nursery pot and its soil.
func nursery(c *art.Canvas) {
	c.Block(art.Block{X0: 19.6, Y0: soilY, X1: 28.4, Y1: 22.4, Tones: soil, Jitter: 0.5})
	c.Block(art.Block{X0: 19.4, Y0: 23.6, X1: 28.6, Y1: 28.0, Tones: plastic, Taper: 1.3})
	c.Block(art.Block{X0: 18.8, Y0: 22.4, X1: 29.2, Y1: 23.6, Tones: plastic[1:]})
}

// bonsaiPot draws the deep clay pot, its rim and feet, and its soil.
func bonsaiPot(c *art.Canvas) {
	c.Block(art.Block{X0: 14.3, Y0: soilY, X1: 33.7, Y1: 22.6, Tones: soil, Jitter: 0.5})
	clay(c, clayBox{x0: 13.2, y0: 22.6, x1: 34.8, y1: 23.6, lit: 0.9, salt: 1})
	clay(c, clayBox{x0: 14.0, y0: 23.6, x1: 34.0, y1: 27.4, taper: 0.6, under: 0.9, salt: 2})
	for _, x := range []float64{15.4, 30.6} {
		clay(c, clayBox{x0: x, y0: 27.4, x1: x + 3.0, y1: 28.4, salt: 3})
	}
}

// A clayBox is a box of unglazed clay, its foot narrowed by taper units on each side.
type clayBox struct {
	x0, y0, x1, y1 float64
	taper          float64
	under          float64 // the height of a darker band below its top edge
	lit            float64 // the height of a lit line along its top edge
	salt           int
}

// clay draws a box of unglazed clay: its left edge lit, its right side in shadow, a darker band or a lit line along
// its top edge, and a grain that shows more at a larger scale.
func clay(c *art.Canvas, b clayBox) {
	for _, p := range c.Scan(b.x0, b.y0, b.x1, b.y1) {
		inset := b.taper * (p.Y - b.y0) / (b.y1 - b.y0)
		left, right := b.x0+inset, b.x1-inset
		if p.Y < b.y0 || p.Y >= b.y1 || p.X < left || p.X >= right {
			continue
		}
		s := 0.55 + (art.Noise(p.I, p.J, b.salt)-0.5)*0.18
		if p.X-left < 1.0 {
			s += 0.4
		} else if p.X-left < 2.0 {
			s += 0.12
		}
		if right-p.X < 2.2 {
			s -= 0.45
		}
		if p.Y-b.y0 < b.under {
			s -= 0.22
		}
		if p.Y-b.y0 < b.lit {
			s += 0.35
		}
		if b.y1-p.Y < 0.8 {
			s -= 0.18
		}
		c.Put(p.I, p.J, pot.Shade(art.Clamp(s), 0.1, art.Noise(p.I, p.J, b.salt+1)))
	}
}

// mossOnSoil draws moss on the soil: patches of it from Refined, and from Established a bed of it.
func mossOnSoil(c *art.Canvas, g float64) {
	if art.During(g, art.Refined, art.Established) {
		c.ScatterBox(14.3, soilY, 33.7, 22.6, art.Scatter{Colors: moss, Density: 0.5})
		for k, x := range []float64{16.5, 19.5, 28.8, 31.8} {
			c.Blob(art.Blob{X: x, Y: 21.7, RX: 1.9, RY: 0.9, Tones: moss, Lump: 0.2, Salt: 20 + k, Where: aboveRim})
		}
	}
	if !art.From(g, art.Established) {
		return
	}
	for k := range 8 {
		c.Blob(art.Blob{X: 15 + 2.8*float64(k), Y: 21.3, RX: 2.4, RY: 1.6, Tones: moss, Lump: 0.2, Salt: k, Where: aboveRim})
	}
	for k := range 3 {
		c.Blob(art.Blob{X: 29.5 + 2.2*float64(k), Y: 20.9, RX: 1.9, RY: 1.5, Tones: moss, Lump: 0.2, Salt: 30 + k, Where: aboveRim})
	}
}

// stone draws a big mossy stone on the soil at the left, standing in front of the lowest tier's underside.
func stone(c *art.Canvas) {
	clip := func(x, y float64) bool { return aboveRim(x, y) && x > 13.9 }
	cover := c.Blob(art.Blob{X: 17.2, Y: 20.6, RX: 4.2, RY: 3.6, Tones: stoneTone, Jitter: 0.3, Lump: 0.12, Salt: 5, Where: clip})
	high := func(_, j int, _ float64) bool { return float64(j) < 19.6*c.Scale() }
	c.ScatterCover(cover, art.Scatter{Colors: moss[1:], Density: 0.3, Salt: 7, Test: high})
	c.Blob(art.Blob{X: 16.6, Y: 17.9, RX: 3.0, RY: 1.3, Tones: moss, Jitter: 0.3, Lump: 0.25, Salt: 6, Where: aboveRim})
}

// accent draws a little slate pot of grass beside the pot, on the stand, two of its blades in flower.
func accent(c *art.Canvas) {
	c.Block(art.Block{X0: 38.6, Y0: 26.4, X1: 43.6, Y1: 28.4, Tones: slate, Taper: 0.8, Rounding: 0.3})
	for k, blade := range []struct{ dx, lean, h float64 }{{-1.4, -0.7, 4.2}, {-0.5, -0.2, 5.6}, {0.5, 0.25, 4.8}, {1.4, 0.7, 3.8}} {
		x := 41.1 + blade.dx
		tip := art.Point{X: x + blade.lean*blade.h, Y: 26.8 - blade.h}
		path := art.Spline([]art.Point{{X: x, Y: 26.8}, {X: x + blade.lean*blade.h*0.4, Y: 26.8 - blade.h*0.65}, tip}, 3)
		c.Stroke(art.Stroke{Path: path, W0: 0.9, W1: 0.3, Tones: grass, Salt: k})
		if k%2 == 1 {
			c.Blob(art.Blob{X: tip.X, Y: tip.Y, RX: 0.6, RY: 0.6, Tones: flower, Jitter: art.NoJitter})
		}
	}
}

// stemPlant draws the sprout to the whip: a thin stem with a whorl of seed needles on it, then sparse dark fascicles
// of needles along it, and on the whip a candle at its tip.
func stemPlant(c *art.Canvas, g float64, lk *look) {
	path := make(art.Path, 13)
	for k := range path {
		path[k] = art.Point{X: axisX + 0.4*art.Wobble(float64(k)*0.5, 3)*lk.top/14, Y: baseY - lk.top*float64(k)/12}
	}
	tones := stem
	if art.From(g, art.Whip) {
		tones = olive
	}
	c.Stroke(art.Stroke{Path: path, W0: lk.base, W1: lk.base * 0.8, Tones: tones})
	tip := path[len(path)-1]
	if art.During(g, art.Sprout, art.Sapling) {
		whorl := burst{
			x: tip.X, y: tip.Y, length: 3.4 + 0.8*b2f(art.From(g, art.Seedling)), count: 6, colors: sprout,
			from: -3.2, to: 0.05, salt: 1, spread: 0.1,
		}
		needles(c, whorl)
		i, j := c.Cell(tip.X, tip.Y)
		c.Put(i, j, seed[1])
	}
	if art.From(g, art.Sapling) {
		count := max(2, int(math.RoundToEven(art.Steps(g, []art.Step{art.At(art.Sapling, 2), art.At(art.Whip, 4)}))))
		for k := range count {
			at := path.Along(0.25 + 0.5*float64(k)/float64(count-1))
			fascicle(c, at.X, at.Y, -1.57+1.2*sign(k%2 == 1), 2.3, 0.55)
		}
		fascicle(c, tip.X, tip.Y, -1.57, 2.8, 0.9)
	}
	if art.From(g, art.Whip) {
		i, j := c.Cell(tip.X, tip.Y-1.0)
		candle(c, i, j)
	}
}

// fascicle draws a small dark tuft of needles, a V of three spread apart with lit tips, from the node (x, y) leaning
// toward angle.
func fascicle(c *art.Canvas, x, y, angle, length, spread float64) {
	for _, da := range []float64{-spread, 0, spread} {
		ex, ey := x+math.Cos(angle+da)*length, y+math.Sin(angle+da)*length
		c.Line(x, y, ex, ey, needle[2], nil)
		i, j := c.Cell(ex, ey)
		c.Put(i, j, needle[4])
	}
}
