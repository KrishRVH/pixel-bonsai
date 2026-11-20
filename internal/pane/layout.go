// Package pane is the live bonsai: the selected variety's day or lifetime tree as large as the terminal holds it,
// over a caption, redrawn as tokens are counted, with keys and clicks to switch variety and view.
package pane

import (
	"math"
	"strconv"
	"strings"
	"unicode"

	"github.com/KrishRVH/pixel-bonsai/art"
	"github.com/KrishRVH/pixel-bonsai/bonsai"
	"github.com/KrishRVH/pixel-bonsai/internal/ansi"
)

// The caption's colors.
const (
	gold    art.Color = 0xe5b850
	text    art.Color = 0xece4d2
	dim     art.Color = 0x7d7a73
	subtext art.Color = 0xb2aa98
)

// A scene is what the pane shows: a variety's tree in a view, grown from a token count.
type scene struct {
	Variety art.Variety
	View    bonsai.View
	Tokens  int64
}

// A clickTarget is the span of the caption a click selects the next variety from: on line Line, columns X0 to X1.
type clickTarget struct {
	Line, X0, X1 int
}

// hit reports whether cell (x, y) is in the target.
func (t clickTarget) hit(x, y int) bool {
	return y == t.Line && t.X0 <= x && x < t.X1
}

// layout lays out a scene in a pane width cells wide and height lines tall: the tree at the largest scale that fits,
// centered over its caption, or the caption alone when the tree does not fit. It also returns the caption's variety
// name as a click target.
func layout(s scene, width, height int, mode ansi.Mode) ([]string, clickTarget) {
	if height <= 0 || width <= 0 {
		return nil, clickTarget{Line: -1}
	}
	g := s.View.Growth(s.Tokens)
	name := s.Variety.Icon + " " + s.Variety.Name
	var parts []part
	for _, word := range []string{" tokens", ""} { // the word goes first when the caption is too wide
		parts = []part{
			{name, text},
			{" · ", dim},
			{art.StageAt(g).String(), subtext},
			{" · " + amount(s.Tokens) + word + " · ", dim},
			{string(s.View), gold},
			{" ⇄", dim},
		}
		if span(parts) <= width {
			break
		}
	}
	left := (width - min(width, span(parts))) / 2
	target := clickTarget{Line: height - 1, X0: left, X1: min(width, left+cells(name))}
	lines := make([]string, height)
	lines[height-1] = row(parts, width, mode)
	if scale := art.Fit(width, height); scale > 0 {
		w, _ := art.Size(scale)
		picture := ansi.HalfBlocks(s.Variety.Render(g, scale), mode)
		pad := strings.Repeat(" ", (width-w)/2)
		top := height - 1 - len(picture)
		for k, line := range picture {
			if top+k >= 0 {
				lines[top+k] = pad + line
			}
		}
	}
	return lines, target
}

// A part is a run of caption text in one color.
type part struct {
	text  string
	color art.Color
}

func span(parts []part) int {
	n := 0
	for _, p := range parts {
		n += cells(p.text)
	}
	return n
}

// row returns the parts as one centered line, clipped and padded to exactly width cells.
func row(parts []part, width int, mode ansi.Mode) string {
	var b strings.Builder
	used := 0
	for _, p := range parts {
		if used >= width {
			break
		}
		t := clip(p.text, width-used)
		b.WriteString(mode.Paint(p.color, t))
		used += cells(t)
	}
	left := (width - used) / 2
	return strings.Repeat(" ", left) + b.String() + strings.Repeat(" ", width-used-left)
}

// clip shortens text to width cells, ending it with an ellipsis when it is cut.
func clip(text string, width int) string {
	if cells(text) <= width {
		return text
	}
	var b strings.Builder
	used := 0
	for _, r := range text {
		if used+runeCells(r) > width-1 {
			break
		}
		b.WriteRune(r)
		used += runeCells(r)
	}
	return b.String() + "…"
}

// cells returns how many terminal cells text takes.
func cells(text string) int {
	n := 0
	for _, r := range text {
		n += runeCells(r)
	}
	return n
}

// runeCells returns the cells a rune takes: none for combining marks and joiners, two for emoji and East Asian wide
// characters, one otherwise.
func runeCells(r rune) int {
	switch {
	case unicode.Is(unicode.Mn, r) || r == 0x200d || r == 0xfe0f:
		return 0
	case r >= 0x1f000,
		0x1100 <= r && r <= 0x115f, 0x2e80 <= r && r <= 0xa4cf, 0xac00 <= r && r <= 0xd7a3,
		0xf900 <= r && r <= 0xfaff, 0xfe30 <= r && r <= 0xfe4f, 0xff00 <= r && r <= 0xff60, 0xffe0 <= r && r <= 0xffe6:
		return 2
	}
	return 1
}

// amount returns a token count in short form: 1234567 is "1.2M", and 999950 is "1.0M" rather than "1000.0K".
func amount(n int64) string {
	units := []struct {
		size float64
		unit string
	}{{1e3, "K"}, {1e6, "M"}, {1e9, "B"}, {1e12, "T"}}
	k := len(units) - 1
	for k >= 0 && float64(n) < units[k].size {
		k--
	}
	if k < 0 {
		return strconv.FormatInt(n, 10)
	}
	v := math.Round(float64(n)/units[k].size*10) / 10
	if v >= 1000 && k < len(units)-1 {
		k++
		v = math.Round(float64(n)/units[k].size*10) / 10
	}
	return strconv.FormatFloat(v, 'f', 1, 64) + units[k].unit
}
