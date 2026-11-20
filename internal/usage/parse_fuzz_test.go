package usage

import "testing"

// FuzzParsers feeds every source's parser arbitrary lines. A parser never panics, and what it counts is never
// negative.
func FuzzParsers(f *testing.F) {
	for _, seed := range []string{
		`{"type":"assistant","requestId":"r","timestamp":"2026-11-10T12:00:00Z","message":{"id":"m","usage":{"input_tokens":5}}}`,
		`{"type":"event_msg","timestamp":"2026-11-10T12:00:00Z","payload":{"type":"token_count","info":{"total_token_usage":{"total_tokens":9},"last_token_usage":{"total_tokens":9}}}}`,
		`{"step_index":3,"source":"MODEL","status":"DONE","created_at":"2026-11-10T12:00:00Z","input_tokens":-4}`,
		`{"type":"message","id":"e","timestamp":1794312000000,"message":{"role":"assistant","usage":{"totalTokens":1e99}}}`,
		`{"source":"s","id":"i","timestamp":"2026-11-10T12:00:00+02:00","tokens":"7"}`,
		`{"type":"session.usage_record","id":"e","timestamp":"2026-11-10T12:00:00Z","data":{"usage":{"inputTokens":5,"accounting":{"sourceSessionId":"s","sequence":-1}}}}`,
		`{"schemaVersion":1,"id":"q","timestamp":"2026-11-10T12:00:00Z","inputTokens":4,"outputTokens":2,"totalTokens":"6"}`,
		`{"type":"usage.record","model":"k","usage":{"inputOther":10,"output":5,"inputCacheRead":-3},"usageScope":"turn","time":1794312000000}`,
		`{"tokens":null,"usage":[],"message":7}`,
		"",
	} {
		f.Add([]byte(seed))
	}
	parsers := []func(string) lineParser{claudeParser, codexParser, copilotParser, antigravityParser, piParser, qwenParser, kimiParser, feedParser}
	f.Fuzz(func(t *testing.T, line []byte) {
		for _, parser := range parsers {
			e, ok, err := parser("/sessions/a/.system_generated/logs/transcript.jsonl")(line)
			if ok && (err != nil || e.Tokens < 0) {
				t.Errorf("parsed %q to %+v, %v", line, e, err)
			}
		}
	})
}
