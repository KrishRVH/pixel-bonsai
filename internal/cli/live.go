package cli

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/KrishRVH/pixel-bonsai/internal/ansi"
	"github.com/KrishRVH/pixel-bonsai/internal/garden"
	"github.com/KrishRVH/pixel-bonsai/internal/pane"
	"github.com/KrishRVH/pixel-bonsai/internal/term"
	"github.com/KrishRVH/pixel-bonsai/internal/usage"
)

// countEvery is how often the live pane counts usage.
const countEvery = time.Minute

var errNoTerminal = errors.New("the live bonsai needs a terminal: use render for one frame")

// live shows the pane in the terminal.
func (a App) live(ctx context.Context, args []string) (err error) {
	fs := a.flags("pixel-bonsai", "pixel-bonsai [--color auto|truecolor|256|none]")
	colors := fs.String("color", "auto", "colors: auto (from the environment), truecolor, 256, or none")
	if err := parseNone(fs, args); err != nil {
		return err
	}
	mode := ansi.Detect(a.Getenv)
	if *colors != "auto" {
		if mode, err = ansi.ParseMode(*colors); err != nil {
			return usageError{err}
		}
	}
	env, err := a.usageEnv()
	if err != nil {
		return err
	}
	out, ok := a.Stdout.(*os.File)
	if !ok {
		return errNoTerminal
	}
	t, err := term.Open(a.Stdin, out)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, t.Close()) }()
	u := usage.New(env)
	count := func(ctx context.Context) pane.Count {
		_ = u.Update(ctx) // a source that fails counts what it read; doctor shows the error
		now := a.Now()
		from, to := day(now)
		return pane.Count{Days: u.Days(), Today: u.Between(from, to), Date: now.Format(time.DateOnly)}
	}
	return pane.Run(ctx, t, garden.Open(env.State), count, pane.Options{Mode: mode, Every: countEvery, Now: a.Now})
}

// day returns the start of t's local day and the start of the next. A day is 23 or 25 hours long when the clocks
// change.
func day(t time.Time) (from, to time.Time) {
	y, m, d := t.Date()
	from = time.Date(y, m, d, 0, 0, 0, 0, t.Location())
	return from, from.AddDate(0, 0, 1)
}
