// Package proofscenarios loads minimal YAML proof scenarios (benefit receipts, not a benchmark framework).
package proofscenarios

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Mode values for scenario execution classification.
const (
	ModeAutomatedCore = "automated_core"
	ModeIntegration   = "integration"
	ModeManual        = "manual"
)

// Scenario is a single proof definition (see docs/proof-scenarios.md).
type Scenario struct {
	ID           string         `yaml:"id"`
	Version      int            `yaml:"version"`
	Title        string         `yaml:"title"`
	Mode         string         `yaml:"mode"`
	Category     string         `yaml:"category"`
	BenefitClaim string         `yaml:"benefit_claim"`
	Tags         []string       `yaml:"tags,omitempty"`
	Seed         map[string]any `yaml:"seed,omitempty"`
	Stimulus     map[string]any `yaml:"stimulus,omitempty"`
	Expected     map[string]any `yaml:"expected,omitempty"`
}

// LoadDir reads all *.yaml files from dir (non-recursive).
func LoadDir(dir string) ([]Scenario, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.HasPrefix(n, "_") || strings.HasPrefix(n, ".") {
			continue // templates / local-only; not part of numbered suite
		}
		if strings.HasSuffix(strings.ToLower(n), ".yaml") || strings.HasSuffix(strings.ToLower(n), ".yml") {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	var out []Scenario
	for _, n := range names {
		path := filepath.Join(dir, n)
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var sc Scenario
		if err := yaml.Unmarshal(b, &sc); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		out = append(out, sc)
	}
	return out, nil
}

// Validate checks required fields and mode.
func Validate(sc *Scenario) error {
	if strings.TrimSpace(sc.ID) == "" {
		return fmt.Errorf("missing id")
	}
	if sc.Version < 1 {
		return fmt.Errorf("id=%s: version must be >= 1", sc.ID)
	}
	if strings.TrimSpace(sc.Title) == "" {
		return fmt.Errorf("id=%s: missing title", sc.ID)
	}
	switch sc.Mode {
	case ModeAutomatedCore, ModeIntegration, ModeManual:
	default:
		return fmt.Errorf("id=%s: invalid mode %q", sc.ID, sc.Mode)
	}
	if strings.TrimSpace(sc.Category) == "" {
		return fmt.Errorf("id=%s: missing category", sc.ID)
	}
	if strings.TrimSpace(sc.BenefitClaim) == "" {
		return fmt.Errorf("id=%s: missing benefit_claim", sc.ID)
	}
	return nil
}

// ValidateUniqueIDs returns an error if duplicate scenario ids appear.
func ValidateUniqueIDs(scenarios []Scenario) error {
	seen := make(map[string]struct{})
	for _, sc := range scenarios {
		if _, ok := seen[sc.ID]; ok {
			return fmt.Errorf("duplicate scenario id: %s", sc.ID)
		}
		seen[sc.ID] = struct{}{}
	}
	return nil
}
