package sakura

import (
	"math"

	"github.com/KrishRVH/pixel-bonsai/art"
)

// soft returns smooth fixed noise in [0, 1) at a point in units: blotches about cell units across.
func soft(x, y float64, salt int, cell float64) float64 {
	fx, fy := x/cell, y/cell
	ix, iy := int(math.Floor(fx)), int(math.Floor(fy))
	tx, ty := fx-float64(ix), fy-float64(iy)
	tx, ty = tx*tx*(3-2*tx), ty*ty*(3-2*ty)
	top := art.Noise(ix, iy, salt)*(1-tx) + art.Noise(ix+1, iy, salt)*tx
	low := art.Noise(ix, iy+1, salt)*(1-tx) + art.Noise(ix+1, iy+1, salt)*tx
	return top*(1-ty) + low*ty
}

// puffs draws the body of a pad, a row of overlapping domes over an underside that curves up at its ends, and returns
// its cover.
func puffs(c *art.Canvas, p pad, tones art.Tones, extra int) *art.Cover {
	under := func(x, y float64) bool {
		u := (x - p.x) / p.rx
		return y < p.y+p.ry*(0.7-0.5*u*u)
	}
	cover := c.Blob(art.Blob{X: p.x, Y: p.y + p.ry*0.1, RX: p.rx, RY: p.ry * 0.7, Tones: tones, Lump: 0.06, Salt: p.salt, Where: under})
	n := 3 + int(p.rx/2.8) + extra
	for m := range n {
		t := (float64(m)+0.5)/float64(n) - 0.5
		size := 1 - 0.9*math.Abs(t)
		k := p.salt
		dome := art.Blob{
			X:     p.x + t*2*p.rx*0.8 + (art.Noise(k, m, 0)-0.5)*1.2,
			Y:     p.y - p.ry*0.18*size + (art.Noise(k, m+5, 0)-0.5)*p.ry*0.3,
			RX:    p.rx * (0.3 + 0.12*art.Noise(k, m+9, 0)) * (0.8 + 0.4*size),
			RY:    p.ry * (0.75 + 0.3*size),
			Tones: tones,
			Lump:  0.1,
			Salt:  k + m,
			Where: under,
		}
		cover.Merge(c.Blob(dome))
	}
	return cover
}

// A bloomStyle is how a pad looks at an age: how far it has flushed into blossom, the blossom and leaf tones, buds,
// the holes that open in it, and how crisply it is lit.
type bloomStyle struct {
	flush     float64
	window    art.Tones
	leaves    art.Tones
	buds      float64
	sharp     float64
	petals    float64
	hearts    float64
	tier      float64
	holes     []hole
	fineScale bool
	veryOld   bool
}

type hole struct{ x, y, rx, ry float64 }

func newBloomStyle(c *art.Canvas, g float64, p pad) bloomStyle {
	count := int(math.RoundToEven(art.Steps(g, []art.Step{art.At(art.Shaped, 4), art.At(art.Ramified, 5), art.At(art.Refined, 6)})))
	lit := art.Clamp(0.8 - (p.x-footX)/60 - (crownY-(crownY-p.y)/stature(g)-8)/30)
	window := leaf
	if count > 0 {
		from := max(0, int(math.RoundToEven(lit*(5.4-float64(count)))))
		window = bloom[min(from, len(bloom)):min(from+count, len(bloom))]
	}
	s := bloomStyle{
		flush:     art.Steps(g, []art.Step{art.At(art.Shaped, 0.92), art.At(art.Ramified, 0.94), art.At(art.Refined, 0.95), art.At(art.Mature, 0.96), art.At(art.Venerable, 0.98)}),
		window:    rich(c, window),
		leaves:    rich(c, leaf),
		buds:      art.Steps(g, []art.Step{art.At(art.YoungTree, 0.01), art.At(art.FirstBranches, 0.03), art.At(art.FirstStyling, 0.06), art.At(art.Shaped, 0.02)}),
		sharp:     art.Since(g, art.Refined, 0.03),
		petals:    0.16,
		hearts:    0.035,
		fineScale: c.Scale() >= 1.5,
		veryOld:   art.From(g, art.Venerable),
	}
	if s.fineScale {
		s.petals = 0.08
	}
	if s.veryOld {
		s.petals *= 1.5
		s.hearts *= 2
	}
	if isTier(p.salt) {
		s.tier = 0.12
	} else if art.From(g, art.Refined) && p.rx > 4.5 {
		out := 1.0
		if p.x <= footX {
			out = -1
		}
		s.holes = []hole{{p.x - out*p.rx*0.25, p.y + p.ry*0.55, 2.2, 1.5}, {p.x + out*p.rx*0.5, p.y + p.ry*0.45, 1.8, 1.3}}
	}
	return s
}

func (s bloomStyle) inHole(x, y float64) bool {
	for _, h := range s.holes {
		dx, dy := (x-h.x)/h.rx, (y-h.y)/h.ry
		if dx*dx+dy*dy < 1 {
			return true
		}
	}
	return false
}

// blossomPad draws a crown pad: a leafy clump that becomes a cloud of blossom with leaves showing through, open holes
// on an old tree, and a halo of loose petals.
func blossomPad(c *art.Canvas, g float64, p pad) {
	s := newBloomStyle(c, g, p)
	var behind *art.Canvas
	if len(s.holes) > 0 {
		behind = c.Clone()
	}
	extra := int(math.RoundToEven(art.Steps(g, []art.Step{art.At(art.Ramified, 1), art.At(art.Mature, 2)})))
	cover := puffs(c, p, s.window, extra)
	shades := art.NewCover()
	for _, cell := range cover.Cells() {
		x, y := c.Center(cell.I, cell.J)
		if s.inHole(x, y) {
			// The puffs painted it: let what is behind show.
			c.Erase(cell.I, cell.J)
			if col, ok := behind.Get(cell.I, cell.J); ok {
				c.Put(cell.I, cell.J, col)
			}
			continue
		}
		shade := padShade(s, p, cell.Shade, x, y)
		shades.Set(cell.I, cell.J, shade)
		c.Put(cell.I, cell.J, padColor(c, s, p, cover, cell.I, cell.J, shade, x, y))
	}
	if s.flush > 0 && s.fineScale {
		flowers(c, shades, p.salt)
	}
	halo(c, g, shades, p.salt)
}

func padShade(s bloomStyle, p pad, shade, x, y float64) float64 {
	shade = 0.7*shade + 0.15*(1+(x-p.x)/p.rx*art.LightX+(y-p.y)/p.ry*art.LightY)
	shade += 0.2*(soft(x, y, p.salt, 2.4)-0.5) - (0.3+0.2*s.sharp)*max(0, (y-p.y)/p.ry-0.1)
	return art.Clamp(0.5 + (1.4+0.5*s.sharp)*(shade-0.5) + s.tier)
}

// padColor picks a pad pixel's color: a leaf showing through the blossom in shadow, a blossom tone, a white petal, a
// flower's pink heart, or a bud.
func padColor(c *art.Canvas, s bloomStyle, p pad, cover *art.Cover, i, j int, shade, x, y float64) art.Color {
	k := p.salt
	n, m := art.Noise(i, j, k), art.Noise(i, j, k+31)
	var col art.Color
	if s.flush == 0 || m < (1-s.flush)*1.4*(1-shade) {
		col = s.leaves.Shade(0.3+0.6*shade, 0.6, n)
		if s.fineScale && cover.Has(i+1, j) {
			c.Put(i+1, j, col)
		}
	} else {
		col = s.window.Shade(shade, 0.35, n)
		if art.Noise(i, j, k+17) < s.petals*shade*shade*(0.3+1.4*soft(x, y, k+3, 1.8)) {
			col = bloom[5+b2i(shade > 0.6)]
		} else if art.Noise(i, j, k+19) < s.hearts && 0.2 < shade && shade < 0.8 {
			col = center[b2i(shade < 0.4)]
		}
	}
	if m > 1-s.buds {
		col = bud
	}
	return col
}

// flowers draws five-dot blossoms, a deep pink heart and four petals, in most cells of a four-pixel grid over a pad.
func flowers(c *art.Canvas, shades *art.Cover, k int) {
	cells := shades.Cells()
	if len(cells) == 0 {
		return
	}
	i0, j0, i1, j1 := cells[0].I, cells[0].J, cells[0].I, cells[0].J
	for _, cell := range cells {
		i0, j0, i1, j1 = min(i0, cell.I), min(j0, cell.J), max(i1, cell.I), max(j1, cell.J)
	}
	for ci := i0 / 4; ci <= i1/4; ci++ {
		for cj := j0 / 4; cj <= j1/4; cj++ {
			i, j := ci*4+int(art.Noise(ci, cj, k)*4), cj*4+int(art.Noise(ci, cj, k+1)*4)
			shade, ok := shades.Shade(i, j)
			if !ok || art.Noise(ci, cj, k+2) > 0.92 {
				continue
			}
			petal := bloom[4+b2i(shade > 0.5)+b2i(shade > 0.75)]
			floret(c, i, j, petal, center[b2i(shade > 0.45)])
		}
	}
}

// halo scatters loose petals just outside a pad's top and sides.
func halo(c *art.Canvas, g float64, shades *art.Cover, k int) {
	density := art.Steps(g, []art.Step{art.At(art.Shaped, 0.1), art.At(art.Refined, 0.16), art.At(art.Venerable, 0.32)})
	if density == 0 {
		return
	}
	for _, cell := range shades.Cells() {
		for _, q := range [3][2]int{{cell.I, cell.J - 1}, {cell.I - 1, cell.J}, {cell.I + 1, cell.J}} {
			if !shades.Has(q[0], q[1]) && !c.Has(q[0], q[1]) && art.Noise(q[0], q[1], k+23) < density {
				c.Put(q[0], q[1], bloom[3+int(art.Noise(q[0], q[1], k+24)*3)])
			}
		}
	}
}

// twigs draws, from Ramified, a fan of fine twigs under a pad, before the pad.
func twigs(c *art.Canvas, g float64, p pad) {
	if p.rx <= 4.5 {
		return
	}
	fan := int(math.RoundToEven(art.Steps(g, []art.Step{art.At(art.Ramified, 3), art.At(art.Mature, 2)}) * max(1, math.Sqrt(c.Scale()))))
	for m := range fan {
		t := (float64(m)+0.5)/float64(fan) - 0.5
		x, y := p.x+t*p.rx, p.y+p.ry*(0.7-0.5*t*t)
		end := art.Point{X: x + t*1.5, Y: y + 1.8 + 1.4*art.Noise(p.salt, m, 0)}
		path := art.Path{{X: p.x + t*p.rx*0.5, Y: y - 0.6}, {X: p.x + t*p.rx*0.8, Y: y + 0.5}, end}
		c.Stroke(art.Stroke{Path: path, W0: 1.0, W1: 0.5, Tones: bark[2:]})
	}
}

// sprigs draws, from Ramified until Established, twig tips with a bud on them poking out of a pad's top and ends.
func sprigs(c *art.Canvas, g float64, p pad) {
	if !art.During(g, art.Ramified, art.Established) || p.rx < 4.5 {
		return
	}
	empty := func(i, j int) bool { return !c.Has(i, j) }
	for m, t := range []float64{-0.75, -0.25, 0.3, 0.8} {
		x, y := p.x+t*p.rx, p.y-p.ry*(0.78-0.4*t*t)
		lean := (art.Noise(p.salt, 60+m, 0) - 0.5) * 2.4
		tip := art.Point{X: x + lean, Y: y - 1.4 - 1.0*art.Noise(p.salt, 64+m, 0)}
		if tip.Y < 1.5 {
			continue
		}
		c.Line(x, y+0.4, tip.X, tip.Y, bark[3], empty)
		i, j := c.Cell(tip.X, tip.Y)
		c.Put(i, j, bloom[3])
	}
}

// branchlets draws, from Ramified, short forked twigs along each limb, reaching into the gaps between the pads.
func branchlets(c *art.Canvas, g float64, branches []limb) {
	if !art.From(g, art.Ramified) {
		return
	}
	for k, b := range branches {
		for m, share := range []float64{0.55, 0.75} {
			at := b.path.Along(share)
			up := float64(2*((k+m)%2) - 1)
			lean := 1.0
			if art.Noise(k, 90+m, 0) >= 0.5 {
				lean = -1
			}
			ex := at.X + lean*(1.4+art.Noise(k, 91+m, 0))
			ey := at.Y + up*(2.4+1.2*art.Noise(k, 92+m, 0))
			fork := art.Path{at, {X: at.X + lean*0.4, Y: at.Y + up*1.4}, {X: ex, Y: ey}}
			c.Stroke(art.Stroke{Path: fork, W0: 0.9, W1: 0.5, Tones: bark[2:]})
			side := art.Path{{X: at.X + lean*0.4, Y: at.Y + up*1.6}, {X: ex + lean*1.6, Y: ey - up*0.2}}
			c.Stroke(art.Stroke{Path: side, W0: 0.7, W1: 0.4, Tones: bark[2:]})
		}
	}
}

// fringe draws, from Mature, weeping chains of blossom hanging from a pad's underside, longer on a show tree.
func fringe(c *art.Canvas, g float64, p pad) {
	if p.rx <= 4.5 || p.y <= 14 {
		return
	}
	n := int(math.RoundToEven(art.Steps(g, []art.Step{art.At(art.Mature, 3), art.At(art.ShowTree, 4)})))
	empty := func(i, j int) bool { return !c.Has(i, j) }
	for m := range n {
		t := (float64(m)+0.5)/float64(n) - 0.5
		x := p.x + t*p.rx*1.5
		y := p.y + p.ry*(0.7-0.5*t*t*2.25)
		drop := (2.6 + 1.8*art.Noise(p.salt, 80+m, 0)) * (1 + art.Steps(g, []art.Step{art.At(art.ShowTree, 0.3)}))
		sway := 0.7
		if art.Noise(p.salt, 70+m, 0) >= 0.5 {
			sway = -0.7
		}
		c.Line(x, y-0.3, x+sway, y+drop, bloom[2], empty)
		i, j := c.Cell(x+sway, y+drop)
		c.Put(i, j, bloom[4+m%2])
	}
}
