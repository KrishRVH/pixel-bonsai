package maple

import "github.com/KrishRVH/pixel-bonsai/art"

// stars are palmate leaf glyphs by width in pixels, each with five lobes, written row by row.
var stars = map[int][]string{
	1: {"#"},
	3: {".#. ### #.#", "#.# ### .#.", "#.# .#. #.#", ".#. ### .#."},
	5: {
		".#.#. .###. .###. #.#.# ..#..",
		"..#.. #.#.# .###. .###. .#.#.",
		"..#.. .###. ##### .###. .#.#.",
	},
}

// tufts are the pixels of a twig-tip tuft on a small canvas, as offsets; below scale 0.8 the first shows alone.
var tufts = [][][2]int{{{0, 0}, {1, 0}}, {{0, 0}, {0, -1}}, {{0, 0}, {-1, -1}}, {{0, 0}, {-1, 0}}, {{0, 0}, {1, -1}}}

// bayer is a 4×4 ordered dither, which breaks the edges between leaf tones into an even weave.
var bayer = [4][4]float64{{0, 8, 2, 10}, {12, 4, 14, 6}, {3, 11, 1, 9}, {15, 7, 13, 5}}

// leafColor returns the color of a leaf pixel at tone level in [0, 1], deep wine to gold, dithered. While the
// foliage is still partly green, by green from 0 to 1, its low tones show spring green.
func leafColor(i, j int, level, green, jitter float64) art.Color {
	if green != 0 && level+0.12*(art.Noise(i, j, 7)-0.5) < green*0.34 {
		return spring.Shade(level/(0.34*green), 0.4, art.Noise(i, j, 8))
	}
	// j&3 and i&3 place the pixel in the dither, also left of or above the canvas.
	return leaf.Shade(level, jitter, 0.65*(bayer[j&3][i&3]+0.5)/16+0.35*art.Noise(i, j, 9))
}

// lobes calls paint for each pixel of a leaf glyph size pixels wide centered on pixel (ci, cj), with its distance in
// steps from the center and how lit it is, from -1 at its lower right to 1 at its upper left.
func lobes(glyph string, size, ci, cj int, paint func(i, j, steps int, lit float64)) {
	half := size / 2
	for k, ch := range glyph {
		if ch != '#' {
			continue
		}
		// Each row is size characters and a space.
		di, dj := k%(size+1)-half, k/(size+1)-half
		lit := (float64(di)*art.LightX + float64(dj)*art.LightY) / float64(max(half, 1))
		paint(ci+di, cj+dj, abs(di)+abs(dj), lit)
	}
}

func abs(n int) int {
	return max(n, -n)
}

// star draws a palmate leaf glyph size pixels wide centered on pixel (ci, cj), lit on its upper left. kind picks
// one of the glyphs of that size.
func star(c *art.Canvas, ci, cj, size int, level, green float64, kind int, jitter float64) {
	shapes := stars[size]
	lobes(shapes[kind%len(shapes)], size, ci, cj, func(i, j, _ int, lit float64) {
		c.Put(i, j, leafColor(i, j, level+0.2*lit, green, jitter))
	})
}

// glyphSize returns the width in pixels of a small palmate leaf on c: 1 on a small canvas, 3 on a normal one, and 5
// on a large one.
func glyphSize(c *art.Canvas) int {
	switch {
	case c.Scale() < 0.8:
		return 1
	case c.Scale() < 1.6:
		return 3
	}
	return 5
}

// singleLeaf draws one leaf at (x, y) in units, glyphSize pixels wide.
func singleLeaf(c *art.Canvas, x, y, level, green float64, kind int) {
	i, j := c.Cell(x, y)
	star(c, i, j, glyphSize(c), level, green, kind, 0.25)
}

// tuft draws the leaves at a twig's tip: a pixel or two on a small canvas, a palmate glyph on a larger one.
func tuft(c *art.Canvas, x, y, level, green float64, kind int) {
	if c.Scale() >= 1.4 {
		singleLeaf(c, x, y, level, green, kind)
		return
	}
	i, j := c.Cell(x, y)
	offsets := tufts[kind%len(tufts)]
	if c.Scale() < 0.8 {
		offsets = offsets[:1]
	}
	for _, o := range offsets {
		c.Put(i+o[0], j+o[1], leafColor(i+o[0], j+o[1], level, green, 0.25))
	}
}
