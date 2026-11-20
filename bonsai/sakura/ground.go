package sakura

import "github.com/KrishRVH/pixel-bonsai/art"

// aboveRim clips what sits on the soil, such as moss, a stone, or roots, to above the pot's rim.
func aboveRim(_, y float64) bool {
	return y < 23.3
}

// nursery draws the plastic nursery pot with its label, and the seed on the bare soil.
func nursery(c *art.Canvas, g float64) {
	c.Block(art.Block{X0: 19, Y0: 24.3, X1: 29, Y1: 25.2, Tones: soil, Rounding: 0.5, Jitter: 1})
	c.Block(art.Block{X0: 20.6, Y0: 25.6, X1: 27.4, Y1: 29.6, Tones: plastic, Taper: 0.1})
	c.Block(art.Block{X0: 18.4, Y0: 24.8, X1: 29.6, Y1: 26.0, Tones: plastic[1:], Rounding: 0.3})
	if c.Scale() >= 1.5 {
		for _, x := range []float64{22, 24, 26} {
			c.Line(x, 26.4, x-0.3, 29.3, plastic[0], nil)
		}
	}
	c.Line(20, 23.2, 20.4, 24.6, stand[2], nil)
	c.Block(art.Block{X0: 19.2, Y0: 22.2, X1: 21.4, Y1: 23.2, Tones: art.Tones{tag}})
	if art.During(g, art.Seed, art.Sprout) {
		c.Blob(art.Blob{X: 24, Y: 24.7, RX: 1.1, RY: 0.65, Tones: art.Tones{seed}})
		c.Line(25.4, 24.9, 27.4, 25.1, soil[0], nil)
	}
}

// bonsaiPot draws the display stand from ShowTree, the soil, moss, and the pot.
func bonsaiPot(c *art.Canvas, g float64) {
	if art.From(g, art.ShowTree) {
		c.Block(art.Block{X0: 7.5, Y0: 27.6, X1: 46.5, Y1: 29.1, Tones: stand, Rounding: 0.2})
		c.Block(art.Block{X0: 10, Y0: 29.1, X1: 13.5, Y1: 30, Tones: stand[:2]})
		c.Block(art.Block{X0: 40.5, Y0: 29.1, X1: 44, Y1: 30, Tones: stand[:2]})
	}
	c.Block(art.Block{X0: 11.8, Y0: 22.2, X1: 36.2, Y1: 23.6, Tones: soil, Rounding: 0.6, Jitter: 1})
	potBody(c)
	grown := art.Steps(g, []art.Step{art.At(art.Refined, 1.0), art.At(art.Established, 1.5)})
	if grown == 0 {
		return
	}
	for k, patch := range []struct{ x, r float64 }{{13.5, 2.2}, {18.5, 1.6}, {28.5, 1.8}, {31.5, 1.6}, {34.5, 2.2}} {
		c.Blob(art.Blob{X: patch.x, Y: 22.5, RX: patch.r * (0.6 + 0.6*grown), RY: 0.7 + 0.8*grown, Tones: moss, Salt: k, Where: aboveRim})
	}
	depth := art.Steps(g, []art.Step{art.At(art.Refined, 0.95), art.At(art.Established, 1.6), art.At(art.Mature, 1.9)})
	c.Blob(art.Blob{X: 24, Y: 22.7, RX: 11.5, RY: depth, Tones: moss, Jitter: 1, Where: aboveRim})
}

// potBody draws the pot's feet, body, and rim. It draws again over the trunk's foot, which goes into the soil behind
// the lip.
func potBody(c *art.Canvas) {
	for _, x := range []float64{15, 30} {
		c.Block(art.Block{X0: x, Y0: 27.2, X1: x + 3, Y1: 28.0, Tones: pot[:2]})
	}
	body := c.Block(art.Block{X0: 12.6, Y0: 24.2, X1: 35.4, Y1: 27.4, Tones: pot, Taper: 1.6, Rounding: 0.55})
	glaze(c, body, 12.6, 35.4, 24.2, 27.4, pot[:4])
	rim := c.Block(art.Block{X0: 10.8, Y0: 23.2, X1: 37.2, Y1: 24.8, Tones: pot[1:], Rounding: 0.7})
	glaze(c, rim, 10.8, 37.2, 23.2, 24.8, pot[1:])
}

// glaze repaints a pot's cover as glaze: lit at the left, shaded to the right and at the foot, with a soft sheen.
func glaze(c *art.Canvas, cover *art.Cover, xl, xr, y0, y1 float64, tones art.Tones) {
	tones = rich(c, tones)
	for _, cell := range cover.Cells() {
		x, y := c.Center(cell.I, cell.J)
		down := (y - y0) / (y1 - y0)
		shade := 0.95 - 0.8*(x-xl)/(xr-xl) - 0.25*down
		shade += 0.25 * max(0, 1-abs(x-xl-0.22*(xr-xl))/2.5) * (1 - down)
		dither := (float64(cell.I%2+2*(cell.J%2)) + 0.5) / 4
		c.Put(cell.I, cell.J, tones.Shade(art.Clamp(shade), 0.18, dither))
	}
}

func abs(v float64) float64 {
	return max(v, -v)
}

// nebari draws the roots flaring over the soil from the trunk's foot, thicker from Mature.
func nebari(c *art.Canvas, g float64) {
	fat := 1 + art.Steps(g, []art.Step{art.At(art.Mature, 0.25)})
	c.Blob(art.Blob{X: footX + 0.4, Y: 22.7, RX: 6.4 * fat, RY: 1.7 * fat, Tones: bark, Lump: 0.1, Where: aboveRim})
	for _, root := range []struct{ side, reach, lift float64 }{{-1, 10.5, 0}, {-1, 6.8, 1.0}, {1, 10.0, 0}, {1, 6.4, 1.0}} {
		path := art.Path{
			{X: footX + root.side*1.5, Y: 21.0 - root.lift},
			{X: footX + root.side*root.reach*0.5, Y: 22.1},
			{X: footX + root.side*root.reach, Y: 22.8},
		}
		c.Stroke(art.Stroke{Path: path, W0: 2.8 * fat, W1: 0.9, Tones: bark[1:], Salt: int(root.side)})
	}
}

// stone draws a big mossy stone on the soil in front of the tree's left side, with a crack and moss on its top.
func stone(c *art.Canvas) {
	inside := func(x, y float64) bool {
		dx, dy := (x-15.2)/4.6, (y-21.2)/3.7
		return dx*dx+dy*dy < 1 && y < 23.3
	}
	c.Blob(art.Blob{X: 15.2, Y: 21.2, RX: 4.6, RY: 3.7, Tones: stoneTone, Lump: 0.12, Salt: 5, Where: aboveRim})
	c.Line(14.2, 20.4, 15.6, 22.0, stoneTone[0], c.Has)
	c.Blob(art.Blob{X: 14.4, Y: 18.3, RX: 4.0, RY: 1.6, Tones: moss, Lump: 0.2, Salt: 6, Where: inside})
	c.Blob(art.Blob{X: 19.4, Y: 22.9, RX: 1.5, RY: 0.9, Tones: moss, Salt: 7, Where: aboveRim})
}

// accent draws a small dish planting beside the pot: a tuft of grass with three flowers on a mossy mound.
func accent(c *art.Canvas) {
	c.Block(art.Block{X0: 40.6, Y0: 25.6, X1: 46.4, Y1: 27.6, Tones: dish, Taper: 0.6, Rounding: 0.6})
	c.Blob(art.Blob{X: 43.5, Y: 25.4, RX: 2.9, RY: 1.0, Tones: moss})
	for _, blade := range []struct{ x, top, lean float64 }{{41.8, 21.8, -0.8}, {42.8, 20.2, -0.2}, {43.9, 19.4, 0.2}, {45.0, 20.6, 0.9}, {44.4, 22.6, 0.4}} {
		c.Line(blade.x, 25.2, blade.x+blade.lean, blade.top, moss[2+b2i(blade.x > 43.5)], nil)
	}
	for _, flower := range []struct {
		x, y  float64
		petal art.Color
	}{{41.0, 21.0, bloom[5]}, {43.9, 18.6, bloom[4]}, {45.9, 19.8, bloom[5]}} {
		i, j := c.Cell(flower.x, flower.y)
		floret(c, i, j, flower.petal, center[1])
	}
}

// floret draws a five-dot flower: four petals round a heart.
func floret(c *art.Canvas, i, j int, petal, heart art.Color) {
	c.Put(i-1, j, petal)
	c.Put(i+1, j, petal)
	c.Put(i, j-1, petal)
	c.Put(i, j+1, petal)
	c.Put(i, j, heart)
}

// stemPlant draws the sprout, seedling, sapling, and whip: a thin stem with seed leaves, then true leaves.
func stemPlant(c *art.Canvas, g float64, foot art.Point) {
	top := art.Lerp(g, height)
	x0, y0 := foot.X+0.5, foot.Y
	hook := art.During(g, art.Sprout, art.Seedling)
	lean := 0.3
	if hook {
		lean = 1.1
	}
	spine := art.Spline([]art.Point{{X: x0, Y: y0}, {X: x0, Y: y0 - top*0.5}, {X: x0 + 0.3, Y: y0 - top*0.8}, {X: x0 + lean, Y: y0 - top}}, 4)
	tones := stem[int(art.Steps(g, []art.Step{art.At(art.Sapling, 1), art.At(art.Whip, 2)})+0.5)]
	c.Stroke(art.Stroke{Path: spine, W0: 0.8, W1: 0.8, Tones: tones, Jitter: 0.3})
	tip := spine[len(spine)-1]
	switch {
	case hook:
		for _, side := range []float64{-1, 1} {
			sproutLeaf(c, tip.X, tip.Y+0.5, side, 3.2, seedLeaf, int(side))
			c.Line(x0+side*1.2, y0, x0+side*2.4, y0, soil[3], nil)
		}
	case art.During(g, art.Seedling, art.Sapling):
		for _, side := range []float64{-1, 1} {
			c.Blob(art.Blob{X: tip.X + side*1.8, Y: tip.Y + top*0.5, RX: 1.5, RY: 0.75, Tones: seedLeaf, Salt: int(side)})
			sproutLeaf(c, tip.X, tip.Y+0.6, side, 2.8, leaf[1:], int(side))
		}
	default:
		sapling := art.During(g, art.Sapling, art.Whip)
		heights := []float64{0.3, 0.42, 0.54, 0.66, 0.78, 0.9}
		if sapling {
			heights = []float64{0.42, 0.58, 0.74, 0.9}
		}
		for k, h := range heights {
			at := spine.Along(h)
			size := 3.0 - 0.4*float64(b2i(k == len(heights)-1))
			sproutLeaf(c, at.X, at.Y, float64(2*(k%2)-1), size, leaf[1:], k)
		}
		if !sapling {
			i, j := c.Cell(tip.X, tip.Y-0.4)
			c.Put(i, j, bud)
		}
	}
}

// sproutLeaf draws a pointed leaf leaving a stem at (x, y), up and out to one side.
func sproutLeaf(c *art.Canvas, x, y, side, size float64, tones art.Tones, salt int) {
	points := []art.Point{
		{X: x, Y: y},
		{X: x + side*size*0.4, Y: y - size*0.5},
		{X: x + side*size*0.9, Y: y - size*0.65},
		{X: x + side*size*0.6, Y: y - size*0.05},
	}
	c.Polygon(art.Polygon{Points: points, Tones: tones, Salt: salt})
}
