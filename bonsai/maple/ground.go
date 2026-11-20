package maple

import (
	"math"

	"github.com/KrishRVH/pixel-bonsai/art"
)

// roots lie over the soil and the stone from the trunk's foot: their points, and their width where they leave it.
var roots = []struct {
	points []art.Point
	w      float64
}{
	{[]art.Point{{X: 25.4, Y: 20.4}, {X: 26.6, Y: 18.2}, {X: 28.0, Y: 18.4}, {X: 28.2, Y: 20.4}, {X: 28.2, Y: 22.6}}, 1.7},
	{[]art.Point{{X: 25.4, Y: 20.2}, {X: 27.4, Y: 17.8}, {X: 29.8, Y: 17.4}, {X: 31.0, Y: 19.4}, {X: 31.2, Y: 22.6}}, 1.7},
	{[]art.Point{{X: 23.2, Y: 21.6}, {X: 20.0, Y: 22.0}, {X: 17.4, Y: 22.8}}, 1.6},
	{[]art.Point{{X: 23.6, Y: 21.9}, {X: 21.2, Y: 22.5}, {X: 19.8, Y: 23.1}}, 1.4},
}

// fallen are the leaves lying on the moss and the stand at Venerable.
var fallen = []looseLeaf{
	{17.0, 21.6, 0.35, 0},
	{19.8, 21.9, 0.7, 1},
	{29.0, 21.0, 0.5, 0},
	{31.2, 21.8, 0.3, 2},
	{10.8, 27.0, 0.65, 1},
	{35.4, 27.2, 0.4, 0},
}

// aboveRim clips what lies on the soil, such as moss and the stone, to above the pot's rim.
func aboveRim(_, y float64) bool {
	return y < 23.3
}

// pots draws the nursery pot with its label and the samara in it, then from ThickeningTrunk the ivory bonsai pot.
func pots(c *art.Canvas, g float64) {
	if art.During(g, art.Seed, art.ThickeningTrunk) {
		c.Block(art.Block{X0: 18.4, Y0: 22.4, X1: 29.6, Y1: 23.4, Tones: soil, Jitter: 0.4})
		nursery(c)
		c.ScatterBox(18.4, 22.4, 29.6, 23.2, art.Scatter{Colors: soil[2:], Density: 0.2, Salt: 2})
		c.Line(28.6, 22.4, 28.6, 20.4, lichen[1], nil) // the label's stake
		c.Block(art.Block{X0: 27.7, Y0: 19.2, X1: 29.5, Y1: 20.6, Tones: art.Tones{label}})
	} else {
		c.Block(art.Block{X0: 16.2, Y0: 22.2, X1: 31.8, Y1: 23.5, Tones: soil, Rounding: 0.4, Jitter: 0.4})
		potBody(c)
		c.Block(art.Block{X0: 14.9, Y0: 23.2, X1: 33.1, Y1: 24.3, Tones: pot, Rounding: 0.5})
		c.Block(art.Block{X0: 14.9, Y0: 24.3, X1: 33.1, Y1: 24.8, Tones: pot[:1]})
		c.Block(art.Block{X0: 18.4, Y0: 27.4, X1: 29.6, Y1: 28.2, Tones: pot[:2], Rounding: 0.3})
	}
	if art.During(g, art.Seed, art.Sprout) {
		c.Blob(art.Blob{X: 23.2, Y: 22.1, RX: 0.7, RY: 0.6, Tones: art.Tones{seed}})
		c.Stroke(art.Stroke{Path: art.Path{{X: 23.6, Y: 21.9}, {X: 26.4, Y: 20.7}}, W0: 0.9, W1: 0.5, Tones: art.Tones{samara}})
		c.Stroke(art.Stroke{Path: art.Path{{X: 23.6, Y: 22.1}, {X: 26.2, Y: 22.4}}, W0: 0.9, W1: 0.5, Tones: art.Tones{samara}})
	}
}

// nursery draws a plastic nursery pot: a lipped tub tapering to its foot, with ribs down its sides.
func nursery(c *art.Canvas) {
	c.Block(art.Block{X0: 17.4, Y0: 23.0, X1: 30.6, Y1: 24.2, Tones: plastic[2:], Jitter: 0.1})
	c.Block(art.Block{X0: 18.4, Y0: 24.2, X1: 29.6, Y1: 28.2, Tones: plastic[1:2], Taper: 1.1})
	c.Line(19.2, 24.6, 19.9, 27.8, plastic[2], nil)
	c.Line(28.3, 24.8, 27.8, 27.8, plastic[0], nil)
	c.Line(22.0, 24.6, 22.2, 27.8, plastic[0], nil)
	c.Line(25.7, 24.6, 25.5, 27.8, plastic[0], nil)
	c.Line(21.0, 28.0, 27.0, 28.0, plastic[0], nil)
}

// potBody draws the bonsai pot's round body: its half width falls from 8.4 units under the rim to 5.4 at the foot
// along a curve, shaded as a cylinder, with an incised band round it.
func potBody(c *art.Canvas) {
	const (
		cx, y0, y1   = 24.0, 23.2, 27.6
		hw0, hw1     = 8.4, 5.4
		curve        = 1.8
		band0, band1 = 25.3, 25.8
	)
	for _, s := range c.Scan(cx-hw0-1, y0, cx+hw0+1, y1) {
		t := art.Clamp((s.Y - y0) / (y1 - y0))
		nx := (s.X - cx) / (hw1 + (hw0-hw1)*math.Sqrt(1-math.Pow(t, curve)))
		if math.Abs(nx) > 1 {
			continue
		}
		shade := 0.62 - 0.5*nx - 0.05*t
		if band0 <= s.Y && s.Y < band1 {
			shade -= 0.16
		}
		c.Put(s.I, s.J, pot.Shade(art.Clamp(shade), 0.04, art.Noise(s.I, s.J, 0)))
	}
}

// ground draws what lies on the soil: moss, a stone with roots clasping it, the nebari, then the stand and its
// accent, and the fallen leaves.
func ground(c *art.Canvas, g float64) {
	if art.From(g, art.Refined) {
		mossBeds(c, g)
	}
	if art.From(g, art.Established) {
		stoneAndRoots(c, g)
	} else if art.From(g, art.ThickeningTrunk) {
		// Until the nebari spreads, the trunk's foot swells into the soil.
		c.Blob(art.Blob{X: baseX, Y: 22.0, RX: 3.2, RY: 1.3, Tones: bark[1:4], Where: func(_, y float64) bool { return y < 23.2 }})
	}
	if art.From(g, art.ShowTree) {
		standAndAccent(c)
	}
	if art.From(g, art.Venerable) {
		for _, f := range fallen {
			singleLeaf(c, f.x, f.y, f.level, 0, f.kind)
		}
	}
}

// mossBeds draws moss flecked over the soil from Refined, thickening at Established, with beds of it either side of
// the trunk that spread at Established.
func mossBeds(c *art.Canvas, g float64) {
	c.ScatterBox(16.2, 21.6, 31.8, 23.2, art.Scatter{Colors: moss, Density: 0.3 + 0.5*art.Since(g, art.Established, 0.004), Salt: 3})
	if !art.From(g, art.Established) {
		for _, x := range []float64{18.2, 29.8} {
			c.Blob(art.Blob{X: x, Y: 22.5, RX: 2.5, RY: 1.0, Tones: moss, Jitter: 0.6, Salt: int(x), Where: aboveRim})
		}
		return
	}
	for _, bed := range []struct{ x, rx float64 }{{17.5, 2.6}, {21, 2.0}, {28, 2.4}, {31, 3.0}} {
		c.Blob(art.Blob{X: bed.x, Y: 22.4, RX: bed.rx, RY: 1.2, Tones: moss, Jitter: 0.6, Salt: int(bed.x), Where: aboveRim})
	}
}

// stoneAndRoots draws the stone, mossy on its lit top, and the roots clasping it and lying over the soil, thicker
// from Ancient.
func stoneAndRoots(c *art.Canvas, g float64) {
	rock := c.Blob(art.Blob{X: 30.4, Y: 20.0, RX: 3.9, RY: 3.4, Tones: stone, Jitter: 0.3, Lump: 0.08, Salt: 5, Where: aboveRim})
	lit := rock.Filter(func(cell art.Cell) bool { return cell.Shade > 0.6 })
	c.ScatterCover(lit, art.Scatter{Colors: moss[1:], Density: 0.25, Salt: 6})
	thick := 1.0
	if art.From(g, art.Ancient) {
		thick = 1.4
	}
	for n, r := range roots {
		root(c, art.Spline(r.points, 4), r.w*thick, n)
	}
}

// root draws a root lying over the soil, w wide where it leaves the trunk and tapering, grey and lit with a shadow
// under it, kept above the pot's rim.
func root(c *art.Canvas, path art.Path, w float64, salt int) {
	ghost := art.NewCanvas(c.Scale())
	shadow := make(art.Path, len(path))
	for k, p := range path {
		shadow[k] = art.Point{X: p.X, Y: p.Y + 0.7}
	}
	ghost.Stroke(art.Stroke{Path: shadow, W0: w, W1: 0.6, Tones: bark[:1]})
	ghost.Stroke(art.Stroke{Path: path, W0: w, W1: 0.6, Tones: bark[2:], Salt: salt})
	c.Overlay(ghost, func(_, j int) bool { return float64(j) < 23*c.Scale() })
}

// standAndAccent draws the display stand and, on it beside the pot, a clay dish with a tuft of grass.
func standAndAccent(c *art.Canvas) {
	c.Block(art.Block{X0: 8, Y0: 28.2, X1: 43, Y1: 29.2, Tones: stand[1:], Jitter: 0.2})
	c.Block(art.Block{X0: 9, Y0: 29.2, X1: 42, Y1: 30, Tones: stand[:2]})
	c.Block(art.Block{X0: 36.6, Y0: 26.2, X1: 41.4, Y1: 28.2, Tones: clay, Taper: 0.8})
	for k, d := range []art.Point{{X: -2.6, Y: -3.4}, {X: -1.4, Y: -4.4}, {X: 0.2, Y: -4.8}, {X: 1.6, Y: -4.0}, {X: 2.8, Y: -2.6}} {
		path := art.Path{{X: 39, Y: 26.4}, {X: 39 + d.X*0.4, Y: 26.4 + d.Y*0.7}, {X: 39 + d.X, Y: 26.4 + d.Y}}
		c.Stroke(art.Stroke{Path: path, W0: 0.9, W1: 0.3, Tones: grass, Salt: k})
	}
}
