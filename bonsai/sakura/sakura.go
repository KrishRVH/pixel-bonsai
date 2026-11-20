// Package sakura draws a cherry bonsai in blossom on the art kit.
//
// A reddish seedling in a nursery pot grows into an informal upright cherry with a gentle S trunk and a broad crown
// of blossom pads on a celadon pot, ending as a gnarled, hollow old tree under a drift of petals. From Shaped to
// ShowTree the crown widens and rises; tiers of pads stack up; and twigs, moss, a stone, bold bark, and a stand
// arrive in turn.
package sakura

import (
	"cmp"
	"math"
	"slices"

	"github.com/KrishRVH/pixel-bonsai/art"
)

var (
	bark      = art.Tones{0x362130, 0x4c2b3c, 0x683d4b, 0x8a5360, 0xb07b85} // glossy purplish red-brown
	gloss     = art.Color(0xdcaab1)                                         // a streak of sheen on the trunk's lit edge
	lenticel  = art.Tones{0xa99a8d, 0xcdbdac}                               // the pale horizontal dashes of cherry bark
	leaf      = art.Tones{0x4a3320, 0x6a5a2c, 0x7a8236, 0x8fa148, 0xa9bd6a} // fresh bronze-green leaves
	seedLeaf  = art.Tones{0x5f8a3c, 0x86b04f, 0xb4d46e}
	bloom     = art.Tones{0x6f3d68, 0x9a527f, 0xc9729b, 0xe895b6, 0xf6bcd1, 0xfddfe9, 0xffffff} // mauve to white
	center    = art.Tones{0xa8407a, 0xcf5f93}                                                   // a flower's deep pink heart
	bud       = art.Color(0xd94a86)
	pot       = art.Tones{0x4e7d76, 0x6fa597, 0x98c8b6, 0xc4e6d6, 0xe2f6ec} // celadon glaze
	soil      = art.Tones{0x231710, 0x33221a, 0x4a3425, 0x5f4632}
	seed      = art.Color(0xc9a46a)
	tag       = art.Color(0xe6d9c8)
	wireTone  = art.Tones{0xa85d2c, 0xd38a45}
	moss      = art.Tones{0x35602a, 0x568c35, 0x7fb348, 0xa6d46b}
	stoneTone = art.Tones{0x55555f, 0x7a7a86, 0xa0a0aa, 0xc4c4cc}
	shari     = art.Tones{0x7f7869, 0xa89f8c, 0xcfc6b2, 0xece4d2}
	stand     = art.Tones{0x3a2619, 0x5b3e29, 0x7d5a3a, 0xa07a52}
	plastic   = art.Tones{0x2e3140, 0x474b60, 0x6a7092, 0x8f97b8}
	dish      = art.Tones{0xa89c86, 0xd9cdb6, 0xf3ead8}
	hollow    = art.Tones{0x120a0e, 0x362130}

	// stem holds a young stem's tones as it reddens and then browns.
	stem = []art.Tones{{0xa64f5c, 0xd0707b, 0xf0a0a8}, {0x8a3f4a, 0xb8586a, 0xd87f8a}, bark[1:4]}
)

// Variety returns the sakura.
func Variety() art.Variety {
	return art.Variety{
		ID:        "sakura",
		Name:      "sakura",
		Icon:      "🌸",
		Signature: []art.Color{bloom[3], bloom[4], leaf[2], leaf[3]},
		Draw:      draw,
	}
}

// The trunk's foot on the nursery soil and on the bonsai pot's soil, and the height the crown rises from: the lower
// pads sit there and the top pads climb away from it.
const (
	footX    = 24
	nurseryY = 24.9
	soilY    = 22.8
	crownY   = 18.0
)

// anchors is the finished S trunk: its sideways shift from the foot at each share of its height.
var anchors = []struct{ dx, share float64 }{
	{0, 0},
	{0.8, 0.08},
	{0.2, 0.2},
	{-2.4, 0.32},
	{-4.2, 0.44},
	{-3.4, 0.56},
	{-0.8, 0.66},
	{2.4, 0.77},
	{3.4, 0.88},
	{1.6, 1.0},
}

var (
	height = []art.Step{
		art.At(art.Sprout, 3.2), art.At(art.Seedling, 6), art.At(art.Sapling, 9), art.At(art.Whip, 13), art.At(art.YoungTree, 14.5),
		art.At(art.FirstBranches, 15.2), art.At(art.FirstStyling, 16), art.At(art.Shaped, 16.6), art.At(art.Mature, 17),
	}
	bend  = []art.Step{art.At(art.Sapling, 0), art.At(art.YoungTree, 0.25), art.At(art.FirstWiring, 0.45), art.At(art.ThickeningTrunk, 1)}
	girth = []art.Step{
		art.At(art.Sprout, 0.9), art.At(art.Whip, 1.1), art.At(art.YoungTree, 1.8), art.At(art.FirstWiring, 2.4), art.At(art.ThickeningTrunk, 3.2),
		art.At(art.Shaped, 4.2), art.At(art.LayeredPads, 4.9), art.At(art.Ramified, 5.3), art.At(art.Refined, 5.6), art.At(art.Established, 6.0),
		art.At(art.Mature, 6.8), art.At(art.ShowTree, 6.9), art.At(art.Ancient, 7.4), art.At(art.Venerable, 7.7),
	}
	limbWidth = []art.Step{
		art.At(art.FirstBranches, 1.0), art.At(art.Shaped, 2.0), art.At(art.LayeredPads, 2.2), art.At(art.Ramified, 2.4), art.At(art.Refined, 2.6),
		art.At(art.Established, 2.9), art.At(art.Mature, 3.3), art.At(art.ShowTree, 3.5), art.At(art.Ancient, 3.9),
	}
	reach = []art.Step{
		art.At(art.FirstBranches, 0.55), art.At(art.FirstStyling, 0.7), art.At(art.Shaped, 0.84), art.At(art.LayeredPads, 0.9),
		art.At(art.Ramified, 0.94), art.At(art.Refined, 0.97), art.At(art.Established, 1.0), art.At(art.Mature, 1.015), art.At(art.ShowTree, 1.03),
		art.At(art.Venerable, 1.07),
	}
	// statureSteps is how much of its full height the crown has above the lower pads.
	statureSteps = []art.Step{
		art.At(art.Shaped, 0.9), art.At(art.LayeredPads, 0.92), art.At(art.Ramified, 0.94), art.At(art.Refined, 0.96),
		art.At(art.Established, 0.98), art.At(art.Mature, 0.99), art.At(art.ShowTree, 1.0),
	}
	droop   = []art.Step{art.At(art.Shaped, 0), art.At(art.Mature, 0.5), art.At(art.Venerable, 1.0)}
	padSize = []art.Step{
		art.At(art.FirstBranches, 0.5), art.At(art.FirstWiring, 0.55), art.At(art.FirstStyling, 0.64), art.At(art.Shaped, 0.9),
		art.At(art.LayeredPads, 0.915), art.At(art.Ramified, 0.93), art.At(art.Refined, 0.94), art.At(art.Established, 0.95),
		art.At(art.Mature, 0.957), art.At(art.ShowTree, 0.965), art.At(art.Venerable, 1.0),
	}
	// capSize is how much of its full size the pad on the apex has.
	capSize = []art.Step{art.At(art.LayeredPads, 0.7), art.At(art.ShowTree, 1.0)}
)

// limbs are the main branches: the trunk share each leaves at, its pad's center and radii when finished, how high
// it arches, and the stage it grows at.
var limbs = []struct {
	share  float64
	pad    art.Point
	rx, ry float64
	rise   float64
	born   art.Stage
}{
	{0.24, art.Point{X: 8.5, Y: 16.8}, 6.2, 2.9, 2.2, art.FirstBranches},
	{0.56, art.Point{X: 39.0, Y: 15.3}, 6.4, 3.0, 2.0, art.FirstBranches},
	{0.66, art.Point{X: 11.2, Y: 8.6}, 6.8, 3.2, 1.8, art.FirstBranches},
	{0.78, art.Point{X: 38.7, Y: 7.2}, 6.6, 3.1, 1.6, art.FirstStyling},
	{0.28, art.Point{X: 31.5, Y: 19.8}, 3.0, 1.6, 1.0, art.Shaped},
}

// layers are pads stacked on the upper limbs' pads: the limb, the shift in that pad's radii, the share of its radii,
// and the stage it grows at.
var layers = []struct {
	limb   int
	ox, oy float64
	sx, sy float64
	born   art.Stage
}{
	{2, -0.25, -1.4, 0.88, 0.82, art.LayeredPads},
	{3, 0.25, -1.4, 0.88, 0.82, art.LayeredPads},
}

// isTier reports whether a pad salt names a pad that caps others: the two over the upper pads and the one over the
// apex. Tiers draw last, over the pads they cap.
func isTier(salt int) bool {
	return salt >= 30 && salt <= 32
}

// A pad is a crown pad: its salt, which keeps its blossoms in place as other pads join it, and its ellipse.
type pad struct {
	salt         int
	x, y, rx, ry float64
}

// A limb is a branch path and its widths at the trunk and at the tip.
type limb struct {
	path   art.Path
	w0, w1 float64
}

// stature returns how much of its full height the crown has: 1 before Shaped and from ShowTree on.
func stature(g float64) float64 {
	if art.From(g, art.Shaped) {
		return art.Lerp(g, statureSteps)
	}
	return 1
}

func trunkPath(g float64, foot art.Point) art.Path {
	top, b := art.Lerp(g, height), art.Lerp(g, bend)
	points := make([]art.Point, len(anchors))
	for k, a := range anchors {
		points[k] = art.Point{X: foot.X + a.dx*b, Y: foot.Y - top*a.share}
	}
	return art.Spline(points, 5)
}

// rich returns, at larger scales, a finer run of tones with a blend between each neighbouring pair.
func rich(c *art.Canvas, tones art.Tones) art.Tones {
	if c.Scale() < 1.5 {
		return tones
	}
	out := art.Tones{tones[0]}
	for k := 1; k < len(tones); k++ {
		out = append(out, art.Mix(tones[k-1], tones[k], 0.5), tones[k])
	}
	return out
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// draw paints the sakura at age g.
func draw(c *art.Canvas, g float64) {
	young := !art.From(g, art.ThickeningTrunk)
	foot := art.Point{X: footX, Y: soilY}
	if young {
		foot.Y = nurseryY
		nursery(c, g)
	} else {
		bonsaiPot(c, g)
	}
	if !art.From(g, art.Sprout) {
		return
	}
	if !art.From(g, art.YoungTree) {
		stemPlant(c, g, foot)
		return
	}
	trunk, width := trunkPath(g, foot), art.Lerp(g, girth)
	branches, pads := crown(g, trunk)
	for k, b := range branches {
		c.Stroke(art.Stroke{Path: b.path, W0: b.w0, W1: b.w1, Tones: rich(c, bark[:4]), Salt: k})
	}
	cover := c.Stroke(art.Stroke{Path: trunk, W0: width, W1: width * 0.42, Tones: rich(c, bark)})
	if art.During(g, art.FirstWiring, art.ThickeningTrunk) {
		wire(c, limb{trunk, width, width * 0.42})
		for _, b := range branches {
			wire(c, b)
		}
	}
	barkDetail(c, g, trunk, width, cover)
	if !young {
		potBody(c)
		if art.From(g, art.Established) {
			nebari(c, g)
		} else {
			c.Blob(art.Blob{X: footX + 0.3, Y: 22.9, RX: 2.8, RY: 0.9, Tones: bark, Lump: 0.1, Where: aboveRim})
		}
	}
	dress(c, g, branches, pads)
}

// dress draws what grows on the branches and, with age, around the tree: branchlets, twigs, and blossom, then the
// stone, the accent planting, and falling petals.
func dress(c *art.Canvas, g float64, branches []limb, pads []pad) {
	branchlets(c, g, branches)
	for _, p := range pads {
		twigs(c, g, p)
	}
	for _, p := range pads {
		blossomPad(c, g, p)
	}
	for _, p := range pads {
		sprigs(c, g, p)
		fringe(c, g, p)
	}
	if art.From(g, art.Established) {
		stone(c)
	}
	if art.From(g, art.ShowTree) {
		accent(c)
	}
	if art.From(g, art.Ancient) {
		petals(c, g)
	}
}

// crown returns the branches and pads of the tree at age g. Each pad keeps its salt, so it keeps its blossoms as
// others join it.
func crown(g float64, trunk art.Path) ([]limb, []pad) {
	final := trunkPath(1, art.Point{X: footX, Y: soilY})
	limbW, stretch, sag, size := art.Lerp(g, limbWidth), art.Lerp(g, reach), art.Lerp(g, droop), art.Lerp(g, padSize)
	tall := stature(g)
	set := art.Since(g, art.FirstStyling, 0.05)
	var branches []limb
	var pads []pad
	made := map[int]pad{}
	for k, l := range limbs {
		if !art.From(g, l.born) {
			continue
		}
		at := int(math.RoundToEven(l.share * float64(len(trunk)-1)))
		a, f := trunk[at], final[at]
		dx := (l.pad.X - f.X) * stretch
		dy := (l.pad.Y-f.Y)*set - math.Abs(l.pad.X-f.X)*0.55*stretch*(1-set)
		dy = crownY - (crownY-(a.Y+dy))*tall - a.Y
		end := art.Point{X: a.X + dx, Y: a.Y + dy + sag + l.ry*0.3*size}
		arch := []art.Point{a, {X: a.X + 0.3*dx, Y: a.Y + 0.3*dy - l.rise*0.6}, {X: a.X + 0.65*dx, Y: a.Y + 0.65*dy - l.rise}, end}
		branches = append(branches, limb{art.Spline(arch, 4), limbW, limbW * 0.4})
		made[k] = pad{10 + k, a.X + dx, a.Y + dy + sag, l.rx * size, l.ry * size}
		pads = append(pads, made[k])
	}
	for k, l := range layers {
		if base, ok := made[l.limb]; ok && art.From(g, l.born) {
			pads = append(pads, pad{30 + k, base.x + l.ox*base.rx, base.y + l.oy*base.ry, base.rx * l.sx, base.ry * l.sy})
		}
	}
	if art.During(g, art.YoungTree, art.FirstBranches) {
		for _, side := range []float64{-1, 1} {
			a := trunk.Along(0.55 + 0.1*side)
			pads = append(pads, pad{20 + int(side), a.X + side*2.6, a.Y - 1.6, 1.8, 1.0})
			path := art.Path{a, {X: a.X + side*1.5, Y: a.Y - 0.8}, {X: a.X + side*2.6, Y: a.Y - 1.2}}
			branches = append(branches, limb{path, 0.8, 0.5})
		}
	}
	tip := trunk[len(trunk)-1]
	top := crownY - (crownY-tip.Y+1.0)*tall
	pads = append(pads, pad{40, tip.X, top, 5.9 * size, 3.4 * size})
	if art.From(g, art.LayeredPads) {
		cp := art.Lerp(g, capSize) * size
		pads = append(pads, pad{32, tip.X - 0.6, top - 2.5*cp, 4.2 * cp, 2.0 * cp})
	}
	if art.From(g, art.Venerable) {
		// A pad hanging in full bloom from a twig low on the left.
		from, to := art.Point{X: 8.0, Y: 18.6}, art.Point{X: 4.6, Y: 22.0}
		mid := art.Point{X: from.X + (to.X-from.X)*0.4, Y: from.Y + (to.Y-from.Y)*0.7}
		branches = append(branches, limb{art.Path{from, mid, to}, 0.8, 0.5})
		pads = append(pads, pad{41, to.X, to.Y, 3.0, 1.6})
	}
	// Pads draw from the top down, and the tiers last, over the pads they cap.
	slices.SortStableFunc(pads, func(a, b pad) int {
		return cmp.Or(cmp.Compare(b2i(isTier(a.salt)), b2i(isTier(b.salt))), cmp.Compare(a.y, b.y))
	})
	if !art.From(g, art.Shaped) {
		// Before Shaped a pad's blossoms follow its place in the crown.
		for n := range pads {
			pads[n].salt = 10 + n
		}
	}
	return branches, pads
}
