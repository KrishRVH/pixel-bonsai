package usage

import (
	"encoding/json"
	"path/filepath"
)

// claudeFiles finds Claude Code's transcripts: every JSONL file under projects/ in ~/.claude and its siblings named
// .claude*, and in CLAUDE_CONFIG_DIR when it is set.
func claudeFiles(env Env) []string {
	roots := glob(filepath.Join(env.Home, ".claude*"))
	if dir := env.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		roots = append(roots, dir)
	}
	var out []string
	for _, root := range unique(roots) {
		out = append(out, walk(filepath.Join(root, "projects"), "*.jsonl")...)
	}
	return unique(out)
}

type claudeLine struct {
	Type      string `json:"type"`
	RequestID string `json:"requestId"`
	Timestamp stamp  `json:"timestamp"`
	Message   *struct {
		ID    string `json:"id"`
		Usage *struct {
			Input         count `json:"input_tokens"`
			CacheCreation count `json:"cache_creation_input_tokens"`
			CacheRead     count `json:"cache_read_input_tokens"`
			Output        count `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

// claudeParser reads an assistant message's usage. A message repeats across the lines of a streamed response and in
// a resumed session's copy of the transcript, so its key is the message and request ids.
func claudeParser(string) lineParser {
	return func(line []byte) (event, bool, error) {
		var l claudeLine
		if err := json.Unmarshal(line, &l); err != nil {
			return event{}, false, err
		}
		m := l.Message
		if l.Type != assistant || m == nil || m.Usage == nil || m.ID == "" {
			return event{}, false, nil
		}
		u := m.Usage
		return event{key(m.ID, l.RequestID), l.Timestamp.Time, sum(u.Input, u.CacheCreation, u.CacheRead, u.Output)}, true, nil
	}
}
