//go:build unix

package term

import (
	"errors"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// enableEscapes does nothing: Unix terminals process escape codes already.
func enableEscapes(*os.File) (func() error, error) {
	return func() error { return nil }, nil
}

// allowMouse does nothing: a Unix terminal reports clicks once the pane asks for mouse reports.
func allowMouse(*os.File) error {
	return nil
}

// readInput waits for input with select, which works on terminals on every Unix, then reads what arrived.
func readInput(in *os.File, buf []byte, timeout time.Duration) (int, error) {
	fd := int(in.Fd()) // #nosec G115 -- File descriptors fit in an int.
	var ready unix.FdSet
	ready.Zero()
	ready.Set(fd)
	tv := unix.NsecToTimeval(timeout.Nanoseconds())
	n, err := unix.Select(fd+1, &ready, nil, nil, &tv)
	if errors.Is(err, unix.EINTR) || (err == nil && n == 0) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return unix.Read(fd, buf)
}
