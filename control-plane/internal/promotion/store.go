package promotion

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// AppendRecord appends one JSON object as a single line (JSONL).
func AppendRecord(path string, rec ExperienceRecord) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(rec); err != nil {
		return err
	}
	return nil
}
