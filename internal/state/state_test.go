package state_test

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/KrishRVH/pixel-bonsai/internal/state"
)

const (
	xdg     = "XDG_STATE_HOME"
	windows = "windows"
	darwin  = "darwin"
	custom  = "custom"
	linux   = "linux"
	xdgDir  = "xdg"
)

func TestTheStateFolderOnEachSystem(t *testing.T) {
	t.Parallel()
	home := filepath.Join("home", "me")
	for _, tc := range []struct {
		goos string
		env  map[string]string
		want string
	}{
		{linux, map[string]string{"BONSAI_STATE_DIR": custom, xdg: xdgDir}, custom},
		{windows, map[string]string{"BONSAI_STATE_DIR": custom}, custom},
		{linux, map[string]string{xdg: xdgDir}, filepath.Join(xdgDir, "pixel-bonsai")},
		{linux, map[string]string{}, filepath.Join(home, ".local", "state", "pixel-bonsai")},
		{"freebsd", map[string]string{}, filepath.Join(home, ".local", "state", "pixel-bonsai")},
		{darwin, map[string]string{}, filepath.Join(home, "Library", "Application Support", "pixel-bonsai")},
		{darwin, map[string]string{xdg: xdgDir}, filepath.Join(xdgDir, "pixel-bonsai")},
		{windows, map[string]string{"LOCALAPPDATA": "local", xdg: xdgDir}, filepath.Join("local", "pixel-bonsai")},
		{windows, map[string]string{}, filepath.Join(home, "AppData", "Local", "pixel-bonsai")},
	} {
		got, err := state.Dir(home, func(k string) string { return tc.env[k] }, tc.goos)
		if err != nil || got != tc.want {
			t.Errorf("Dir on %s with %v = %q, %v; want %q", tc.goos, tc.env, got, err, tc.want)
		}
	}
	for _, goos := range []string{linux, darwin, windows} {
		if _, err := state.Dir("", func(string) string { return "" }, goos); err == nil {
			t.Errorf("Dir on %s with no home and no settings is not an error", goos)
		}
	}
}

func TestWriteAtomicReplacesAWholeFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "garden.json")
	for _, data := range []string{"first, and longer", "second"} {
		if err := state.WriteAtomic(path, []byte(data)); err != nil {
			t.Fatal(err)
		}
		if got, _ := os.ReadFile(path); string(got) != data { // #nosec G304 -- A test's own temporary file.
			t.Errorf("read %q, want %q", got, data)
		}
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("%d files in the folder, want the one: a temporary file was left behind", len(entries))
	}
	if err := state.WriteAtomic(filepath.Join(path, "below-a-file"), nil); err == nil {
		t.Error("writing below a file is not an error")
	}
}

func TestLockedSerializesItsHolders(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lock, counter := filepath.Join(dir, "sub", "count.lock"), filepath.Join(dir, "count")
	const holders = 8
	var wg sync.WaitGroup
	for range holders {
		wg.Go(func() {
			err := state.Locked(lock, func() error {
				data, _ := os.ReadFile(counter) // #nosec G304 -- A test's own temporary file.
				n, _ := strconv.Atoi(string(data))
				time.Sleep(5 * time.Millisecond) // a holder that interleaved would lose a count here
				return os.WriteFile(counter, []byte(strconv.Itoa(n+1)), 0o600)
			})
			if err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if data, _ := os.ReadFile(counter); string(data) != strconv.Itoa(holders) { // #nosec G304 -- A test's own temporary file.
		t.Errorf("count = %s, want %d", data, holders)
	}
	blocker := filepath.Join(dir, "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := state.Locked(filepath.Join(blocker, "x.lock"), func() error { return nil }); err == nil {
		t.Error("a lock below a file is not an error")
	}
}
