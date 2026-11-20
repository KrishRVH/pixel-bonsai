package usage

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
)

// A lineParser reads one JSON log line into an event. It reports false for a line that holds no usage. A parser may
// keep state across the lines of one file.
type lineParser func(line []byte) (event, bool, error)

// A tailer reads a JSONL source: it finds the source's files and reads each from where it left off. Logs only grow,
// so a file that shrank or was replaced is read again from its start; the ledger's dedupe makes that harmless.
type tailer struct {
	env    Env
	find   func(Env) []string
	marker []byte                       // a cheap test that skips lines without usage before decoding; nil decodes every line
	parser func(path string) lineParser // a fresh parser per file
	files  map[string]*tailFile
}

type tailFile struct {
	info   os.FileInfo
	offset int64
	parse  lineParser
}

func newTailer(env Env, find func(Env) []string, marker []byte, parser func(string) lineParser) *tailer {
	return &tailer{env: env, find: find, marker: marker, parser: parser, files: map[string]*tailFile{}}
}

// read reads every file's new lines, several files at once, then adds their events in path order, so the ledger
// sees the same events in the same order however the reads interleave. A canceled read adds nothing and forgets
// every file, so the next read starts over; the ledger's dedupe makes that harmless. A file that vanished is
// forgotten; one that could not be read is left to the next update, as it may be mid-rotation or briefly locked.
func (t *tailer) read(ctx context.Context, add func(event)) (Stats, error) {
	paths := t.find(t.env)
	prev := make([]*tailFile, len(paths))
	for k, path := range paths {
		prev[k] = t.files[path]
	}
	reads := make([]fileRead, len(paths))
	var next atomic.Int64
	var wg sync.WaitGroup
	for range min(runtime.GOMAXPROCS(0), len(paths)) {
		wg.Go(func() {
			for k := int(next.Add(1) - 1); k < len(paths) && ctx.Err() == nil; k = int(next.Add(1) - 1) {
				reads[k] = t.readFile(ctx, paths[k], prev[k])
			}
		})
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		t.files = map[string]*tailFile{}
		return Stats{Places: paths}, err
	}
	stats := Stats{Places: paths}
	files := make(map[string]*tailFile, len(paths))
	for k, r := range reads {
		if r.state != nil {
			files[paths[k]] = r.state
		}
		for _, e := range r.events {
			add(e)
		}
		stats.Malformed += r.malformed
	}
	t.files = files
	return stats, nil
}

// A fileRead is what one read of a file found: the file's state after it, the events in its new lines, and how many
// of those lines were malformed.
type fileRead struct {
	state     *tailFile
	events    []event
	malformed int
}

// readFile reads a file's new whole lines from where prev, its state after the last read, left off. It returns prev
// when the file cannot be opened, and the state as far as it read when a read fails midway.
func (t *tailer) readFile(ctx context.Context, path string, prev *tailFile) fileRead {
	f, err := os.Open(path) // #nosec G304 -- Paths come from the source's own discovery.
	if err != nil {
		return fileRead{state: prev}
	}
	defer func() { _ = f.Close() }() // a read-only file has nothing to flush
	info, err := f.Stat()
	if err != nil {
		return fileRead{state: prev}
	}
	s := &tailFile{info: info, parse: t.parser(path)}
	if prev != nil && os.SameFile(prev.info, info) && info.Size() >= prev.offset {
		s.offset, s.parse = prev.offset, prev.parse
	}
	if info.Size() == s.offset {
		return fileRead{state: s}
	}
	if _, err := f.Seek(s.offset, io.SeekStart); err != nil {
		return fileRead{state: prev}
	}
	r := fileRead{state: s}
	t.scan(ctx, io.LimitReader(f, info.Size()-s.offset), &r)
	return r
}

// scan reads whole lines into r, advancing its state's offset past each. A line still being written, without its
// newline yet, waits for the next update.
func (t *tailer) scan(ctx context.Context, in io.Reader, r *fileRead) {
	add := func(e event) { r.events = append(r.events, e) }
	br := bufio.NewReaderSize(in, 1<<20)
	var long []byte // a line longer than the buffer, gathered across reads
	for n := 0; n%4096 != 0 || ctx.Err() == nil; n++ {
		part, err := br.ReadSlice('\n')
		if errors.Is(err, bufio.ErrBufferFull) {
			long = append(long, part...)
			continue
		}
		if err != nil { // the end of the whole lines, or a failed read the next update tries again
			return
		}
		line := part
		if long != nil {
			long = append(long, part...)
			line, long = long, nil
		}
		r.state.offset += int64(len(line))
		if !t.consume(line[:len(line)-1], r.state.parse, add) {
			r.malformed++
		}
	}
}

// consume parses one line and adds its event. It reports false for a line that held the marker but did not decode.
// Blank lines are skipped.
func (t *tailer) consume(line []byte, parse lineParser, add func(event)) bool {
	if len(bytes.TrimSpace(line)) == 0 || !bytes.Contains(line, t.marker) {
		return true
	}
	e, ok, err := parse(line)
	if err != nil {
		return false
	}
	if ok {
		add(e)
	}
	return true
}

// walk returns the files under root whose names match pattern, sorted. A missing root has none.
func walk(root, pattern string) []string {
	var out []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // An unreadable folder holds no usage; the walk goes on.
		}
		if ok, _ := filepath.Match(pattern, d.Name()); ok && d.Type().IsRegular() {
			out = append(out, path)
		}
		return nil
	})
	slices.Sort(out)
	return out
}

// glob returns the paths matching a pattern, sorted. A malformed pattern matches nothing.
func glob(pattern string) []string {
	out, _ := filepath.Glob(pattern)
	slices.Sort(out)
	return out
}

// envDir returns the folder an environment variable names, or a default under home.
func envDir(env Env, variable string, def ...string) string {
	if dir := env.Getenv(variable); dir != "" {
		return dir
	}
	return filepath.Join(append([]string{env.Home}, def...)...)
}

// unique returns paths without repeats, keeping the first of each.
func unique(paths []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}
