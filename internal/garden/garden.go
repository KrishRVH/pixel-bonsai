// Package garden keeps the bonsai's saved state: which variety and view are selected, the tokens each variety has
// grown in its lifetime and today, and each source's daily token high-water marks.
//
// Tokens go to the variety selected when they are counted, for its day tree and its lifetime tree alike. The
// high-water marks keep the lifetime total from shrinking when a harness prunes old logs: each UTC day keeps the
// highest count any read has seen.
package garden

import (
	"maps"
	"time"

	"github.com/KrishRVH/pixel-bonsai/bonsai"
)

// schema is the version of the garden file this build writes. A file with a higher version came from a newer build,
// and this one leaves it alone.
const schema = 1

// A Garden is the saved state.
type Garden struct {
	Schema  int              `json:"schema"`
	Variety string           `json:"variety"` // the selected variety's id
	View    bonsai.View      `json:"view"`
	Grown   map[string]int64 `json:"grown"`   // lifetime tokens by variety id, keeping ids this build does not know
	Counted int64            `json:"counted"` // lifetime tokens already handed out
	Day     Day              `json:"day"`

	// History is each source's highest token count seen for each UTC date, "2006-01-02".
	History map[string]map[string]int64 `json:"history"`
}

// A Day is the day tree's record: its local date, the day's tokens already handed out, and each variety's share.
type Day struct {
	Date    string           `json:"date"`
	Counted int64            `json:"counted"`
	Grown   map[string]int64 `json:"grown"`
}

// New returns an empty garden with the default variety selected.
func New() Garden {
	return Garden{
		Schema:  schema,
		Variety: bonsai.Default().ID,
		View:    bonsai.Today,
		Grown:   map[string]int64{},
		Day:     Day{Grown: map[string]int64{}},
		History: map[string]map[string]int64{},
	}
}

// normalize repairs what a hand-edited file may lack: maps, a known variety, and a known view.
func (g *Garden) normalize() {
	if g.Grown == nil {
		g.Grown = map[string]int64{}
	}
	if g.Day.Grown == nil {
		g.Day.Grown = map[string]int64{}
	}
	if g.History == nil {
		g.History = map[string]map[string]int64{}
	}
	if _, ok := bonsai.Lookup(g.Variety); !ok {
		g.Variety = bonsai.Default().ID
	}
	if g.View != bonsai.Lifetime {
		g.View = bonsai.Today
	}
	g.Schema = schema
}

// Total returns the lifetime tokens: the sum of every source's daily highs.
func (g *Garden) Total() int64 {
	total := int64(0)
	for _, days := range g.History {
		for _, n := range days {
			total += n
		}
	}
	return total
}

// Record raises each source's daily highs to the counts read, and reports whether any rose.
func (g *Garden) Record(counts map[string]map[string]int64) bool {
	changed := false
	for source, days := range counts {
		kept := g.History[source]
		if kept == nil {
			kept = map[string]int64{}
		}
		for day, n := range days {
			if n > kept[day] {
				kept[day] = n
				changed = true
			}
		}
		if len(kept) > 0 {
			g.History[source] = kept
		}
	}
	return changed
}

// Accrue hands the selected variety the lifetime tokens counted since the last accrual, and the day's tokens since
// the last accrual today. The same totals twice add nothing, and smaller ones are ignored. The first accrual on a new
// local date starts the day afresh and gives the selected variety all of today, since no record says which variety
// was selected earlier in the day. A count of the day before the record's arrives late, as from a pane that counted
// just before midnight and saved after another started the new day, and leaves the day record alone. It reports
// whether anything changed.
func (g *Garden) Accrue(today int64, date string) bool {
	total := g.Total()
	grown := max(0, total-g.Counted)
	g.Grown[g.Variety] += grown
	g.Counted = max(g.Counted, total)
	changed := grown > 0
	if g.Day.follows(date) {
		return changed
	}
	if g.Day.Date != date {
		g.Day = Day{Date: date, Grown: map[string]int64{}}
		changed = true
	}
	fresh := max(0, today-g.Day.Counted)
	g.Day.Grown[g.Variety] += fresh
	g.Day.Counted = max(g.Day.Counted, today)
	return changed || fresh > 0
}

// follows reports whether the record's date is the day after date.
func (d Day) follows(date string) bool {
	t, err := time.Parse(time.DateOnly, d.Date)
	return err == nil && t.AddDate(0, 0, -1).Format(time.DateOnly) == date
}

// Choose selects a variety and reports whether the selection changed. It hands out nothing: tokens counted later go
// to it.
func (g *Garden) Choose(id string) bool {
	if _, ok := bonsai.Lookup(id); !ok || g.Variety == id {
		return false
	}
	g.Variety = id
	return true
}

// SetView selects a view and reports whether it changed.
func (g *Garden) SetView(v bonsai.View) bool {
	if g.View == v {
		return false
	}
	g.View = v
	return true
}

// Tokens returns the tokens a variety's tree has grown in a view: its share of the day record when the record is for
// date, 0 when the record is from an earlier day, and its lifetime tokens otherwise.
func (g *Garden) Tokens(id string, v bonsai.View, date string) int64 {
	if v == bonsai.Lifetime {
		return g.Grown[id]
	}
	if g.Day.Date != date {
		return 0
	}
	return g.Day.Grown[id]
}

// Clone returns a deep copy, so a snapshot shared with the pane cannot change under it.
func (g *Garden) Clone() Garden {
	c := *g
	c.Grown = maps.Clone(g.Grown)
	c.Day.Grown = maps.Clone(g.Day.Grown)
	c.History = make(map[string]map[string]int64, len(g.History))
	for source, days := range g.History {
		c.History[source] = maps.Clone(days)
	}
	return c
}
