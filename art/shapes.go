package art

import (
	"image"
	"math"
)

// NoJitter is the Jitter of a shape shaded with no dither at all, since a Jitter left at zero takes the kit's default.
const NoJitter = -1

// jitterOr returns a shape's dither: none for [NoJitter], def when left at zero, and v otherwise.
func jitterOr(v, def float64) float64 {
	if v < 0 {
		return 0
	}
	return orDefault(v, def)
}

// orDefault returns v, or def when v is zero: a shape option left out takes the kit's default.
func orDefault(v, def float64) float64 {
	if v == 0 {
		return def
	}
	return v
}

// A StrokeCell is one pixel of a stroke: U, how far along the stroke it lies in units, and F, that as a share of the
// stroke's length; Across, where it lies from -1 at one edge to 1 at the other; and its shade.
type StrokeCell struct {
	I, J                int
	U, F, Across, Shade float64
}

type segment struct{ ax, ay, dx, dy, n, u0 float64 }

// Trace measures a stroke along a path, w0 units wide at its start and w1 at its end, without painting it. A stroke is
// at least a pixel wide, and each pixel appears once, measured from its nearest segment.
func (c *Canvas) Trace(path Path, w0, w1 float64) []StrokeCell {
	segs, total := segments(path)
	floor := 0.6 / c.scale
	reach := max(w0, w1)/2 + floor
	var cells []StrokeCell
	var dist []float64
	index := map[cellKey]int{}
	for _, s := range segs {
		x0, x1 := min(s.ax, s.ax+s.dx)-reach, max(s.ax, s.ax+s.dx)+reach
		y0, y1 := min(s.ay, s.ay+s.dy)-reach, max(s.ay, s.ay+s.dy)+reach
		for _, p := range c.Scan(x0, y0, x1, y1) {
			t := Clamp(((p.X-s.ax)*s.dx + (p.Y-s.ay)*s.dy) / (s.n * s.n))
			f := (s.u0 + t*s.n) / total
			half := max((w0+(w1-w0)*f)/2, floor)
			ox, oy := (p.X-s.ax-t*s.dx)/half, (p.Y-s.ay-t*s.dy)/half
			d := ox*ox + oy*oy
			if d > 1 {
				continue
			}
			cell := StrokeCell{p.I, p.J, s.u0 + t*s.n, f, (ox*-s.dy + oy*s.dx) / s.n, Lit(ox, oy)}
			key := cellKey{p.I, p.J}
			if k, ok := index[key]; !ok {
				index[key] = len(cells)
				cells = append(cells, cell)
				dist = append(dist, d)
			} else if d < dist[k] {
				cells[k], dist[k] = cell, d
			}
		}
	}
	return cells
}

// segments returns a path's segments, each with its length and its start's distance along the path, and the path's
// length. A path with no length becomes a tiny horizontal segment, so it still marks its pixel.
func segments(path Path) ([]segment, float64) {
	var segs []segment
	total := 0.0
	for k := 1; k < len(path); k++ {
		a, b := path[k-1], path[k]
		if n := math.Hypot(b.X-a.X, b.Y-a.Y); n > 0 {
			segs = append(segs, segment{a.X, a.Y, b.X - a.X, b.Y - a.Y, n, total})
			total += n
		}
	}
	if len(segs) == 0 {
		a := path[0]
		return []segment{{a.X, a.Y, 1e-6, 0, 1e-6, 0}}, 1e-6
	}
	return segs, total
}

// A Stroke is a tapered line along a path: a trunk, a branch, a twig, or a root, lit on its upper left edge.
type Stroke struct {
	Path   Path
	W0, W1 float64 // width in units at the start and at the end
	Tones  Tones   // nil measures the stroke without painting it
	Jitter float64 // dither between tones; 0 takes 0.25, NoJitter none
	Salt   int
}

// Stroke paints a stroke and returns its cover.
func (c *Canvas) Stroke(s Stroke) *Cover {
	jitter := jitterOr(s.Jitter, 0.25)
	cover := NewCover()
	for _, cell := range c.Trace(s.Path, s.W0, s.W1) {
		cover.push(cell.I, cell.J, cell.Shade)
		if len(s.Tones) > 0 {
			c.Put(cell.I, cell.J, s.Tones.Shade(cell.Shade, jitter, Noise(cell.I, cell.J, s.Salt)))
		}
	}
	return cover
}

// A Coil is wire wound round a stroke: a diagonal stripe every Pitch units along it.
type Coil struct {
	Path   Path
	W0, W1 float64 // the widths of the stroke it winds round
	Color  Color
	Paint  func(across, shade float64) Color // when set, colors each pixel by its side and shade instead of Color
	Pitch  float64                           // units between turns; 0 takes 3
	Twist  float64                           // how far a turn slants across the stroke; 0 takes 0.35
	Duty   float64                           // the share of each turn the wire covers; 0 takes 0.45
	Grow   float64                           // width the wire adds to the stroke's, in units
	Test   func(i, j int) bool               // when set, a pixel is wound only where it reports true
}

// Coil paints wire and returns the pixels it set.
func (c *Canvas) Coil(w Coil) []image.Point {
	pitch, twist, duty := orDefault(w.Pitch, 3), orDefault(w.Twist, 0.35), orDefault(w.Duty, 0.45)
	var out []image.Point
	for _, cell := range c.Trace(w.Path, w.W0+w.Grow, w.W1+w.Grow) {
		if Fract(cell.U/pitch+twist*cell.Across) >= duty || (w.Test != nil && !w.Test(cell.I, cell.J)) {
			continue
		}
		col := w.Color
		if w.Paint != nil {
			col = w.Paint(cell.Across, cell.Shade)
		}
		c.Put(cell.I, cell.J, col)
		out = append(out, image.Point{X: cell.I, Y: cell.J})
	}
	return out
}

// A Blob is a filled ellipse, shaded lit on its upper left with a dithered edge between tones: foliage, moss, stones.
type Blob struct {
	X, Y, RX, RY float64
	Tones        Tones
	Jitter       float64                 // dither between tones; 0 takes 0.5, NoJitter none
	Lump         float64                 // 0 to about 0.4 makes the outline uneven
	Gaps         float64                 // the share of the inside left open to the sky
	Salt         int                     // varies the dither, lumps, and gaps
	Where        func(x, y float64) bool // when set, clips the blob to the points in units where it reports true
}

// Blob paints a blob and returns its cover.
func (c *Canvas) Blob(b Blob) *Cover {
	jitter := jitterOr(b.Jitter, 0.5)
	p, q := Noise(b.Salt, 3, 0)*2*math.Pi, Noise(b.Salt, 4, 0)*2*math.Pi
	reach := 1 + b.Lump
	cover := NewCover()
	for _, s := range c.Scan(b.X-b.RX*reach, b.Y-b.RY*reach, b.X+b.RX*reach, b.Y+b.RY*reach) {
		nx, ny := (s.X-b.X)/b.RX, (s.Y-b.Y)/b.RY
		edge := 1.0
		if b.Lump != 0 {
			a := math.Atan2(ny, nx)
			edge += b.Lump * (0.6*math.Sin(3*a+p) + 0.4*math.Sin(5*a+q))
		}
		r2 := nx*nx + ny*ny
		if r2 > edge*edge || (b.Where != nil && !b.Where(s.X, s.Y)) {
			continue
		}
		if b.Gaps != 0 && r2 < 0.55 && Noise(s.I, s.J, b.Salt+101) < b.Gaps {
			continue
		}
		shade := Lit(nx, ny)
		cover.push(s.I, s.J, shade)
		c.Put(s.I, s.J, b.Tones.Shade(shade, jitter, Noise(s.I, s.J, b.Salt)))
	}
	return cover
}

// A Block is a shaded box lit on its left and top: a pot, a stand, a label.
type Block struct {
	X0, Y0, X1, Y1 float64
	Tones          Tones
	Taper          float64 // narrows the foot by this many units on each side
	Rounding       float64 // rounds the corners, from 0 for a box to 1 for an ellipse
	Jitter         float64 // dither between tones; 0 is none
	Salt           int
}

// Block paints a block and returns its cover.
func (c *Canvas) Block(b Block) *Cover {
	cover := NewCover()
	for _, s := range c.Scan(b.X0, b.Y0, b.X1, b.Y1) {
		if s.Y < b.Y0 || s.Y >= b.Y1 {
			continue
		}
		inset := b.Taper * (s.Y - b.Y0) / (b.Y1 - b.Y0)
		left, right := b.X0+inset, b.X1-inset
		if s.X < left || s.X >= right {
			continue
		}
		nx, ny := 2*(s.X-left)/(right-left)-1, 2*(s.Y-b.Y0)/(b.Y1-b.Y0)-1
		if b.Rounding != 0 && superellipse(nx, ny, 2+8*(1-b.Rounding)) > 1 {
			continue
		}
		shade := Lit(nx, ny)
		cover.push(s.I, s.J, shade)
		c.Put(s.I, s.J, b.Tones.Shade(shade, b.Jitter, Noise(s.I, s.J, b.Salt)))
	}
	return cover
}

func superellipse(nx, ny, e float64) float64 {
	return math.Pow(math.Abs(nx), e) + math.Pow(math.Abs(ny), e)
}

// A Polygon is a filled shape through points in units, shaded lit on its upper left across its extent: a pad's flat
// underside, a jin, a leaf.
type Polygon struct {
	Points []Point
	Tones  Tones
	Jitter float64 // dither between tones; 0 takes 0.25, NoJitter none
	Salt   int
}

// Polygon paints a polygon and returns its cover.
func (c *Canvas) Polygon(p Polygon) *Cover {
	jitter := jitterOr(p.Jitter, 0.25)
	x0, y0, x1, y1 := bounds(p.Points)
	cx, cy := (x0+x1)/2, (y0+y1)/2
	hw, hh := orDefault((x1-x0)/2, 1), orDefault((y1-y0)/2, 1)
	cover := NewCover()
	for _, s := range c.Scan(x0, y0, x1, y1) {
		if !insidePolygon(p.Points, s.X, s.Y) {
			continue
		}
		shade := Lit((s.X-cx)/hw, (s.Y-cy)/hh)
		cover.push(s.I, s.J, shade)
		c.Put(s.I, s.J, p.Tones.Shade(shade, jitter, Noise(s.I, s.J, p.Salt)))
	}
	return cover
}

func bounds(points []Point) (x0, y0, x1, y1 float64) {
	x0, y0, x1, y1 = points[0].X, points[0].Y, points[0].X, points[0].Y
	for _, p := range points[1:] {
		x0, y0, x1, y1 = min(x0, p.X), min(y0, p.Y), max(x1, p.X), max(y1, p.Y)
	}
	return x0, y0, x1, y1
}

// insidePolygon reports whether (x, y) is inside the polygon, by the even-odd rule.
func insidePolygon(points []Point, x, y float64) bool {
	inside := false
	for k := range points {
		a, b := points[k], points[(k+1)%len(points)]
		if (a.Y > y) != (b.Y > y) && x < a.X+(y-a.Y)*(b.X-a.X)/(b.Y-a.Y) {
			inside = !inside
		}
	}
	return inside
}

// Line paints a one-pixel line between two points in units: a needle, a crack, a blade of grass. When test is set, a
// pixel is painted only where it reports true; every pixel is tested before any is painted. Returns the pixels it set.
func (c *Canvas) Line(x0, y0, x1, y1 float64, col Color, test func(i, j int) bool) []image.Point {
	i0, j0 := c.Cell(x0, y0)
	i1, j1 := c.Cell(x1, y1)
	steps := max(abs(i1-i0), abs(j1-j0), 1)
	out := make([]image.Point, 0, steps+1)
	for k := 0; k <= steps; k++ {
		t := float64(k) / float64(steps)
		i := int(math.RoundToEven(float64(i0) + float64(i1-i0)*t))
		j := int(math.RoundToEven(float64(j0) + float64(j1-j0)*t))
		if test == nil || test(i, j) {
			out = append(out, image.Point{X: i, Y: j})
		}
	}
	for _, p := range out {
		c.Put(p.X, p.Y, col)
	}
	return out
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// A Scatter is per-pixel detail: blossoms, bark flecks, petals. It is placed per pixel, so a larger scale shows more
// of it.
type Scatter struct {
	Colors  Tones   // each marked pixel takes one, picked by noise
	Density float64 // the chance that a pixel is marked
	Salt    int
	Run     int                                // pixels in each horizontal dash; 0 takes 1
	Test    func(i, j int, shade float64) bool // when set, a pixel is marked only where it reports true
}

// ScatterCover marks pixels of a cover, keeping each dash inside it, and returns the pixels it set. Test sees each
// pixel's shade in the cover.
func (c *Canvas) ScatterCover(cover *Cover, s Scatter) []image.Point {
	return c.scatter(cover.Cells(), cover, s)
}

// ScatterBox marks pixels whose centers fall inside a box in units and returns the pixels it set. Test sees a shade of
// 0.5, and a dash may run past the box's right edge.
func (c *Canvas) ScatterBox(x0, y0, x1, y1 float64, s Scatter) []image.Point {
	var cells []Cell
	for _, p := range c.Scan(x0, y0, x1, y1) {
		if x0 <= p.X && p.X < x1 && y0 <= p.Y && p.Y < y1 {
			cells = append(cells, Cell{p.I, p.J, 0.5})
		}
	}
	return c.scatter(cells, nil, s)
}

func (c *Canvas) scatter(cells []Cell, within *Cover, s Scatter) []image.Point {
	run := max(1, s.Run)
	var out []image.Point
	for _, cell := range cells {
		if Noise(cell.I, cell.J, s.Salt) >= s.Density || (s.Test != nil && !s.Test(cell.I, cell.J, cell.Shade)) {
			continue
		}
		col := s.Colors[int(Noise(cell.I, cell.J, s.Salt+1)*float64(len(s.Colors)))]
		for k := range run {
			if within == nil || within.Has(cell.I+k, cell.J) {
				c.Put(cell.I+k, cell.J, col)
				out = append(out, image.Point{X: cell.I + k, Y: cell.J})
			}
		}
	}
	return out
}
