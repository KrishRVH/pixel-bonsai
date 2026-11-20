package usage

import (
	"encoding/json"
	"path/filepath"
	"strconv"
)

// codexFiles finds Codex's rollouts, active and archived, under CODEX_HOME (default ~/.codex).
func codexFiles(env Env) []string {
	root := envDir(env, "CODEX_HOME", ".codex")
	return append(walk(filepath.Join(root, "sessions"), "rollout-*.jsonl"), walk(filepath.Join(root, "archived_sessions"), "rollout-*.jsonl")...)
}

type codexLine struct {
	Type      string `json:"type"`
	Timestamp stamp  `json:"timestamp"`
	Payload   *struct {
		Type string `json:"type"`
		Info *struct {
			Total *struct {
				Tokens count `json:"total_tokens"`
			} `json:"total_token_usage"`
			Last *struct {
				Tokens count `json:"total_tokens"`
			} `json:"last_token_usage"`
		} `json:"info"`
	} `json:"payload"`
}

// codexParser reads token-count events. Rate-limit refreshes repeat the same counters, so an event whose running total
// matches the file's last one is skipped. A fork copies its parent's events into a new rollout, so the key is the
// event's time and counters, which the copy shares.
func codexParser(string) lineParser {
	var lastTotal int64
	return func(line []byte) (event, bool, error) {
		var l codexLine
		if err := json.Unmarshal(line, &l); err != nil {
			return event{}, false, err
		}
		p := l.Payload
		if l.Type != "event_msg" || p == nil || p.Type != "token_count" || p.Info == nil || p.Info.Total == nil {
			return event{}, false, nil
		}
		total := int64(p.Info.Total.Tokens)
		if total == 0 || total == lastTotal || p.Info.Last == nil || p.Info.Last.Tokens == 0 {
			return event{}, false, nil
		}
		lastTotal = total
		last := int64(p.Info.Last.Tokens)
		return event{key(l.Timestamp.Raw, strconv.FormatInt(total, 10), strconv.FormatInt(last, 10)), l.Timestamp.Time, last}, true, nil
	}
}
