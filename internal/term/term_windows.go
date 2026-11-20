//go:build windows

package term

import (
	"os"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

// enableEscapes turns on escape-code processing for the console, which Windows 10 and later support, and returns a
// function that restores the console's mode. Output that is not a console needs nothing.
func enableEscapes(out *os.File) (func() error, error) {
	h := windows.Handle(out.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(h, &mode); err != nil {
		return func() error { return nil }, nil //nolint:nilerr // Not a console: a pipe or a terminal emulator's pty.
	}
	if err := windows.SetConsoleMode(h, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING|windows.ENABLE_PROCESSED_OUTPUT); err != nil {
		return nil, err
	}
	return func() error { return windows.SetConsoleMode(h, mode) }, nil
}

// allowMouse turns off the console's QuickEdit mode, in which a click selects text instead of reaching the program,
// and turns on mouse input. Close restores the mode raw input saved, QuickEdit included.
func allowMouse(in *os.File) error {
	h := windows.Handle(in.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(h, &mode); err != nil {
		return err
	}
	return windows.SetConsoleMode(h, mode&^windows.ENABLE_QUICK_EDIT_MODE|windows.ENABLE_EXTENDED_FLAGS|windows.ENABLE_MOUSE_INPUT)
}

// readConsoleInput is kernel32's ReadConsoleInputW, which golang.org/x/sys/windows does not wrap.
var readConsoleInput = windows.NewLazySystemDLL("kernel32.dll").NewProc("ReadConsoleInputW")

// inputRecord is the console's INPUT_RECORD holding a KEY_EVENT_RECORD, the only kind read here.
type inputRecord struct {
	eventType uint16
	_         uint16
	keyDown   int32
	repeat    uint16
	keyCode   uint16
	scanCode  uint16
	char      uint16
	control   uint32
}

const keyEvent = 0x0001

// readInput waits for console input, then reads its records and returns the characters of key presses. Raw mode
// turns on virtual terminal input, so arrows and mouse reports arrive as the same escape sequences as on Unix. Other
// records, such as focus and resize events, are consumed and dropped, so they never block a read.
func readInput(in *os.File, buf []byte, timeout time.Duration) (int, error) {
	h := windows.Handle(in.Fd())
	event, err := windows.WaitForSingleObject(h, uint32(timeout.Milliseconds())) // #nosec G115 -- Timeouts are short.
	if err != nil {
		return 0, err
	}
	if event == uint32(windows.WAIT_TIMEOUT) {
		return 0, nil
	}
	var records [64]inputRecord
	var n uint32
	ok, _, err := readConsoleInput.Call(uintptr(h), uintptr(unsafe.Pointer(&records[0])), uintptr(len(records)), uintptr(unsafe.Pointer(&n))) // #nosec G103 -- A documented Win32 call with buffers it fills.
	if ok == 0 {
		return 0, err
	}
	var units []uint16
	for _, r := range records[:n] {
		if r.eventType == keyEvent && r.keyDown != 0 && r.char != 0 {
			for range max(1, r.repeat) {
				units = append(units, r.char)
			}
		}
	}
	return copy(buf, string(utf16.Decode(units))), nil
}
