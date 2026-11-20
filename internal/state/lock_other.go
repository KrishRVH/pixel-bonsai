//go:build !(darwin || freebsd || linux || netbsd || openbsd || windows)

package state

import "os"

// lock does nothing on systems without flock or LockFileEx: two processes updating the state at the same instant
// there can lose one update, which the next count repairs.
func lock(*os.File) error { return nil }

func unlock(*os.File) error { return nil }
