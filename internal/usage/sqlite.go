package usage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	_ "modernc.org/sqlite" // Registers the pure-Go "sqlite" driver, so builds need no C compiler.
)

// databases are the SQLite files a source reads, each with the watermarks the source reached in it.
type databases map[string]*database

type database struct {
	info  os.FileInfo
	marks map[string]int64
}

// A dbReader reads a database's rows past its watermarks, raising each watermark only once its rows were read.
type dbReader func(ctx context.Context, db *sql.DB, marks map[string]int64) (malformed int, err error)

// read reads each database among paths that exists, and forgets the ones that vanished. A missing database has
// recorded nothing yet. A replaced file is read again from its start; one that is busy or migrating returns its error
// and is read again on the next update.
func (d *databases) read(ctx context.Context, paths []string, readDB dbReader) (Stats, error) {
	var stats Stats
	var errs []error
	next := databases{}
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		stats.Places = append(stats.Places, path)
		s := (*d)[path]
		if s == nil || !os.SameFile(s.info, info) {
			s = &database{marks: map[string]int64{}}
		}
		s.info = info
		next[path] = s
		malformed, err := readDatabase(ctx, path, s.marks, readDB)
		stats.Malformed += malformed
		if err != nil {
			errs = append(errs, err)
		}
	}
	*d = next
	return stats, errors.Join(errs...)
}

func readDatabase(ctx context.Context, path string, marks map[string]int64, readDB dbReader) (malformed int, err error) {
	db, err := sql.Open("sqlite", readOnlyURI(path))
	if err != nil {
		return 0, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { err = errors.Join(err, db.Close()) }()
	malformed, err = readDB(ctx, db, marks)
	if err != nil {
		return malformed, fmt.Errorf("%s: %w", path, err)
	}
	return malformed, nil
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

// hasTable reports whether a database has a table. A database a harness has not migrated yet may lack one.
func hasTable(ctx context.Context, db *sql.DB, name string) (bool, error) {
	var n int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = ?", name).Scan(&n); err != nil {
		return false, fmt.Errorf("list tables: %w", err)
	}
	return n > 0, nil
}
