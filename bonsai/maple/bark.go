package maple

import (
	"math"

	"github.com/KrishRVH/pixel-bonsai/art"
)

var (
	// flare is how far the trunk's foot spreads, as a multiple of the trunk's width.
	flare = []art.Step{art.At(art.Established, 1.15), art.At(art.Mature, 1.6), art.At(art.Old, 1.55), art.At(art.Ancient, 1.45)}
	// mottling is the share of the bark in pale and dark patches.
	mottling = []art.Step{art.At(art.Mature, 0.42), art.At(art.Old, 0.44), art.At(art.Ancient, 0.1)}
)

// barkStroke draws a trunk, root, or limb of smooth gray bark: shaded as a cylinder lit on the left, with faint
// streaks along it and, as it ages, mottling over a share of it.
func barkStroke(c *art.Canvas, path art.Path, w0, w1, mottle float64, salt int) {
	for _, cell := range c.Trace(path, w0, w1) {
		shade := cell.Shade
		// Streaks run along the bark, in bands about 0.8 units across and 5 long.
		if art.Noise(int(math.RoundToEven(cell.Across*w0/1.6)), int(cell.U/5), salt+4) < 0.2 {
			shade -= 0.16
		}
		col := bark.Shade(shade, 0.25, art.Noise(cell.I, cell.J, salt))
		if mottle != 0 {
			col = mottled(cell, mottle, salt, col)
		}
		c.Put(cell.I, cell.J, col)
	}
}

// mottled returns the color of a bark pixel in blotches two pixels across: pale patches on its lit side and dark ones
// anywhere, each over half of mottle's share. Elsewhere the bark keeps col.
func mottled(cell art.StrokeCell, mottle float64, salt int, col art.Color) art.Color {
	blot := art.Noise(cell.I/2, cell.J/2, salt+5)
	switch {
	case cell.Shade > 0.3 && blot < 0.5*mottle:
		return patch[b2i(art.Noise(cell.I, cell.J, salt+6) < 0.3)]
	case blot > 1-0.5*mottle:
		dark := art.Tones{twig[2], bark[1], bark[0]} // lit to shadow
		return dark[min(2, int((1-cell.Shade)*3))]
	}
	return col
}

// branches draws the gray limbs, and from Ramified the forks of fine twig on them.
func branches(c *art.Canvas, g float64, limbs []limb, width float64) {
	for _, l := range limbs {
		share := 0.2
		if pads[l.pad].parent < 0 {
			share = 0.34
		}
		barkStroke(c, l.path, max(0.9, width*share), 0.5, 0, l.pad)
		if !art.From(g, art.Ramified) {
			continue
		}
		for n, at := range []float64{0.45, 0.75} {
			p := l.path.Along(at)
			side := float64(2*((l.pad+n)%2) - 1)
			c.Line(p.X, p.Y, p.X+side*1.6, p.Y-1.4-float64(n), twig[2], nil)
		}
	}
}

// wire draws the copper wire coiled round the trunk and the limbs while they are first being bent.
func wire(c *art.Canvas, g float64, path art.Path, width float64, limbs []limb) {
	if !art.During(g, art.FirstWiring, art.ThickeningTrunk) {
		return
	}
	c.Coil(art.Coil{Path: path, W0: width + 0.5, W1: width*0.4 + 0.5, Color: wireTone})
	for _, l := range limbs {
		c.Coil(art.Coil{Path: l.path, W0: 1.4, W1: 1.0, Color: wireTone})
	}
}

// trunk draws the trunk: a seedling's red stem, then gray bark with its root flare, which broadens from Mature, the
// hollow of Old, and the shari of Ancient, a bleached stripe up most of its length.
func trunk(c *art.Canvas, g float64, path art.Path, width float64) {
	taper := max(width*0.4, 0.9)
	if !art.From(g, art.YoungTree) {
		c.Stroke(art.Stroke{Path: path, W0: width, W1: taper, Tones: young})
		return
	}
	mottle := art.Steps(g, mottling)
	barkStroke(c, path, width, taper, mottle, 0)
	if art.From(g, art.ThickeningTrunk) {
		barkStroke(c, path[:7], width*1.3, width*1.02, mottle, 3)
	}
	if spread := art.Steps(g, flare); spread != 0 {
		foot := art.NewCanvas(c.Scale())
		splayed := make(art.Path, 10)
		for k, p := range path[:10] {
			splayed[k] = art.Point{X: p.X - 2.8*(spread-1), Y: p.Y}
		}
		barkStroke(foot, splayed, width*spread, width*1.02, mottle, 3)
		c.Overlay(foot, func(_, j int) bool { return float64(j) < 23.3*c.Scale() })
	}
	if art.From(g, art.Old) {
		at := path.Along(0.34)
		c.Blob(art.Blob{X: at.X + width*0.1, Y: at.Y, RX: width * 0.34, RY: width * 0.5, Tones: lichen, Jitter: 0.2})
		c.Blob(art.Blob{X: at.X + width*0.12, Y: at.Y + 0.2, RX: width * 0.22, RY: width * 0.36, Tones: art.Tones{scar}})
	}
	if art.From(g, art.Ancient) {
		shares := []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7}
		vein := make(art.Path, len(shares))
		for k, s := range shares {
			p := path.Along(s)
			vein[k] = art.Point{X: p.X - width*0.2*(1-s) + 0.5*art.Wobble(s*4, 0), Y: p.Y}
		}
		c.Stroke(art.Stroke{Path: vein, W0: width * 0.34, W1: width * 0.2, Tones: deadwood, Salt: 3})
	}
}
