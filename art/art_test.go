package art_test

import (
	"math"
	"testing"

	"github.com/KrishRVH/pixel-bonsai/art"
)

func TestTheLadderHasTwentyStagesInOrder(t *testing.T) {
	t.Parallel()
	if art.Stages != 20 {
		t.Fatalf("stages = %d, want 20", art.Stages)
	}
	for k := 1; k < art.Stages; k++ {
		s := art.Stage(k)
		if s.Start() <= art.Stage(k-1).Start() {
			t.Errorf("%v starts at %v, not after %v", s, s.Start(), art.Stage(k-1))
		}
		if art.StageAt(s.Start()) != s || art.StageAt(s.Start()-1e-9) != art.Stage(k-1) {
			t.Errorf("%v does not begin exactly at its start age", s)
		}
	}
	if art.StageAt(1) != art.Venerable || art.StageAt(0) != art.Seed {
		t.Error("ages 0 and 1 are not seed and venerable")
	}
}

func TestStageFunctionsFollowTheLadder(t *testing.T) {
	t.Parallel()
	shaped := art.Shaped.Start()
	if art.Since(shaped-0.01, art.Shaped, 0.05) != 0 || math.Abs(art.Since(shaped+0.025, art.Shaped, 0.05)-0.5) > 1e-9 || art.Since(1, art.Shaped, 0.05) != 1 {
		t.Error("Since does not ramp from 0 at the stage to 1 over its span")
	}
	if art.From(shaped-1e-9, art.Shaped) || !art.From(shaped, art.Shaped) {
		t.Error("From does not switch at the stage's start")
	}
	if !art.During(shaped, art.Shaped, art.Mature) || art.During(art.Mature.Start(), art.Shaped, art.Mature) {
		t.Error("During does not cover first and stop at last")
	}
	table := []art.Step{art.At(art.Sapling, 1), art.At(art.Whip, 2)}
	for _, c := range []struct {
		g    float64
		want float64
	}{{0, 0}, {art.Sapling.Start() + art.Plus, 1}, {art.Whip.Start() + art.Plus, 2}, {1, 2}} {
		if got := art.Steps(c.g, table); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("Steps(%v) = %v, want %v", c.g, got, c.want)
		}
	}
	lerp := []art.Step{art.At(art.Sapling, 10), art.At(art.Whip, 20)}
	mid := (art.Sapling.Start() + art.Whip.Start()) / 2
	if art.Lerp(0, lerp) != 10 || math.Abs(art.Lerp(mid, lerp)-15) > 1e-9 || art.Lerp(1, lerp) != 20 {
		t.Error("Lerp does not hold its ends and move linearly between steps")
	}
}

func TestNoiseIsFixedInRangeAndWellSpread(t *testing.T) {
	t.Parallel()
	buckets := make([]int, 10)
	for k := range 10000 {
		x, y := k/100, k%100
		n := art.Noise(x, y, 7)
		if n < 0 || n >= 1 || n != art.Noise(x, y, 7) {
			t.Fatalf("Noise(%d, %d) = %v, outside [0, 1) or not fixed", x, y, n)
		}
		buckets[int(n*10)]++
	}
	for k, n := range buckets {
		if n < 800 || n > 1200 {
			t.Errorf("bucket %d holds %d of 10000 values, want about 1000", k, n)
		}
	}
	if art.Noise(1, 2, 3) == art.Noise(1, 2, 4) {
		t.Error("the salt does not change the value")
	}
	if art.Noise(-5, -9, 1) < 0 {
		t.Error("negative inputs leave the range")
	}
}

func TestColorsBlendAndTonesRunShadowToLit(t *testing.T) {
	t.Parallel()
	if got := art.Mix(0x000000, 0xffffff, 0.5); got != 0x808080 {
		t.Errorf("Mix halfway = %v, want 808080", got)
	}
	if art.Mix(0x102030, 0x405060, 0) != 0x102030 || art.Mix(0x102030, 0x405060, 1) != 0x405060 {
		t.Error("Mix does not hold its ends")
	}
	tones := art.Tones{1, 2, 3, 4}
	if tones.Shade(0, 0, 0.5) != 1 || tones.Shade(1, 0, 0.5) != 4 || tones.Shade(0.6, 0, 0.5) != 3 {
		t.Error("Shade does not run from the first tone in shadow to the last lit")
	}
	if tones.Shade(-5, 1, 0) != 1 || tones.Shade(5, 1, 0.99) != 4 {
		t.Error("Shade leaves the tones")
	}
	if art.Color(0x0a0b0c).Hex() != "0a0b0c" {
		t.Error("Hex is not six lowercase digits")
	}
}

func TestTheSizeOfACanvas(t *testing.T) {
	t.Parallel()
	if w, h := art.Size(1); w != art.W || h != art.H {
		t.Errorf("Size(1) = %d×%d, want %d×%d", w, h, art.W, art.H)
	}
	if w, h := art.Size(0.6); w != 29 || h != 18 {
		t.Errorf("Size(0.6) = %d×%d, want 29×18", w, h)
	}
	if w, h := art.Size(0.58); w != 28 || h != 18 {
		t.Errorf("Size(0.58) = %d×%d, want 28×18: the height rounds up to even", w, h)
	}
	for _, s := range []float64{0.5, 0.6, 0.75, 1.3, 2, 2.75, 3} {
		if _, h := art.Size(s); h%2 != 0 {
			t.Errorf("Size(%v) height %d is odd: half blocks need pairs of rows", s, h)
		}
	}
}

func TestAPaneFitsTheLargestScaleInEighths(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		width, rows int
		want        float64
	}{
		{48, 16, 1},     // exactly the base canvas over a caption
		{100, 16, 1},    // limited by height
		{60, 100, 1.25}, // limited by width, floored to an eighth
		{500, 500, 3},   // capped
		{20, 16, 0},     // too small: the caption shows alone
		{54, 20, 1.125}, // a narrow side column
		{100, 50, 2},
		{20, 40, 0}, // too narrow
		{54, 8, 0},  // too short
		{54, 9, 0.5},
	} {
		if got := art.Fit(c.width, c.rows); got != c.want {
			t.Errorf("Fit(%d, %d) = %v, want %v", c.width, c.rows, got, c.want)
		}
		if s := art.Fit(c.width, c.rows); s > 0 {
			if w, h := art.Size(s); w > c.width || h/2 > c.rows-1 {
				t.Errorf("Fit(%d, %d) = %v does not fit: %d×%d", c.width, c.rows, s, w, h)
			}
		}
	}
}

func TestASplineRunsThroughItsPoints(t *testing.T) {
	t.Parallel()
	points := []art.Point{{X: 0, Y: 0}, {X: 10, Y: 5}, {X: 20, Y: 0}}
	path := art.Spline(points, 4)
	if len(path) != 9 {
		t.Fatalf("len = %d, want 9: 4 segments between each pair and the end", len(path))
	}
	for k, p := range points {
		if path[k*4] != p {
			t.Errorf("point %d = %v, want %v", k, path[k*4], p)
		}
	}
	if path.Along(0) != points[0] || path.Along(1) != points[2] || path.Along(0.5) != points[1] {
		t.Error("Along does not walk the path from start to end")
	}
}
