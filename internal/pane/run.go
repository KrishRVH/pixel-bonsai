package pane

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/KrishRVH/pixel-bonsai/bonsai"
	"github.com/KrishRVH/pixel-bonsai/internal/ansi"
	"github.com/KrishRVH/pixel-bonsai/internal/garden"
)

// A Count is the usage read in one pass: each source's tokens by UTC date, and every source's tokens on the local
// date Date, "2006-01-02". Date comes with the count, read from the same clock reading as Today, so a count taken just
// before midnight goes to the day it measured.
type Count struct {
	Days  map[string]map[string]int64
	Today int64
	Date  string
}

// A Terminal is what the pane draws on and reads from.
type Terminal interface {
	io.Writer
	Size() (width, height int, err error)
	// Read returns 0 bytes and no error when no input arrives by the timeout, and can return so sooner.
	Read(buf []byte, timeout time.Duration) (int, error)
	Interactive() bool
}

// Options configure the pane.
type Options struct {
	Mode  ansi.Mode
	Every time.Duration    // how often usage is counted
	Now   func() time.Time // the clock, for the day tree's date
}

// The escape codes that enter and leave the pane's screen: the alternate screen, a hidden cursor, and SGR mouse
// reports for clicks.
const (
	enter = "\x1b[?1049h\x1b[?25l\x1b[?1000h\x1b[?1006h\x1b[2J"
	leave = "\x1b[?1006l\x1b[?1000l\x1b[0m\x1b[?25h\x1b[?1049l"
)

// Run shows the pane until ctx ends or the user quits. count reads usage; it runs at once and then every
// opts.Every, beside the pane, and hands its result to the garden, which gives the new tokens to the selected
// variety. The pane redraws when the terminal's size, the variety, the view, or the token count changes.
func Run(ctx context.Context, t Terminal, store *garden.Store, count func(context.Context) Count, opts Options) (err error) {
	g, err := store.Load()
	if errors.Is(err, garden.ErrNewer) {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	counts := make(chan Count)
	inputs := make(chan []input)
	var wg sync.WaitGroup
	defer func() {
		cancel()
		wg.Wait()
	}()
	wg.Go(func() { countEvery(ctx, count, opts.Every, counts) })
	if t.Interactive() {
		wg.Go(func() { readInput(ctx, t, inputs) })
	}
	if _, err := io.WriteString(t, enter); err != nil {
		return fmt.Errorf("draw: %w", err)
	}
	defer func() {
		_, leaveErr := io.WriteString(t, leave)
		err = errors.Join(err, leaveErr)
	}()
	p := &pane{t: t, store: store, opts: opts, garden: g}
	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()
	for {
		if err := p.draw(); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case c := <-counts:
			if err := p.counted(c); err != nil {
				return err
			}
		case in := <-inputs:
			quit, err := p.handle(in)
			if quit || err != nil {
				return err
			}
		case <-tick.C: // the terminal may have resized, or the day turned
		}
	}
}

func countEvery(ctx context.Context, count func(context.Context) Count, every time.Duration, out chan<- Count) {
	for {
		c := count(ctx)
		if ctx.Err() != nil {
			return
		}
		select {
		case out <- c:
		case <-ctx.Done():
			return
		}
		select {
		case <-time.After(every):
		case <-ctx.Done():
			return
		}
	}
}

// escapeWait is how long input pauses before an unfinished escape sequence ends, and a lone ESC is the Escape key.
const escapeWait = 100 * time.Millisecond

// readInput sends the inputs the terminal reads until ctx ends. It waits on input in short slices, so it stops soon
// after the pane does. A read error means the input is gone; the pane goes on drawing without it.
func readInput(ctx context.Context, t Terminal, out chan<- []input) {
	buf := make([]byte, 1024)
	var d decoder
	last := time.Now()
	for ctx.Err() == nil {
		n, err := t.Read(buf, escapeWait)
		if err != nil {
			return
		}
		var inputs []input
		switch {
		case n > 0:
			inputs, last = d.decode(buf[:n]), time.Now()
		case time.Since(last) >= escapeWait: // a read can return nothing early, as when a signal interrupts it
			inputs = d.idle()
		}
		if len(inputs) == 0 {
			continue
		}
		select {
		case out <- inputs:
		case <-ctx.Done():
			return
		}
	}
}

// pane is the running pane's state, owned by Run's goroutine.
type pane struct {
	t      Terminal
	store  *garden.Store
	opts   Options
	garden garden.Garden
	target clickTarget
	drawn  struct {
		shown         scene
		width, height int
	}
}

func (p *pane) showing() scene {
	v, _ := bonsai.Lookup(p.garden.Variety)
	date := p.opts.Now().Format(time.DateOnly)
	return scene{Variety: v, View: p.garden.View, Tokens: p.garden.Tokens(v.ID, p.garden.View, date)}
}

// draw redraws the pane when what it shows has changed. The last column stays empty, so no terminal wraps a full
// line onto the next.
func (p *pane) draw() error {
	width, height, err := p.t.Size()
	if err != nil {
		return fmt.Errorf("terminal size: %w", err)
	}
	s := p.showing()
	if s.Variety.ID == p.drawn.shown.Variety.ID && s.View == p.drawn.shown.View && s.Tokens == p.drawn.shown.Tokens &&
		width == p.drawn.width && height == p.drawn.height {
		return nil
	}
	lines, target := layout(s, width-1, height, p.opts.Mode)
	var b bytes.Buffer
	for k, line := range lines {
		b.WriteString("\x1b[" + strconv.Itoa(k+1) + ";1H")
		b.WriteString(line)
		b.WriteString("\x1b[K")
	}
	if _, err := p.t.Write(b.Bytes()); err != nil {
		return fmt.Errorf("draw: %w", err)
	}
	p.target = target
	p.drawn.shown, p.drawn.width, p.drawn.height = s, width, height
	return nil
}

// update applies a change to the garden. A garden saved by a newer build stops the pane. After any other error the
// pane shows the garden the store returns, and the next update tries the file again.
func (p *pane) update(change func(*garden.Garden) bool) error {
	g, err := p.store.Update(change)
	if errors.Is(err, garden.ErrNewer) {
		return err
	}
	p.garden = g
	return nil
}

func (p *pane) counted(c Count) error {
	return p.update(func(g *garden.Garden) bool {
		recorded := g.Record(c.Days)
		return g.Accrue(c.Today, c.Date) || recorded
	})
}

func (p *pane) handle(inputs []input) (quit bool, err error) {
	a := act(inputs, p.target)
	if a.Quit {
		return true, nil
	}
	if a.Step != 0 {
		ids := bonsai.IDs()
		next := ids[mod(indexOf(ids, p.garden.Variety)+a.Step, len(ids))]
		if err := p.update(func(g *garden.Garden) bool { return g.Choose(next) }); err != nil {
			return false, err
		}
	}
	if a.Flip {
		view := p.garden.View.Next()
		if err := p.update(func(g *garden.Garden) bool { return g.SetView(view) }); err != nil {
			return false, err
		}
	}
	return false, nil
}

func indexOf(ids []string, id string) int {
	for k, v := range ids {
		if v == id {
			return k
		}
	}
	return 0
}

// mod returns a modulo b in [0, b).
func mod(a, b int) int {
	return (a%b + b) % b
}
