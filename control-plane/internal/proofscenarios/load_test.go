package proofscenarios

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	dir := filepath.Dir(file)
	for i := 0; i < 10; i++ {
		mod := filepath.Join(dir, "go.mod")
		if st, err := os.Stat(mod); err == nil && !st.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("go.mod not found")
	return ""
}

func TestLoadDir_proofScenariosValid(t *testing.T) {
	root := moduleRoot(t)
	dir := filepath.Join(root, "proof-scenarios")
	scenarios, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	if len(scenarios) < 6 {
		t.Fatalf("expected at least 6 scenarios, got %d", len(scenarios))
	}
	if err := ValidateUniqueIDs(scenarios); err != nil {
		t.Fatal(err)
	}
	for i := range scenarios {
		if err := Validate(&scenarios[i]); err != nil {
			t.Errorf("scenario %d: %v", i, err)
		}
	}
}
