package maple

import (
	"cmp"
	"math"
	"slices"

	"github.com/KrishRVH/pixel-bonsai/art"
)

var (
	// spread is how many of the leaf tones a pad spans, from its shadowed lower right to its lit upper left.
	spread = []art.Step{art.At(art.YoungTree, 0.5), art.At(art.Shaped, 0.8), art.At(art.Ramified, 0.9), art.At(art.Refined, 1.4)}
	// split is how far a pad's lit crest and shadowed flank pull apart.
	split = []art.Step{art.At(art.Refined, 0.24)}
	// gaps is how many holes open in the pads, on a canvas of scale 1.4 and up.
	gaps = []art.Step{art.At(art.Refined, 0.5), art.At(art.Old, 0.8)}
	// tint is how far the foliage creeps toward gold with age.
	tint = []art.Step{art.At(art.Ramified, 0.02), art.At(art.Refined, 0.05), art.At(art.Mature, 0.09), art.At(art.Venerable, 0.12)}
)

// A hole is a gap in a pad where the sky shows: its center and its half width; it is 0.7 as tall.
type hole struct{ x, y, r float64 }

// A clump is one rounded clump of a pad's leaves: its offset from the pad's center and its radii.
type clump struct{ ox, oy, rx, ry float64 }

// foliage draws the pads from the top down, so each lower pad's lit crown overlaps the dark underside above it.
func foliage(c *art.Canvas, g float64, limbs []limb) {
	green := greenness(g)
	span, open, apart := spreadAt(g), art.Steps(g, gaps), art.Steps(g, split)
	drop := 1 + b2i(art.From(g, art.Refined))
	order := slices.Clone(limbs)
	slices.SortStableFunc(order, func(a, b limb) int { return cmp.Compare(a.y, b.y) })
	for _, l := range order {
		level := 0.40 + art.Steps(g, tint) - 0.1*(l.x-24)/20 - 0.08*(l.y-10)/10
		shadeBehind(c, l.x, l.y, l.r, l.r*0.58, drop)
		cover := pad(c, l.x, l.y, l.r, level, green, span, open, l.pad*11)
		if apart != 0 {
			splitLight(c, cover, l.x, l.y, l.r, apart, green)
		}
		underside(c, cover, green)
		fringe(c, cover, green, l.pad*5)
		if c.Scale() >= 1.4 {
			leaves(c, cover, green, l.pad*3)
		}
	}
}

// greenness returns how much green shows in the foliage: all of it until FirstBranches, fading, but never below a
// share until Shaped, the first full autumn color, drops it.
func greenness(g float64) float64 {
	fade := 1 - art.Since(g, art.FirstBranches, art.Shaped.Start()-art.FirstBranches.Start())
	return max(fade, 0.45*(1-art.Since(g, art.Shaped, 0.004)))
}

// spreadAt returns the spread of the pads at age g. Each of its steps eases in over 0.03 of age rather than showing
// at once.
func spreadAt(g float64) float64 {
	v := 0.0
	for _, step := range spread {
		v += (step.Value - v) * art.Since(g, step.Stage, 0.03)
	}
	return v
}

// lump draws a rounded clump of leaves centered on (cx, cy) with the clump's radii, lit on its upper left and ragged
// at its edge, with level its mean tone and span the tones it spans. It returns its cover, each pixel's shade being
// its tone level.
func lump(c *art.Canvas, cx, cy float64, cl clump, level, green, span float64, holes []hole, salt int) *art.Cover {
	cover := art.NewCover()
	depth := 0.2 * art.Clamp(c.Scale()-1) // on a larger canvas the leaf glyphs are lit over a darker mass
	jitter := 0.25 - 0.1*art.Clamp(c.Scale()-1)
	for _, s := range c.Scan(cx-cl.rx, cy-cl.ry, cx+cl.rx, cy+cl.ry) {
		nx, ny := (s.X-cx)/cl.rx, (s.Y-cy)/cl.ry
		d := nx*nx + ny*ny
		if d > 1 || (d > 0.6 && art.Noise(s.I, s.J, salt) < (d-0.6)*1.2) || inHole(holes, s.X, s.Y) {
			continue
		}
		lv := art.Clamp(level + span*0.32*(nx*art.LightX+ny*art.LightY) - 0.1*d - depth)
		cover.Set(s.I, s.J, lv)
		c.Put(s.I, s.J, leafColor(s.I, s.J, lv, green, jitter))
	}
	return cover
}

func inHole(holes []hole, x, y float64) bool {
	for _, h := range holes {
		dx, dy := (x-h.x)/h.r, (y-h.y)/(0.7*h.r)
		if dx*dx+dy*dy < 1 {
			return true
		}
	}
	return false
}

// pad draws a pad of foliage r wide: a ring of leaf clumps round a heart, darker below, with sky showing through holes
// in it on a larger canvas, as many as open allows. It returns the pad's cover, each pixel's shade being its tone
// level.
func pad(c *art.Canvas, cx, cy, r, level, green, span, open float64, salt int) *art.Cover {
	rx, ry := r, r*0.58
	n := 0
	if c.Scale() >= 1.4 {
		n = int(math.RoundToEven(open * 4))
	}
	holes := make([]hole, n)
	for k := range holes {
		holes[k] = hole{
			cx + rx*(art.Noise(salt, 20+k, 0)-0.5)*1.3,
			cy + ry*(art.Noise(salt, 30+k, 0)-0.3)*0.8,
			0.8 + 0.4*art.Noise(salt, 40+k, 0),
		}
	}
	clumps := []clump{{0, 0.1 * ry, 0.8 * rx, 0.75 * ry}}
	for k := range 8 {
		a := 6.283 * (float64(k) + 0.6*art.Noise(salt, k, 0)) / 8
		clumps = append(clumps, clump{0.66 * rx * math.Cos(a), 0.62 * ry * math.Sin(a), 0.42 * rx, 0.55 * ry})
	}
	// The lower clumps draw first, so the higher ones overlap them.
	slices.SortStableFunc(clumps, func(a, b clump) int { return cmp.Compare(b.oy, a.oy) })
	cover := art.NewCover()
	for k, cl := range clumps {
		lv := level - 0.34*cl.oy/ry - 0.12*cl.ox/rx
		cover.Merge(lump(c, cx+cl.ox, cy+cl.oy, cl, lv, green, span, holes, salt+k))
	}
	return cover
}

// shadeBehind casts a pad's shadow on the foliage already painted behind its upper rim: those leaves drop tones.
func shadeBehind(c *art.Canvas, cx, cy, rx, ry float64, drop int) {
	for _, s := range c.Scan(cx-rx-1.5, cy-ry-1.5, cx+rx+1.5, cy) {
		nx, ny := (s.X-cx)/(rx+1.2), (s.Y-cy+0.4)/(ry+1.2)
		col, ok := c.Get(s.I, s.J)
		if !ok || nx*nx+ny*ny >= 1 {
			continue
		}
		if k := slices.Index(leaf, col); k >= 0 {
			c.Put(s.I, s.J, leaf[max(0, k-drop)])
		}
	}
}

// splitLight pulls a pad's lit crest and its shadowed flank apart: pixels toward the light climb the leaf tones and
// those away from it fall, with a ragged line between.
func splitLight(c *art.Canvas, cover *art.Cover, x, y, r, apart, green float64) {
	for _, cell := range cover.Cells() {
		px, py := c.Center(cell.I, cell.J)
		t := (px-x)/r*art.LightX + (py-y)/(0.58*r)*art.LightY + 0.4*(art.Noise(cell.I, cell.J, 16)-0.5)
		if -0.1 <= t && t <= 0.35 {
			continue
		}
		pull := -0.8
		if t > 0.35 {
			pull = 1
		}
		lv := min(cell.Shade+apart*pull, 0.8)
		cover.Set(cell.I, cell.J, lv)
		c.Put(cell.I, cell.J, leafColor(cell.I, cell.J, lv, green, 0.25))
	}
}

// underside darkens most of a pad's bottom edge to its deepest tones.
func underside(c *art.Canvas, cover *art.Cover, green float64) {
	for _, cell := range cover.Cells() {
		if !cover.Has(cell.I, cell.J+1) && art.Noise(cell.I, cell.J, 12) < 0.8 {
			c.Put(cell.I, cell.J, leafColor(cell.I, cell.J, min(cell.Shade, 0.15), green, 0.25))
		}
	}
}

// fringe draws leaves hanging from a pad's underside, longer on a larger canvas.
func fringe(c *art.Canvas, cover *art.Cover, green float64, salt int) {
	drop := max(1, int(math.RoundToEven(0.9*c.Scale())))
	for _, cell := range cover.Cells() {
		i, j := cell.I, cell.J
		if cover.Has(i, j+1) || art.Noise(i, j, salt) >= 0.2 {
			continue
		}
		for k := 1; k < 2+int(art.Noise(i, j, salt+1)*float64(drop)); k++ {
			c.Put(i, j+k, leafColor(i, j+k, cell.Shade-0.15, green, 0.25))
		}
	}
}

// leaves draws leaf glyphs in a staggered grid over a pad and just past its rim, thicker where it is lit. Only a
// canvas of scale 1.4 and up has them.
func leaves(c *art.Canvas, cover *art.Cover, green float64, salt int) {
	size := 3
	if c.Scale() >= 2.6 {
		size = 5
	}
	sx := size + 1 - size/5
	cells := withRim(cover)
	if cells.Len() == 0 {
		return
	}
	i0, j0, i1, j1 := extent(cells)
	for row, j := 0, j0; j <= j1; row, j = row+1, j+size {
		for i := i0 + row%2*(sx/2); i <= i1; i += sx {
			ci := i + int(math.RoundToEven(art.Noise(i, j, salt)*2-1))
			cj := j + int(math.RoundToEven(art.Noise(i, j, salt+1)*2-1))
			lv, ok := cells.Shade(ci, cj)
			if !ok || art.Noise(i, j, salt+2) > lv*1.6+0.1 {
				continue
			}
			lv += 0.26 + 0.36*(art.Noise(i, j, salt+3)-0.5)
			n := art.Noise(i, j, salt+4)
			star(c, ci, cj, size, lv, green, int(n*n*3), 0.1)
		}
	}
}

// withRim returns a pad's cover with a rim one pixel wide round its top and sides, a tenth of a tone darker than the
// pixel inside.
func withRim(cover *art.Cover) *art.Cover {
	out := art.NewCover()
	out.Merge(cover)
	for _, cell := range cover.Cells() {
		for _, p := range [3][2]int{{cell.I + 1, cell.J}, {cell.I - 1, cell.J}, {cell.I, cell.J - 1}} {
			if !out.Has(p[0], p[1]) {
				out.Set(p[0], p[1], cell.Shade-0.1)
			}
		}
	}
	return out
}

// extent returns the box of pixels a cover spans, inclusive. The cover must not be empty.
func extent(cover *art.Cover) (i0, j0, i1, j1 int) {
	cells := cover.Cells()
	i0, j0, i1, j1 = cells[0].I, cells[0].J, cells[0].I, cells[0].J
	for _, cell := range cells {
		i0, j0, i1, j1 = min(i0, cell.I), min(j0, cell.J), max(i1, cell.I), max(j1, cell.J)
	}
	return i0, j0, i1, j1
}
