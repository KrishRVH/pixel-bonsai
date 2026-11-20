package maple

import (
	"image"
	"math"
	"slices"

	"github.com/KrishRVH/pixel-bonsai/art"
)

// windows are sky showing between the tiers of the crown, with a limb in it: the center, the radii, and the stage it
// opens at.
var windows = []struct {
	x, y, rx, ry float64
	opens        art.Stage
}{
	{16.5, 9.6, 2.8, 0.9, art.LayeredPads},
	{31.5, 9.2, 2.8, 0.9, art.LayeredPads},
	{13.4, 12.8, 3.9, 1.8, art.Refined},
	{34.8, 12.6, 3.9, 1.8, art.Refined},
	{20.0, 6.6, 2.8, 2.1, art.Refined},
	{29.2, 6.2, 2.8, 2.1, art.Refined},
	{21.2, 5.6, 1.0, 1.6, art.Mature},
	{28.4, 5.2, 1.0, 1.6, art.Mature},
}

// lace is the share of the twigs on the crown's edge that have grown.
var lace = []art.Step{art.At(art.Ramified, 0.55), art.At(art.Refined, 0.6), art.At(art.Established, 0.65), art.At(art.Mature, 0.8)}

// A looseLeaf is a leaf off the tree: where it lies or floats in units, its tone, and its glyph.
type looseLeaf struct {
	x, y, level float64
	kind        int
}

// drift are the leaves in the air beside the crown: the first six from Ancient, and all of them from Venerable.
var drift = []looseLeaf{
	{3.0, 1.6, 0.85, 0},
	{42.0, 2.4, 0.75, 1},
	{45.5, 17.0, 0.7, 0},
	{3.5, 20.0, 0.8, 1},
	{41.0, 20.5, 0.6, 0},
	{10.0, 20.5, 0.45, 1},
	{12.5, 1.5, 0.55, 1},
	{34.5, 1.8, 0.8, 0},
	{2.5, 8.0, 0.6, 0},
	{44.5, 22.5, 0.7, 1},
}

// carve opens the windows of sky between the tiers of the crown and shows the limbs that cross them; the far rim of
// each window catches the light.
func carve(c *art.Canvas, g float64, limbs []limb, width float64) {
	opened := openWindows(c, g)
	if len(opened) == 0 {
		return
	}
	ghost := art.NewCanvas(c.Scale())
	branches(ghost, g, limbs, width)
	var sky []image.Point
	for _, p := range opened {
		if col, ok := ghost.Get(p.X, p.Y); ok {
			c.Put(p.X, p.Y, col)
		} else {
			sky = append(sky, p)
		}
	}
	for _, p := range sky {
		// The far rim of a window faces the light.
		for _, q := range [2]image.Point{{X: p.X + 1, Y: p.Y}, {X: p.X, Y: p.Y + 1}} {
			col, ok := c.Get(q.X, q.Y)
			if k := slices.Index(leaf, col); ok && k >= 0 {
				c.Put(q.X, q.Y, leaf[min(6, k+2)])
			}
		}
	}
}

// openWindows erases the painted pixels inside the windows open at age g, with ragged edges, and returns them.
func openWindows(c *art.Canvas, g float64) []image.Point {
	var opened []image.Point
	for _, w := range windows {
		if !art.From(g, w.opens) {
			continue
		}
		for _, s := range c.Scan(w.x-w.rx-1, w.y-w.ry-1, w.x+w.rx+1, w.y+w.ry+1) {
			dx, dy := (s.X-w.x)/w.rx, (s.Y-w.y)/w.ry
			if dx*dx+dy*dy < 1-0.4*art.Noise(s.I, s.J, 14) && c.Has(s.I, s.J) {
				c.Erase(s.I, s.J)
				opened = append(opened, image.Point{X: s.I, Y: s.J})
			}
		}
	}
	return opened
}

// outside returns a test for the empty pixels the sky reaches from the canvas's border, so a twig can tell the
// crown's edge from its gaps.
func outside(c *art.Canvas) func(i, j int) bool {
	w, h := c.Width(), c.Height()
	seen := make([]bool, w*h)
	todo := make([]image.Point, 0, 2*(w+h))
	for i := range w {
		todo = append(todo, image.Point{X: i, Y: 0}, image.Point{X: i, Y: h - 1})
	}
	for j := range h {
		todo = append(todo, image.Point{X: 0, Y: j}, image.Point{X: w - 1, Y: j})
	}
	for len(todo) > 0 {
		p := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		if p.X < 0 || p.X >= w || p.Y < 0 || p.Y >= h || seen[p.Y*w+p.X] || c.Has(p.X, p.Y) {
			continue
		}
		seen[p.Y*w+p.X] = true
		todo = append(todo, image.Point{X: p.X + 1, Y: p.Y}, image.Point{X: p.X - 1, Y: p.Y},
			image.Point{X: p.X, Y: p.Y + 1}, image.Point{X: p.X, Y: p.Y - 1})
	}
	return func(i, j int) bool {
		return 0 <= i && i < w && 0 <= j && j < h && seen[j*w+i]
	}
}

// spokes returns where the fine twigs of a pad of radius r end, relative to its center: a few fan out over its top
// and sides past the rim.
func spokes(r float64, salt int) []art.Point {
	out := make([]art.Point, 6)
	for k := range out {
		a := -3.3 + 0.7*float64(k) + 0.25*art.Noise(salt, k, 0)
		far := 1.4 + 1.8*art.Noise(salt, k+9, 0)
		out[k] = art.Point{X: math.Cos(a) * (r + far), Y: math.Sin(a) * (r*0.58 + far*0.8)}
	}
	return out
}

// edge draws the crown's lacy edge: fine twigs fan out past the rims of the pads on its outline, each ending in a
// tuft of leaves. They fill only what is still empty, so none crosses a pad.
func edge(c *art.Canvas, g float64, limbs []limb) {
	green, grown := greenness(g), art.Steps(g, lace)
	sky, rim := outside(c), art.NewCanvas(c.Scale())
	ink := twig[1+b2i(c.Scale() >= 1.4)] // the darker twig keeps it fine at scale 1
	for _, l := range limbs {
		for n, tip := range spokes(l.r, l.pad) {
			if !sky(c.Cell(l.x+tip.X, l.y+tip.Y)) || art.Noise(l.pad, n, 15) > grown {
				continue
			}
			rim.Line(l.x+0.6*tip.X, l.y+0.6*tip.Y, l.x+tip.X, l.y+tip.Y, ink, nil)
			lit := (tip.X*art.LightX + tip.Y*art.LightY) / math.Hypot(tip.X, tip.Y)
			tuft(rim, l.x+tip.X, l.y+tip.Y-0.3, 0.5+0.28*lit+0.12*(art.Noise(l.pad, n, 0)-0.5), green, n)
		}
	}
	c.Overlay(rim, func(i, j int) bool { return !c.Has(i, j) })
}
