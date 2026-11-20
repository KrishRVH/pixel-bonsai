package maple

import "github.com/KrishRVH/pixel-bonsai/art"

// nodes are a young stem's pairs of true leaves: the height on the stem of each, and the stage it unfolds at.
var nodes = []struct {
	h    float64
	born art.Stage
}{{4.2, art.Seedling}, {8.0, art.Sapling}, {11.8, art.Whip}}

// shoot draws the hooked shoot of Sprout with its two seed leaves.
func shoot(c *art.Canvas) {
	hook := art.Path{{X: 24, Y: 22.6}, {X: 24, Y: 21.0}, {X: 24.5, Y: 19.7}, {X: 25.8, Y: 19.1}, {X: 26.9, Y: 19.8}, {X: 27.1, Y: 21.0}}
	c.Stroke(art.Stroke{Path: hook, W0: 0.9, W1: 0.9, Tones: young})
	for _, tip := range []art.Point{{X: 26.0, Y: 22.0}, {X: 28.4, Y: 21.9}} {
		c.Stroke(art.Stroke{Path: art.Path{{X: 27.1, Y: 20.8}, tip}, W0: 1.0, W1: 0.5, Tones: spring})
	}
}

// stemLeaves draws a young stem's leaves: its seed leaves until Whip, and the opposite pairs of true leaves along it,
// each on a short stalk and a tiny palmate shape where the scale allows.
func stemLeaves(c *art.Canvas, g float64, path art.Path, top float64) {
	size := glyphSize(c)
	stalk := size / 2
	for n, node := range nodes {
		if !art.From(g, node.born) || node.h > top+0.5 {
			continue
		}
		at := path.Along(min(node.h, top) / top)
		ci, cj := c.Cell(at.X, at.Y)
		lift := n % 2
		for _, side := range []int{-1, 1} {
			for k := 1; k <= stalk; k++ {
				c.Put(ci+side*k, cj-lift*b2i(k > 1), spring[1])
			}
			blade(c, ci+side*(stalk+1+size/2), cj-lift, size, 0.3*float64(n))
		}
	}
	if art.During(g, art.Sprout, art.Whip) {
		at := path.Along(2.4 / top)
		for _, side := range []float64{-1, 1} {
			c.Stroke(art.Stroke{Path: art.Path{at, {X: at.X + side*2.2, Y: at.Y - 0.4}}, W0: 1.0, W1: 0.6, Tones: spring})
		}
	}
}

// blade draws a small palmate leaf of a young plant centered on pixel (ci, cj): crimson at its lobes, blushing deeper
// by blush, and green at its heart.
func blade(c *art.Canvas, ci, cj, size int, blush float64) {
	heart := -1 // how far the green heart reaches from the center: none on a one-pixel leaf
	if size > 1 {
		heart = b2i(size > 3)
	}
	lobes(stars[size][0], size, ci, cj, func(i, j, steps int, lit float64) {
		if steps <= heart {
			c.Put(i, j, spring.Shade(0.55+0.2*lit, 0.1, art.Noise(i, j, 0)))
		} else {
			c.Put(i, j, leaf.Shade(0.45+0.12*lit+0.12*blush, 0.1, art.Noise(i, j, 0)))
		}
	})
}
