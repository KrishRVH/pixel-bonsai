package pane

import "bytes"

// An inputKind is the kind of an input event.
type inputKind int

// The input events the pane reads.
const (
	keyChar inputKind = iota
	keyUp
	keyDown
	keyLeft
	keyRight
	keyEnter
	keyEscape
	mouseClick
	mouseWheel
)

// An input is one key press or mouse event.
type input struct {
	Kind inputKind
	Char byte // the ASCII character, for keyChar
	X, Y int  // the cell clicked, zero-based, for a click
}

var arrows = []struct {
	seq  string
	kind inputKind
}{
	{"\x1b[A", keyUp},
	{"\x1b[B", keyDown},
	{"\x1b[C", keyRight},
	{"\x1b[D", keyLeft},
	{"\x1bOA", keyUp},
	{"\x1bOB", keyDown},
	{"\x1bOC", keyRight},
	{"\x1bOD", keyLeft},
}

// parseInput returns the events in a read of terminal input: ASCII characters, arrows, Enter, Escape, left clicks,
// and the wheel in SGR mouse reports. Other escape sequences, such as focus reports, and the bytes of non-ASCII
// characters are skipped.
func parseInput(data []byte) []input {
	var out []input
	for i := 0; i < len(data); {
		rest := data[i:]
		if in, n, ok := parseMouse(rest); ok {
			if in.Kind == mouseClick || in.Kind == mouseWheel {
				out = append(out, in)
			}
			i += n
			continue
		}
		if kind, n, ok := parseArrow(rest); ok {
			out = append(out, input{Kind: kind})
			i += n
			continue
		}
		if n := csiLength(rest); n > 0 {
			i += n
			continue
		}
		switch c := rest[0]; {
		case c == 0x1b:
			out = append(out, input{Kind: keyEscape})
		case c == '\r' || c == '\n':
			out = append(out, input{Kind: keyEnter})
		case c < 0x80:
			out = append(out, input{Kind: keyChar, Char: c})
		}
		i++
	}
	return out
}

func parseArrow(data []byte) (inputKind, int, bool) {
	for _, a := range arrows {
		if bytes.HasPrefix(data, []byte(a.seq)) {
			return a.kind, len(a.seq), true
		}
	}
	return 0, 0, false
}

// parseMouse reads a mouse report: SGR, ESC [ < button ; x ; y then M for a press or m for a release, or the older
// X10 form, ESC [ M then the button, x, and y each as one byte offset by 32, from terminals without SGR reports. A press
// of any button is a click and buttons 64 and 65 are the wheel; other reports parse with kind keyChar, which the
// caller drops.
func parseMouse(data []byte) (input, int, bool) {
	if bytes.HasPrefix(data, []byte("\x1b[M")) && len(data) >= 6 {
		return mouse(int(data[3])-32, int(data[4])-33, int(data[5])-33, true), 6, true
	}
	if !bytes.HasPrefix(data, []byte("\x1b[<")) {
		return input{}, 0, false
	}
	var nums [3]int
	k := 0
	for i := 3; i < len(data); i++ {
		c := data[i]
		switch {
		case '0' <= c && c <= '9':
			nums[k] = min(nums[k]*10+int(c-'0'), 1<<20)
		case c == ';' && k < 2:
			k++
		case (c == 'M' || c == 'm') && k == 2:
			return mouse(nums[0], nums[1]-1, nums[2]-1, c == 'M'), i + 1, true
		default:
			return input{}, 0, false
		}
	}
	return input{}, 0, false
}

// mouse returns the input a mouse report of a button at cell (x, y) is. The button's bits 4, 8, and 16 are the Shift,
// Alt, and Ctrl keys, which change nothing here; in X10 reports a release is button 3.
func mouse(button, x, y int, press bool) input {
	switch button &^ (4 | 8 | 16) {
	case 0, 1, 2:
		if press {
			return input{Kind: mouseClick, X: x, Y: y}
		}
	case 64, 65:
		return input{Kind: mouseWheel}
	}
	return input{Kind: keyChar}
}

// csiLength returns the length of a control sequence, ESC [ then parameters and one final letter or ~, or 0.
func csiLength(data []byte) int {
	if !bytes.HasPrefix(data, []byte("\x1b[")) {
		return 0
	}
	for i := 2; i < len(data); i++ {
		c := data[i]
		switch {
		case '0' <= c && c <= '9' || c == ';' || c == '?':
		case 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' || c == '~':
			return i + 1
		default:
			return 0
		}
	}
	return 0
}

// An action is what the inputs ask of the pane.
type action struct {
	Quit bool
	Step int  // varieties to step through: +1 next, -1 previous
	Flip bool // switch between today and lifetime
}

// act returns what a run of inputs asks for. q, Ctrl+C, and Ctrl+D quit. A click on the caption's variety, v, Tab, or
// the right arrow steps to the next variety, and the left arrow to the previous. The wheel does nothing. Any other
// key or click switches between today and lifetime.
func act(inputs []input, target clickTarget) action {
	var a action
	for _, in := range inputs {
		switch {
		case in.Kind == keyChar && (in.Char == 'q' || in.Char == 'Q' || in.Char == 0x03 || in.Char == 0x04):
			a.Quit = true
		case in.Kind == mouseWheel:
		case in.Kind == keyRight || in.Kind == keyChar && (in.Char == 'v' || in.Char == 'V' || in.Char == '\t'):
			a.Step++
		case in.Kind == keyLeft:
			a.Step--
		case in.Kind == mouseClick && target.hit(in.X, in.Y):
			a.Step++
		default:
			a.Flip = !a.Flip
		}
	}
	return a
}
