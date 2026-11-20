package usage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"time"
)

// goose reads the usage ledger in Goose's sessions database, one row for each model call. Rows are only ever added,
// under ids that rise, so a read takes the rows after the highest id it has seen.
type goose struct {
	env Env
	dbs databases
}

func newGoose(env Env) *goose {
	return &goose{env: env, dbs: databases{}}
}

// gooseFiles returns Goose's sessions database: under GOOSE_PATH_ROOT/data when that is absolute, or else in Goose's
// data folder, XDG_DATA_HOME/goose (default ~/.local/share/goose) on Linux and macOS and APPDATA/Block/goose/data on
// Windows.
func gooseFiles(env Env) []string {
	if root := env.Getenv("GOOSE_PATH_ROOT"); filepath.IsAbs(root) {
		return []string{filepath.Join(root, "data", "sessions", "sessions.db")}
	}
	return unique([]string{
		filepath.Join(envDir(env, "XDG_DATA_HOME", ".local", "share"), "goose", "sessions", "sessions.db"),
		filepath.Join(envDir(env, "APPDATA", "AppData", "Roaming"), "Block", "goose", "data", "sessions", "sessions.db"),
	})
}

// gooseQuery reads the ledger's rows after an id. A carried-forward row lumps the usage from before the ledger existed
// into one row stamped when it was written, so it is left out. Input already holds the cache reads and writes.
const gooseQuery = `SELECT id, session_id, created_timestamp, COALESCE(input_tokens, 0), COALESCE(output_tokens, 0),
COALESCE(total_tokens, 0) FROM usage_ledger WHERE id > ? AND cost_source IS NOT 'carried_forward' ORDER BY id`

func (g *goose) read(ctx context.Context, add func(event)) (Stats, error) {
	return g.dbs.read(ctx, gooseFiles(g.env), func(ctx context.Context, db *sql.DB, marks map[string]int64) (_ int, err error) {
		if ok, err := hasTable(ctx, db, "usage_ledger"); err != nil || !ok {
			return 0, err
		}
		watermark := marks["usage_ledger"]
		rows, err := db.QueryContext(ctx, gooseQuery, watermark)
		if err != nil {
			return 0, fmt.Errorf("read usage_ledger: %w", err)
		}
		defer func() { err = errors.Join(err, rows.Close()) }()
		for rows.Next() {
			var id, created, input, output, total int64
			var session string
			if err := rows.Scan(&id, &session, &created, &input, &output, &total); err != nil {
				return 0, fmt.Errorf("read usage_ledger: %w", err)
			}
			watermark = max(watermark, id)
			tokens := max(input, 0) + max(output, 0)
			if tokens == 0 {
				tokens = total
			}
			add(event{key(strconv.FormatInt(id, 10), session, strconv.FormatInt(created, 10)), time.Unix(created, 0).UTC(), tokens})
		}
		if err := rows.Err(); err != nil {
			return 0, fmt.Errorf("read usage_ledger: %w", err)
		}
		marks["usage_ledger"] = watermark
		return 0, nil
	})
}
