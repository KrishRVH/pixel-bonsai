package art_test

import (
	"image"
	"math"
	"testing"

	"github.com/KrishRVH/pixel-bonsai/art"
)

var tones = art.Tones{0x100000, 0x200000, 0x300000, 0x400000, 0x500000}

func TestAStrokeTapersAndStaysOnTheCanvas(t *testing.T) {
	t.Parallel()
	c := art.NewCanvas(1)
	cover := c.Stroke(art.Stroke{Path: art.Path{{X: 24, Y: -10}, {X: 24, Y: 40}}, W0: 8, W1: 2, Tones: tones})
	widths := map[int]int{}
	for _, cell := range cover.Cells() {
		if cell.I < 0 || cell.I >= c.Width() || cell.J < 0 || cell.J >= c.Height() {
			t.Fatalf("pixel (%d, %d) is off the canvas", cell.I, cell.J)
		}
		widths[cell.J]++
	}
	if widths[2] <= widths[27] {
		t.Errorf("width near the start %d is not wider than near the end %d", widths[2], widths[27])
	}
	if f := c.Frame(); f.Len() != cover.Len() {
		t.Errorf("painted %d pixels, cover holds %d", f.Len(), cover.Len())
	}
}

func TestAStrokeIsLitOnItsUpperLeftEdge(t *testing.T) {
	t.Parallel()
	c := art.NewCanvas(2)
	cover := c.Stroke(art.Stroke{Path: art.Path{{X: 24, Y: 5}, {X: 24, Y: 25}}, W0: 6, W1: 6, Tones: tones})
	left, _ := cover.Shade(c.Cell(21.6, 15))
	right, _ := cover.Shade(c.Cell(26.4, 15))
	if left <= right {
		t.Errorf("left edge shade %v is not brighter than right %v", left, right)
	}
}

func TestAThinStrokeIsUnbrokenAtAnyAngle(t *testing.T) {
	t.Parallel()
	for angle := 0.0; angle < math.Pi; angle += math.Pi / 12 {
		c := art.NewCanvas(1)
		dx, dy := math.Cos(angle)*15, math.Sin(angle)*15
		cover := c.Stroke(art.Stroke{Path: art.Path{{X: 24 - dx, Y: 15 - dy}, {X: 24 + dx, Y: 15 + dy}}, W0: 0.1, W1: 0.1, Tones: tones})
		if !connected(cover) {
			t.Errorf("a hairline at %.0f° breaks apart", angle*180/math.Pi)
		}
	}
}

// connected reports whether a cover's pixels touch each other, diagonals included, as one piece.
func connected(cover *art.Cover) bool {
	cells := cover.Cells()
	if len(cells) == 0 {
		return false
	}
	seen := map[image.Point]bool{{X: cells[0].I, Y: cells[0].J}: true}
	queue := []image.Point{{X: cells[0].I, Y: cells[0].J}}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				q := image.Point{X: p.X + dx, Y: p.Y + dy}
				if !seen[q] && cover.Has(q.X, q.Y) {
					seen[q] = true
					queue = append(queue, q)
				}
			}
		}
	}
	return len(seen) == len(cells)
}

func TestABlobFillsItsEllipseLitFromTheUpperLeft(t *testing.T) {
	t.Parallel()
	c := art.NewCanvas(1)
	cover := c.Blob(art.Blob{X: 24, Y: 15, RX: 10, RY: 5, Tones: tones})
	if n := cover.Len(); math.Abs(float64(n)-math.Pi*50) > 15 {
		t.Errorf("filled %d pixels, want about %.0f", n, math.Pi*50)
	}
	ul, _ := cover.Shade(c.Cell(18, 12.5))
	lr, _ := cover.Shade(c.Cell(30, 17.5))
	if ul <= lr {
		t.Errorf("upper left %v is not lit more than lower right %v", ul, lr)
	}
	for _, cell := range cover.Cells() {
		x, y := c.Center(cell.I, cell.J)
		if ((x-24)/10)*((x-24)/10)+((y-15)/5)*((y-15)/5) > 1 {
			t.Fatalf("pixel (%d, %d) is outside the ellipse", cell.I, cell.J)
		}
	}
}

func TestABlobCanBeLumpyHoledAndClipped(t *testing.T) {
	t.Parallel()
	plain := art.NewCanvas(1).Blob(art.Blob{X: 24, Y: 15, RX: 10, RY: 6, Tones: tones})
	lumpy := art.NewCanvas(1).Blob(art.Blob{X: 24, Y: 15, RX: 10, RY: 6, Tones: tones, Lump: 0.3})
	holed := art.NewCanvas(1).Blob(art.Blob{X: 24, Y: 15, RX: 10, RY: 6, Tones: tones, Gaps: 0.5})
	clipped := art.NewCanvas(1).Blob(art.Blob{X: 24, Y: 15, RX: 10, RY: 6, Tones: tones, Where: func(x, _ float64) bool { return x < 24 }})
	if same(plain, lumpy) {
		t.Error("lump does not change the outline")
	}
	if holed.Len() >= plain.Len()*9/10 {
		t.Errorf("gaps 0.5 left %d of %d pixels", holed.Len(), plain.Len())
	}
	for _, cell := range clipped.Cells() {
		if cell.I >= 24 {
			t.Fatalf("clipped blob has pixel (%d, %d) right of the clip", cell.I, cell.J)
		}
	}
}

func same(a, b *art.Cover) bool {
	if a.Len() != b.Len() {
		return false
	}
	for _, cell := range a.Cells() {
		if !b.Has(cell.I, cell.J) {
			return false
		}
	}
	return true
}

func TestABlockAPolygonAndALineStayInTheirShapes(t *testing.T) {
	t.Parallel()
	c := art.NewCanvas(1)
	block := c.Block(art.Block{X0: 10, Y0: 10, X1: 20, Y1: 14, Tones: tones, Taper: 2})
	for _, cell := range block.Cells() {
		x, y := c.Center(cell.I, cell.J)
		inset := 2 * (y - 10) / 4
		if x < 10+inset || x >= 20-inset || y < 10 || y >= 14 {
			t.Fatalf("block pixel (%d, %d) is outside its tapered box", cell.I, cell.J)
		}
	}
	if rounded := art.NewCanvas(1).Block(art.Block{X0: 10, Y0: 10, X1: 20, Y1: 14, Tones: tones, Rounding: 1}); rounded.Has(10, 10) {
		t.Error("a rounded block keeps its corner")
	}
	triangle := c.Polygon(art.Polygon{Points: []art.Point{{X: 30, Y: 5}, {X: 40, Y: 5}, {X: 30, Y: 15}}, Tones: tones})
	if !triangle.Has(31, 6) || triangle.Has(39, 14) {
		t.Error("the polygon does not fill its triangle alone")
	}
	line := c.Line(0, 0, 10, 5, 0xabcdef, nil)
	if len(line) != 11 || line[0] != (image.Point{}) || line[10] != (image.Point{X: 10, Y: 5}) {
		t.Errorf("the line runs %v, want 11 pixels from (0, 0) to (10, 5)", line)
	}
	vetoed := c.Line(0, 20, 10, 20, 0xabcdef, func(i, _ int) bool { return i%2 == 0 })
	if len(vetoed) != 6 {
		t.Errorf("a vetoing line painted %d pixels, want 6", len(vetoed))
	}
}

func TestScatterMarksOnlyItsCoverAndABoxGainsPixelsWithScale(t *testing.T) {
	t.Parallel()
	c := art.NewCanvas(1)
	cover := c.Blob(art.Blob{X: 24, Y: 15, RX: 6, RY: 4, Tones: tones})
	marked := c.ScatterCover(cover, art.Scatter{Colors: art.Tones{0xffffff}, Density: 0.5, Run: 3})
	if len(marked) == 0 {
		t.Fatal("scatter marked nothing")
	}
	for _, p := range marked {
		if !cover.Has(p.X, p.Y) {
			t.Fatalf("scatter marked (%d, %d) outside its cover", p.X, p.Y)
		}
	}
	small := art.NewCanvas(1).ScatterBox(0, 0, 48, 30, art.Scatter{Colors: art.Tones{1}, Density: 0.1})
	big := art.NewCanvas(2).ScatterBox(0, 0, 48, 30, art.Scatter{Colors: art.Tones{1}, Density: 0.1})
	if len(big) < 3*len(small) {
		t.Errorf("scale 2 scattered %d pixels and scale 1 %d: detail does not grow with scale", len(big), len(small))
	}
}

func TestWireWindsRoundAStrokeInStripes(t *testing.T) {
	t.Parallel()
	path := art.Path{{X: 24, Y: 3}, {X: 24, Y: 27}}
	c := art.NewCanvas(2)
	wound := c.Coil(art.Coil{Path: path, W0: 5, W1: 5, Color: 0xc0773a})
	all := len(c.Trace(path, 5, 5))
	if len(wound) < all/5 || len(wound) > all*7/10 {
		t.Errorf("wire covers %d of %d pixels: stripes with gaps, not a solid wrap", len(wound), all)
	}
	sides := map[art.Color]bool{}
	c2 := art.NewCanvas(2)
	paint := func(across, _ float64) art.Color {
		if across < 0 {
			return 0xaa0000
		}
		return 0x0000aa
	}
	for _, p := range c2.Coil(art.Coil{Path: path, W0: 5, W1: 5, Paint: paint, Test: func(_, j int) bool { return j < 30 }}) {
		if p.Y >= 30 {
			t.Fatal("a vetoed pixel is wound")
		}
		col, _ := c2.Get(p.X, p.Y)
		sides[col] = true
	}
	if len(sides) != 2 {
		t.Errorf("wire colors %v, want one for each side", sides)
	}
	if grown := art.NewCanvas(2).Coil(art.Coil{Path: path, W0: 5, W1: 5, Color: 1, Grow: 2}); len(grown) <= len(wound) {
		t.Error("Grow does not widen the wire")
	}
}

func TestACoverKeepsDrawingOrderThroughMerges(t *testing.T) {
	t.Parallel()
	a := art.NewCover()
	a.Set(1, 1, 0.1)
	a.Set(2, 2, 0.2)
	b := art.NewCover()
	b.Set(3, 3, 0.3)
	b.Set(1, 1, 0.9)
	a.Merge(b)
	got := a.Cells()
	want := []art.Cell{{I: 1, J: 1, Shade: 0.9}, {I: 2, J: 2, Shade: 0.2}, {I: 3, J: 3, Shade: 0.3}}
	if len(got) != len(want) {
		t.Fatalf("cells = %v, want %v", got, want)
	}
	for k := range want {
		if got[k] != want[k] {
			t.Errorf("cell %d = %v, want %v: a merged pixel keeps its place and takes the new shade", k, got[k], want[k])
		}
	}
	if f := a.Filter(func(c art.Cell) bool { return c.Shade > 0.25 }); f.Len() != 2 || f.Cells()[0].I != 1 {
		t.Error("Filter does not keep the matching cells in order")
	}
}

func TestACanvasCopiesOverlaysAndErases(t *testing.T) {
	t.Parallel()
	c := art.NewCanvas(1)
	c.Put(1, 1, 0x111111)
	c.Put(-1, 0, 0x222222)
	c.Put(c.Width(), 0, 0x222222)
	if f := c.Frame(); f.Len() != 1 {
		t.Errorf("an off-canvas put landed: %d pixels set", f.Len())
	}
	before := c.Clone()
	c.Erase(1, 1)
	c.PutIfEmpty(2, 2, 0x333333)
	c.PutIfEmpty(2, 2, 0x444444)
	if c.Has(1, 1) || !before.Has(1, 1) {
		t.Error("Erase reached the clone, or missed the canvas")
	}
	if col, _ := c.Get(2, 2); col != 0x333333 {
		t.Errorf("PutIfEmpty overwrote a pixel: %v", col)
	}
	c.Overlay(before, func(i, _ int) bool { return i < 2 })
	if col, ok := c.Get(1, 1); !ok || col != 0x111111 {
		t.Error("Overlay did not bring the pixel back")
	}
}
