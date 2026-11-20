// Package usage reads the token usage coding agents record on this machine: each supported harness's local logs, and
// the event feed any other harness can append to. Each source has a [Ledger] that reads incrementally, counts every
// call once at the largest count reported for it, and totals tokens by UTC day and by quarter hour.
package usage

import (
	"context"
	"errors"
	"hash/fnv"
	"sync"
	"time"
)

// An event is one model call's tokens.
type event struct {
	Key    string // identifies the call within its source: a retry or a copied log line has the same key
	Time   time.Time
	Tokens int64
}

// Env is what sources read from the machine: the home folder, environment variables, and the bonsai's state folder,
// which holds the event feed. Tests point it at temporary folders.
type Env struct {
	Home   string
	Getenv func(string) string
	State  string
}

// Stats describe what a ledger has read, for diagnosis.
type Stats struct {
	Places    []string // the files or databases the source found
	Events    int      // distinct events counted
	Malformed int      // lines that looked like usage but did not decode
}

// A reader adds a source's new events since its last read.
type reader interface {
	read(ctx context.Context, add func(event)) (Stats, error)
}

// A Ledger is one source's usage: every call it has read, once each, by quarter hour. A call reported again with
// more tokens, as a streamed response is, counts at the larger number. Updates read only what the source added since
// the last update, so a long-running pane pays for new usage, not for history.
type Ledger struct {
	Source string

	reader   reader
	calls    map[uint64]call // by a hash of the event key
	quarters map[int64]int64 // tokens by quarter hour, as Unix seconds / quarter
	stats    Stats
	err      error
}

// A call is what a ledger counted for one event key: the quarter hour of its first event and its largest count. A
// call stays in that quarter hour because the garden keeps each day's highest total: a call that moved to a later day
// would count on both.
type call struct {
	quarter, tokens int64
}

// quarter is the bucket ledgers total tokens in. Every time zone's midnight falls on a quarter hour in UTC, so the
// tokens since local midnight are a sum of whole buckets.
const quarter = 15 * 60

func newLedger(source string, r reader) *Ledger {
	return &Ledger{Source: source, reader: r, calls: map[uint64]call{}, quarters: map[int64]int64{}}
}

// Update reads what the source added since the last update. An unreadable or missing source is not an error: it
// counts nothing until it appears. The error reports a failure the next update may not repair, such as a canceled
// context or a database it cannot open.
func (l *Ledger) Update(ctx context.Context) error {
	stats, err := l.reader.read(ctx, l.add)
	l.stats.Places = stats.Places
	l.stats.Malformed += stats.Malformed
	l.err = err
	return err
}

func (l *Ledger) add(e event) {
	if e.Key == "" || e.Tokens <= 0 || e.Time.IsZero() {
		return
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(e.Key)) // hash.Hash writes never fail
	key := h.Sum64()
	c, seen := l.calls[key]
	if !seen {
		c.quarter = floorDiv(e.Time.Unix(), quarter)
		l.stats.Events++
	}
	if e.Tokens <= c.tokens {
		return
	}
	l.quarters[c.quarter] += e.Tokens - c.tokens
	c.tokens = e.Tokens
	l.calls[key] = c
}

func floorDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// Between returns the tokens from from until to. Both must fall on a quarter hour, as every local midnight does.
func (l *Ledger) Between(from, to time.Time) int64 {
	first, end := floorDiv(from.Unix(), quarter), floorDiv(to.Unix(), quarter)
	total := int64(0)
	for q, n := range l.quarters {
		if first <= q && q < end {
			total += n
		}
	}
	return total
}

// Days returns the tokens by UTC date, "2006-01-02".
func (l *Ledger) Days() map[string]int64 {
	out := map[string]int64{}
	for q, n := range l.quarters {
		out[time.Unix(q*quarter, 0).UTC().Format(time.DateOnly)] += n
	}
	return out
}

// Stats returns what the ledger has read, and the error of its last update.
func (l *Ledger) Stats() (Stats, error) {
	return l.stats, l.err
}

// Usage is every source's ledger.
type Usage struct {
	Ledgers []*Ledger
}

// New returns a ledger for each source, in a fixed order, reading nothing yet.
func New(env Env) *Usage {
	return &Usage{Ledgers: []*Ledger{
		newLedger("claude", newTailer(env, claudeFiles, []byte(`"usage"`), claudeParser)),
		newLedger("codex", newTailer(env, codexFiles, []byte(`"token_count"`), codexParser)),
		newLedger("copilot", newTailer(env, copilotFiles, []byte(`"session.usage_record"`), copilotParser)),
		newLedger("antigravity", newTailer(env, antigravityFiles, []byte(`"step_index"`), antigravityParser)),
		newLedger("pi", newTailer(env, piFiles, []byte(`"usage"`), piParser)),
		newLedger("opencode", newOpenCode(env, openCodeFiles)),
		newLedger("kilo", newOpenCode(env, kiloFiles)),
		newLedger("qwen", newTailer(env, qwenFiles, []byte(`"totalTokens"`), qwenParser)),
		newLedger("kimi", newTailer(env, kimiFiles, []byte(`"usage.record"`), kimiParser)),
		newLedger("goose", newGoose(env)),
		newLedger("events", newTailer(env, feedFiles, nil, feedParser)),
	}}
}

// Update updates every ledger at once and returns their errors joined.
func (u *Usage) Update(ctx context.Context) error {
	var wg sync.WaitGroup
	errs := make([]error, len(u.Ledgers))
	for k, l := range u.Ledgers {
		wg.Go(func() { errs[k] = l.Update(ctx) })
	}
	wg.Wait()
	return errors.Join(errs...)
}

// Between returns every source's tokens from from until to.
func (u *Usage) Between(from, to time.Time) int64 {
	total := int64(0)
	for _, l := range u.Ledgers {
		total += l.Between(from, to)
	}
	return total
}

// Days returns each source's tokens by UTC date.
func (u *Usage) Days() map[string]map[string]int64 {
	out := make(map[string]map[string]int64, len(u.Ledgers))
	for _, l := range u.Ledgers {
		out[l.Source] = l.Days()
	}
	return out
}
