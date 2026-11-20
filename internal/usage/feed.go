package usage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/KrishRVH/pixel-bonsai/internal/state"
)

// The event feed is how any harness reports usage: one JSON object per completed model call, one per line, in
// monthly files the record command writes under the state folder and in the files BONSAI_USAGE_FILE lists. Its
// format is a published contract: see docs/integrations.md.

// FeedEvent is one line of the event feed.
type FeedEvent struct {
	Source    string `json:"source"`    // a stable name for the harness
	ID        string `json:"id"`        // unique within the source, and the same when a call is retried
	Timestamp string `json:"timestamp"` // RFC 3339 with a time zone offset
	Tokens    int64  `json:"tokens"`    // the call's tokens, at least 0
}

// feedDir returns the folder the record command appends to.
func feedDir(stateDir string) string {
	return filepath.Join(stateDir, "events")
}

// feedFiles finds the feed's monthly files and the files BONSAI_USAGE_FILE lists, separated as in PATH.
func feedFiles(env Env) []string {
	out := glob(filepath.Join(feedDir(env.State), "*.jsonl"))
	for _, p := range filepath.SplitList(env.Getenv("BONSAI_USAGE_FILE")) {
		if p != "" {
			out = append(out, p)
		}
	}
	return unique(out)
}

// feedLine is a line of the feed as read, with tokens a pointer so that a line without them is malformed rather
// than zero.
type feedLine struct {
	Source    string `json:"source"`
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Tokens    *int64 `json:"tokens"`
}

// feedParser reads feed events. Unlike a harness's log, every line of the feed is an event, so a line that breaks
// the format is malformed. IDs are scoped to their source, so two harnesses may use the same ID.
func feedParser(string) lineParser {
	return func(line []byte) (event, bool, error) {
		var l feedLine
		if err := json.Unmarshal(line, &l); err != nil {
			return event{}, false, err
		}
		if l.Tokens == nil {
			return event{}, false, fmt.Errorf("%w: tokens is missing", ErrInvalidEvent)
		}
		e := FeedEvent{Source: l.Source, ID: l.ID, Timestamp: l.Timestamp, Tokens: *l.Tokens}
		when, err := e.check()
		if err != nil {
			return event{}, false, err
		}
		return event{key(e.Source, e.ID), when, e.Tokens}, true, nil
	}
}

// ErrInvalidEvent reports an event that breaks the feed's format.
var ErrInvalidEvent = errors.New("invalid event")

// check returns the event's time, or an error wrapping ErrInvalidEvent that names the rule the event breaks.
func (e *FeedEvent) check() (time.Time, error) {
	if strings.TrimSpace(e.Source) == "" || strings.TrimSpace(e.ID) == "" {
		return time.Time{}, fmt.Errorf("%w: source and id must not be blank", ErrInvalidEvent)
	}
	if e.Tokens < 0 {
		return time.Time{}, fmt.Errorf("%w: tokens must be at least 0", ErrInvalidEvent)
	}
	when, err := parseTime(e.Timestamp)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: timestamp %q is not RFC 3339 with a time zone: %w", ErrInvalidEvent, e.Timestamp, err)
	}
	return when, nil
}

// Record appends an event to this month's feed file in the state folder, under a lock shared with other recorders.
// A blank timestamp means now.
func Record(stateDir string, e FeedEvent, now time.Time) error {
	if e.Timestamp == "" {
		e.Timestamp = now.UTC().Format(time.RFC3339Nano)
	}
	when, err := e.check()
	if err != nil {
		return err
	}
	e.Timestamp = when.Format(time.RFC3339Nano)
	line, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("record: %w", err)
	}
	dir := feedDir(stateDir)
	path := filepath.Join(dir, now.UTC().Format("2006-01")+".jsonl")
	return state.Locked(filepath.Join(dir, ".lock"), func() (err error) {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) // #nosec G304 -- The path is in the state folder.
		if err != nil {
			return fmt.Errorf("record: %w", err)
		}
		defer func() { err = errors.Join(err, f.Close()) }()
		if _, err := f.Write(append(line, '\n')); err != nil {
			return fmt.Errorf("record: %w", err)
		}
		return nil
	})
}
