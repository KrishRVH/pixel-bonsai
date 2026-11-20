package pane

import (
	"bytes"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/KrishRVH/pixel-bonsai/art"
	"github.com/KrishRVH/pixel-bonsai/bonsai"
	"github.com/KrishRVH/pixel-bonsai/internal/ansi"
)

var escapes = regexp.MustCompile("\x1b\\[[0-9;]*m")

// plain returns a line without its color codes.
func plain(line string) string {
	return escapes.ReplaceAllString(line, "")
}

func variety(t *testing.T, id string) art.Variety {
	t.Helper()
	v, ok := bonsai.Lookup(id)
	if !ok {
		t.Fatalf("no variety %q", id)
	}
	return v
}

func caption(t *testing.T, id string, view bonsai.View, tokens int64, width int) string {
	t.Helper()
	lines, _ := layout(scene{variety(t, id), view, tokens}, width, 20, ansi.NoColor)
	return lines[len(lines)-1]
}

// drawn returns the lines that hold the tree: every line but the caption that is not empty.
func drawn(lines []string) []string {
	var out []string
	for _, l := range lines[:len(lines)-1] {
		if l != "" {
			out = append(out, plain(l))
		}
	}
	return out
}

func TestTheCaptionNamesVarietyStageAmountAndView(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		id     string
		view   bonsai.View
		tokens int64
		want   string
	}{
		{"sakura", bonsai.Lifetime, 13e9, "🌸 sakura · shaped · 13.0B tokens · lifetime ⇄"},
		{"maple", bonsai.Today, 100e6, "🍁 maple · first branches · 100.0M tokens · today ⇄"},
		{"black-pine", bonsai.Today, 0, "🌲 black pine · seed · 0 tokens · today ⇄"},
	} {
		if got := strings.TrimSpace(caption(t, c.id, c.view, c.tokens, 60)); got != c.want {
			t.Errorf("caption = %q, want %q", got, c.want)
		}
	}
}

func TestANarrowCaptionDropsTheWordTokensFirst(t *testing.T) {
	t.Parallel()
	if got := strings.TrimSpace(caption(t, "sakura", bonsai.Lifetime, 13e9, 45)); got != "🌸 sakura · shaped · 13.0B · lifetime ⇄" {
		t.Errorf("caption at 45 = %q", got)
	}
	got := caption(t, "sakura", bonsai.Lifetime, 13e9, 20)
	if cells(got) != 20 || !strings.Contains(got, "sakura") || !strings.HasSuffix(got, "…") {
		t.Errorf("caption at 20 = %q (%d cells), want it clipped to 20 with an ellipsis", got, cells(got))
	}
	for width := 1; width <= 60; width++ {
		if got := caption(t, "black-pine", bonsai.Today, 4e11, width); cells(got) != width {
			t.Fatalf("caption at %d is %d cells: %q", width, cells(got), got)
		}
	}
}

func TestTheVarietyInTheCaptionIsAClickTarget(t *testing.T) {
	t.Parallel()
	lines, target := layout(scene{variety(t, "sakura"), bonsai.Lifetime, 13e9}, 60, 20, ansi.TrueColor)
	line := plain(lines[19])
	at := strings.Index(line, "🌸")
	if at < 0 {
		t.Fatalf("caption %q has no icon", line)
	}
	x0 := cells(line[:at])
	if want := (clickTarget{Line: 19, X0: x0, X1: x0 + cells("🌸 sakura")}); target != want {
		t.Fatalf("target = %+v, want %+v", target, want)
	}
	if !target.hit(x0, 19) || target.hit(target.X1, 19) || target.hit(x0, 18) {
		t.Error("the target is not exactly the variety's name")
	}
	if lines, target := layout(scene{variety(t, "sakura"), bonsai.Today, 0}, 60, 0, ansi.TrueColor); lines != nil || target.hit(0, -1) {
		t.Error("a pane with no rows draws something or has a target")
	}
}

func TestABiggerPaneDrawsABiggerCanvas(t *testing.T) {
	t.Parallel()
	sc := scene{variety(t, "sakura"), bonsai.Today, 688e6}
	small, _ := layout(sc, 54, 20, ansi.TrueColor)
	big, _ := layout(sc, 100, 50, ansi.TrueColor)
	if len(small) != 20 || len(big) != 50 {
		t.Fatalf("heights %d and %d, want 20 and 50", len(small), len(big))
	}
	if len(drawn(small)) != 17 || len(drawn(big)) != 30 {
		t.Errorf("tree rows %d and %d, want 17 and 30: scale 1.125, then 2", len(drawn(small)), len(drawn(big)))
	}
	widest := func(lines []string) int {
		n := 0
		for _, l := range lines {
			n = max(n, cells(l))
		}
		return n
	}
	if widest(drawn(small)) > 54 || widest(drawn(big)) <= 90 {
		t.Errorf("widths %d and %d", widest(drawn(small)), widest(drawn(big)))
	}
	if tiny, _ := layout(sc, 54, 9, ansi.TrueColor); len(drawn(tiny)) != 8 {
		t.Errorf("the smallest tree has %d rows, want 8 at half scale", len(drawn(tiny)))
	}
}

func TestTheTreeIsTheSelectedVarietyAtTheViewsAge(t *testing.T) {
	t.Parallel()
	pictures := map[string]bool{}
	for _, v := range bonsai.Varieties() {
		lines, _ := layout(scene{v, bonsai.Lifetime, 13e9}, 60, 24, ansi.TrueColor)
		pictures[strings.Join(lines[:23], "\n")] = true
	}
	if len(pictures) != len(bonsai.Varieties()) {
		t.Error("the varieties do not draw different pictures")
	}
	maple := variety(t, "maple")
	day, _ := layout(scene{maple, bonsai.Today, 100e6}, 60, 24, ansi.TrueColor)
	life, _ := layout(scene{maple, bonsai.Lifetime, 100e6}, 60, 24, ansi.TrueColor)
	if slices.Equal(day[:23], life[:23]) {
		t.Error("the same tokens draw the same tree in both views")
	}
	scale := art.Fit(60, 24)
	w, _ := art.Size(scale)
	want := ansi.HalfBlocks(maple.Render(bonsai.LifeGrowth(13e9), scale), ansi.TrueColor)
	got, _ := layout(scene{maple, bonsai.Lifetime, 13e9}, 60, 24, ansi.TrueColor)
	pad := strings.Repeat(" ", (60-w)/2)
	for k, line := range want {
		if got[23-len(want)+k] != pad+line {
			t.Fatalf("row %d is not the centered frame just above the caption", k)
		}
	}
}

func TestTooSmallAPaneShowsTheCaptionAloneAndNoColorASilhouette(t *testing.T) {
	t.Parallel()
	sc := scene{variety(t, "sakura"), bonsai.Today, 688e6}
	short, _ := layout(sc, 54, 3, ansi.TrueColor)
	narrow, _ := layout(sc, 20, 40, ansi.TrueColor)
	if short[0] != "" || short[1] != "" || narrow[38] != "" {
		t.Error("a pane too small for the tree draws some of it")
	}
	plainLines, _ := layout(sc, 54, 20, ansi.NoColor)
	body := strings.Join(plainLines, "\n")
	if strings.Contains(body, "\x1b") || !strings.Contains(body, "█") {
		t.Error("no color is not a plain silhouette")
	}
}

func TestEveryVarietyAndViewDrawsAtAnyPaneSize(t *testing.T) {
	t.Parallel()
	for _, view := range []bonsai.View{bonsai.Today, bonsai.Lifetime} {
		for _, v := range bonsai.Varieties() {
			for _, tokens := range []int64{0, 1e6, 3e8, 5e9, 13e9, 4e11} {
				for _, size := range [][2]int{{54, 18}, {80, 30}, {130, 70}, {1, 1}, {7, 2}} {
					checkFits(t, scene{v, view, tokens}, size[0], size[1])
				}
			}
		}
	}
}

// checkFits checks that a scene's lines fill a pane's height and none is wider than the pane.
func checkFits(t *testing.T, s scene, width, height int) {
	t.Helper()
	lines, _ := layout(s, width, height, ansi.TrueColor)
	if len(lines) != height {
		t.Fatalf("%+v at %d×%d: %d lines", s, width, height, len(lines))
	}
	for _, l := range lines {
		if cells(plain(l)) > width {
			t.Fatalf("%s %s %d at %d×%d: a line of %d cells", s.Variety.ID, s.View, s.Tokens, width, height, cells(plain(l)))
		}
	}
}

func TestTheAmountIsShort(t *testing.T) {
	t.Parallel()
	for n, want := range map[int64]string{0: "0", 999: "999", 1000: "1.0K", 999949: "999.9K", 999950: "1.0M", 1234567: "1.2M", 13e9: "13.0B", 2.5e12: "2.5T", 5e15: "5000.0T"} {
		if got := amount(n); got != want {
			t.Errorf("amount(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestInputDecodesKeysArrowsAndMouseReportsAcrossReads(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		reads []string
		want  []input
	}{
		{[]string{"\x1b[C\x1b[D\x1bOC\x1bOD"}, []input{{Kind: keyRight}, {Kind: keyLeft}, {Kind: keyRight}, {Kind: keyLeft}}},
		{[]string{"\x1b[A\x1b[B\t"}, []input{{Kind: keyUp}, {Kind: keyDown}, {Kind: keyChar, Char: '\t'}}},
		{[]string{"\x1b[<0;5;20M\x1b[<0;5;20m"}, []input{{Kind: mouseClick, X: 4, Y: 19}}},
		{[]string{"\x1b[<64;1;1M\x1b[<65;1;1M\x1b[<35;3;3M"}, []input{{Kind: mouseWheel}, {Kind: mouseWheel}}},
		{[]string{"\x1b[<2;3;4M\x1b[<16;1;1M"}, []input{{Kind: mouseClick, X: 2, Y: 3}, {Kind: mouseClick}}},     // right, Ctrl+left
		{[]string{"\x1b[M Qa\x1b[M#Qa\x1b[M`!!"}, []input{{Kind: mouseClick, X: 48, Y: 64}, {Kind: mouseWheel}}}, // X10: press, release, wheel
		{[]string{"\x1b[I\x1b[O\x1b[1;5C\x1b[15~\x1b[<-1;5;20M"}, nil},
		{[]string{"\x1bOQ\x1bOP\x1b[[A"}, nil}, // F2 and F1 on the Windows console, and F1 on the Linux console
		{[]string{"\x1b"}, []input{{Kind: keyEscape}}},
		{[]string{"\x1b\x1b[C"}, []input{{Kind: keyEscape}, {Kind: keyRight}}},
		{[]string{"\x1b[1;"}, nil}, // unfinished when input pauses
		{[]string{"\x1b[", "C"}, []input{{Kind: keyRight}}},
		{[]string{"\x1b", "OD"}, []input{{Kind: keyLeft}}},
		{[]string{"\x1b[<0;5", ";20M"}, []input{{Kind: mouseClick, X: 4, Y: 19}}},
		{[]string{"\x1b[M ", "Qa"}, []input{{Kind: mouseClick, X: 48, Y: 64}}},
		{[]string{"\r\n"}, []input{{Kind: keyEnter}, {Kind: keyEnter}}},
		{[]string{"é q"}, []input{{Kind: keyChar, Char: ' '}, {Kind: keyChar, Char: 'q'}}},
	} {
		var d decoder
		var got []input
		for _, r := range c.reads {
			got = append(got, d.decode([]byte(r))...)
		}
		got = append(got, d.idle()...) // input pauses
		if !slices.Equal(got, c.want) {
			t.Errorf("decode(%q) = %v, want %v", c.reads, got, c.want)
		}
	}
}

// FuzzDecoder checks that input decodes the same however reads split it.
func FuzzDecoder(f *testing.F) {
	for _, seed := range []string{"q", "\x1b[<0;5;20M", "\x1b[<64;99999999999;1M", "\x1b[A\x1bOD", "\x1b[<", "\x1b[1;", "\x1bOQ", "\x1b[M a", "\x1b[[A"} {
		f.Add([]byte(seed), uint8(2))
	}
	f.Fuzz(func(t *testing.T, data []byte, cut uint8) {
		var whole, split decoder
		want := whole.decode(data)
		k := int(cut) % (len(data) + 1)
		got := append(split.decode(data[:k]), split.decode(data[k:])...)
		if !slices.Equal(got, want) || !bytes.Equal(split.pending, whole.pending) {
			t.Errorf("%q split at %d decoded to %v and kept %q, whole to %v and kept %q", data, k, got, split.pending, want, whole.pending)
		}
		if len(want) > len(data) {
			t.Errorf("%q decoded to %d inputs", data, len(want))
		}
	})
}

var target = clickTarget{Line: 19, X0: 4, X1: 15}

func TestTheCaptionClickVTabAndTheArrowsStepThroughTheVarieties(t *testing.T) {
	t.Parallel()
	char := func(c byte) input { return input{Kind: keyChar, Char: c} }
	for _, c := range []struct {
		inputs []input
		step   int
	}{
		{[]input{char('v')}, 1},
		{[]input{char('V')}, 1},
		{[]input{char('\t')}, 1},
		{[]input{{Kind: keyRight}}, 1},
		{[]input{{Kind: keyLeft}}, -1},
		{[]input{{Kind: mouseClick, X: 4, Y: 19}}, 1},
		{[]input{char('v'), char('v'), {Kind: keyLeft}}, 1},
	} {
		if a := act(c.inputs, target); a.Step != c.step || a.Flip || a.Quit {
			t.Errorf("act(%v) = %+v, want step %d", c.inputs, a, c.step)
		}
	}
}

func TestAnyOtherClickOrKeySwitchesTheViewAndTheWheelNothing(t *testing.T) {
	t.Parallel()
	for _, in := range []input{
		{Kind: mouseClick, X: 3, Y: 4},
		{Kind: mouseClick, X: 15, Y: 19},
		{Kind: mouseClick, X: 4, Y: 18},
		{Kind: keyChar, Char: ' '},
		{Kind: keyChar, Char: 'x'},
		{Kind: keyUp},
		{Kind: keyDown},
		{Kind: keyEnter},
		{Kind: keyEscape},
	} {
		if a := act([]input{in}, target); !a.Flip || a.Step != 0 || a.Quit {
			t.Errorf("act(%v) = %+v, want a flip", in, a)
		}
	}
	if a := act([]input{{Kind: mouseWheel}}, target); a != (action{}) {
		t.Errorf("the wheel acts: %+v", a)
	}
	if a := act([]input{{Kind: keyChar, Char: ' '}, {Kind: keyChar, Char: ' '}}, target); a.Flip {
		t.Error("two flips do not cancel")
	}
	if a := act([]input{{Kind: keyChar, Char: 'v'}, {Kind: keyChar, Char: ' '}}, target); a.Step != 1 || !a.Flip {
		t.Errorf("v then space = %+v, want both", a)
	}
	for _, c := range []byte{'q', 'Q', 0x03, 0x04} {
		if !act([]input{{Kind: keyChar, Char: c}}, target).Quit {
			t.Errorf("%q does not quit", c)
		}
	}
}
