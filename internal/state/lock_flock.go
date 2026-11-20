//go:build darwin || freebsd || linux || netbsd || openbsd

package state

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func lock(f *os.File) error {
	for {
		err := unix.Flock(int(f.Fd()), unix.LOCK_EX) // #nosec G115 -- File descriptors fit in an int.
		if !errors.Is(err, unix.EINTR) {
			return err
		}
	}
}

func unlock(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_UN) // #nosec G115 -- File descriptors fit in an int.
}
