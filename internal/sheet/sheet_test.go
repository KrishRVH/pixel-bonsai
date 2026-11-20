package sheet_test

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/KrishRVH/pixel-bonsai/art"
	"github.com/KrishRVH/pixel-bonsai/bonsai"
	"github.com/KrishRVH/pixel-bonsai/internal/sheet"
)

func decode(t *testing.T, path string) image.Image {
	t.Helper()
	f, err := os.Open(path) // #nosec G304 -- A test's own temporary file.
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return img
}

func TestItWritesLabelledPNGsAndANSIFramesForAVariety(t *testing.T) {
	t.Parallel()
	out := t.TempDir()
	pine, _ := bonsai.Lookup("black-pine")
	files, err := sheet.Write(out, pine, 1, 6)
	if err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(out, "black-pine")
	names := make([]string, len(files))
	for k, f := range files {
		names[k] = filepath.Base(f)
		if filepath.Dir(f) != folder {
			t.Errorf("%s is outside %s", f, folder)
		}
	}
	slices.Sort(names)
	if want := []string{"day-100M.png", "frames.txt", "life-13B.png", "scales.png", "stages.png"}; !slices.Equal(names, want) {
		t.Fatalf("files = %v, want %v", names, want)
	}
	stages := decode(t, filepath.Join(folder, "stages.png"))
	if b := stages.Bounds(); b.Dx() <= 5*art.W*6 || b.Dy() <= 4*art.H*6 {
		t.Errorf("stages sheet is %v, too small for 20 frames at zoom 6", b)
	}
	if r, g, b, _ := stages.At(0, 0).RGBA(); r>>8 != 0x1e || g>>8 != 0x1e || b>>8 != 0x2e {
		t.Error("the sheet is not on the terminal's background")
	}
	if !holds(stages, pine.Signature[0]) {
		t.Error("the tree's signature color is not on the sheet")
	}
	if b := decode(t, filepath.Join(folder, "day-100M.png")).Bounds(); b.Dx() <= art.W*6 {
		t.Errorf("the day sheet is %v wide", b)
	}
	checkFrames(t, filepath.Join(folder, "frames.txt"))
}

// checkFrames checks that frames.txt holds every stage of the black pine and the view and scale frames, in color.
func checkFrames(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path) // #nosec G304 -- A test's own temporary file.
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for k := range art.Stages {
		if !strings.Contains(text, " "+art.Stage(k).String()+" (age ") {
			t.Errorf("frames.txt lacks %v", art.Stage(k))
		}
	}
	if n := strings.Count(text, "== black-pine "); n != art.Stages+2+3 {
		t.Errorf("frames.txt holds %d frames, want %d", n, art.Stages+5)
	}
	for _, s := range []string{"13b at 0.6", "\x1b[38;2;", "▀"} {
		if !strings.Contains(text, s) {
			t.Errorf("frames.txt lacks %q", s)
		}
	}
}

func holds(img image.Image, c art.Color) bool {
	r0, g0, b0 := c.RGB()
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if r>>8 == uint32(r0) && g>>8 == uint32(g0) && b>>8 == uint32(b0) {
				return true
			}
		}
	}
	return false
}
