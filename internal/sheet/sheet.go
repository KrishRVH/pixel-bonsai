// Package sheet writes contact sheets of a variety, for reviewing its art without a terminal pane: labeled PNG grids
// of every stage and the pane's ANSI frames.
package sheet

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/KrishRVH/pixel-bonsai/art"
	"github.com/KrishRVH/pixel-bonsai/bonsai"
	"github.com/KrishRVH/pixel-bonsai/internal/ansi"
)

var (
	background = color.NRGBA{R: 0x1e, G: 0x1e, B: 0x2e, A: 0xff} // the terminal's background
	ink        = color.NRGBA{R: 0xb2, G: 0xaa, B: 0x98, A: 0xff} // the labels
)

const (
	columns    = 5
	gap        = 8
	labelZoom  = 2
	labelSpace = 12
)

// A shot is one labeled frame on a sheet.
type shot struct {
	label string
	g     float64
	scale float64
}

func stageShots(scale float64) []shot {
	out := make([]shot, art.Stages)
	for k := range out {
		s := art.Stage(k)
		out[k] = shot{fmt.Sprintf("%02d %s", k, s), s.Start() + art.Plus, scale}
	}
	return out
}

func viewShots() []shot {
	return []shot{{"100m day", bonsai.DayGrowth(100e6), 1}, {"13b lifetime", bonsai.LifeGrowth(13e9), 1}}
}

func scaleShots() []shot {
	g := bonsai.LifeGrowth(13e9)
	return []shot{{"13b at 0.6", g, 0.6}, {"13b at 1", g, 1}, {"13b at 2", g, 2}}
}

// Write writes a folder for the variety in dir, named for its id, and returns the files it wrote:
//
//   - stages.png: every stage just past its start, labeled, in a grid at the given scale
//   - day-100M.png and life-13B.png: the day's tree at 100M tokens and the lifetime tree at 13B
//   - scales.png: the 13B frame at scales 0.6, 1, and 2
//   - frames.txt: the same frames as the pane draws them, in ANSI half blocks
//
// Each pixel of a frame is zoom PNG pixels square.
func Write(dir string, v art.Variety, scale float64, zoom int) ([]string, error) {
	folder := filepath.Join(dir, v.ID)
	if err := os.MkdirAll(folder, 0o750); err != nil {
		return nil, fmt.Errorf("create sheet folder: %w", err)
	}
	views := viewShots()
	sheets := []struct {
		name  string
		shots []shot
	}{
		{"stages.png", stageShots(scale)},
		{"day-100M.png", views[:1]},
		{"life-13B.png", views[1:]},
		{"scales.png", scaleShots()},
	}
	var written []string
	for _, s := range sheets {
		path := filepath.Join(folder, s.name)
		if err := writePNG(path, grid(v, s.shots, zoom)); err != nil {
			return written, err
		}
		written = append(written, path)
	}
	path := filepath.Join(folder, "frames.txt")
	frames := frameText(v, append(append(stageShots(1), views...), scaleShots()...))
	if err := os.WriteFile(path, []byte(frames), 0o600); err != nil {
		return written, fmt.Errorf("write %s: %w", path, err)
	}
	return append(written, path), nil
}

// grid lays the shots out in rows of columns cells, each frame under its label.
func grid(v art.Variety, shots []shot, zoom int) *image.NRGBA {
	frames := make([]*art.Frame, len(shots))
	cellW, cellH := 0, 0
	for k, s := range shots {
		frames[k] = v.Render(s.g, s.scale)
		cellW, cellH = max(cellW, frames[k].Width()*zoom+gap), max(cellH, frames[k].Height()*zoom+labelSpace+gap)
	}
	cols := min(columns, len(shots))
	rows := (len(shots) + cols - 1) / cols
	img := image.NewNRGBA(image.Rect(0, 0, cellW*cols+gap, cellH*rows+gap))
	draw.Draw(img, img.Bounds(), image.NewUniform(background), image.Point{}, draw.Src)
	for k, f := range frames {
		x, y := gap+k%cols*cellW, gap+k/cols*cellH
		label(img, x, y, shots[k].label)
		for _, p := range f.Pixels() {
			r, g, b := p.Color.RGB()
			fill(img, x+p.X*zoom, y+labelSpace+p.Y*zoom, zoom, color.NRGBA{R: r, G: g, B: b, A: 0xff})
		}
	}
	return img
}

func fill(img *image.NRGBA, x, y, size int, c color.NRGBA) {
	draw.Draw(img, image.Rect(x, y, x+size, y+size), image.NewUniform(c), image.Point{}, draw.Src)
}

// label writes text in a 3×5 pixel font. Characters the font lacks are left blank.
func label(img *image.NRGBA, x, y int, text string) {
	for k, r := range strings.ToLower(text) {
		for j, row := range strings.Fields(font[r]) {
			for i, bit := range row {
				if bit == '#' {
					fill(img, x+(4*k+i)*labelZoom, y+j*labelZoom, labelZoom, ink)
				}
			}
		}
	}
}

func writePNG(path string, img image.Image) (err error) {
	f, err := os.Create(path) // #nosec G304 -- The caller chose the output folder.
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	return nil
}

// frameText returns the shots as the pane draws them: a heading and the half-block rows of each.
func frameText(v art.Variety, shots []shot) string {
	var b strings.Builder
	for _, s := range shots {
		fmt.Fprintf(&b, "== %s %s (age %.3f, scale %g) ==\n", v.ID, s.label, s.g, s.scale)
		for _, line := range ansi.HalfBlocks(v.Render(s.g, s.scale), ansi.TrueColor) {
			b.WriteString(line)
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// font holds the label glyphs: five rows of three pixels each, top to bottom, separated by spaces.
var font = map[rune]string{
	'a': ".#. #.# ### #.# #.#", 'b': "##. #.# ##. #.# ##.",
	'c': ".## #.. #.. #.. .##", 'd': "##. #.# #.# #.# ##.",
	'e': "### #.. ##. #.. ###", 'f': "### #.. ##. #.. #..",
	'g': ".## #.. #.# #.# .##", 'h': "#.# #.# ### #.# #.#",
	'i': "### .#. .#. .#. ###", 'j': "..# ..# ..# #.# .#.",
	'k': "#.# #.# ##. #.# #.#", 'l': "#.. #.. #.. #.. ###",
	'm': "#.# ### ### #.# #.#", 'n': "### #.# #.# #.# #.#",
	'o': ".#. #.# #.# #.# .#.", 'p': "##. #.# ##. #.. #..",
	'q': ".#. #.# #.# ##. .##", 'r': "##. #.# ##. #.# #.#",
	's': ".## #.. .#. ..# ##.", 't': "### .#. .#. .#. .#.",
	'u': "#.# #.# #.# #.# ###", 'v': "#.# #.# #.# #.# .#.",
	'w': "#.# #.# ### ### #.#", 'x': "#.# #.# .#. #.# #.#",
	'y': "#.# #.# .#. .#. .#.", 'z': "### ..# .#. #.. ###",
	'0': "### #.# #.# #.# ###", '1': ".#. ##. .#. .#. ###",
	'2': "##. ..# .#. #.. ###", '3': "##. ..# .#. ..# ##.",
	'4': "#.# #.# ### ..# ..#", '5': "### #.. ##. ..# ##.",
	'6': ".## #.. ### #.# ###", '7': "### ..# .#. .#. .#.",
	'8': "### #.# ### #.# ###", '9': "### #.# ### ..# ##.",
	'.': "... ... ... ... .#.", '-': "... ... ### ... ...",
	':': "... .#. ... .#. ...", '=': "... ### ... ### ...",
}
