package agentobedience

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

// LoadCases reads obedience fixtures from testdata.
func LoadCases(path string) ([]ObedienceCase, error) {
	if path == "" {
		path = filepath.Join(DefaultCasesDir(), "cases.json")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f CasesFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	return f.Cases, nil
}

// DefaultCasesDir returns testdata/agent_contract_obedience.
func DefaultCasesDir() string {
	root := repoRoot()
	return filepath.Join(root, "control-plane", "testdata", "agent_contract_obedience")
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	dir := filepath.Dir(file)
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "control-plane", "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Dir(file)
}
