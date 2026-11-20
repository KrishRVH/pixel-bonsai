package usage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
)

// openCode reads completed assistant messages from the SQLite databases of OpenCode and of Kilo Code, its fork, in
// either table they keep messages in, message or session_message. It keeps each table's highest update time and reads
// from there on.
type openCode struct {
	env  Env
	find func(Env) []string
	dbs  databases
}

func newOpenCode(env Env, find func(Env) []string) *openCode {
	return &openCode{env: env, find: find, dbs: databases{}}
}

// openCodeFiles returns OpenCode's databases: OPENCODE_DB, or opencode.db and the opencode-CHANNEL.db of other
// release channels in its data folder.
func openCodeFiles(env Env) []string {
	return dataDatabases(env, "opencode", "OPENCODE_DB", "opencode-*.db")
}

// kiloFiles returns Kilo Code's databases, which keep OpenCode's layout under Kilo's name: KILO_DB, or kilo.db,
// kilo-CHANNEL.db, and the opencode-CHANNEL.db a channel build keeps when one exists, in its data folder.
func kiloFiles(env Env) []string {
	return dataDatabases(env, "kilo", "KILO_DB", "kilo-*.db", "opencode-*.db")
}

// dataDatabases returns the database an environment variable names, relative to the app's data folder unless
// absolute, or else app.db and the files matching patterns there. The data folder is XDG_DATA_HOME/app, default
// ~/.local/share/app, on every system.
func dataDatabases(env Env, app, variable string, patterns ...string) []string {
	base := filepath.Join(envDir(env, "XDG_DATA_HOME", ".local", "share"), app)
	if name := env.Getenv(variable); name != "" {
		if !filepath.IsAbs(name) {
			name = filepath.Join(base, name)
		}
		return []string{name}
	}
	out := []string{filepath.Join(base, app+".db")}
	for _, pattern := range patterns {
		out = append(out, glob(filepath.Join(base, pattern))...)
	}
	return unique(out)
}

func (o *openCode) read(ctx context.Context, add func(event)) (Stats, error) {
	return o.dbs.read(ctx, o.find(o.env), func(ctx context.Context, db *sql.DB, marks map[string]int64) (int, error) {
		malformed := 0
		for _, table := range openCodeTables {
			n, err := readOpenCodeTable(ctx, db, table, marks, add)
			malformed += n
			if err != nil {
				return malformed, err
			}
		}
		return malformed, nil
	})
}

type openCodeData struct {
	Role       string `json:"role"`
	ProviderID string `json:"providerID"`
	ModelID    string `json:"modelID"`
	Model      *struct {
		ID         string `json:"id"`
		ProviderID string `json:"providerID"`
	} `json:"model"` // the session_message table's form of providerID and modelID
	Time struct {
		Created   count `json:"created"`
		Completed count `json:"completed"`
	} `json:"time"`
	Tokens *struct {
		Total     count `json:"total"`
		Input     count `json:"input"`
		Output    count `json:"output"`
		Reasoning count `json:"reasoning"`
		Cache     struct {
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
	{"session_message", "SELECT time_updated, data FROM session_message WHERE time_updated >= ? AND type = 'assistant'", false},
	{"message", "SELECT time_updated, data FROM message WHERE time_updated >= ?", true},
}

// readOpenCodeTable reads a table's messages updated from its watermark on, and raises the watermark to the latest. A
// table the database lacks has nothing to read.
func readOpenCodeTable(ctx context.Context, db *sql.DB, table openCodeTable, marks map[string]int64, add func(event)) (malformed int, err error) {
	if ok, err := hasTable(ctx, db, table.name); err != nil || !ok {
		return 0, err
	}
	watermark := marks[table.name]
	rows, err := db.QueryContext(ctx, table.query, watermark)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", table.name, err)
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var updated int64
		var data []byte
		if err := rows.Scan(&updated, &data); err != nil {
			return malformed, fmt.Errorf("read %s: %w", table.name, err)
		}
		watermark = max(watermark, updated)
		e, ok, decodeErr := openCodeEvent(data, table.roleInData)
		if decodeErr != nil {
			malformed++
		} else if ok {
			add(e)
		}
	}
	if err := rows.Err(); err != nil {
		return malformed, fmt.Errorf("read %s: %w", table.name, err)
	}
	marks[table.name] = watermark
	return malformed, nil
}

// openCodeEvent reads a completed assistant message's tokens, checking the role in the data when checkRole is set. A
// fork copies a session's messages under new ids with their times and tokens, so the key is the call's model, times,
// and tokens, which the copy shares.
func openCodeEvent(data []byte, checkRole bool) (event, bool, error) {
	var d openCodeData
	if err := json.Unmarshal(data, &d); err != nil {
		return event{}, false, err
	}
	if (checkRole && d.Role != assistant) || d.Time.Completed == 0 || d.Tokens == nil {
		return event{}, false, nil
	}
	provider, model := d.ProviderID, d.ModelID
	if d.Model != nil {
		provider, model = d.Model.ProviderID, d.Model.ID
	}
	t := d.Tokens
	tokens := int64(t.Total)
	if tokens == 0 {
		tokens = sum(t.Input, t.Output, t.Reasoning, t.Cache.Read, t.Cache.Write)
	}
	id := key(provider, model, counts(d.Time.Created, d.Time.Completed, t.Total, t.Input, t.Output, t.Reasoning, t.Cache.Read, t.Cache.Write))
	return event{id, timeFromMillis(int64(d.Time.Completed)), tokens}, true, nil
}
