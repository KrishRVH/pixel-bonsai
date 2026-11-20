package ansi_test

import (
	"strings"
	"testing"

	"github.com/KrishRVH/pixel-bonsai/art"
	"github.com/KrishRVH/pixel-bonsai/internal/ansi"
)

const (
	a art.Color = 0xaa0000
	b art.Color = 0x00bb00
	c art.Color = 0x0000cc
	d art.Color = 0xdddddd
)

func frame() *art.Frame {
	cv := art.NewCanvas(1)
	for _, p := range []struct {
		x, y int
		col  art.Color
	}{{0, 0, a}, {0, 1, b}, {1, 0, a}, {2, 1, c}, {3, 0, d}, {3, 1, d}} {
		cv.Put(p.x, p.y, p.col)
	}
	return cv.Frame()
}

func TestHalfBlocksStackTwoPixelsToACell(t *testing.T) {
	t.Parallel()
	m := ansi.TrueColor
	lines := ansi.HalfBlocks(frame(), m)
	if len(lines) != art.Rows {
		t.Fatalf("%d lines, want %d", len(lines), art.Rows)
	}
	want := "\x1b[38;2;170;0;0m\x1b[48;2;0;187;0m▀" + "\x1b[49m▀" + "\x1b[38;2;0;0;204m▄" + "\x1b[38;2;221;221;221m█" +
		strings.Repeat(" ", art.W-4) + "\x1b[0m"
	if lines[0] != want {
		t.Errorf("line 0 = %q, want %q: one code per change of color", lines[0], want)
	}
	if lines[1] != strings.Repeat(" ", art.W) {
		t.Errorf("an empty row = %q, want plain spaces", lines[1])
	}
	if got := ansi.HalfBlocks(frame(), ansi.NoColor)[0]; got != "█▀▄█"+strings.Repeat(" ", art.W-4) {
		t.Errorf("no color = %q, want the silhouette", got)
	}
}

func TestColorCodesForEachMode(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mode ansi.Mode
		col  art.Color
		fg   string
	}{
		{ansi.TrueColor, 0x0a0b0c, "\x1b[38;2;10;11;12m"},
		{ansi.Color256, 0xff0000, "\x1b[38;5;196m"},
		{ansi.Color256, 0x808080, "\x1b[38;5;244m"},
		{ansi.Color256, 0xffffff, "\x1b[38;5;231m"},
		{ansi.Color256, 0x000000, "\x1b[38;5;16m"},
		{ansi.NoColor, 0xffffff, ""},
	} {
		want := "x"
		if tc.fg != "" {
			want = tc.fg + "x\x1b[0m"
		}
		if got := tc.mode.Paint(tc.col, "x"); got != want {
			t.Errorf("%v Paint(%v) = %q, want %q", tc.mode, tc.col, got, want)
		}
	}
}

func TestTheModeComesFromTheEnvironmentOrAName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		env  map[string]string
		want ansi.Mode
	}{
		{map[string]string{}, ansi.TrueColor},
		{map[string]string{"NO_COLOR": "1", "COLORTERM": "truecolor"}, ansi.NoColor},
		{map[string]string{"TERM": "dumb"}, ansi.NoColor},
		{map[string]string{"COLORTERM": "24bit", "TERM_PROGRAM": "Apple_Terminal"}, ansi.TrueColor},
		{map[string]string{"TERM_PROGRAM": "Apple_Terminal"}, ansi.Color256},
	} {
		if got := ansi.Detect(func(k string) string { return tc.env[k] }); got != tc.want {
			t.Errorf("Detect(%v) = %v, want %v", tc.env, got, tc.want)
		}
	}
	for _, m := range []ansi.Mode{ansi.TrueColor, ansi.Color256, ansi.NoColor} {
		if got, err := ansi.ParseMode(m.String()); err != nil || got != m {
			t.Errorf("ParseMode(%q) = %v, %v", m.String(), got, err)
		}
	}
	if _, err := ansi.ParseMode("sepia"); err == nil {
		t.Error("an unknown mode parses")
	}
}
