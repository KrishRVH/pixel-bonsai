package usage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	_ "modernc.org/sqlite" // Registers the pure-Go "sqlite" driver, so builds need no C compiler.
)

// openCode reads completed assistant messages from OpenCode's SQLite database, in either table OpenCode keeps them in,
// message or session_message. It keeps each table's highest update time and reads from there on; a replaced database
// is read again from its start.
type openCode struct {
	env        Env
	info       os.FileInfo
	watermarks map[string]int64
}

func newOpenCode(env Env) *openCode {
	return &openCode{env: env, watermarks: map[string]int64{}}
}

// openCodePath returns the database: OPENCODE_DB, relative to OpenCode's data folder unless absolute, or opencode.db
// there. The data folder is XDG_DATA_HOME/opencode, default ~/.local/share/opencode, on every system.
func openCodePath(env Env) string {
	base := filepath.Join(envDir(env, "XDG_DATA_HOME", ".local", "share"), "opencode")
	name := env.Getenv("OPENCODE_DB")
	if name == "" {
		name = "opencode.db"
	}
	if filepath.IsAbs(name) {
		return name
	}
	return filepath.Join(base, name)
}

// read reads the messages updated since the last read. A database that is missing counts nothing; one that is busy
// or migrating returns its error and is read again on the next update.
func (o *openCode) read(ctx context.Context, add func(event)) (stats Stats, err error) {
	path := openCodePath(o.env)
	info, statErr := os.Stat(path)
	if statErr != nil {
		return Stats{}, nil // no database means OpenCode has recorded nothing yet
	}
	stats.Places = []string{path}
	if o.info == nil || !os.SameFile(o.info, info) {
		clear(o.watermarks)
	}
	o.info = info
	db, err := sql.Open("sqlite", readOnlyURI(path))
	if err != nil {
		return stats, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { err = errors.Join(err, db.Close()) }()
	for _, table := range openCodeTables {
		malformed, err := o.readTable(ctx, db, table, add)
		stats.Malformed += malformed
		if err != nil {
			return stats, fmt.Errorf("opencode %s: %w", path, err)
		}
	}
	return stats, nil
}

// readOnlyURI returns a SQLite URI that opens path read-only.
func readOnlyURI(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	p := filepath.ToSlash(abs)
	if runtime.GOOS == "windows" && !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return (&url.URL{Scheme: "file", Path: p, RawQuery: "mode=ro"}).String()
}

type openCodeData struct {
	Role string `json:"role"`
	Time struct {
		Completed count `json:"completed"`
	} `json:"time"`
	Tokens *struct {
		Total     count `json:"total"`
		Input     count `json:"input"`
		Output    count `json:"output"`
		Reasoning count `json:"reasoning"`
		Cache     *struct {
			Read  count `json:"read"`
			Write count `json:"write"`
		} `json:"cache"`
	} `json:"tokens"`
}

// An openCodeTable is a table OpenCode keeps messages in, with the query that reads its assistant messages updated
// from a time on.
type openCodeTable struct {
	name, query string
	roleInData  bool // the message table keeps a message's role in its data
}

var openCodeTables = []openCodeTable{
	{"session_message", "SELECT id, time_updated, data FROM session_message WHERE time_updated >= ? AND type = 'assistant'", false},
	{"message", "SELECT id, time_updated, data FROM message WHERE time_updated >= ?", true},
}

// readTable reads a table's messages updated from its watermark on, and raises the watermark to the latest. A table
// the database lacks has nothing to read.
func (o *openCode) readTable(ctx context.Context, db *sql.DB, table openCodeTable, add func(event)) (malformed int, err error) {
	var exists int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table.name).Scan(&exists); err != nil {
		return 0, fmt.Errorf("list tables: %w", err)
	}
	if exists == 0 {
		return 0, nil
	}
	watermark := o.watermarks[table.name]
	rows, err := db.QueryContext(ctx, table.query, watermark)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", table.name, err)
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var id string
		var updated int64
		var data []byte
		if err := rows.Scan(&id, &updated, &data); err != nil {
			return malformed, fmt.Errorf("read %s: %w", table.name, err)
		}
		watermark = max(watermark, updated)
		e, ok, decodeErr := openCodeEvent(id, data, table.roleInData)
		if decodeErr != nil {
			malformed++
		} else if ok {
			add(e)
		}
	}
	if err := rows.Err(); err != nil {
		return malformed, fmt.Errorf("read %s: %w", table.name, err)
	}
	o.watermarks[table.name] = watermark
	return malformed, nil
}

// openCodeEvent reads a completed assistant message's tokens, checking the role in the data when checkRole is set.
func openCodeEvent(id string, data []byte, checkRole bool) (event, bool, error) {
	var d openCodeData
	if err := json.Unmarshal(data, &d); err != nil {
		return event{}, false, err
	}
	if (checkRole && d.Role != assistant) || d.Time.Completed == 0 || d.Tokens == nil {
		return event{}, false, nil
	}
	t := d.Tokens
	tokens := int64(t.Total)
	if tokens == 0 {
		tokens = sum(t.Input, t.Output, t.Reasoning)
		if t.Cache != nil {
			tokens += sum(t.Cache.Read, t.Cache.Write)
		}
	}
	return event{id, timeFromMillis(int64(d.Time.Completed)), tokens}, true, nil
}
