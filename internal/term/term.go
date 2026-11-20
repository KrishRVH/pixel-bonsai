// Package term runs the terminal for the live pane on Windows, macOS, Linux, and the BSDs: raw input, escape-code
// output, its size, and input reads that wait no longer than a timeout, so the reader can stop when the pane does.
package term

import (
	"errors"
	"fmt"
	"os"
	"time"

	"golang.org/x/term"
)

// A Term is an open terminal. Close restores the modes Open changed.
type Term struct {
	in, out    *os.File
	saved      *term.State // the input mode before raw mode; nil when input is not a terminal
	restoreOut func() error
}

// isTerminal reports whether f is a terminal.
func isTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd())) // #nosec G115 -- File descriptors fit in an int.
}

// Open puts the terminal in the modes the pane needs: raw input when in is a terminal, so keys and clicks arrive at
// once without echo, and escape-code processing on out. out must be a terminal.
func Open(in, out *os.File) (*Term, error) {
	if !isTerminal(out) {
		return nil, errors.New("the live bonsai needs a terminal: use render for one frame")
	}
	restoreOut, err := enableEscapes(out)
	if err != nil {
		return nil, fmt.Errorf("enable escape codes: %w", err)
	}
	t := &Term{in: in, out: out, restoreOut: restoreOut}
	if isTerminal(in) {
		t.saved, err = term.MakeRaw(int(in.Fd())) // #nosec G115 -- File descriptors fit in an int.
		if err != nil {
			return nil, errors.Join(fmt.Errorf("raw input: %w", err), restoreOut())
		}
		_ = allowMouse(in) // clicks are a shortcut: every action has a key, so the pane runs without them
	}
	return t, nil
}

// Interactive reports whether the terminal has input to read.
func (t *Term) Interactive() bool {
	return t.saved != nil
}

// Write writes to the terminal.
func (t *Term) Write(p []byte) (int, error) {
	return t.out.Write(p)
}

// Size returns the terminal's width and height in cells.
func (t *Term) Size() (width, height int, err error) {
	return term.GetSize(int(t.out.Fd())) // #nosec G115 -- File descriptors fit in an int.
}

// Read reads input, waiting at most timeout for it to arrive. It returns 0 bytes and no error when none did.
func (t *Term) Read(buf []byte, timeout time.Duration) (int, error) {
	if t.saved == nil {
		time.Sleep(timeout)
		return 0, nil
	}
	return readInput(t.in, buf, timeout)
}

// Close restores the terminal's modes.
func (t *Term) Close() error {
	var err error
	if t.saved != nil {
		err = term.Restore(int(t.in.Fd()), t.saved) // #nosec G115 -- File descriptors fit in an int.
	}
	return errors.Join(err, t.restoreOut())
}
