package garden

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/KrishRVH/pixel-bonsai/internal/state"
)

// ErrNewer reports a garden file written by a newer build. The store reads nothing from it and never overwrites it.
var ErrNewer = errors.New("the garden was saved by a newer pixel-bonsai")

// A Store is the garden file in a state folder, garden.json, with its lock beside it. Every read and write holds the
// lock, so panes and other processes sharing the folder see each other's updates in order.
//
// The file is the record. When it cannot be locked or read, an update goes on from the garden this store last saw,
// so a pane keeps working, and returns the error; the next update that reads the file starts from the file again, so
// a change that no write saved is lost rather than written over another process's updates.
type Store struct {
	dir    string
	memory *Garden // the garden as this store last saw it, to go on from while the file cannot be read
}

// Open returns the store in a state folder. It touches no files.
func Open(dir string) *Store {
	return &Store{dir: dir}
}

// path returns the garden file's path.
func (s *Store) path() string {
	return filepath.Join(s.dir, "garden.json")
}

func (s *Store) lockPath() string {
	return s.path() + ".lock"
}

// Load returns the saved garden, or a new one when none is saved. It writes nothing: a damaged file is reported as
// an error, with a new garden, and stays where it is until an update sets it aside.
func (s *Store) Load() (Garden, error) {
	if _, err := os.Stat(s.path()); errors.Is(err, fs.ErrNotExist) {
		return New(), nil // nothing saved, and no state folder or lock to create for it
	}
	var g Garden
	err := state.Locked(s.lockPath(), func() error {
		var err error
		g, _, err = s.read(false)
		return err
	})
	switch {
	case err == nil:
		return g, nil
	case errors.Is(err, ErrNewer):
		return Garden{}, err
	}
	return s.fallback(), err
}

// Update applies change to the saved garden and saves the result when change reports a change or no garden was
// saved. It returns the garden as it is after. A damaged file is set aside as garden.json.damaged and a new garden
// starts.
func (s *Store) Update(change func(*Garden) bool) (Garden, error) {
	var g Garden
	applied := false
	err := state.Locked(s.lockPath(), func() error {
		saved, fresh, err := s.read(true)
		if err != nil {
			return err
		}
		g, applied = saved, true
		if change(&g) || fresh {
			return s.write(g)
		}
		return nil
	})
	switch {
	case err == nil:
		s.memory = nil
		return g, nil
	case errors.Is(err, ErrNewer):
		return Garden{}, err
	}
	if !applied {
		g = s.fallback()
		change(&g)
	}
	kept := g.Clone()
	s.memory = &kept
	return g, err
}

// fallback returns the garden to go on from when the file cannot be locked or read: the one in memory, else the
// file as it reads without the lock, else a new one.
func (s *Store) fallback() Garden {
	if s.memory != nil {
		return s.memory.Clone()
	}
	if data, err := os.ReadFile(s.path()); err == nil {
		if g, err := parse(data); err == nil {
			return g
		}
	}
	return New()
}

// read returns the garden on disk and whether it is new: absent, or damaged and set aside when setAside allows.
func (s *Store) read(setAside bool) (Garden, bool, error) {
	data, err := os.ReadFile(s.path())
	if errors.Is(err, fs.ErrNotExist) {
		return New(), true, nil
	}
	if err != nil {
		return Garden{}, false, fmt.Errorf("read garden: %w", err)
	}
	g, err := parse(data)
	if errors.Is(err, ErrNewer) || (err != nil && !setAside) {
		return Garden{}, false, err
	}
	if err != nil {
		if err := os.Rename(s.path(), s.path()+".damaged"); err != nil {
			return Garden{}, false, fmt.Errorf("set aside damaged garden: %w", err)
		}
		return New(), true, nil
	}
	return g, false, nil
}

func parse(data []byte) (Garden, error) {
	var g Garden
	if err := json.Unmarshal(data, &g); err != nil {
		return Garden{}, fmt.Errorf("decode garden: %w", err)
	}
	if g.Schema > schema {
		return Garden{}, fmt.Errorf("%w (schema %d): upgrade to keep growing it", ErrNewer, g.Schema)
	}
	g.normalize()
	return g, nil
}

func (s *Store) write(g Garden) error {
	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return fmt.Errorf("encode garden: %w", err)
	}
	if err := state.WriteAtomic(s.path(), append(data, '\n')); err != nil {
		return fmt.Errorf("save garden: %w", err)
	}
	return nil
}
