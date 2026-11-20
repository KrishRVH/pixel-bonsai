package sakura

import (
	"math"

	"github.com/KrishRVH/pixel-bonsai/art"
)

// knots are burls on the trunk: the share of its height, the side, and the stage each grows at.
var knots = []struct {
	share float64
	side  float64
	born  art.Stage
}{{0.12, -1, art.Old}, {0.34, 1, art.Old}, {0.46, -1, art.Ancient}, {0.07, 1, art.Ancient}}

// wire draws copper wire wound round a limb in thin diagonal turns.
func wire(c *art.Canvas, b limb) {
	paint := func(across, _ float64) art.Color { return wireTone[b2i(across < 0)] }
	c.Coil(art.Coil{Path: b.path, W0: b.w0, W1: b.w1, Paint: paint, Pitch: 3.2, Twist: 0.4, Duty: 0.32, Grow: 0.6})
}

// barkDetail draws gloss, lenticels, and knots on the trunk, then from Old a hollow and from Ancient a vein of shari.
func barkDetail(c *art.Canvas, g float64, trunk art.Path, width float64, cover *art.Cover) {
	if art.From(g, art.ThickeningTrunk) {
		c.ScatterCover(cover, art.Scatter{Colors: art.Tones{bark[4]}, Density: 0.22, Salt: 8, Test: func(_, _ int, s float64) bool { return s > 0.78 }})
	}
	if art.From(g, art.Refined) {
		c.ScatterCover(cover, art.Scatter{Colors: art.Tones{gloss}, Density: 1, Salt: 15, Test: func(_, _ int, s float64) bool { return s > 0.9 }})
	}
	if art.From(g, art.Mature) {
		lenticels(c, g, cover)
	}
	for _, knot := range knots {
		if art.From(g, knot.born) {
			at := trunk.Along(knot.share)
			w := width * (1 - 0.58*knot.share)
			c.Blob(art.Blob{X: at.X + knot.side*w*0.42, Y: at.Y, RX: 1.3, RY: 1.7, Tones: bark, Lump: 0.2, Salt: int(knot.share * 99)})
		}
	}
	if art.From(g, art.Old) {
		oldBark(c, g, trunk, width, cover)
	}
}

// lenticels draws the pale horizontal dashes of cherry bark, bold on a mature tree and fading with age.
func lenticels(c *art.Canvas, g float64, cover *art.Cover) {
	row := 2 + int(math.RoundToEven(c.Scale()))
	density, run := 0.3, 2
	if art.During(g, art.Mature, art.Old) {
		density, run = 0.65, 3
	}
	for _, band := range []struct {
		color     art.Color
		low, high float64
	}{{lenticel[0], 0.25, 0.55}, {lenticel[1], 0.5, 0.92}} {
		test := func(_, j int, s float64) bool { return j%row == 0 && band.low < s && s < band.high }
		c.ScatterCover(cover, art.Scatter{Colors: art.Tones{band.color}, Density: density, Salt: 5, Run: run, Test: test})
	}
}

// oldBark draws a strip of shari torn down the trunk, deep fissures, a hollow near the foot, and from Ancient a vein
// of deadwood up from it.
func oldBark(c *art.Canvas, g float64, trunk art.Path, width float64, cover *art.Cover) {
	at := trunk.Along(0.44)
	x := at.X - 0.4*width*(1-0.58*0.44)
	k := width / 5.8
	strip := art.Path{{X: x + 0.5, Y: at.Y + 1.2}, {X: x - 2.6*k, Y: at.Y - 0.2*k}, {X: x - 5.6*k, Y: at.Y - 3.4*k}}
	c.Stroke(art.Stroke{Path: strip, W0: 3.0 * k, W1: 0.5, Tones: shari, Salt: 7})
	fissures := art.Steps(g, []art.Step{art.At(art.Old, 0.03), art.At(art.Ancient, 0.06)})
	c.ScatterCover(cover, art.Scatter{Colors: art.Tones{bark[0]}, Density: fissures, Salt: 14, Test: func(_, _ int, s float64) bool { return 0.2 < s && s < 0.8 }})
	foot := int(math.RoundToEven(0.2 * float64(len(trunk)-1)))
	p := trunk[foot]
	w := width * (1 - 0.58*0.2)
	c.Blob(art.Blob{X: p.X - 0.15*w, Y: p.Y, RX: 0.2*w + 0.7, RY: 2.3, Tones: shari[1:], Lump: 0.1, Salt: 3})
	c.Blob(art.Blob{X: p.X - 0.1*w + 0.2, Y: p.Y + 0.2, RX: 0.2 * w, RY: 1.8, Tones: hollow, Salt: 4})
	if art.From(g, art.Ancient) {
		end := int(math.RoundToEven(0.46 * float64(len(trunk)-1)))
		vein := make(art.Path, 0, end-foot+2)
		for _, q := range trunk[foot-2 : end] {
			vein = append(vein, art.Point{X: q.X - 0.22*w, Y: q.Y})
		}
		c.Stroke(art.Stroke{Path: vein, W0: 1.4, W1: 0.6, Tones: shari, Salt: 6})
	}
}

// petals draws petals in the air from Ancient, and from Venerable fallen on the moss, the pot, and the stand.
func petals(c *art.Canvas, g float64) {
	drift := art.Steps(g, []art.Step{art.At(art.Ancient, 0.04), art.At(art.Venerable, 0.055)})
	free := func(i, j int, _ float64) bool { return !c.Has(i, j) && !c.Has(i+1, j) }
	c.ScatterBox(0, 1, 48, 27, art.Scatter{Colors: bloom[3:6], Density: drift, Salt: 9, Test: free})
	c.ScatterBox(0, 1, 48, 27, art.Scatter{Colors: bloom[4:6], Density: drift * 0.4, Salt: 10, Run: 2, Test: free})
	if !art.From(g, art.Venerable) {
		return
	}
	onSomething := func(i, j int, _ float64) bool { return c.Has(i, j) }
	c.ScatterBox(11.8, 22.0, 36.2, 23.4, art.Scatter{Colors: bloom[4:6], Density: 0.3, Salt: 11, Test: onSomething})
	c.ScatterBox(10.8, 23.2, 37.2, 24.0, art.Scatter{Colors: bloom[5:], Density: 0.12, Salt: 12, Test: onSomething})
	c.ScatterBox(8, 27.6, 40, 28.2, art.Scatter{Colors: bloom[4:6], Density: 0.2, Salt: 13, Test: onSomething})
}
