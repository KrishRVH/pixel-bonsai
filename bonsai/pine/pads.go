package pine

import (
	"math"
	"slices"

	"github.com/KrishRVH/pixel-bonsai/art"
)

// A padPlan is a pad of the finished tree at its full size. A pad grown from another measures inner and base from
// that pad's outer end.
type padPlan struct {
	inner   float64 // the distance from the trunk's axis to the pad's inner end, negative to its left
	base    float64 // the y of its underside at its inner end
	hw, th  float64 // its half width and the height of its dome
	rise    float64 // how far its outer end stands higher
	born    art.Stage
	candles []candlePlan // the candles on its top
	parent  int          // the plan it grows from, or fromTrunk
	back    bool         // whether it stands behind the trunk, in shadow
	floor   float64      // the lowest y any of it reaches
}

// A candlePlan is a candle on a pad's top: the stage it stands from, and its place as a share of the pad's half width
// from its middle.
type candlePlan struct {
	born  art.Stage
	share float64
}

// fromTrunk is the parent of a pad on a branch from the trunk.
const fromTrunk = -1

// dip is how far below a pad its branch leaves the trunk.
const dip = 1.4

var (
	plans = []padPlan{
		{-3.6, 18.9, 6.0, 3.0, 1.0, art.YoungTree, []candlePlan{{art.Shaped, -0.8}}, fromTrunk, false, under},
		{3.0, 12.6, 5.0, 2.6, 0.8, art.YoungTree, []candlePlan{{art.Shaped, 0.3}, {art.Venerable, -0.5}}, fromTrunk, false, under},
		{3.6, 18.5, 5.4, 3.0, 0.8, art.FirstBranches, []candlePlan{{art.Ramified, 0.3}}, fromTrunk, false, under},
		{-3.0, 12.8, 4.6, 2.6, 0.8, art.FirstBranches, []candlePlan{{art.LayeredPads, -0.4}}, fromTrunk, false, under},
		{0.0, -1.8, 2.3, 2.2, 0.6, art.LayeredPads, []candlePlan{{art.Ramified, 0.6}}, 1, false, under},
		{0.0, -1.8, 2.2, 1.8, 0.6, art.LayeredPads, []candlePlan{{art.Venerable, 0.0}}, 3, false, under},
		{-2.2, 2.2, 2.7, 2.2, -0.8, art.LayeredPads, nil, 0, false, 21.6},
		{0.5, 2.2, 2.7, 2.2, -0.8, art.LayeredPads, []candlePlan{{art.Refined, 0.1}}, 2, false, 21.6},
		{-0.4, 15.4, 5.6, 2.2, 0.0, art.LayeredPads, nil, fromTrunk, true, under},
		{0.6, 9.8, 4.6, 1.8, 0.0, art.LayeredPads, nil, fromTrunk, true, under},
	}
	// crownPlan is the pad at the top of the trunk.
	crownPlan = padPlan{
		0.0, 5.9, 4.2, 2.6, 0.0, art.YoungTree, []candlePlan{{art.Shaped, 0.0}, {art.Venerable, -0.6}}, fromTrunk, false, under,
	}
	// lowestTier is the lowest tier's pads in plans.
	lowestTier = []int{0, 2}
)

// A padShape is a pad of needles' outline: a flat underside at y = base at its inner end, its outer end rise higher
// (a negative rise puts the outer end on the left), and a domed, scalloped top th tall over a half width hw about x.
type padShape struct {
	x, base, hw, th, rise float64
	peak                  float64 // how square the dome's shoulders are; a higher peak is flatter on top
	point                 float64 // how much longer the leader of needles on a pointed top is; 0 for a round top
	floor                 float64 // the lowest y any of it reaches
}

// with returns the shape with a dome's peak and point.
func (s padShape) with(peak, point float64) padShape {
	s.peak, s.point = peak, point
	return s
}

// out returns the side the pad's outer end is on: 1 for the right, -1 for the left.
func (s padShape) out() float64 {
	return sign(s.rise >= 0)
}

// lift returns how far above base the underside is at t, -1 at the pad's left end to 1 at its right.
func (s padShape) lift(t float64) float64 {
	return math.Abs(s.rise) * (t*s.out() + 1) / 2
}

// A pad is a pad where it stands at an age, its branch leaving the trunk, or its parent pad, at (ax, ay).
type pad struct {
	padShape
	ax, ay float64
	plan   *padPlan
	crown  bool
}

// place returns where each pad that has grown stands at an age, in the order of plans and the crown last.
func place(g float64, lk *look) []pad {
	k := lk.top / fullHeight
	var out []pad
	made := map[int]pad{}
	for n := range len(plans) + 1 {
		plan, crown := &crownPlan, n == len(plans)
		if !crown {
			plan = &plans[n]
		}
		if !art.From(g, plan.born) {
			continue
		}
		wide := 1 + lk.lowest*b2f(slices.Contains(lowestTier, n))
		if plan.parent != fromTrunk {
			wide = lk.child
		}
		hw, th := max(plan.hw*lk.size*wide, 0.9), max(plan.th*math.Sqrt(lk.size)*lk.thick, 1.0)
		var p pad
		switch {
		case plan.parent != fromTrunk:
			p = onPad(made[plan.parent], plan, hw)
		case crown:
			p = atTop(lk, plan, k)
		default:
			p = onTrunk(lk, plan, hw, k)
		}
		p.hw, p.th, p.floor, p.plan, p.crown = hw, th, plan.floor, plan, crown
		p.rise = plan.rise * sign(p.x >= p.ax)
		made[n] = p
		out = append(out, p)
	}
	return out
}

// onPad places a pad grown from the outer end of another.
func onPad(near pad, plan *padPlan, hw float64) pad {
	side := sign(near.x > near.ax)
	var p pad
	p.ax, p.ay = near.x+side*near.hw*0.8, near.base-0.5
	p.x, p.base = p.ax+side*(plan.inner+hw), p.ay+0.4+plan.base
	return p
}

// atTop places the crown on the trunk's top.
func atTop(lk *look, plan *padPlan, k float64) pad {
	var p pad
	top := trunkAt(lk.top, lk.bend)
	p.ax, p.ay = top.X, top.Y
	p.x, p.base = p.ax+plan.inner, baseY-(baseY-plan.base)*k
	return p
}

// onTrunk places a pad on a branch from the trunk, out to its side; a pad behind the trunk stays close to it.
func onTrunk(lk *look, plan *padPlan, hw, k float64) pad {
	var p pad
	h := (baseY - plan.base) * k
	tx := trunkX(h/fullHeight, lk.bend)
	side := sign(plan.inner > 0)
	p.x, p.base = tx+plan.inner*lk.reach+side*(hw+0.25*max(lk.base-5.0, 0.0)), baseY-h
	if plan.back {
		p.x = tx + plan.inner
	}
	at := trunkAt(max(h-dip*k, 0.5), lk.bend)
	p.ax, p.ay = at.X, at.Y
	return p
}

// A clump is a sub-pad: its first pixel column and its width in pixels.
type clump struct{ first, w int }

// subPads returns the sub-pads a pad x ± hw across breaks into from Ramified, each about clumps pixels wide.
func subPads(c *art.Canvas, x, hw, clumps float64, salt int) []clump {
	var out []clump
	end := int(math.Ceil((x + hw) * c.Scale()))
	for i := int(math.Floor((x - hw) * c.Scale())); i < end; {
		w := max(2, int(math.RoundToEven(clumps*math.Sqrt(c.Scale())*(0.7+0.6*art.Noise(len(out), salt+70, 0)))))
		out = append(out, clump{i, w})
		i += w
	}
	return out
}

// dome returns how far into its clump the pixel column i is: 0 at a seam, 1 in the middle, and 1 outside them all.
func dome(parts []clump, i int) float64 {
	for _, part := range parts {
		if part.first <= i && i < part.first+part.w {
			return math.Sin(math.Pi * (float64(i-part.first) + 0.5) / float64(part.w))
		}
	}
	return 1.0
}

// A padBrush paints one pad of needles: its shape, its look, its salt, the phase of its scalloped top, and the
// sub-pads it breaks into.
type padBrush struct {
	padShape
	lk    *look
	salt  int
	phase float64
	parts []clump
}

// needlePad draws a pad of needles: its body, fans of lit needles along its top and at its ends, a leader of them on
// a pointed top, and from Refined holes through it. From Ramified it breaks into sub-pads, each a dome with its own
// fan. A larger scale draws more fans. It returns the pixel row of the pad's top in each column it covers.
func needlePad(c *art.Canvas, s padShape, lk *look, salt int) map[int]int {
	p := &padBrush{padShape: s, lk: lk, salt: salt, phase: art.Noise(salt, 11, 0) * 2 * math.Pi}
	if lk.clumps != 0 {
		p.parts = subPads(c, s.x, s.hw, lk.clumps, salt)
	}
	p.body(c)
	p.fans(c)
	p.fill(c)
	p.holes(c)
	p.sprigs(c)
	return p.tops(c)
}

// ridge returns the y of the pad's top at t, -1 at its left end to 1 at its right.
func (p *padBrush) ridge(t float64) float64 {
	scallop := 1 + 0.16*math.Sin(t*p.hw*1.9+p.phase) + 0.1*p.lk.fringe*math.Sin(t*p.hw*3.3+2*p.phase)
	return p.base - p.lift(t) - p.th*math.Pow(1-math.Pow(math.Abs(t), p.peak), 0.6)*scallop
}

// body fills the pad's mass, lit along its top and dark at its underside, each sub-pad a dome with a dark seam.
func (p *padBrush) body(c *art.Canvas) {
	for _, s := range c.Scan(p.x-p.hw, p.base-p.th-2, p.x+p.hw, p.base) {
		t := (s.X - p.x) / p.hw
		if math.Abs(t) >= 1 {
			continue
		}
		u := dome(p.parts, s.I)
		bottom := p.base - p.lift(t) - 0.5*p.lk.notch*(1-u)
		high := p.ridge(t) + p.lk.notch*(1-math.Pow(u, 0.6))
		if s.Y < high || s.Y > min(bottom, p.floor) {
			continue
		}
		d := (s.Y - high) / max(bottom-high, 1.0)
		shade := 0.74 + p.lk.lit - 0.75*d - 0.12*t - 0.35*math.Pow(1-u, 1.5)
		c.Put(s.I, s.J, p.lk.body.Shade(art.Clamp(shade), 0.5, art.Noise(s.I, s.J, p.salt)))
	}
}

// fans draws fans of needles along the pad's top, and one splayed out sideways at each end.
func (p *padBrush) fans(c *art.Canvas) {
	fans := max(3, int(math.RoundToEven(p.hw*2*c.Scale()*(1+0.5*p.lk.fringe)/2.2)))
	ends := [2]struct{ t, a float64 }{{-0.99, -3.0}, {0.99, -0.14 + 0.2*p.out()}}
	for k := range fans + 2 {
		var t, y, a float64
		if k < fans {
			t = -0.96 + 1.92*(float64(k)+0.15+0.7*art.Noise(k, p.salt+2, 0))/float64(fans)
			y = (p.base - p.lift(t) + p.ridge(t)) / 2
			a = -1.57 + (0.7+0.5*t*t)*t
		} else {
			t, a = ends[k-fans].t, ends[k-fans].a
			y = p.base - p.lift(t) - 0.45*p.th
		}
		bottom := p.base - p.lift(t)
		length := (1.0 + 0.6*art.Noise(k, p.salt+3, 0)) * (1 + 0.4*p.lk.fringe*b2f(math.Abs(t) > 0.6))
		if p.point != 0 && math.Abs(t) < 0.3 {
			a, length = -1.57+t, length+p.point
		}
		needles(c, burst{
			x: p.x + t*p.hw, y: y, length: length, count: 4, colors: p.lk.tips, from: a - 0.6, to: a + 0.6,
			salt: p.salt + k, bias: p.lk.glow, test: above(c, min(bottom, p.floor)),
		})
	}
}

// fill draws short needles over the pad's face, where a scale from 1.3 has the pixels to show them.
func (p *padBrush) fill(c *art.Canvas) {
	if c.Scale() < 1.3 {
		return
	}
	for n := range int(math.RoundToEven(p.hw * c.Scale() * 0.8)) {
		t := (art.Noise(n, p.salt+40, 0) - 0.5) * 1.7
		bottom := p.base - p.lift(t)
		y := bottom - (bottom-p.ridge(t))*(0.35+0.35*art.Noise(n, p.salt+41, 0))
		a := -1.57 + 0.7*t
		needles(c, burst{
			x: p.x + t*p.hw, y: y, length: 1.3, count: 2, colors: p.lk.tips, from: a - 0.4, to: a + 0.4,
			salt: p.salt + 60 + n, bias: p.lk.glow - 0.1, test: above(c, min(bottom, p.floor)),
		})
	}
}

// holes opens the gaps of an older pad, two pixels wide, where the sky shows through its needles.
func (p *padBrush) holes(c *art.Canvas) {
	for n := range int(math.RoundToEven(p.lk.gaps * p.hw * c.Scale() / 2)) {
		i, j := c.Cell(p.x+(art.Noise(n, p.salt+20, 0)-0.5)*1.3*p.hw, p.base-p.th*(0.25+0.3*art.Noise(n, p.salt+21, 0)))
		for _, hole := range [2]int{i, i + 1} {
			if col, ok := c.Get(hole, j); ok && isNeedle(col) {
				c.Erase(hole, j)
			}
		}
	}
}

// sprigs draws a small fan of needles standing on top of most sub-pads.
func (p *padBrush) sprigs(c *art.Canvas) {
	for _, part := range p.parts {
		mid := part.first + part.w/2
		x, _ := c.Center(mid, 0)
		if math.Abs(x-p.x) >= p.hw || art.Noise(mid, p.salt+73, 0) >= 0.8 {
			continue
		}
		top := p.ridge((x-p.x)/p.hw) + p.lk.notch*0.1
		needles(c, burst{
			x: x, y: top, length: 0.8 + 0.5*art.Noise(mid, p.salt+74, 0), count: 3, colors: p.lk.tips, from: -2.0,
			to: -1.1, salt: p.salt + mid, bias: p.lk.glow, test: above(c, min(p.base, p.floor)),
		})
	}
}

// tops returns the row of the pad's highest needle pixel in each column it covers, at or above its underside.
func (p *padBrush) tops(c *art.Canvas) map[int]int {
	s := c.Scale()
	tops := map[int]int{}
	j0 := max(0, int(math.Floor((p.base-p.th-3)*s)))
	for i := int(math.Floor((p.x - p.hw) * s)); i < int(math.Ceil((p.x+p.hw)*s)); i++ {
		for j := j0; j < c.Height() && (float64(j)+0.5)/s <= p.base; j++ {
			if col, ok := c.Get(i, j); ok && isNeedle(col) {
				tops[i] = j
				break
			}
		}
	}
	return tops
}

// A burst is a starburst of needles from (x, y), length units long, fanned from angle from to angle to (radians, y
// down), in colors darker at the root and lighter at the tip. They are lit toward the upper left, more so by bias.
type burst struct {
	x, y, length float64
	count        int // needles at scale 1; a larger scale draws more of them
	colors       art.Tones
	from, to     float64
	salt         int
	bias         float64
	test         func(i, j int) bool // when set, a needle pixel is drawn only where it reports true
	spread       float64             // how much the needles' tones vary; 0 takes 0.5
}

// needles draws a burst, each needle a darker line from the root to its middle and a lighter one on to its tip.
func needles(c *art.Canvas, b burst) {
	spread := b.spread
	if spread == 0 {
		spread = 0.5
	}
	n := max(3, int(math.RoundToEven(float64(b.count)*(0.55+0.45*c.Scale()))))
	for k := range n {
		a := b.from + (b.to-b.from)*(float64(k)+0.5)/float64(n)
		reach := b.length * (0.75 + 0.5*art.Noise(k, b.salt, 0))
		cos, sin := math.Cos(a), math.Sin(a)
		lit := math.Pow(art.Lit(cos, sin), 1.6)*0.9 + b.bias
		shift := (art.Noise(k, b.salt+1, 0) - 0.5) * spread
		mx, my := b.x+cos*reach*0.55, b.y+sin*reach*0.55
		c.Line(b.x, b.y, mx, my, b.colors.Shade(lit-0.1+shift, 0, 0.5), b.test)
		c.Line(mx, my, b.x+cos*reach, b.y+sin*reach, b.colors.Shade(lit+0.12+shift, 0, 0.5), b.test)
	}
}

// above returns a test for [art.Canvas.Line] that keeps the pixels centered at or above the line y.
func above(c *art.Canvas, y float64) func(i, j int) bool {
	return func(i, j int) bool {
		_, cy := c.Center(i, j)
		return cy <= y
	}
}

// twigs draws bare twigs reaching down and out from a pad's flat underside: the ramified branch tips showing.
func twigs(c *art.Canvas, p pad, salt int) {
	count := max(2, int(math.RoundToEven(p.hw*(0.3+0.3*c.Scale()))))
	free := func(i, j int) bool {
		_, y := c.Center(i, j)
		return !c.Has(i, j) && y < soilY-0.5
	}
	for n := range count {
		t := -0.85 + 1.7*(float64(n)+art.Noise(n, salt+80, 0))/float64(count)
		x, y := p.x+t*p.hw, p.base-p.lift(t)+0.3
		reach := 1.0 + 1.0*art.Noise(n, salt+81, 0)
		a := 1.57 + (art.Noise(n, salt+82, 0)-0.5)*1.6 + 0.5*t
		c.Line(x, y, x+math.Cos(a)*reach, y+math.Sin(a)*reach, bark[2+n%2], free)
	}
}

// puff draws a small pad perched on top of a big one, toward its inner end: the pads of a ramified tree grow pads.
func puff(c *art.Canvas, p pad, lk *look, salt int) {
	t := -0.45 * p.out()
	top := p.base - p.lift(t) - 0.9*p.th
	shape := padShape{x: p.x + t*p.hw, base: top + 1.0, hw: p.hw * 0.38, th: 1.8, peak: 2.0, floor: top + 1.0}
	needlePad(c, shape, lk, salt)
}

// candles stands a pad's candles that have grown on its top.
func candles(c *art.Canvas, g float64, p pad, tops map[int]int) {
	for _, cp := range p.plan.candles {
		if !art.From(g, cp.born) {
			continue
		}
		i := int(math.Floor((p.x + cp.share*p.hw) * c.Scale()))
		if j, ok := tops[i]; ok {
			candle(c, i, j)
		}
	}
}

// candle draws a candle, a new silvery shoot, standing with its foot just above pixel (i, j): pale silver with a tan
// tip, and from scale 1.75 two pixels wide, lit on its left.
func candle(c *art.Canvas, i, j int) {
	tall := max(2, int(math.RoundToEven(2*c.Scale())))
	wide := 1
	if c.Scale() >= 1.75 {
		wide = 2
	}
	for k := range tall {
		for w := range wide {
			col := candleBody[1]
			switch {
			case k == tall-1:
				col = candleTip
			case wide > 1:
				col = candleBody[1-w]
			}
			c.Put(i+w, j-1-k, col)
		}
	}
}

// tuft draws a young pine's foliage: a ball of needles r units long, dark in its heart and lit at the tips.
func tuft(c *art.Canvas, x, y, r float64, lk *look, salt int) {
	y = min(y, under-1.0)
	c.Blob(art.Blob{X: x, Y: y, RX: r * 0.3, RY: r * 0.28, Tones: lk.tones[:2], Jitter: 0.5, Salt: salt})
	needles(c, burst{
		x: x, y: y, length: r * 1.1, count: int(math.RoundToEven(2.6 * r)), colors: lk.tones[1:5], from: -3.5, to: 2.8,
		salt: salt, bias: 0.1, test: above(c, under),
	})
}

// drift draws old needles falling through the air, in the sky around the tree.
func drift(c *art.Canvas) {
	const salt = 21
	for n := range int(math.RoundToEven(7 * (0.5 + 0.5*c.Scale()))) {
		x, y := 2+44*art.Noise(n, salt, 0), 4+17*art.Noise(n, salt+1, 0)
		if c.Has(c.Cell(x, y)) {
			continue
		}
		reach := 0.8 / c.Scale() * (1 + 0.5*art.Noise(n, salt+2, 0))
		c.Line(x, y, x+reach, y+1.6/c.Scale(), oldNeedle[n%len(oldNeedle)], nil)
	}
}
