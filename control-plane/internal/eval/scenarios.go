package eval

import (
	"embed"
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed scenarios/*.json
var scenarioFS embed.FS

func LoadScenarios() ([]Scenario, error) {
	entries, err := scenarioFS.ReadDir("scenarios")
	if err != nil {
		return nil, err
	}
	out := make([]Scenario, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		// proof-*.json is the REST adversarial harness only (see proof_rest_runner.go).
		if strings.HasPrefix(e.Name(), "proof-") {
			continue
		}
		b, err := scenarioFS.ReadFile("scenarios/" + e.Name())
		if err != nil {
			return nil, err
		}
		var s Scenario
		if err := json.Unmarshal(b, &s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
