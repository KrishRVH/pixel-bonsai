package pane

import (
	"strconv"
	"strings"
)

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

// A decoder reads terminal input as a stream: ASCII characters, arrows, Enter, Escape, and the clicks and wheel turns
// of mouse reports. It skips other escape sequences, such as focus reports and function keys, and the bytes of
// non-ASCII characters. A read can end partway through an escape sequence, so the decoder keeps an unfinished one
// until the next read finishes it.
type decoder struct {
	pending []byte // the start of an unfinished escape sequence
}

// decode returns the inputs that data finishes.
func (d *decoder) decode(data []byte) []input {
	data = append(d.pending, data...)
	d.pending = nil
	var out []input
	for len(data) > 0 {
		in, n, ok := next(data)
		if n == 0 {
			d.pending = data
			break
		}
		if ok {
			out = append(out, in)
		}
		data = data[n:]
	}
	return out
}

// idle returns what an unfinished sequence was once input pauses: a lone ESC is the Escape key, and anything longer
// is dropped.
func (d *decoder) idle() []input {
	lone := len(d.pending) == 1
	d.pending = nil
	if lone {
		return []input{{Kind: keyEscape}}
	}
	return nil
}

// maxSequence bounds an escape sequence the decoder waits for. A longer one is no key or mouse report.
const maxSequence = 32

// next reads the input at the start of data. It returns the bytes it used, 0 when data ends partway through an
// escape sequence, and false for bytes that are no input the pane reads.
func next(data []byte) (input, int, bool) {
	switch c := data[0]; {
	case c == 0x1b:
		return escape(data)
	case c == '\r' || c == '\n':
		return input{Kind: keyEnter}, 1, true
	case c < 0x80:
		return input{Kind: keyChar, Char: c}, 1, true
	}
	return input{}, 1, false
}

// escape reads what starts with ESC: a control sequence, ESC [; ESC O and one byte, as some terminals send arrows and
// function keys; or else the Escape key.
func escape(data []byte) (input, int, bool) {
	switch {
	case len(data) == 1:
		return input{}, 0, false
	case data[1] == '[':
		return control(data)
	case data[1] != 'O':
		return input{Kind: keyEscape}, 1, true
	case len(data) == 2:
		return input{}, 0, false
	}
	kind, ok := arrow(data[2])
	return input{Kind: kind}, 3, ok
}

// arrow returns the arrow key a sequence's final byte names.
func arrow(final byte) (inputKind, bool) {
	switch final {
	case 'A':
		return keyUp, true
	case 'B':
		return keyDown, true
	case 'C':
		return keyRight, true
	case 'D':
		return keyLeft, true
	}
	return 0, false
}

// control reads a control sequence: ESC [, parameter and intermediate bytes, and a final byte. Terminals without SGR
// mouse reports send the older X10 form instead: ESC [ M, then the button, x, and y each as one byte offset by 32. The
// Linux console sends F1 to F5 as ESC [ [ and a letter.
func control(data []byte) (input, int, bool) {
	if len(data) > 2 && data[2] == '[' {
		if len(data) < 4 {
			return input{}, 0, false
		}
		return input{}, 4, false
	}
	if len(data) > 2 && data[2] == 'M' {
		if len(data) < 6 {
			return input{}, 0, false
		}
		in, ok := mouse(int(data[3])-32, int(data[4])-33, int(data[5])-33, true)
		return in, 6, ok
	}
	end := min(len(data), maxSequence)
	i := 2
	for i < end && 0x20 <= data[i] && data[i] <= 0x3f {
		i++
	}
	switch {
	case i == maxSequence:
		return input{}, i, false
	case i == len(data):
		return input{}, 0, false
	case data[i] < 0x40 || data[i] > 0x7e:
		return input{}, i, false // no sequence after all: the stray byte reads on its own
	}
	params, final := string(data[2:i]), data[i]
	switch {
	case params == "":
		kind, ok := arrow(final)
		return input{Kind: kind}, i + 1, ok
	case final == 'M' || final == 'm':
		in, ok := sgrMouse(params, final == 'M')
		return in, i + 1, ok
	}
	return input{}, i + 1, false
}

// sgrMouse reads an SGR mouse report's parameters: <, then the button, x, and y separated by semicolons. The final
// byte M is a press and m a release.
func sgrMouse(params string, press bool) (input, bool) {
	rest, ok := strings.CutPrefix(params, "<")
	fields := strings.Split(rest, ";")
	if !ok || len(fields) != 3 {
		return input{}, false
	}
	var n [3]int
	for k, f := range fields {
		v, err := strconv.Atoi(f)
		if err != nil || v < 0 {
			return input{}, false
		}
		n[k] = v
	}
	return mouse(n[0], n[1]-1, n[2]-1, press)
}

// mouse returns the input a report of a button at cell (x, y) is: a press of any button is a click, and buttons 64
// and 65 are the wheel. The button's bits 4, 8, and 16 are the Shift, Alt, and Ctrl keys, which change nothing here;
// in X10 reports a release is button 3.
func mouse(button, x, y int, press bool) (input, bool) {
	switch button &^ (4 | 8 | 16) {
	case 0, 1, 2:
		return input{Kind: mouseClick, X: x, Y: y}, press
	case 64, 65:
		return input{Kind: mouseWheel}, true
	}
	return input{}, false
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
