package usage_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KrishRVH/pixel-bonsai/internal/usage"
)

const (
	stamp = "2026-11-10T12:00:00Z"
	day1  = "2026-11-10"
	day2  = "2026-11-11"
	tool  = "my-tool"
)

// A fixture is a home folder of fake logs, a state folder, and an environment of their own.
type fixture struct {
	t           *testing.T
	home, state string
	env         map[string]string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	return &fixture{t: t, home: t.TempDir(), state: t.TempDir(), env: map[string]string{}}
}

func (f *fixture) usage() *usage.Usage {
	return usage.New(usage.Env{Home: f.home, Getenv: func(k string) string { return f.env[k] }, State: f.state})
}

// ledger returns a fresh ledger for a source, updated once.
func (f *fixture) ledger(source string) *usage.Ledger {
	f.t.Helper()
	for _, l := range f.usage().Ledgers {
		if l.Source == source {
			update(f.t, l)
			return l
		}
	}
	f.t.Fatalf("no source %q", source)
	return nil
}

func update(t *testing.T, l *usage.Ledger) *usage.Ledger {
	t.Helper()
	if err := l.Update(t.Context()); err != nil {
		t.Fatalf("update %s: %v", l.Source, err)
	}
	return l
}

// write appends text to a file under the home folder.
func (f *fixture) write(rel, text string) string {
	f.t.Helper()
	path := filepath.Join(f.home, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		f.t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) // #nosec G304 -- A test's own temporary file.
	if err != nil {
		f.t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	if _, err := file.WriteString(text); err != nil {
		f.t.Fatal(err)
	}
	return path
}

// append appends lines.
func (f *fixture) append(rel string, lines ...string) string {
	f.t.Helper()
	return f.write(rel, strings.Join(lines, "\n")+"\n")
}

func total(days map[string]int64) int64 {
	n := int64(0)
	for _, v := range days {
		n += v
	}
	return n
}

// claude returns a transcript line for an API message, stamped at noon UTC on a day or at a full stamp.
func claude(message, when string, tokens int64) string {
	if !strings.Contains(when, "T") {
		when += "T12:00:00Z"
	}
	return fmt.Sprintf(`{"type":"assistant","requestId":"r","timestamp":%q,"message":{"id":%q,"usage":{"input_tokens":%d,"output_tokens":0}}}`, when, message, tokens)
}

func TestClaudeCountsEachMessageOnceAndKeepsWhatWasPruned(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	dir := ".claude/projects/p/"
	first := f.append(dir+"a.jsonl", claude("m1", day1, 100), claude("m1", day1, 100))
	f.write(dir+"a.jsonl", "not json\n")
	f.append(dir+"resumed.jsonl", claude("m1", day1, 100), claude("m2", day2, 5))
	agent := f.append(dir+"sid/subagents/agent-x.jsonl", claude("m3", day1, 7))
	f.append(".claude-work/projects/q/b.jsonl", claude("m4", day2, 1000))
	f.env["CLAUDE_CONFIG_DIR"] = filepath.Join(f.home, "elsewhere")
	f.append("elsewhere/projects/r/c.jsonl", claude("m5", day2, 20000))
	l := f.ledger("claude")
	if want := map[string]int64{day1: 107, day2: 21005}; !maps.Equal(l.Days(), want) {
		t.Fatalf("days = %v, want %v: a message repeated in a line or a resumed copy counts once", l.Days(), want)
	}
	for _, p := range []string{first, agent} {
		if err := os.Remove(p); err != nil {
			t.Fatal(err)
		}
	}
	f.append(dir+"b.jsonl", claude("m6", day2, 30))
	if got := total(update(t, l).Days()); got != 21142 {
		t.Errorf("total = %d, want 21142: pruned transcripts keep their count", got)
	}
	stats, err := l.Stats()
	if err != nil || stats.Events != 6 || len(stats.Places) != 4 {
		t.Errorf("stats = %+v, %v: want 6 events in 4 files", stats, err)
	}
}

// streamed returns a line of a streamed Claude response, with the usage known when the line was written.
func streamed(when string, output int64, stop string) string {
	return fmt.Sprintf(`{"type":"assistant","requestId":"r1","timestamp":%q,"message":{"id":"m1","stop_reason":%s,`+
		`"usage":{"input_tokens":2,"cache_creation_input_tokens":26140,"cache_read_input_tokens":18036,"output_tokens":%d}}}`, when, stop, output)
}

func TestClaudeCountsAStreamedMessageAtItsFinalUsage(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	rel := ".claude/projects/p/a.jsonl"
	start, end := "2026-11-10T23:59:58Z", "2026-11-11T00:00:04Z"
	f.append(rel, streamed(start, 8, "null")) // the response's first block, with the output so far
	l := f.ledger("claude")
	f.append(rel, streamed(end, 251, `"tool_use"`))
	f.append(".claude/projects/p/resumed.jsonl", streamed(start, 8, "null"))
	if want := map[string]int64{day1: 2 + 26140 + 18036 + 251}; !maps.Equal(update(t, l).Days(), want) {
		t.Errorf("days = %v, want %v: a message counts at its final usage, read later or copied, on the day it began", l.Days(), want)
	}
	if stats, _ := l.Stats(); stats.Events != 1 {
		t.Errorf("events = %d, want 1", stats.Events)
	}
}

func TestALedgerReadsOnlyWhatWasAppended(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	rel := ".claude/projects/p/a.jsonl"
	half := claude("m2", day1, 5)
	path := f.append(rel, claude("m1", day1, 100))
	f.write(rel, half[:20]) // m2 still being written
	l := f.ledger("claude")
	if want := map[string]int64{day1: 100}; !maps.Equal(l.Days(), want) {
		t.Fatalf("days = %v, want %v: a half-written line waits", l.Days(), want)
	}
	f.write(rel, half[20:]+"\n")
	f.append(rel, claude("m1", day1, 100))
	if want := map[string]int64{day1: 105}; !maps.Equal(update(t, l).Days(), want) {
		t.Fatalf("days = %v, want %v: the rest of the line, and m1 again once", l.Days(), want)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	f.append(rel, claude("m3", day2, 7))
	if want := map[string]int64{day1: 105, day2: 7}; !maps.Equal(update(t, l).Days(), want) {
		t.Errorf("days = %v, want %v: a replaced transcript is read again", l.Days(), want)
	}
}

func TestTheDayCountsFromLocalMidnight(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	zone := time.FixedZone("India", 5*3600+1800) // a half-hour offset still falls on a quarter hour
	midnight := time.Date(2026, 11, 19, 0, 0, 0, 0, zone)
	at := func(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }
	noon := midnight.Add(12 * time.Hour)
	f.append(".claude/projects/p/a.jsonl", claude("m1", at(midnight.Add(-time.Minute)), 50), claude("m2", at(noon), 7))
	f.append(".claude/projects/p/b.jsonl", claude("m2", at(noon), 7), claude("m3", at(midnight), 3), claude("m4", at(midnight.AddDate(0, 0, 1)), 900))
	u := f.usage()
	if err := u.Update(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := u.Between(midnight, midnight.AddDate(0, 0, 1)); got != 10 {
		t.Errorf("today = %d, want 10: yesterday's and tomorrow's messages are left out", got)
	}
	if got := u.Days()["claude"]["2026-11-18"]; got != 50+3 {
		t.Errorf("UTC 2026-11-18 = %d, want 53: days are UTC dates", got)
	}
}

func codex(when string, total, last int64) string {
	return fmt.Sprintf(`{"type":"event_msg","timestamp":%q,"payload":{"type":"token_count","info":{"total_token_usage":{"total_tokens":%d},"last_token_usage":{"total_tokens":%d}}}}`, when, total, last)
}

func TestCodexCountsUsageOnceAcrossRefreshForkAndArchive(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.env["CODEX_HOME"] = filepath.Join(f.home, "codex")
	rel := "codex/sessions/2026/11/10/rollout-session.jsonl"
	first := codex(stamp, 100, 100)
	f.append(rel, first, codex("2026-11-10T12:01:00Z", 100, 100))
	f.append("codex/archived_sessions/rollout-fork.jsonl", first)
	f.append("codex/sessions/2026/11/10/other.jsonl", codex(stamp, 900, 900))
	l := f.ledger("codex")
	if want := map[string]int64{day1: 100}; !maps.Equal(l.Days(), want) {
		t.Fatalf("days = %v, want %v", l.Days(), want)
	}
	f.append(rel, codex("2026-11-11T12:02:00Z", 108, 8))
	if want := map[string]int64{day1: 100, day2: 8}; !maps.Equal(update(t, l).Days(), want) {
		t.Errorf("days = %v, want %v", l.Days(), want)
	}
}

func TestAntigravityPiAndOhMyPiSessions(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	for _, status := range []string{"ACTIVE", "DONE", "DONE"} {
		f.append(".gemini/antigravity-cli/brain/conversation/.system_generated/logs/transcript.jsonl", fmt.Sprintf(
			`{"step_index":3,"source":"MODEL","status":%q,"created_at":%q,"input_tokens":21,"cache_read_tokens":9,"output_tokens":3}`, status, stamp))
	}
	pi := `{"type":"message","id":"entry","timestamp":"` + stamp + `","message":{"role":"assistant","usage":{"totalTokens":100,"cacheRead":98}}}`
	f.append(".pi/agent/sessions/project/a.jsonl", pi)
	f.append(".pi/agent/sessions/project/fork.jsonl", pi)
	f.append(".omp/agent/sessions/project/a.jsonl", `{"type":"message","id":"other","timestamp":"`+stamp+`","message":{"role":"assistant","usage":{"input":2,"cacheWrite":4}}}`)
	f.env["PI_CODING_AGENT_SESSION_DIR"] = filepath.Join(f.home, "custom")
	f.append("custom/x.jsonl", `{"type":"message","timestamp":"`+stamp+`","message":{"role":"assistant","responseId":"resp","usage":{"totalTokens":1000}}}`)
	if got := total(f.ledger("antigravity").Days()); got != 33 {
		t.Errorf("antigravity = %d, want 33: a step counts once it is done", got)
	}
	if got := total(f.ledger("pi").Days()); got != 1106 {
		t.Errorf("pi = %d, want 1106: a fork's copy counts once", got)
	}
}

func TestOpenCodeReadsCompletedMessagesFromEitherTable(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	path := filepath.Join(f.home, "custom.sqlite")
	f.env["OPENCODE_DB"] = path
	exec(t, path,
		"create table message (id text, time_updated integer, data text)",
		"insert into message values ('a', 1, '"+openCodeMessage(1)+"')",
		"create table session_message (id text, type text, time_updated integer, data text)",
		"insert into session_message values ('b', 'assistant', 10, '"+openCodeMessage(2)+"')",
		`insert into session_message values ('next', 'assistant', 10, '{"time":{},"role":"assistant","tokens":{"input":20}}')`,
	)
	l := f.ledger("opencode")
	if got := total(l.Days()); got != 40 {
		t.Fatalf("total = %d, want 40: an unfinished message waits", got)
	}
	exec(t, path, "update session_message set data='"+openCodeMessage(3)+"', time_updated=11 where id='next'")
	if got := total(update(t, l).Days()); got != 60 {
		t.Errorf("total = %d, want 60", got)
	}
	if got := total(update(t, l).Days()); got != 60 {
		t.Errorf("total = %d after a read with nothing new, want 60", got)
	}
}

// openCodeMessage returns a completed assistant message's data with 20 tokens, finished n seconds after noon UTC on
// day1.
func openCodeMessage(n int64) string {
	return fmt.Sprintf(`{"time":{"completed":%d},"role":"assistant","tokens":{"input":2,"output":3,"reasoning":4,"cache":{"read":5,"write":6}}}`, 1794312000000+n*1000)
}

func exec(t *testing.T, path string, statements ...string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	for _, s := range statements {
		if _, err := db.ExecContext(t.Context(), s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
}

func TestOpenCodeCountsAForkedSessionOnce(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	path := filepath.Join(f.home, ".local", "share", "opencode", "opencode.db")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	call := `{"time":{"created":1794311990000,"completed":1794312000000},"role":"assistant","providerID":"p","modelID":"m","tokens":{"input":2,"output":3}}`
	exec(t, path,
		"create table message (id text, time_updated integer, data text)",
		"insert into message values ('msg_1', 1, '"+call+"')",
		"insert into message values ('msg_fork', 2, '"+call+"')",
	)
	if got := total(f.ledger("opencode").Days()); got != 5 {
		t.Errorf("total = %d, want 5: a fork copies a session's messages under new ids", got)
	}
}

func TestKiloReadsTheDatabaseOfEachChannel(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	dir := filepath.Join(f.home, ".local", "share", "kilo")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	for k, name := range []string{"kilo.db", "kilo-rc.db", "opencode-dev.db", "session-export.db"} {
		exec(t, filepath.Join(dir, name),
			"create table message (id text, time_updated integer, data text)",
			"insert into message values ('m', 1, '"+openCodeMessage(int64(k))+"')",
		)
	}
	if got := total(f.ledger("kilo").Days()); got != 60 {
		t.Errorf("total = %d, want 60: an export is not a channel's database", got)
	}
	f.env["KILO_DB"] = "kilo-rc.db"
	if got := total(f.ledger("kilo").Days()); got != 20 {
		t.Errorf("total = %d, want 20 from the one database KILO_DB names", got)
	}
}

func TestCopilotCountsEachCallOnceAcrossRotationAndForks(t *testing.T) {
	t.Parallel()
	call := func(event, session string, sequence, tokens int) string {
		return fmt.Sprintf(`{"type":"session.usage_record","id":%q,"timestamp":%q,"data":{"usage":{"model":"m","inputTokens":%d,"outputTokens":1,"cacheReadTokens":%d,"accounting":{"sourceSessionId":%q,"sequence":%d}}}}`, event, stamp, tokens-1, tokens-2, session, sequence)
	}
	f := newFixture(t)
	dir := ".copilot/session-state/"
	f.append(dir+"s1/events.1.jsonl", call("e1", "s1", 0, 100), `{"type":"assistant.message","data":{"outputTokens":7}}`)
	f.append(dir+"s1/events.jsonl", call("e1", "s1", 0, 100), call("e2", "s1", 1, 20))
	f.append(dir+"s2/events.jsonl", call("e8", "s1", 0, 100), call("e9", "s1", 1, 20),
		`{"type":"session.usage_record","id":"e3","timestamp":"`+stamp+`","data":{"usage":{"inputTokens":3}}}`)
	if got := total(f.ledger("copilot").Days()); got != 123 {
		t.Errorf("total = %d, want 123: a rotated journal and a fork copy their calls", got)
	}
	g := newFixture(t)
	g.env["COPILOT_HOME"] = filepath.Join(g.home, "elsewhere")
	g.append("elsewhere/session-state/s/events.jsonl", call("e", "s", 1, 9))
	if got := total(g.ledger("copilot").Days()); got != 9 {
		t.Errorf("total = %d, want 9 from COPILOT_HOME", got)
	}
}

func TestQwenReadsTheLedgersOfEachRuntimeFolder(t *testing.T) {
	t.Parallel()
	line := func(id string, total int) string {
		return fmt.Sprintf(`{"schemaVersion":1,"id":%q,"timestamp":%q,"inputTokens":%d,"outputTokens":10,"cachedTokens":40,"totalTokens":%d}`, id, stamp, total-10, total)
	}
	f := newFixture(t)
	f.append(".qwen/usage/token-usage-2026-11.jsonl", line("a", 100), line("a", 100), line("b", 10))
	f.env["QWEN_RUNTIME_DIR"] = filepath.Join(f.home, "runtime")
	f.append("runtime/usage/token-usage-2026-10.jsonl", line("c", 1000))
	if got := total(f.ledger("qwen").Days()); got != 1110 {
		t.Errorf("total = %d, want 1110: each id counts once, from every runtime folder", got)
	}
}

func TestKimiCodeCountsAForkedJournalOnce(t *testing.T) {
	t.Parallel()
	record := func(ms int64, input int) string {
		return fmt.Sprintf(`{"type":"usage.record","agentId":"main","model":"kimi-k2","usage":{"inputOther":%d,"output":5,"inputCacheRead":100,"inputCacheCreation":0},"usageScope":"turn","time":%d}`, input, ms)
	}
	f := newFixture(t)
	dir := ".kimi-code/sessions/wd_proj_0123456789ab/"
	f.append(dir+"s1/agents/main/wire.jsonl", `{"type":"metadata","protocol_version":"1.5"}`, record(1794312000000, 10),
		`{"type":"context.append_loop_event","event":{"type":"step.end","usage":{"inputOther":10}}}`)
	f.append(dir+"s2/agents/main/wire.jsonl", record(1794312000000, 10), record(1794312001000, 20))
	f.append(dir+"s1/agents/sub-1/wire.jsonl", record(1794312000000, 10))
	if got := total(f.ledger("kimi").Days()); got != 355 {
		t.Errorf("total = %d, want 355: a fork copies the journal, and each agent's calls are its own", got)
	}
}

func TestGooseReadsItsUsageLedger(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.env["APPDATA"] = filepath.Join(f.home, "roaming")
	path := filepath.Join(f.home, ".local", "share", "goose", "sessions", "sessions.db")
	windows := filepath.Join(f.home, "roaming", "Block", "goose", "data", "sessions", "sessions.db")
	const table = "create table usage_ledger (id integer primary key autoincrement, session_id text not null, " +
		"created_timestamp integer not null, input_tokens integer, output_tokens integer, total_tokens integer, " +
		"cache_read_tokens integer, cost_source text)"
	const insert = "insert into usage_ledger (session_id, created_timestamp, input_tokens, output_tokens, total_tokens, cache_read_tokens, cost_source) values "
	for _, p := range []string{path, windows} {
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	exec(t, path, table,
		insert+"('20261110_1', 1794312000, null, null, 900, null, 'carried_forward')",
		insert+"('20261110_1', 1794312000, 120, 30, 150, 100, 'estimated')",
		insert+"('20261110_1', 1794312060, null, null, 40, null, null)",
	)
	exec(t, windows, table, insert+"('20261110_1', 1794312000, 1, 1, 2, 0, null)")
	l := f.ledger("goose")
	if got := l.Days(); !maps.Equal(got, map[string]int64{day1: 192}) {
		t.Fatalf("days = %v, want 192 on %s: input holds the cache reads, and carried-forward usage is not a call", got, day1)
	}
	exec(t, path, insert+"('20261110_2', 1794312120, 5, 5, 10, 0, 'provider_reported')")
	if got := total(update(t, l).Days()); got != 202 {
		t.Errorf("total = %d, want 202 after a new row", got)
	}
	fresh := filepath.Join(f.home, "fresh.db")
	exec(t, fresh, table, insert+"('20261110_3', 1794312180, 6, 6, 12, 0, null)")
	if err := os.Rename(fresh, path); err != nil {
		t.Fatal(err)
	}
	if got := total(update(t, l).Days()); got != 214 {
		t.Errorf("total = %d, want 214: a replaced database is read from its first row", got)
	}
	g := newFixture(t)
	g.env["GOOSE_PATH_ROOT"] = filepath.Join(g.home, "root")
	rooted := filepath.Join(g.home, "root", "data", "sessions", "sessions.db")
	if err := os.MkdirAll(filepath.Dir(rooted), 0o750); err != nil {
		t.Fatal(err)
	}
	exec(t, rooted, table, insert+"('s', 1794312000, 3, 4, 7, 0, null)")
	if got := total(g.ledger("goose").Days()); got != 7 {
		t.Errorf("total = %d, want 7 from GOOSE_PATH_ROOT", got)
	}
}

func TestTheEventFeedTakesAnyHarnessOnceEach(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	now := time.Date(2026, 11, 19, 9, 0, 0, 0, time.UTC)
	e := usage.FeedEvent{Source: tool, ID: "turn42", Tokens: 71, Timestamp: "2026-11-11T00:30:00+02:00"}
	for range 2 {
		if err := usage.Record(f.state, e, now); err != nil {
			t.Fatal(err)
		}
	}
	if err := usage.Record(f.state, usage.FeedEvent{Source: tool, ID: "now", Tokens: 1}, now); err != nil {
		t.Fatal(err)
	}
	extra := filepath.Join(f.home, "extra.jsonl")
	f.env["BONSAI_USAGE_FILE"] = extra + string(os.PathListSeparator) + filepath.Join(f.home, "missing.jsonl")
	f.append("extra.jsonl",
		`{"source":"second-tool","id":"turn42","tokens":3,"timestamp":"`+stamp+`"}`,
		`{"source":"my-tool","id":"true-count","tokens":true,"timestamp":"`+stamp+`"}`,
		`{"source":"my-tool","id":"string","tokens":"1250","timestamp":"`+stamp+`"}`,
		`{"source":"my-tool","id":"float","tokens":1250.0,"timestamp":"`+stamp+`"}`,
		`{"source":"my-tool","id":"negative","tokens":-5,"timestamp":"`+stamp+`"}`,
		`{"source":"my-tool","id":"words","tokens":5,"timestamp":"yesterday"}`,
		`{"source":"my-tool","id":"no-zone","tokens":5,"timestamp":"2026-11-11T12:00:00"}`,
		`{"source":"my-tool","id":"millis","tokens":5,"timestamp":1762862400000}`,
		`{"source":"my-tool","id":"no-tokens","timestamp":"`+stamp+`"}`,
		`not json at all`,
		" \r",
		`{"source":" ","id":"blank","tokens":5,"timestamp":"`+stamp+`"}`,
		`{"tokens": broken`)
	l := f.ledger("events")
	if want := map[string]int64{day1: 74, "2026-11-19": 1}; !maps.Equal(l.Days(), want) {
		t.Fatalf("days = %v, want %v", l.Days(), want)
	}
	f.append("extra.jsonl", `{"source":"my-tool","id":"next","tokens":7,"timestamp":"`+stamp+`"}`)
	if got := total(update(t, l).Days()); got != 82 {
		t.Errorf("total = %d, want 82", got)
	}
	f.append("extra.jsonl",
		`{"source":"my-tool","id":"next","tokens":9,"timestamp":"`+stamp+`"}`,
		`{"source":"my-tool","id":"next","tokens":2,"timestamp":"`+stamp+`"}`)
	if got := total(update(t, l).Days()); got != 84 {
		t.Errorf("total = %d, want 84: a call counts at the largest count it reports", got)
	}
	if stats, _ := l.Stats(); stats.Malformed != 11 {
		t.Errorf("malformed = %d, want 11: every line that breaks the feed's format, and no blank one", stats.Malformed)
	}
	files, _ := filepath.Glob(filepath.Join(f.state, "events", "*.jsonl"))
	if len(files) != 1 || filepath.Base(files[0]) != "2026-11.jsonl" {
		t.Errorf("feed files = %v, want this month's", files)
	}
}

func TestRecordRefusesAnEventThatBreaksTheFormat(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for _, e := range []usage.FeedEvent{
		{Source: "", ID: "a", Tokens: 1},
		{Source: "s", ID: " ", Tokens: 1},
		{Source: "s", ID: "a", Tokens: -1},
		{Source: "s", ID: "a", Tokens: 1, Timestamp: "2026-11-10 12:00"},
		{Source: "s", ID: "a", Tokens: 1, Timestamp: "2026-11-10T12:00:00"},
	} {
		if err := usage.Record(dir, e, time.Now()); !errors.Is(err, usage.ErrInvalidEvent) {
			t.Errorf("Record(%+v) = %v, want ErrInvalidEvent", e, err)
		}
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Error("a refused event wrote files")
	}
}

func TestAnUpdateStopsWhenItsContextEnds(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.append(".claude/projects/p/a.jsonl", claude("m1", day1, 100))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	u := f.usage()
	if err := u.Update(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("update error = %v, want context.Canceled", err)
	}
	if err := u.Update(t.Context()); err != nil || u.Days()["claude"][day1] != 100 {
		t.Errorf("the next update = %v, %v: want the whole count", err, u.Days())
	}
}

func BenchmarkFirstRead(b *testing.B) {
	home := b.TempDir()
	dir := filepath.Join(home, ".claude", "projects", "p")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		b.Fatal(err)
	}
	for file := range 20 {
		var lines strings.Builder
		for k := range 2000 {
			lines.WriteString(claude(fmt.Sprintf("m%d-%d", file, k), day1, 100))
			lines.WriteString("\n{\"type\":\"user\",\"message\":{\"content\":\"a prompt with no usage in it\"}}\n")
		}
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("%d.jsonl", file)), []byte(lines.String()), 0o600); err != nil {
			b.Fatal(err)
		}
	}
	env := usage.Env{Home: home, Getenv: func(string) string { return "" }, State: b.TempDir()}
	for b.Loop() {
		if err := usage.New(env).Update(b.Context()); err != nil {
			b.Fatal(err)
		}
	}
}
