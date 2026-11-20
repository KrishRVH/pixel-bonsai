// Package pine draws a Japanese black pine bonsai (Pinus thunbergii) on the art kit: flat tiers of dark needle pads
// on a plated, near-black trunk with rust in its cracks, silver candles standing on the pad tops, in a deep unglazed
// clay pot.
//
// A winged seed in a nursery pot sprouts a whorl of seed needles, then a stem with dark fascicles and a candle at its
// tip. The young tree's tufts of needles are wired and styled into tiers of pads; from Ramified the pads break into
// domed sub-pads with bare twigs below, and moss, a stone, nebari, and a stand arrive in turn. From Old bleached jin
// stand out of the pads and the trunk, and from Ancient a vein of shari runs up the trunk under lichen and falling
// needles.
package pine

import (
	"slices"

	"github.com/KrishRVH/pixel-bonsai/art"
)

var (
	needle     = art.Tones{0x15323a, 0x1c4440, 0x26574a, 0x336a55, 0x4a8967, 0x7fb47c} // shadow to lit tips
	midNeedle  = art.Color(0x3e7a5e)                                                   // halfway from needle[3] to needle[4]
	paleTip    = art.Color(0x9ccf8f)                                                   // the palest tips of a venerable pad
	sprout     = art.Tones{0x5f9a62, 0x8cc476, 0xbfe29a}                               // seed needles and fresh growth
	stem       = art.Tones{0x6f6a42, 0x9a9358, 0xc2b878}                               // a young stem
	olive      = art.Tones{0x5d5a3a, 0x857e4e, 0xaaa268}                               // the same, going woody
	candleBody = art.Tones{0xb4c1b2, 0xdbe4d5}                                         // a candle, shadow to lit
	candleTip  = art.Color(0xc6a468)
	bark       = art.Tones{0x251e1f, 0x342c2c, 0x463b39, 0x5d5046, 0x85756a} // fissure shadow, plates to lit edge
	rust       = art.Tones{0x6a3a2a, 0x8a4c30}
	oldNeedle  = art.Tones{0x7a5a34, 0xa07a45}
	dead       = art.Tones{0x8a8472, 0xbdb7a4, 0xe9e4d3, 0xfbf8ee} // deadwood, shadow to bleached
	lichen     = art.Tones{0x9fb87a, 0xc4d29a}
	pot        = art.Tones{0x33251f, 0x523c2f, 0x7a5c47, 0xa58565} // unglazed clay: shadow, body, lit rim
	plastic    = art.Tones{0x2a303c, 0x3d4655, 0x5a6578}
	soil       = art.Tones{0x2b2118, 0x3d3022, 0x574429}
	moss       = art.Tones{0x2c5428, 0x3f7430, 0x5a963e, 0x7fb552}
	stoneTone  = art.Tones{0x4d4d55, 0x6e6e76, 0x9a98a0, 0xc0bdc2}
	stand      = art.Tones{0x4a3322, 0x6b4a32, 0x8f6a4a, 0xb08a62}
	slate      = art.Tones{0x3d5a6e, 0x5d8199, 0x8fb0c4}
	grass      = art.Tones{0x4f7d3a, 0x78a84a, 0xa8cf72, 0xd7e8a0}
	flower     = art.Tones{0xe8c76a, 0xfbf0b8} // the accent's flowers
	wireTone   = art.Tones{0xa85d2a, 0xd58a46, 0xf0b472}
	seed       = art.Tones{0x5c3d22, 0x8a6538}
	wing       = art.Color(0xc9b890) // a seed's papery wing

	// padTones holds a pad's tones, shadow to lit, from each stage on: more of them as it fills out.
	padTones = []struct {
		from  art.Stage
		tones art.Tones
	}{
		{art.YoungTree, needle[:5]},
		{art.Ramified, art.Tones{needle[0], needle[1], needle[2], needle[3], midNeedle, needle[4]}},
		{art.Refined, art.Tones{needle[0], needle[1], needle[2], needle[3], midNeedle, needle[4], needle[5]}},
		{art.Venerable, art.Tones{needle[0], needle[1], needle[2], needle[3], midNeedle, needle[4], needle[5], paleTip}},
	}
)

// Variety returns the black pine.
func Variety() art.Variety {
	return art.Variety{
		ID:        "black-pine",
		Name:      "black pine",
		Icon:      "🌲",
		Signature: []art.Color{needle[1], needle[2], needle[3]},
		Draw:      draw,
	}
}

// isNeedle reports whether a color is one of a pad's tones, so holes and candles find the pad under them.
func isNeedle(col art.Color) bool {
	return slices.Contains(needle, col) || col == midNeedle || col == paleTip
}

// The soil line at the pot's top; the trunk's foot in the soil, its axis there, and its full height; and the lowest y
// a pad reaches, which leaves two pixels of air over the soil.
const (
	soilY      = 21.0
	baseY      = 22.0
	axisX      = 24.0
	fullHeight = 19.8
	under      = soilY - 2.0
)

// sway is the trunk's informal S, which leans back over its foot: its sideways shift at each share of its height.
var sway = []struct{ share, dx float64 }{
	{0.0, 0.0}, {0.1, -0.8}, {0.22, -2.0}, {0.4, -0.3}, {0.55, 2.0}, {0.68, 2.4}, {0.8, 0.8}, {0.9, -0.4}, {1.0, 0.3},
}

var (
	height = []art.Step{
		art.At(art.Sprout, 3.0), art.At(art.Seedling, 6.0), art.At(art.Sapling, 9.0), art.At(art.Whip, 14.0),
		art.At(art.YoungTree, 15.5), art.At(art.FirstBranches, 16.8), art.At(art.ThickeningTrunk, 17.8),
		art.At(art.FirstStyling, 18.8), art.At(art.Shaped, fullHeight),
	}
	girth = []art.Step{
		art.At(art.Sprout, 0.7), art.At(art.Whip, 1.2), art.At(art.YoungTree, 1.9), art.At(art.FirstBranches, 2.3),
		art.At(art.ThickeningTrunk, 4.0), art.At(art.Shaped, 5.0), art.At(art.LayeredPads, 5.5), art.At(art.Ramified, 6.0),
		art.At(art.Refined, 6.5), art.At(art.Established, 7.0), art.At(art.Mature, 7.8), art.At(art.ShowTree, 8.2),
		art.At(art.Ancient, 8.8),
	}
	taper = []art.Step{
		art.At(art.Sprout, 0.9), art.At(art.YoungTree, 0.6), art.At(art.ThickeningTrunk, 0.42), art.At(art.Shaped, 0.4),
	}
	bend = []art.Step{
		art.At(art.YoungTree, 0.15), art.At(art.FirstWiring, 0.6), art.At(art.ThickeningTrunk, 0.85),
		art.At(art.FirstStyling, 1.0),
	}
	roughness = []art.Step{art.At(art.ThickeningTrunk, 0.25), art.At(art.Shaped, 0.5), art.At(art.Mature, 1.0)}
	reach     = []art.Step{
		art.At(art.YoungTree, 0.3), art.At(art.FirstBranches, 0.45), art.At(art.ThickeningTrunk, 0.6),
		art.At(art.FirstStyling, 0.8), art.At(art.Shaped, 1.0),
	}
	padSize = []art.Step{
		art.At(art.YoungTree, 0.22), art.At(art.FirstBranches, 0.27), art.At(art.ThickeningTrunk, 0.34),
		art.At(art.FirstStyling, 0.62), art.At(art.Shaped, 0.98), art.At(art.LayeredPads, 0.99), art.At(art.Ramified, 1.0),
		art.At(art.Refined, 1.01), art.At(art.Mature, 1.02), art.At(art.Venerable, 1.04),
	}
	gaps      = []art.Step{art.At(art.Refined, 0.3), art.At(art.Mature, 0.45)}
	childSize = []art.Step{
		art.At(art.LayeredPads, 0.5), art.At(art.Ramified, 0.62), art.At(art.Refined, 0.74), art.At(art.Established, 0.8),
		art.At(art.Mature, 0.88),
	}
	// lowest is how much wider than the rest the lowest tier's pads are.
	lowest = []art.Step{art.At(art.Established, 0.1), art.At(art.ShowTree, 0.12)}
	fringe = []art.Step{art.At(art.Ramified, 0.5), art.At(art.Refined, 0.8), art.At(art.Mature, 1.0)}
)

// A look is the tree's features at an age, each keyed to a stage.
type look struct {
	top, base, taper float64   // the trunk's height, its width at the foot, and the share of that left at the top
	widths           []float64 // the trunk's width at its foot, a third and two thirds up, and its top
	bend             float64   // how far the trunk sways into its S
	rough            float64   // how plated and fissured the bark is, 0 smooth to 1 old
	lift             float64   // how much the bark plates stand out
	flare            float64   // how tall the trunk's flare over the soil is
	spread           float64   // how far the nebari reaches
	reach            float64   // how far out from the trunk the pads stand
	size             float64   // the pads' size
	clumps           float64   // the sub-pads' width in pixels at scale 1; 0 for a whole pad
	notch            float64   // how deep the seams between sub-pads are
	thick            float64   // how tall the pads' domes are
	child            float64   // the size of a pad that grows from another
	lowest           float64   // how much wider the lowest tier is
	gaps             float64   // how many holes open through the pads
	lit              float64   // how lit the pads are
	fringe           float64   // how ragged the pads' tops and ends are
	tones            art.Tones // a pad's tones, shadow to lit
	body, tips       art.Tones // the tones of a pad's mass and of its needle tips
	glow             float64   // how lit the needle tips are
}

func newLook(g float64) *look {
	lk := &look{
		top:    art.Steps(g, height),
		base:   art.Steps(g, girth),
		taper:  art.Steps(g, taper),
		bend:   art.Steps(g, bend),
		rough:  art.Steps(g, roughness),
		flare:  1.3 + art.Steps(g, []art.Step{art.At(art.Established, 0.7), art.At(art.Mature, 1.2)}),
		spread: 1 + art.Steps(g, []art.Step{art.At(art.Mature, 0.14)}),
		reach:  art.Steps(g, reach),
		size:   art.Steps(g, padSize),
		clumps: art.Steps(g, []art.Step{art.At(art.Ramified, 3.4)}),
		notch:  art.Steps(g, []art.Step{art.At(art.Ramified, 2.2)}),
		thick:  1 + art.Steps(g, []art.Step{art.At(art.LayeredPads, 0.3)}),
		child:  art.Steps(g, childSize),
		lowest: art.Steps(g, lowest),
		gaps:   art.Steps(g, gaps),
		lit:    art.Steps(g, []art.Step{art.At(art.Refined, 0.16)}),
		fringe: art.Steps(g, fringe),
		glow:   -0.12 + art.Steps(g, []art.Step{art.At(art.Venerable, 0.06)}),
		tones:  padTones[0].tones,
	}
	for _, f := range []float64{0, 1.0 / 3, 2.0 / 3, 1} {
		lk.widths = append(lk.widths, trunkWidth(lk, f))
	}
	// The plates of an old bark stand out.
	lk.lift = 0.15*(1-lk.rough) + art.Steps(g, []art.Step{art.At(art.Mature, 0.13)})
	for _, t := range padTones {
		if art.From(g, t.from) {
			lk.tones = t.tones
		}
	}
	lk.body, lk.tips = lk.tones[1:5], lk.tones[2:]
	return lk
}

// behind returns the look of a pad behind the trunk, in the shade of the pads in front.
func (lk *look) behind() *look {
	back := *lk
	back.body, back.tips, back.glow = lk.tones[0:4], lk.tones[1:4], lk.glow-0.1
	return &back
}

func b2f(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// sign returns 1 when b is true and -1 when it is not: the side a pad or a branch is on.
func sign(b bool) float64 {
	if b {
		return 1
	}
	return -1
}

func draw(c *art.Canvas, g float64) {
	lk := newLook(g)
	ground(c, g)
	switch {
	case art.From(g, art.YoungTree):
		tree(c, g, lk)
	case art.From(g, art.Sprout):
		stemPlant(c, g, lk)
	}
}

// tree draws the young tree on: its trunk, branches, pads, and what grows on them.
func tree(c *art.Canvas, g float64, lk *look) {
	trunk := make(art.Path, 19)
	for k := range trunk {
		trunk[k] = trunkAt(lk.top*float64(k)/18, lk.bend)
	}
	placed := place(g, lk)
	young := !art.From(g, art.FirstStyling)
	limbs := branches(c, lk, placed, young)
	backPads(c, lk, placed)
	rootFlare(c, g, lk)
	style := barkStyle{rough: lk.rough, lift: lk.lift, floor: baseY}
	if art.From(g, art.Ancient) {
		style.bared = shari
	}
	barkStroke(c, trunk, lk.widths, style)
	if art.During(g, art.FirstWiring, art.ThickeningTrunk) {
		wire(c, trunk, lk.base, lk.base*lk.taper)
		for _, limb := range limbs {
			wire(c, limb, 0.9, 0.6)
		}
	}
	if young {
		tufts(c, lk, placed)
	} else {
		pads(c, g, lk, placed)
	}
	if art.From(g, art.Mature) {
		// From here the trunk stands in front of the pads, so its width and plates show.
		style.ceiling = crownPlan.base + 0.6
		front := barkStroke(c, trunk, lk.widths, style)
		if art.From(g, art.Ancient) {
			bare := func(i, j int, s float64) bool { return s > 0.3 && c.Has(i, j) }
			c.ScatterCover(front, art.Scatter{Colors: lichen, Density: 0.05, Salt: 4, Test: bare})
		}
	}
	if art.From(g, art.Established) {
		stone(c)
		nebari(c, g, lk)
	}
	deadwood(c, g, placed, lk)
	if art.From(g, art.Ancient) {
		drift(c)
	}
}

// branches draws the branches from the trunk out to its pads, before the pads, and returns their paths. A young
// tree's branch runs into the middle of its tuft; a styled one ends under its pad's inner end.
func branches(c *art.Canvas, lk *look, placed []pad, young bool) []art.Path {
	var out []art.Path
	for k, p := range placed {
		if p.crown || p.plan.parent != fromTrunk || p.plan.back {
			continue
		}
		end := art.Point{X: p.x - sign(p.x > p.ax)*p.hw*0.7, Y: p.base - 0.8}
		if young {
			end = art.Point{X: p.x, Y: p.base - 0.3*p.hw}
		}
		mid := art.Point{X: (p.ax + end.X) / 2, Y: max(p.ay, end.Y) + 0.2}
		path := art.Spline([]art.Point{{X: p.ax, Y: p.ay}, mid, end}, 4)
		c.Stroke(art.Stroke{Path: path, W0: 0.18*lk.base + 0.4, W1: 0.6, Tones: bark[1:], Salt: k})
		out = append(out, path)
	}
	return out
}

// backPads draws the pads that stand behind the trunk, in the shade of the pads in front.
func backPads(c *art.Canvas, lk *look, placed []pad) {
	back := lk.behind()
	for k, p := range placed {
		if p.plan.back {
			needlePad(c, p.with(2.0, 0), back, 10*k)
		}
	}
}

// tufts draws a young tree's foliage: a tuft of needles where each pad will be, and a larger one at the top.
func tufts(c *art.Canvas, lk *look, placed []pad) {
	for k, p := range placed {
		if p.crown {
			tuft(c, p.ax, p.ay-0.5, max(p.hw, 3.0), lk, 10*k)
		} else {
			tuft(c, p.x, p.base-0.3*p.hw, p.hw, lk, 10*k)
		}
	}
}

// pads draws a styled tree's pads in front of the trunk, with from Ramified their twigs and the small pads perched on
// them, and the candles on their tops.
func pads(c *art.Canvas, g float64, lk *look, placed []pad) {
	for k, p := range placed {
		if p.plan.back {
			continue
		}
		shape := p.with(2.4, 0)
		if p.crown {
			shape = p.with(1.5, 0.8)
		}
		tops := needlePad(c, shape, lk, 10*k)
		if art.From(g, art.Ramified) && !p.crown {
			twigs(c, p, 10*k)
			if p.plan.parent == fromTrunk {
				puff(c, p, lk, 10*k+5)
			}
		}
		candles(c, g, p, tops)
	}
}
