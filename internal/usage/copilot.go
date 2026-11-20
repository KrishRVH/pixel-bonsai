package usage

import (
	"encoding/json"
	"path/filepath"
	"strconv"
)

// copilotFiles finds GitHub Copilot CLI's session journals, live and rotated, under COPILOT_HOME (default ~/.copilot).
func copilotFiles(env Env) []string {
	return glob(filepath.Join(envDir(env, "COPILOT_HOME", ".copilot"), "session-state", "*", "events*.jsonl"))
}

type copilotLine struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	Timestamp stamp  `json:"timestamp"`
	Data      *struct {
		Usage *struct {
			Input      count `json:"inputTokens"`
			Output     count `json:"outputTokens"`
			Accounting *struct {
				Session  string `json:"sourceSessionId"`
				Sequence *count `json:"sequence"`
			} `json:"accounting"`
		} `json:"usage"`
	} `json:"data"`
}

// copilotParser reads usage records, one for each model call. Input already holds the cache reads and writes, and
// output the reasoning. A rotated journal starts with copies of earlier records and a fork copies its parent's, so the
// key is the session that first recorded the call and the call's place in it, or the event's id for a record without
// them.
func copilotParser(string) lineParser {
	return func(line []byte) (event, bool, error) {
		var l copilotLine
		if err := json.Unmarshal(line, &l); err != nil {
			return event{}, false, err
		}
		if l.Type != "session.usage_record" || l.Data == nil || l.Data.Usage == nil {
			return event{}, false, nil
		}
		u := l.Data.Usage
		var id string
		switch a := u.Accounting; {
		case a != nil && a.Session != "" && a.Sequence != nil:
			id = key("call", a.Session, strconv.FormatInt(int64(*a.Sequence), 10))
		case l.ID != "":
			id = key("event", l.ID)
		default:
			return event{}, false, nil
		}
		return event{id, l.Timestamp.Time, sum(u.Input, u.Output)}, true, nil
	}
}
