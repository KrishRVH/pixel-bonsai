package usage

import (
	"encoding/json"
	"path/filepath"
	"strconv"
)

// antigravityFiles finds the transcripts of Antigravity's app, CLI, and IDE conversations.
func antigravityFiles(env Env) []string {
	return glob(filepath.Join(env.Home, ".gemini", "antigravity*", "brain", "*", ".system_generated", "logs", "transcript.jsonl"))
}

type antigravityLine struct {
	Source        string `json:"source"`
	Status        string `json:"status"`
	Step          *int64 `json:"step_index"`
	CreatedAt     stamp  `json:"created_at"`
	Total         count  `json:"total_tokens"`
	Input         count  `json:"input_tokens"`
	Output        count  `json:"output_tokens"`
	CacheRead     count  `json:"cache_read_tokens"`
	CacheCreation count  `json:"cache_creation_input_tokens"`
}

// antigravityParser reads completed model steps. A step is logged as it runs and again when done, so the key is the
// conversation, named by the transcript's folder, and the step's index.
func antigravityParser(path string) lineParser {
	conversation := filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(path))))
	return func(line []byte) (event, bool, error) {
		var l antigravityLine
		if err := json.Unmarshal(line, &l); err != nil {
			return event{}, false, err
		}
		if l.Source != "MODEL" || l.Status != "DONE" || l.Step == nil {
			return event{}, false, nil
		}
		tokens := int64(l.Total)
		if tokens == 0 {
			tokens = sum(l.Input, l.Output, l.CacheRead, l.CacheCreation)
		}
		return event{key(conversation, strconv.FormatInt(*l.Step, 10)), l.CreatedAt.Time, tokens}, true, nil
	}
}
