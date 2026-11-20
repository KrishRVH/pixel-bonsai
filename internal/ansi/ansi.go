// Package ansi turns frames into terminal text: half-block cells colored with escape codes at the depth the terminal
// supports.
package ansi

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/KrishRVH/pixel-bonsai/art"
)

// A Mode is a terminal's color depth.
type Mode int

// The color depths a frame can be drawn at. NoColor draws the tree's silhouette.
const (
	TrueColor Mode = iota
	Color256
	NoColor
)

// ParseMode returns the mode a --color value names: "truecolor" (or "24bit"), "256", or "none".
func ParseMode(s string) (Mode, error) {
	if s == "24bit" {
		return TrueColor, nil
	}
	for m, name := range modeNames {
		if s == name {
			return Mode(m), nil
		}
	}
	return 0, fmt.Errorf("unknown color mode %q: use truecolor, 256, or none", s)
}

var modeNames = [...]string{TrueColor: "truecolor", Color256: "256", NoColor: "none"}

// String returns the mode's name, as ParseMode reads it.
func (m Mode) String() string {
	return modeNames[m]
}

// Detect returns the color depth the environment asks for: none with NO_COLOR or a dumb terminal, 24-bit with
// COLORTERM=truecolor, 256 colors in Apple's Terminal, and 24-bit otherwise, which every current terminal on Windows,
// macOS, and Linux supports.
func Detect(getenv func(string) string) Mode {
	switch {
	case getenv("NO_COLOR") != "" || getenv("TERM") == "dumb":
		return NoColor
	case getenv("COLORTERM") == "truecolor" || getenv("COLORTERM") == "24bit":
		return TrueColor
	case getenv("TERM_PROGRAM") == "Apple_Terminal":
		return Color256
	}
	return TrueColor
}

// fg returns the escape code that sets the foreground color, or "" in NoColor mode.
func (m Mode) fg(c art.Color) string {
	return m.sgr(38, c)
}

// bg returns the escape code that sets the background color, or "" in NoColor mode.
func (m Mode) bg(c art.Color) string {
	return m.sgr(48, c)
}

func (m Mode) sgr(layer int, c art.Color) string {
	switch m {
	case TrueColor:
		r, g, b := c.RGB()
		return "\x1b[" + strconv.Itoa(layer) + ";2;" + strconv.Itoa(int(r)) + ";" + strconv.Itoa(int(g)) + ";" + strconv.Itoa(int(b)) + "m"
	case Color256:
		return "\x1b[" + strconv.Itoa(layer) + ";5;" + strconv.Itoa(xterm256(c)) + "m"
	}
	return ""
}

// reset is the escape code that clears all colors and styles.
const reset = "\x1b[0m"

// Paint returns text in a color followed by a reset, or the text alone in NoColor mode.
func (m Mode) Paint(c art.Color, text string) string {
	if m == NoColor {
		return text
	}
	return m.fg(c) + text + reset
}

// xterm256 returns the nearest color of the xterm 256-color palette: the 6×6×6 cube or the 24-step gray ramp.
func xterm256(c art.Color) int {
	r, g, b := c.RGB()
	levels := [6]int{0, 95, 135, 175, 215, 255}
	step := func(v uint8) int {
		switch {
		case v < 48:
			return 0
		case v < 115:
			return 1
		}
		return (int(v) - 35) / 40
	}
	cr, cg, cb := step(r), step(g), step(b)
	cube := 16 + 36*cr + 6*cg + cb
	cubeDist := distance(r, g, b, levels[cr], levels[cg], levels[cb])
	avg := (int(r) + int(g) + int(b)) / 3
	gray := max(0, min(23, (avg-3)/10))
	level := 8 + 10*gray
	if distance(r, g, b, level, level, level) < cubeDist {
		return 232 + gray
	}
	return cube
}

func distance(r, g, b uint8, r2, g2, b2 int) int {
	dr, dg, db := int(r)-r2, int(g)-g2, int(b)-b2
	return dr*dr + dg*dg + db*db
}

// HalfBlocks returns a frame as terminal lines, two stacked pixels to a cell: the upper pixel as the glyph's color
// and the lower as its background. A frame of odd height leaves its last pixel row out.
func HalfBlocks(f *art.Frame, m Mode) []string {
	lines := make([]string, f.Height()/2)
	var b strings.Builder
	for r := range lines {
		b.Reset()
		p := pen{mode: m}
		for x := range f.Width() {
			top, hasTop := f.Pixel(x, 2*r)
			low, hasLow := f.Pixel(x, 2*r+1)
			switch {
			case hasTop && hasLow && (top == low || m == NoColor):
				p.fill(&b, top)
				b.WriteString("█")
			case hasTop && hasLow:
				p.fill(&b, top)
				p.back(&b, low)
				b.WriteString("▀")
			case hasTop:
				p.fill(&b, top)
				p.clear(&b)
				b.WriteString("▀")
			case hasLow:
				p.fill(&b, low)
				p.clear(&b)
				b.WriteString("▄")
			default:
				p.clear(&b)
				b.WriteByte(' ')
			}
		}
		if p.hasFg || p.hasBg {
			b.WriteString(reset)
		}
		lines[r] = b.String()
	}
	return lines
}

// A pen tracks the colors a line has set, so a run of cells in one color carries one escape code.
type pen struct {
	mode         Mode
	fg, bg       art.Color
	hasFg, hasBg bool
}

// fill sets the glyph color.
func (p *pen) fill(b *strings.Builder, c art.Color) {
	if p.mode != NoColor && (!p.hasFg || p.fg != c) {
		b.WriteString(p.mode.fg(c))
		p.fg, p.hasFg = c, true
	}
}

// back sets the background color.
func (p *pen) back(b *strings.Builder, c art.Color) {
	if p.mode != NoColor && (!p.hasBg || p.bg != c) {
		b.WriteString(p.mode.bg(c))
		p.bg, p.hasBg = c, true
	}
}

// clear returns the background to the terminal's own.
func (p *pen) clear(b *strings.Builder) {
	if p.hasBg {
		b.WriteString("\x1b[49m")
		p.hasBg = false
	}
}
