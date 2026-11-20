// Package maple draws a red Japanese maple (Acer palmatum) bonsai on the art kit.
//
// A seed in a nursery pot grows into a venerable specimen on a display stand. It is fine, dense, and fiery: tiers of
// palmate leaves from deep crimson through scarlet and orange to gold, on a smooth gray trunk with a buttressed
// nebari, in an ivory pot.
//
// The tree is a trunk, the gray limbs that carry its pads of foliage, and the ground it stands on. A pad appears at
// its stage and grows for a while, and the wire is on between two stages. The crown is tiers of pads with sky and
// limbs showing through windows between them and, from Ramified, a lacy edge of twigs that fills out with age. From
// Refined the windows open wide, each pad splits into a lit crest and a shadowed flank, and moss shows on the soil.
// Ancient adds a bleached shari up the trunk, a broad buttressed foot, and leaves drifting in the air. At LayeredPads
// the crown reaches out into low side tiers and its apex rises; at Established the lower pads lift to show the nebari
// and the stone; and at Mature they swing outward and every pad swells round a much thicker, mottled, buttressed
// trunk. Foliage is lit on its upper left; a larger scale adds leaf glyphs, bark streaks, finer twigs, and holes in
// the pads rather than bigger blocks.
package maple

import "github.com/KrishRVH/pixel-bonsai/art"

var (
	leaf     = art.Tones{0x5c0f2c, 0x8e1530, 0xb81c2e, 0xdc2f2a, 0xee5026, 0xf6802c, 0xfab238, 0xfee08a} // deep wine to pale gold
	spring   = art.Tones{0x2e5f2e, 0x4c8c38, 0x7fb648}                                                   // the green showing in young foliage
	bark     = art.Tones{0x34303a, 0x524c54, 0x75706e, 0x958f88, 0xb5ada0}                               // smooth gray
	twig     = art.Tones{0x34262a, 0x54403a, 0x7a5e4e}
	young    = art.Tones{0x6a3430, 0xa04a3c, 0xd06a52} // a seedling's red stem
	lichen   = art.Tones{0x9a9c88, 0xc8c8b4}
	scar     = art.Color(0x241818)
	patch    = art.Tones{0xa89f90, 0xc2baa8}                     // the pale mottling of old gray bark
	pot      = art.Tones{0xa89f88, 0xcfc6ac, 0xebe3cc, 0xfaf5e6} // ivory glaze
	soil     = art.Tones{0x2b1f19, 0x3d2c22, 0x57402f}
	moss     = art.Tones{0x2f5a2a, 0x4f8a38, 0x7cb450, 0xa8d878}
	grass    = art.Tones{0x3f6e28, 0x6e9c38, 0xa8cc58}
	stone    = art.Tones{0x3a4252, 0x56627a, 0x7c8aa4, 0xa4b0c8}
	deadwood = art.Tones{0xc4bba2, 0xece4cc, 0xfffaea} // bleached shari, near white
	stand    = art.Tones{0x3a281c, 0x5a3e2a, 0x7e5a3c, 0xa07a52}
	plastic  = art.Tones{0x2a2f3a, 0x454e60, 0x6a7690, 0x8a96b0}
	clay     = art.Tones{0x6a3e30, 0x8e5a44, 0xb4785a}
	wireTone = art.Color(0xc0773a)
	seed     = art.Color(0x6e4424)
	samara   = art.Color(0xcaa468) // the seed's wings
	label    = art.Color(0xe4e0d0)
)

// Variety returns the maple.
func Variety() art.Variety {
	return art.Variety{
		ID:        "maple",
		Name:      "maple",
		Icon:      "🍁",
		Signature: []art.Color{leaf[2], leaf[3]},
		Draw:      draw,
	}
}

// The trunk's foot on the soil, and its full height, in units.
const (
	baseX = 24.0
	baseY = 22.6
	full  = 15.0
)

// anchors is the trunk's line: its sideways offset at each share of its height.
var anchors = []struct{ share, dx float64 }{{0, 0}, {0.2, 1.6}, {0.4, -1.2}, {0.6, -2.6}, {0.8, -1.0}, {1, 1.5}}

var (
	height = []art.Step{
		art.At(art.Sprout, 3), art.At(art.Seedling, 6), art.At(art.Sapling, 9), art.At(art.Whip, 14),
		art.At(art.YoungTree, 14.5), art.At(art.FirstBranches, 15), art.At(art.Shaped, full),
	}
	girth = []art.Step{
		art.At(art.Sprout, 0.6), art.At(art.YoungTree, 1.7), art.At(art.FirstBranches, 2.0), art.At(art.FirstWiring, 2.4),
		art.At(art.ThickeningTrunk, 3.8), art.At(art.FirstStyling, 4.2), art.At(art.Shaped, 4.6), art.At(art.Ramified, 4.7),
		art.At(art.Mature, 7.8), art.At(art.Old, 8.2), art.At(art.Ancient, 9.2),
	}
	bend = []art.Step{art.At(art.YoungTree, 0.25), art.At(art.FirstWiring, 1.0)}
	// swell is the share every pad grows beyond its radius.
	swell = []art.Step{art.At(art.Mature, 0.1)}
)

// pads are the pads of foliage and the limbs that carry them: the parent limb (-1 for the trunk), the share along it
// where this one leaves, the pad's final center and radius, and the stage it appears at. A parent comes before the
// limbs it carries.
var pads = []struct {
	parent  int
	share   float64
	x, y, r float64
	born    art.Stage
}{
	{-1, 1.00, 24.5, 4.2, 4.2, art.YoungTree},
	{-1, 0.58, 33.5, 11.0, 4.4, art.YoungTree},
	{-1, 0.52, 15.5, 11.4, 4.4, art.YoungTree},
	{-1, 0.82, 18.0, 7.4, 4.4, art.FirstBranches},
	{-1, 0.78, 31.0, 7.0, 4.4, art.FirstBranches},
	{-1, 0.30, 17.0, 14.8, 3.8, art.FirstStyling},
	{-1, 0.25, 31.0, 15.2, 4.0, art.FirstStyling},
	{-1, 0.72, 24.0, 9.2, 3.6, art.Shaped},
	{2, 0.6, 9.5, 10.0, 4.4, art.LayeredPads},
	{1, 0.6, 39.0, 9.6, 4.4, art.LayeredPads},
	{2, 0.6, 8.0, 18.4, 4.0, art.LayeredPads},
	{1, 0.6, 40.5, 18.0, 4.0, art.LayeredPads},
}

// shifts move or swell a pad when a stage starts: its number in pads, the stage, and how far its center moves and
// its radius grows.
var shifts = []struct {
	pad        int
	at         art.Stage
	dx, dy, dr float64
}{
	{0, art.LayeredPads, 0.3, -0.9, 0.3},
	{3, art.LayeredPads, 0, -0.5, 0},
	{4, art.LayeredPads, 0, -0.5, 0},
	{5, art.Established, 0, -0.8, 0},
	{6, art.Established, 0, -0.8, 0},
	{5, art.Mature, -1.8, -0.4, 0},
	{6, art.Mature, 1.8, -0.4, 0},
	{8, art.Mature, -1.6, 0, 0.5},
	{9, art.Mature, 1.6, 0, 0.5},
	{10, art.Mature, -1.8, 0.4, 0.4},
	{11, art.Mature, 1.8, 0.4, 0.4},
}

// A limb is a grown limb and the pad it carries: the pad's number in pads, the limb's path from its parent to the
// pad's center, and the pad's center and radius at the tree's age.
type limb struct {
	pad     int
	path    art.Path
	x, y, r float64
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// draw paints the maple at age g.
func draw(c *art.Canvas, g float64) {
	if !art.From(g, art.Seedling) {
		pots(c, g)
		if art.From(g, art.Sprout) {
			shoot(c)
		}
		return
	}
	top, width := art.Steps(g, height), art.Steps(g, girth)
	path := trunkPath(top, art.Steps(g, bend))
	stem := !art.From(g, art.YoungTree)
	if stem {
		// A slender stem sits on one pixel column, not between two.
		for k := range path {
			path[k].X += 0.5
		}
	}
	limbs := grow(g, path)
	branches(c, g, limbs, width)
	trunk(c, g, path, width)
	wire(c, g, path, width, limbs)
	pots(c, g)
	ground(c, g)
	if stem {
		stemLeaves(c, g, path, top)
	}
	foliage(c, g, limbs)
	carve(c, g, limbs, width)
	if art.From(g, art.Ramified) {
		edge(c, g, limbs)
	}
	if art.From(g, art.Ancient) {
		n := 6
		if art.From(g, art.Venerable) {
			n = len(drift)
		}
		for _, d := range drift[:n] {
			singleLeaf(c, d.x, d.y, d.level, 0, d.kind)
		}
	}
}

// trunkPath returns the trunk's center line from its foot up to height top, leaning out by lean of its full curve.
func trunkPath(top, lean float64) art.Path {
	points := make([]art.Point, len(anchors))
	for k, a := range anchors {
		points[k] = art.Point{X: baseX + lean*a.dx, Y: baseY - top*a.share}
	}
	return art.Spline(points, 5)
}

// grow returns the limbs of the pads that have appeared, in the order of pads. Each runs from its parent to its pad's
// center as it stands at age g.
func grow(g float64, path art.Path) []limb {
	paths := make([]art.Path, len(pads))
	var limbs []limb
	for k, p := range pads {
		if !art.From(g, p.born) {
			continue
		}
		x, y, r := shifted(g, k)
		parent := path
		if p.parent >= 0 {
			parent = paths[p.parent]
		}
		a := parent.Along(p.share)
		grown := grownShare(g, p.born)
		end := art.Point{X: a.X + (x-a.X)*(0.8+0.2*grown), Y: a.Y + (y-a.Y)*(0.85+0.15*grown)}
		mid := art.Point{X: a.X + 0.55*(end.X-a.X), Y: a.Y + 0.35*(end.Y-a.Y) - 1.0}
		paths[k] = art.Spline([]art.Point{a, mid, end}, 4)
		limbs = append(limbs, limb{k, paths[k], end.X, end.Y, r * (0.75 + 0.25*grown) * (1 + art.Steps(g, swell))})
	}
	return limbs
}

// shifted returns pad k's center and radius, moved by the shifts of the stages g has reached.
func shifted(g float64, k int) (x, y, r float64) {
	x, y, r = pads[k].x, pads[k].y, pads[k].r
	for _, s := range shifts {
		if s.pad == k {
			t := art.Since(g, s.at, 0.004)
			x, y, r = x+s.dx*t, y+s.dy*t, r+s.dr*t
		}
	}
	return x, y, r
}

// grownShare returns how far a pad born at a stage has grown, from 0 to 1. A pad of the later tiers arrives nearly
// full, so its stage shows at once.
func grownShare(g float64, born art.Stage) float64 {
	if born < art.LayeredPads {
		return art.Since(g, born, 0.12)
	}
	return 0.5 + 0.5*art.Since(g, born, 0.04)
}
