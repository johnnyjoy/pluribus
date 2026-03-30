package eval

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// ProofScenario is an adversarial REST-level proof definition (scenarios/proof-*.json).
type ProofScenario struct {
	ID              string            `json:"id"`
	Suite           string            `json:"suite"`
	Description     string            `json:"description"`
	AfterInvariants []string          `json:"after_invariants,omitempty"`
	Steps           []ProofStep       `json:"steps"`
}

// ProofStep is one HTTP call in a proof scenario.
type ProofStep struct {
	ID              string            `json:"id"`
	Method          string            `json:"method"`
	Path            string            `json:"path"`
	Body            json.RawMessage   `json:"body,omitempty"`
	ExpectStatus    int               `json:"expect_status"`
	CaptureFields   map[string]string `json:"capture_fields,omitempty"`   // var name -> top-level JSON field in response
	CaptureJSONPath map[string]string `json:"capture_json_path,omitempty"` // var name -> dot path (e.g. candidates.0.candidate_id)
	StoreAs         string            `json:"store_as,omitempty"`         // raw response body stored for after_invariants
	Asserts         []ProofAssert     `json:"asserts,omitempty"`
}

// ProofAssert is a lightweight check on the response body (string-level for stability across minor JSON formatting).
type ProofAssert struct {
	Kind    string   `json:"kind"` // body_contains | body_not_contains | substring_order | body_contains_any
	Value   string   `json:"value,omitempty"`
	Values  []string `json:"values,omitempty"` // body_contains_any: at least one substring must match
	Before  string   `json:"before,omitempty"` // substring_order: "before" must appear before "after" in body
	After   string   `json:"after,omitempty"`
}

// ProofRunContext carries substitution and captured state across steps.
type ProofRunContext struct {
	RunID  string
	Vars   map[string]string
	Stored map[string][]byte
}

// ProofStepResult records one executed step.
type ProofStepResult struct {
	ScenarioID string
	StepID     string
	Path       string
	Status     int
	Pass       bool
	Detail     string
}

// ProofInvariantResult is the outcome of a named invariant.
type ProofInvariantResult struct {
	Name   string
	Pass   bool
	Detail string
}

// ProofScenarioReport aggregates one scenario run.
type ProofScenarioReport struct {
	ScenarioID  string
	Suite       string
	Steps       []ProofStepResult
	Invariants  []ProofInvariantResult
	AllPassed   bool
	Description string
}

// ProofHarnessReport is the full adversarial run.
type ProofHarnessReport struct {
	RunID     string
	Scenarios []ProofScenarioReport
	AllPassed bool
	// DeterminismPass is set when RunProofHarnessRESTDeterminism compares two identical runs.
	DeterminismPass bool
	DeterminismNote string
}

// LoadProofScenarios loads only embedded scenarios/proof-*.json files (same FS as LoadScenarios).
func LoadProofScenarios() ([]ProofScenario, error) {
	entries, err := scenarioFS.ReadDir("scenarios")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.HasPrefix(n, "proof-") && filepath.Ext(n) == ".json" {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	out := make([]ProofScenario, 0, len(names))
	for _, n := range names {
		b, err := scenarioFS.ReadFile("scenarios/" + n)
		if err != nil {
			return nil, err
		}
		var s ProofScenario
		if err := json.Unmarshal(b, &s); err != nil {
			return nil, fmt.Errorf("proof scenario %s: %w", n, err)
		}
		if strings.TrimSpace(s.ID) == "" {
			return nil, fmt.Errorf("proof scenario %s: missing id", n)
		}
		out = append(out, s)
	}
	return out, nil
}
