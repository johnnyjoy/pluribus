package promotion

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"control-plane/internal/merge"
	"control-plane/internal/runmulti"
	"control-plane/internal/signal"
)

func TestAppendRecord_createsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "exp.jsonl")
	rec := ExperienceRecord{
		Type:      experienceType,
		Timestamp: time.Now().UTC(),
		Content:   "hello",
		Score:     1.5,
	}
	if err := AppendRecord(path, rec); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got ExperienceRecord
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Content != "hello" || got.Score != 1.5 {
		t.Errorf("decoded %+v", got)
	}
}

func TestPromoteFromMerge_writesWhenHighSignal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "e.jsonl")
	m := merge.MergeResult{
		MergedOutput: strings.TrimSpace(`
[CORE AGREEMENTS]
- first long agreement line for scoring purposes here
- second long agreement line for scoring purposes here

[VALID UNIQUE ADDITIONS]
(none)

[REFINED STRUCTURE]
ok`),
		UsedVariants: []string{"balanced", "failure_heavy"},
		Agreements: []string{
			"first long agreement line for scoring purposes here",
			"second long agreement line for scoring purposes here",
		},
		FallbackUsed: false,
		Drift:        runmulti.DriftResult{},
	}
	ok, err := PromoteFromMerge(context.Background(), m, PromotionInput{StorePath: path}, signal.DefaultSignalConfig())
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected promoted")
	}
	b, err := os.ReadFile(path)
	if err != nil || len(b) == 0 {
		t.Fatalf("read: %v len=%d", err, len(b))
	}
}

func TestPromoteFromMerge_noWriteOnFallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "e.jsonl")
	m := merge.MergeResult{FallbackUsed: true, MergedOutput: "x"}
	ok, err := PromoteFromMerge(context.Background(), m, PromotionInput{StorePath: path}, signal.DefaultSignalConfig())
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("should not promote")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should not exist")
	}
}
