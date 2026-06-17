package guardrails

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var deleteFromMemoriesPattern = regexp.MustCompile(`(?i)DELETE\s+FROM\s+memories`)

var deleteFromMemoriesAllowlist = []string{
	filepath.Join("internal", "experiments", "pgtextsearch", "seed.go"),
}

func TestNoProductionDeleteFromMemories(t *testing.T) {
	root := controlPlaneModuleRoot(t)
	var violations []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			base := info.Name()
			if base == "vendor" || base == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		for _, allow := range deleteFromMemoriesAllowlist {
			if rel == allow {
				return nil
			}
		}
		if strings.HasSuffix(rel, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if deleteFromMemoriesPattern.Match(b) {
			violations = append(violations, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(violations) > 0 {
		t.Fatalf("production DELETE FROM memories found in: %v", violations)
	}
}
