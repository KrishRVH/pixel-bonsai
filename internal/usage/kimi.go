package usage

import (
	"encoding/json"
	"path/filepath"
)

// kimiFiles finds Kimi Code's journals, one for each agent of each session, under KIMI_CODE_HOME (default
// ~/.kimi-code).
func kimiFiles(env Env) []string {
	return glob(filepath.Join(envDir(env, "KIMI_CODE_HOME", ".kimi-code"), "sessions", "*", "*", "agents", "*", "wire.jsonl"))
}

type kimiLine struct {
	Type  string `json:"type"`
	Model string `json:"model"`
	Scope string `json:"usageScope"`
	Time  stamp  `json:"time"`
	Usage *struct {
		Input      count `json:"inputOther"`
		CacheRead  count `json:"inputCacheRead"`
		CacheWrite count `json:"inputCacheCreation"`
		Output     count `json:"output"`
	} `json:"usage"`
}

// kimiParser reads usage records, one for each model call; its input leaves out the cache reads and writes. A record
// has no id, and a fork copies an agent's journal whole into a new session, so the key is the agent, named by its
// folder, with the record's time, model, scope, and counts.
func kimiParser(path string) lineParser {
	agent := filepath.Base(filepath.Dir(path))
	return func(line []byte) (event, bool, error) {
		var l kimiLine
		if err := json.Unmarshal(line, &l); err != nil {
			return event{}, false, err
		}
		if l.Type != "usage.record" || l.Usage == nil {
			return event{}, false, nil
		}
		u := l.Usage
		id := key(agent, l.Time.Raw, l.Model, l.Scope, counts(u.Input, u.CacheRead, u.CacheWrite, u.Output))
		return event{id, l.Time.Time, sum(u.Input, u.CacheRead, u.CacheWrite, u.Output)}, true, nil
	}
}
