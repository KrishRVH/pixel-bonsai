package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/KrishRVH/pixel-bonsai/bonsai"
	"github.com/KrishRVH/pixel-bonsai/internal/ansi"
	"github.com/KrishRVH/pixel-bonsai/internal/garden"
	"github.com/KrishRVH/pixel-bonsai/internal/usage"
)

// report is doctor's findings.
type report struct {
	Version string         `json:"version"`
	System  string         `json:"system"`
	State   string         `json:"state"`
	Color   string         `json:"color"`
	Garden  *garden.Garden `json:"garden,omitempty"`
	Error   string         `json:"garden_error,omitempty"`
	Sources []sourceReport `json:"sources"`
}

type sourceReport struct {
	Name      string   `json:"name"`
	Places    []string `json:"places"`
	Events    int      `json:"events"`
	Tokens    int64    `json:"tokens"`
	Today     int64    `json:"today"`
	Malformed int      `json:"malformed"`
	Error     string   `json:"error,omitempty"`
}

// doctor reads every source once and reports where it found usage and what it counted, for setting up a harness or
// reporting a problem. It changes nothing.
func (a App) doctor(ctx context.Context, args []string) error {
	fs := a.flags("doctor", "pixel-bonsai doctor [--json]")
	asJSON := fs.Bool("json", false, "print the report as JSON")
	if err := parseNone(fs, args); err != nil {
		return err
	}
	env, err := a.usageEnv()
	if err != nil {
		return err
	}
	r := report{
		Version: version(),
		System:  runtime.GOOS + "/" + runtime.GOARCH,
		State:   env.State,
		Color:   ansi.Detect(a.Getenv).String(),
	}
	if g, err := garden.Open(env.State).Load(); err != nil {
		r.Error = err.Error()
	} else {
		r.Garden = &g
	}
	u := usage.New(env)
	updateErr := u.Update(ctx) // each source's own error goes in the report
	if ctx.Err() != nil {
		return updateErr
	}
	from, to := day(a.Now())
	for _, l := range u.Ledgers {
		stats, err := l.Stats()
		s := sourceReport{
			Name:      l.Source,
			Places:    append([]string{}, stats.Places...), // [] rather than null in JSON
			Events:    stats.Events,
			Today:     l.Between(from, to),
			Malformed: stats.Malformed,
		}
		for _, n := range l.Days() {
			s.Tokens += n
		}
		if err != nil {
			s.Error = err.Error()
		}
		r.Sources = append(r.Sources, s)
	}
	if *asJSON {
		enc := json.NewEncoder(a.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(r)
	}
	return r.write(a.Stdout, a.Now().Format(time.DateOnly))
}

func (r report) write(w io.Writer, today string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "version\t%s\nsystem\t%s\nstate\t%s\ncolor\t%s\n", r.Version, r.System, r.State, r.Color)
	if g := r.Garden; g != nil {
		fmt.Fprintf(&b, "garden\t%s, %s view, %d tokens today and %d in its lifetime\n", g.Variety, g.View,
			g.Tokens(g.Variety, bonsai.Today, today), g.Tokens(g.Variety, bonsai.Lifetime, today))
	} else {
		fmt.Fprintf(&b, "garden\t%s\n", r.Error)
	}
	b.WriteString("\nsource\tplaces\tevents\ttokens\ttoday\tmalformed lines\n")
	for _, s := range r.Sources {
		fmt.Fprintf(&b, "%s\t%d\t%d\t%d\t%d\t%d\n", s.Name, len(s.Places), s.Events, s.Tokens, s.Today, s.Malformed)
		if s.Error != "" {
			fmt.Fprintf(&b, "\t  error: %s\n", s.Error)
		}
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := io.WriteString(tw, b.String()); err != nil {
		return err
	}
	return tw.Flush()
}
