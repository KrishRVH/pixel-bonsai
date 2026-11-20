package cli

import (
	"errors"

	"github.com/KrishRVH/pixel-bonsai/internal/usage"
)

// record appends one event to the event feed, for a harness the bonsai does not read on its own.
func (a App) record(args []string) error {
	fs := a.flags("record", "pixel-bonsai record --source NAME --id ID --tokens N [--timestamp TIME]")
	source := fs.String("source", "", "a stable name for your harness; required")
	id := fs.String("id", "", "the call's ID, unique within the source and reused on retry; required")
	tokens := fs.Int64("tokens", -1, "the call's tokens; required")
	timestamp := fs.String("timestamp", "", "the call's time, RFC 3339 with a time zone offset (default: now)")
	if err := parseNone(fs, args); err != nil {
		return err
	}
	if *source == "" || *id == "" || *tokens < 0 {
		return usageError{errors.New("record needs --source, --id, and a --tokens count of at least 0")}
	}
	dir, err := a.stateDir()
	if err != nil {
		return err
	}
	e := usage.FeedEvent{Source: *source, ID: *id, Timestamp: *timestamp, Tokens: *tokens}
	err = usage.Record(dir, e, a.Now())
	if errors.Is(err, usage.ErrInvalidEvent) {
		return usageError{err}
	}
	return err
}
