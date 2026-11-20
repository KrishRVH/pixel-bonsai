package pane

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/KrishRVH/pixel-bonsai/internal/ansi"
	"github.com/KrishRVH/pixel-bonsai/internal/garden"
)

// fakeTerm is a terminal of a set size whose input the test types.
type fakeTerm struct {
	mu            sync.Mutex
	out           strings.Builder
	width, height int
	keys          chan []byte
}

func newFakeTerm() *fakeTerm {
	return &fakeTerm{width: 60, height: 24, keys: make(chan []byte, 8)}
}

func (f *fakeTerm) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.out.Write(p)
}

func (f *fakeTerm) Size() (int, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.width, f.height, nil
}

func (f *fakeTerm) Read(buf []byte, timeout time.Duration) (int, error) {
	select {
	case data := <-f.keys:
		return copy(buf, data), nil
	case <-time.After(timeout):
		return 0, nil
	}
}

func (f *fakeTerm) Interactive() bool { return true }

func (f *fakeTerm) output() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.out.String()
}

// waitFor waits until the terminal has shown text.
func (f *fakeTerm) waitFor(t *testing.T, text string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !strings.Contains(plain(f.output()), text) {
		if time.Now().After(deadline) {
			t.Fatalf("the pane never showed %q", text)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

var noon = time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)

// countDate is the local date of every count in these tests.
const countDate = "2026-11-19"

// counted is the usage every count in these tests reads: 500 tokens, 40 of them on the local date 2026-11-19.
func counted(context.Context) Count {
	return Count{Days: map[string]map[string]int64{"claude": {countDate: 500}}, Today: 40, Date: countDate}
}

func start(ctx context.Context, t *testing.T, term *fakeTerm, store *garden.Store, now time.Time) <-chan error {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, term, store, counted, Options{Mode: ansi.NoColor, Every: time.Hour, Now: func() time.Time { return now }})
	}()
	return done
}

func finish(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the pane did not stop")
	}
}

func TestThePaneGrowsTheTreeAndSwitchesItUntilQuit(t *testing.T) {
	t.Parallel()
	term := newFakeTerm()
	store := garden.Open(t.TempDir())
	done := start(t.Context(), t, term, store, noon)
	term.waitFor(t, "🌸 sakura · seed · 40 tokens · today ⇄")
	term.keys <- []byte("\x1b[") // a left arrow split across reads; from the first variety it wraps round to the last
	term.keys <- []byte{}        // a read that returns nothing early, as when a signal interrupts it
	term.keys <- []byte("D")
	term.waitFor(t, "black pine")
	term.keys <- []byte("v")
	term.keys <- []byte("\x1b") // Escape, once input pauses
	term.waitFor(t, "🌸 sakura · seed · 500 tokens · lifetime ⇄")
	term.mu.Lock()
	term.width, term.height = 100, 40
	term.mu.Unlock()
	before := len(term.output())
	term.waitFor(t, "\x1b[40;1H") // redrawn to the taller pane's last line
	if len(term.output()) == before {
		t.Error("a resize did not redraw")
	}
	term.keys <- []byte("q")
	finish(t, done)
	out := term.output()
	if !strings.HasPrefix(out, enter) || !strings.HasSuffix(out, leave) {
		t.Error("the pane did not enter and leave its screen")
	}
	g, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if g.Variety != "sakura" || g.View != "lifetime" || g.Grown["sakura"] != 500 || g.Day.Grown["sakura"] != 40 {
		t.Errorf("saved garden = %+v", g)
	}
}

func TestThePaneStopsWithItsContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	term := newFakeTerm()
	done := start(ctx, t, term, garden.Open(t.TempDir()), noon)
	term.waitFor(t, "40 tokens")
	cancel()
	finish(t, done)
}

func TestACountGoesToTheDayItMeasured(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	term := newFakeTerm()
	store := garden.Open(t.TempDir())
	done := start(ctx, t, term, store, time.Date(2026, 11, 20, 0, 0, 1, 0, time.UTC)) // handled just after midnight
	term.waitFor(t, "0 tokens · today")                                               // the new day has none yet
	deadline := time.Now().Add(10 * time.Second)
	for {
		g, _ := store.Load()
		if g.Day.Date != "" {
			if g.Day.Date != countDate || g.Day.Counted != 40 {
				t.Errorf("day record = %+v, want the 40 tokens counted on 2026-11-19", g.Day)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the count was never saved")
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	finish(t, done)
}
