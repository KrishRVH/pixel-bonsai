// Package state holds the primitives for the bonsai's files on disk: where they live on each operating system, a
// lock that serializes processes sharing them, and atomic replacement.
package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Dir returns the folder the bonsai keeps its state in. BONSAI_STATE_DIR overrides it. Otherwise it is
// %LOCALAPPDATA%\pixel-bonsai on Windows, ~/Library/Application Support/pixel-bonsai on macOS, and
// $XDG_STATE_HOME/pixel-bonsai (default ~/.local/state/pixel-bonsai) elsewhere. XDG_STATE_HOME also applies on macOS
// when it is set.
func Dir(home string, getenv func(string) string, goos string) (string, error) {
	if dir := getenv("BONSAI_STATE_DIR"); dir != "" {
		return dir, nil
	}
	if dir := getenv("XDG_STATE_HOME"); dir != "" && goos != "windows" {
		return filepath.Join(dir, "pixel-bonsai"), nil
	}
	switch goos {
	case "windows":
		if dir := getenv("LOCALAPPDATA"); dir != "" {
			return filepath.Join(dir, "pixel-bonsai"), nil
		}
		if home == "" {
			return "", errors.New("find the state folder: neither LOCALAPPDATA nor a home folder is set")
		}
		return filepath.Join(home, "AppData", "Local", "pixel-bonsai"), nil
	case "darwin", "ios":
		if home == "" {
			return "", errors.New("find the state folder: no home folder")
		}
		return filepath.Join(home, "Library", "Application Support", "pixel-bonsai"), nil
	}
	if home == "" {
		return "", errors.New("find the state folder: neither XDG_STATE_HOME nor a home folder is set")
	}
	return filepath.Join(home, ".local", "state", "pixel-bonsai"), nil
}

// Locked runs fn while holding an exclusive lock on the file at path, which it creates. It waits for other processes
// to release the lock first, so the bonsai pane, a second pane, and the record command never interleave their
// updates. Locks guard files that are replaced whole, so the lock is a sibling file rather than the file itself.
func Locked(path string, fn func() error) (err error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600) // #nosec G304 -- The path names a lock in the state folder.
	if err != nil {
		return fmt.Errorf("open lock %s: %w", path, err)
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	if err := lock(f); err != nil {
		return fmt.Errorf("lock %s: %w", path, err)
	}
	defer func() { err = errors.Join(err, unlock(f)) }()
	return fn()
}

// WriteAtomic replaces the file at path with data: it writes a temporary file beside it, syncs it, and renames it over
// the old one, so a reader sees the old contents or the new, never a mix.
func WriteAtomic(path string, data []byte) (err error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, os.Remove(tmp.Name()))
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		return errors.Join(fmt.Errorf("write %s: %w", path, err), tmp.Close())
	}
	if err := tmp.Sync(); err != nil {
		return errors.Join(fmt.Errorf("sync %s: %w", path, err), tmp.Close())
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close %s: %w", path, err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}
