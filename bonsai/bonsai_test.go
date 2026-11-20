package bonsai_test

import (
	"fmt"
	"math"
	"slices"
	"testing"

	"github.com/KrishRVH/pixel-bonsai/art"
	"github.com/KrishRVH/pixel-bonsai/bonsai"
)

// These tests hold every variety to the same bar, however it draws. A variety faces them once it is registered.

// pixels is a frame as a map from position to color, for comparing frames.
type pixels map[[2]int]art.Color

func render(v art.Variety, g, scale float64) pixels {
	out := pixels{}
	for _, p := range v.Render(g, scale).Pixels() {
		out[[2]int{p.X, p.Y}] = p.Color
	}
	return out
}

// differs returns how many pixels two frames disagree on, a pixel set in only one counting.
func differs(a, b pixels) int {
	n := 0
	for p, c := range a {
		if d, ok := b[p]; !ok || d != c {
			n++
		}
	}
	for p := range b {
		if _, ok := a[p]; !ok {
			n++
		}
	}
	return n
}

func colors(px pixels) map[art.Color]bool {
	out := map[art.Color]bool{}
	for _, c := range px {
		out[c] = true
	}
	return out
}

func top(px pixels) int {
	y := math.MaxInt
	for p := range px {
		y = min(y, p[1])
	}
	return y
}

// atStart is the age a stage is judged at: just past its start, where its new feature has arrived.
func atStart(s art.Stage) float64 {
	return s.Start() + art.Plus
}

var scales = []float64{0.6, 1, 2}

func TestTheRegistryListsTheVarietiesInSwitchOrder(t *testing.T) {
	t.Parallel()
	if got, want := bonsai.IDs(), []string{"sakura", "maple", "black-pine"}; !slices.Equal(got, want) {
		t.Fatalf("ids = %v, want %v", got, want)
	}
	if bonsai.Default().ID != "sakura" {
		t.Error("a new garden does not start with sakura")
	}
	for _, v := range bonsai.Varieties() {
		got, ok := bonsai.Lookup(v.ID)
		if !ok || got.ID != v.ID || v.Name == "" || v.Icon == "" || len(v.Signature) == 0 || v.Draw == nil {
			t.Errorf("%q is incomplete or cannot be looked up", v.ID)
		}
	}
	if _, ok := bonsai.Lookup("baobab"); ok {
		t.Error("an unknown id is found")
	}
}

func TestEachStageShowsSomethingNew(t *testing.T) {
	t.Parallel()
	const minChange = 10 // pixels a stage's frame must differ by from the stage before's, at scale 1
	for _, v := range bonsai.Varieties() {
		for k := 1; k < art.Stages; k++ {
			s, before := art.Stage(k), art.Stage(k-1)
			if n := differs(render(v, atStart(s), 1), render(v, atStart(before), 1)); n < minChange {
				t.Errorf("%s: %v differs from %v by %d pixels, want at least %d", v.ID, s, before, n, minChange)
			}
		}
	}
}

func TestEveryPixelIsOnTheCanvasAndTheSameEveryTime(t *testing.T) {
	t.Parallel()
	for _, v := range bonsai.Varieties() {
		for _, scale := range append(scales, art.MinScale, 1.125, 2.625, art.MaxScale) {
			for _, g := range []float64{0, 0.02, 0.1, 0.2, 0.31, 0.412, 0.55, 0.7, 0.85, 1} {
				checkFrame(t, v, g, scale)
			}
		}
	}
}

// checkFrame checks that a variety's frame is the canvas's size, draws something, keeps to the canvas, and is the
// same when drawn again after another.
func checkFrame(t *testing.T, v art.Variety, g, scale float64) {
	t.Helper()
	w, h := art.Size(scale)
	f := v.Render(g, scale)
	if f.Width() != w || f.Height() != h {
		t.Fatalf("%s at %v: frame %d×%d, want %d×%d", v.ID, scale, f.Width(), f.Height(), w, h)
	}
	px := f.Pixels()
	if len(px) == 0 {
		t.Errorf("%s at age %v, scale %v draws nothing", v.ID, g, scale)
	}
	for _, p := range px {
		if p.X < 0 || p.X >= w || p.Y < 0 || p.Y >= h || p.Color > 0xffffff {
			t.Fatalf("%s at age %v, scale %v: pixel %v is off the canvas", v.ID, g, scale, p)
		}
	}
	v.Render(g/2, scale) // another tree in between leaves no trace
	if !slices.Equal(px, v.Render(g, scale).Pixels()) {
		t.Errorf("%s at age %v, scale %v: a second render differs", v.ID, g, scale)
	}
}

func TestALargerScaleDrawsMoreAndRicher(t *testing.T) {
	t.Parallel()
	for _, v := range bonsai.Varieties() {
		for _, g := range []float64{bonsai.LifeGrowth(13e9), 0.7, 1} {
			small, one, big := render(v, g, scales[0]), render(v, g, scales[1]), render(v, g, scales[2])
			if len(small) >= len(one) {
				t.Errorf("%s at %v: scale 0.6 draws %d pixels, not fewer than scale 1's %d", v.ID, g, len(small), len(one))
			}
			if float64(len(big)) <= 2.5*float64(len(one)) {
				t.Errorf("%s at %v: scale 2 draws %d pixels, want over 2.5× scale 1's %d: more pixels, not bigger ones", v.ID, g, len(big), len(one))
			}
			if len(colors(big)) < len(colors(one)) {
				t.Errorf("%s at %v: scale 2 has fewer colors than scale 1", v.ID, g)
			}
		}
	}
}

func TestATreeGrowsAndGainsDetailWithAge(t *testing.T) {
	t.Parallel()
	for _, v := range bonsai.Varieties() {
		var sizes []int
		for _, g := range []float64{0.02, 0.07, 0.14, 0.24, 0.34, 0.45, 0.63, 0.81, 0.94} {
			sizes = append(sizes, len(render(v, g, 1)))
		}
		if !slices.IsSorted(sizes) {
			t.Errorf("%s shrinks with age: %v pixels", v.ID, sizes)
		}
		young, old := render(v, 0.15, 1), render(v, 1, 1)
		if len(colors(old)) <= len(colors(young)) {
			t.Errorf("%s: an old tree has %d tones, not more than a young one's %d", v.ID, len(colors(old)), len(colors(young)))
		}
		if top(old) >= top(young) {
			t.Errorf("%s: an old tree does not stand taller than a young one", v.ID)
		}
	}
}

func TestTheVarietiesAreDistinctAtTheSameAge(t *testing.T) {
	t.Parallel()
	vs := bonsai.Varieties()
	for _, g := range []float64{0.2, bonsai.LifeGrowth(13e9), 0.7, 1} {
		for k, a := range vs {
			for _, b := range vs[k+1:] {
				pa, pb := render(a, g, 1), render(b, g, 1)
				if n := differs(pa, pb); n < 50 {
					t.Errorf("%s and %s at %v differ by %d pixels, want at least 50", a.ID, b.ID, g, n)
				}
				checkOwnColors(t, a, pa, b, pb, g)
				checkOwnColors(t, b, pb, a, pa, g)
			}
		}
	}
}

// checkOwnColors checks that variety a's frame pa has colors of its own that b's frame pb lacks, and that pb shows
// none of a's signature colors.
func checkOwnColors(t *testing.T, a art.Variety, pa pixels, b art.Variety, pb pixels, g float64) {
	t.Helper()
	theirs := colors(pb)
	own := 0
	for c := range colors(pa) {
		if !theirs[c] {
			own++
		}
	}
	if own < 3 {
		t.Errorf("%s at %v has %d colors %s lacks, want at least 3", a.ID, g, own, b.ID)
	}
	for _, c := range a.Signature {
		if theirs[c] {
			t.Errorf("%s at %v uses %s's signature color %v", b.ID, g, a.ID, c)
		}
	}
}

func TestEachIsRecognizableSmallFromAYoungTreeOn(t *testing.T) {
	t.Parallel()
	for _, v := range bonsai.Varieties() {
		for g := atStart(art.YoungTree); g <= 1; g += 0.03 {
			found := colors(render(v, g, 0.6))
			if !slices.ContainsFunc(v.Signature, func(c art.Color) bool { return found[c] }) {
				t.Errorf("%s at %.3f shows no signature color at scale 0.6", v.ID, g)
			}
		}
	}
}

func TestEachViewLandsOnItsOwnStage(t *testing.T) {
	t.Parallel()
	got := []art.Stage{
		art.StageAt(bonsai.DayGrowth(100e6)), art.StageAt(bonsai.DayGrowth(232e6)),
		art.StageAt(bonsai.LifeGrowth(13e9)), art.StageAt(bonsai.DayGrowth(1e9)),
	}
	want := []art.Stage{art.FirstBranches, art.ThickeningTrunk, art.Shaped, art.Refined}
	if !slices.Equal(got, want) {
		t.Errorf("a 100M day, a median day, a 13B lifetime, and a 1B day are %v, want %v", got, want)
	}
	if bonsai.Today.Growth(100e6) != bonsai.DayGrowth(100e6) || bonsai.Lifetime.Growth(13e9) != bonsai.LifeGrowth(13e9) {
		t.Error("a view does not grow by its own curve")
	}
	if bonsai.Today.Next() != bonsai.Lifetime || bonsai.Lifetime.Next() != bonsai.Today {
		t.Error("the views do not alternate")
	}
}

func TestTheDayTreeKeepsGrowingThroughEveryStage(t *testing.T) {
	t.Parallel()
	if bonsai.DayGrowth(0) != 0 || bonsai.DayGrowth(-5) != 0 {
		t.Error("no tokens is not a seed")
	}
	seen := map[art.Stage]bool{}
	for k := range 259 { // 10K tokens up to 3B
		seen[art.StageAt(bonsai.DayGrowth(int64(1e4*math.Pow(1.05, float64(k)))))] = true
	}
	if len(seen) != art.Stages {
		t.Errorf("a day reaches %d of %d stages by 3B tokens", len(seen), art.Stages)
	}
	if art.StageAt(bonsai.DayGrowth(2.7e9)) != art.Venerable {
		t.Error("2.7B tokens is not a venerable day")
	}
	heavy := []float64{bonsai.DayGrowth(1e9), bonsai.DayGrowth(1.5e9), bonsai.DayGrowth(2e9), bonsai.DayGrowth(2.5e9)}
	for k := 1; k < len(heavy); k++ {
		if heavy[k] <= heavy[k-1] {
			t.Errorf("a heavier day is not an older tree: %v", heavy)
		}
	}
	early := (bonsai.DayGrowth(1e8) - bonsai.DayGrowth(1e7)) / 9e7
	late := (bonsai.DayGrowth(6e8) - bonsai.DayGrowth(3e8)) / 3e8
	if early <= late {
		t.Error("an early token does not grow the day's tree most")
	}
	if bonsai.LifeGrowth(0) != 0 || bonsai.LifeGrowth(bonsai.MatureTokens) != 1 || bonsai.LifeGrowth(50*bonsai.MatureTokens) != 1 {
		t.Error("the lifetime tree does not run from seed to venerable at maturity")
	}
	if bonsai.LifeGrowth(0.4e9*365) >= 1 {
		t.Error("a year at 0.4B tokens a day leaves no growing to do")
	}
}

func BenchmarkRender(b *testing.B) {
	for _, v := range bonsai.Varieties() {
		for _, scale := range []float64{1, art.MaxScale} {
			b.Run(fmt.Sprintf("%s/scale=%g", v.ID, scale), func(b *testing.B) {
				for b.Loop() {
					v.Render(1, scale)
				}
			})
		}
	}
}
