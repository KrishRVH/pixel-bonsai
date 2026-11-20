package usage

import (
	"encoding/json"
	"path/filepath"
)

// piFiles finds the sessions of Pi and Oh My Pi, which share a format: under ~/.pi/agent/sessions and
// ~/.omp/agent/sessions, and under PI_CODING_AGENT_SESSION_DIR or the sessions folder in PI_CODING_AGENT_DIR.
func piFiles(env Env) []string {
	custom := env.Getenv("PI_CODING_AGENT_SESSION_DIR")
	if custom == "" {
		custom = filepath.Join(envDir(env, "PI_CODING_AGENT_DIR", ".pi", "agent"), "sessions")
	}
	roots := unique([]string{filepath.Join(env.Home, ".pi", "agent", "sessions"), filepath.Join(env.Home, ".omp", "agent", "sessions"), custom})
	var out []string
	for _, root := range roots {
		out = append(out, walk(root, "*.jsonl")...)
	}
	return out
}

type piLine struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	Timestamp *stamp `json:"timestamp"`
	Message   *struct {
		Role       string `json:"role"`
		ResponseID string `json:"responseId"`
		Timestamp  *stamp `json:"timestamp"`
		Usage      *struct {
			Total      count `json:"totalTokens"`
			Input      count `json:"input"`
			Output     count `json:"output"`
			CacheRead  count `json:"cacheRead"`
			CacheWrite count `json:"cacheWrite"`
		} `json:"usage"`
	} `json:"message"`
}

// piParser reads assistant messages' usage. A forked session copies its parent's entries, so the key is the
// provider's response id or, without one, the entry's id and time.
func piParser(string) lineParser {
	return func(line []byte) (event, bool, error) {
		var l piLine
		if err := json.Unmarshal(line, &l); err != nil {
			return event{}, false, err
		}
		m := l.Message
		if l.Type != "message" || m == nil || m.Role != assistant || m.Usage == nil {
			return event{}, false, nil
		}
		when := l.Timestamp
		if when == nil || when.Time.IsZero() {
			when = m.Timestamp
		}
		if when == nil {
			when = &stamp{}
		}
		u := m.Usage
		tokens := int64(u.Total)
		if tokens == 0 {
			tokens = sum(u.Input, u.Output, u.CacheRead, u.CacheWrite)
		}
		switch {
		case m.ResponseID != "":
			return event{key("response", m.ResponseID), when.Time, tokens}, true, nil
		case l.ID != "":
			return event{key("entry", l.ID, when.Raw), when.Time, tokens}, true, nil
		}
		return event{}, false, nil
	}
}
