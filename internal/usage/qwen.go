package usage

import (
	"encoding/json"
	"path/filepath"
)

// qwenFiles finds Qwen Code's monthly usage ledgers in each runtime folder it may use: ~/.qwen, QWEN_HOME, and
// QWEN_RUNTIME_DIR.
func qwenFiles(env Env) []string {
	roots := []string{filepath.Join(env.Home, ".qwen")}
	for _, variable := range []string{"QWEN_HOME", "QWEN_RUNTIME_DIR"} {
		if dir := env.Getenv(variable); dir != "" {
			roots = append(roots, dir)
		}
	}
	var out []string
	for _, root := range unique(roots) {
		out = append(out, glob(filepath.Join(root, "usage", "token-usage-*.jsonl"))...)
	}
	return out
}

type qwenLine struct {
	ID        string `json:"id"`
	Timestamp stamp  `json:"timestamp"`
	Total     count  `json:"totalTokens"`
}

// qwenParser reads the ledger's records, one for each model call under an id of its own. The total already holds the
// cached input.
func qwenParser(string) lineParser {
	return func(line []byte) (event, bool, error) {
		var l qwenLine
		if err := json.Unmarshal(line, &l); err != nil {
			return event{}, false, err
		}
		return event{l.ID, l.Timestamp.Time, int64(l.Total)}, true, nil
	}
}
