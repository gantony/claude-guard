package decision

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Entry is a single decision record written to the JSONL log.
type Entry struct {
	Timestamp time.Time `json:"ts"`
	Tool      string    `json:"tool"`
	Input     string    `json:"input"`
	Rule      string    `json:"rule,omitempty"`
	Decision  string    `json:"decision"` // "allow" or "skip"
	SessionID string    `json:"session,omitempty"`
}

// Log appends an entry to the JSONL file at path.
func Log(path string, entry Entry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(entry)
}
