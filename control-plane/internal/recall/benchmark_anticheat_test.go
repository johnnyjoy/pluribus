package recall_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestRecallScoringDoesNotReferenceBenchmarkFixtures scans recall scoring sources for
// benchmark case IDs, fixture labels, and fixture statement fragments.
func TestRecallScoringDoesNotReferenceBenchmarkFixtures(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	recallDir := filepath.Dir(thisFile)
	benchDir := filepath.Join(recallDir, "..", "..", "testdata", "recall_benchmark")

	casesRaw, err := os.ReadFile(filepath.Join(benchDir, "cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	memRaw, err := os.ReadFile(filepath.Join(benchDir, "memories.json"))
	if err != nil {
		t.Fatal(err)
	}

	var cases struct {
		Cases []struct {
			ID string `json:"id"`
		} `json:"cases"`
	}
	var mems struct {
		Memories []struct {
			Label     string `json:"label"`
			Statement string `json:"statement"`
		} `json:"memories"`
	}
	if err := json.Unmarshal(casesRaw, &cases); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(memRaw, &mems); err != nil {
		t.Fatal(err)
	}

	forbidden := map[string]string{}
	for _, c := range cases.Cases {
		if c.ID != "" {
			forbidden[c.ID] = "case_id"
		}
	}
	for _, m := range mems.Memories {
		if m.Label != "" {
			forbidden[m.Label] = "fixture_label"
		}
		stmt := strings.TrimSpace(m.Statement)
		if len(stmt) >= 24 {
			forbidden[stmt[:24]] = "fixture_statement_prefix"
		}
	}

	// Product-legitimate vocabulary may appear in general domain logic; document exceptions here.
	allowedSubstrings := []string{
		"pluribus", // product name in general enforcement/query affinity helpers
		"onguard",  // allowed only if absent — checked separately below
	}
	_ = allowedSubstrings

	scoringFiles := []string{
		"relevance_scoring.go",
		"compound_phrases.go",
		"query_modes.go",
		"bundle_rank.go",
		"scorer.go",
		"lifecycle_candidates.go",
	}
	for _, name := range scoringFiles {
		path := filepath.Join(recallDir, name)
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		lower := strings.ToLower(string(body))
		for needle, kind := range forbidden {
			if strings.Contains(needle, "onguard") {
				if strings.Contains(lower, strings.ToLower(needle)) {
					t.Errorf("%s contains benchmark %s %q", name, kind, needle)
				}
				continue
			}
			if kind == "case_id" || kind == "fixture_label" {
				if strings.Contains(lower, strings.ToLower(needle)) {
					t.Errorf("%s contains benchmark %s %q", name, kind, needle)
				}
			}
			if kind == "fixture_statement_prefix" {
				if strings.Contains(lower, strings.ToLower(needle)) {
					t.Errorf("%s contains benchmark %s %q", name, kind, needle)
				}
			}
		}
	}
}
