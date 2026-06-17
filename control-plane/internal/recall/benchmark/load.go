package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

// FixtureMemory is one labeled memory in the benchmark corpus.
type FixtureMemory struct {
	Label         string   `json:"label"`
	Domain        string   `json:"domain"`
	Kind          string   `json:"kind"`
	Statement     string   `json:"statement"`
	Authority     int      `json:"authority"`
	Tags          []string `json:"tags"`
	Status        string   `json:"status,omitempty"`
	ExpectedUse   string   `json:"expected_use,omitempty"`
	ForbiddenUse  string   `json:"forbidden_use,omitempty"`
	Applicability string   `json:"applicability,omitempty"`
	OccurredAt    string   `json:"occurred_at,omitempty"`
	CreatedAt     string   `json:"created_at,omitempty"`
	UtilityScore  float64  `json:"utility_score,omitempty"`
	Refuted       bool     `json:"refuted,omitempty"`
}

// FixtureCase is one benchmark query with expected/forbidden labels.
type FixtureCase struct {
	ID                  string   `json:"id"`
	Description         string   `json:"description"`
	Query               string   `json:"query"`
	Tags                []string `json:"tags"`
	RepoRootHint        string   `json:"repo_root_hint,omitempty"`
	ExpectedDomain      string   `json:"expected_domain,omitempty"`
	ExpectedLabels      []string `json:"expected_labels"`
	AcceptableLabels    []string `json:"acceptable_labels,omitempty"`
	ForbiddenLabels     []string `json:"forbidden_labels"`
	K                   int      `json:"k"`
	MinimumRecallAtK    float64  `json:"minimum_recall_at_k"`
	MinimumPrecisionAtK float64  `json:"minimum_precision_at_k"`
	MaximumForbiddenHits int     `json:"maximum_forbidden_hits"`
	RequireResults      bool     `json:"require_results,omitempty"`
	Hostile             string            `json:"hostile,omitempty"`
	RecallMode          string            `json:"recall_mode,omitempty"`
	IncludeStatus       []string          `json:"include_status,omitempty"`
	OccurredAfter       string            `json:"occurred_after,omitempty"`
	OccurredBefore      string            `json:"occurred_before,omitempty"`
	HybridOnly          bool              `json:"hybrid_only,omitempty"`
	MaxRankForLabel     map[string]int    `json:"max_rank_for_label,omitempty"`
	RequiredBeforeLabel map[string]string `json:"required_before_label,omitempty"`
}

// Corpus holds loaded fixtures.
type Corpus struct {
	Memories []FixtureMemory `json:"memories"`
	Cases    []FixtureCase   `json:"cases"`
}

// LoadedCorpus maps labels to objects for recall compile.
type LoadedCorpus struct {
	Corpus
	LabelToID   map[string]uuid.UUID
	IDToLabel   map[uuid.UUID]string
	IDToDomain  map[uuid.UUID]string
	IDToFixture map[uuid.UUID]FixtureMemory
	Objects     []memory.MemoryObject
}

// DefaultDir returns testdata/recall_benchmark relative to repo root.
func DefaultDir() string {
	if dir := strings.TrimSpace(os.Getenv("RECALL_BENCHMARK_DIR")); dir != "" {
		return dir
	}
	root := findRepoRoot()
	return filepath.Join(root, "control-plane", "testdata", "recall_benchmark")
}

func findRepoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	// .../control-plane/internal/recall/benchmark/load.go -> repo root
	dir := filepath.Dir(file)
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "control-plane", "testdata", "recall_benchmark")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "..")
}

// Load reads memories.json and cases.json from dir.
func Load(dir string) (*LoadedCorpus, error) {
	memPath := filepath.Join(dir, "memories.json")
	casePath := filepath.Join(dir, "cases.json")
	memBytes, err := os.ReadFile(memPath)
	if err != nil {
		return nil, fmt.Errorf("read memories: %w", err)
	}
	caseBytes, err := os.ReadFile(casePath)
	if err != nil {
		return nil, fmt.Errorf("read cases: %w", err)
	}
	var memFile struct {
		Memories []FixtureMemory `json:"memories"`
	}
	if err := json.Unmarshal(memBytes, &memFile); err != nil {
		return nil, fmt.Errorf("parse memories: %w", err)
	}
	var caseFile struct {
		Cases []FixtureCase `json:"cases"`
	}
	if err := json.Unmarshal(caseBytes, &caseFile); err != nil {
		return nil, fmt.Errorf("parse cases: %w", err)
	}
	return BuildLoaded(memFile.Memories, caseFile.Cases)
}

// LoadSemantic reads base corpus plus semantic_memories.json and semantic_cases.json.
func LoadSemantic(dir string) (*LoadedCorpus, error) {
	base, err := Load(dir)
	if err != nil {
		return nil, err
	}
	semMemPath := filepath.Join(dir, "semantic_memories.json")
	semCasePath := filepath.Join(dir, "semantic_cases.json")
	semMemBytes, err := os.ReadFile(semMemPath)
	if err != nil {
		return nil, fmt.Errorf("read semantic memories: %w", err)
	}
	semCaseBytes, err := os.ReadFile(semCasePath)
	if err != nil {
		return nil, fmt.Errorf("read semantic cases: %w", err)
	}
	var semMemFile struct {
		Memories []FixtureMemory `json:"memories"`
	}
	if err := json.Unmarshal(semMemBytes, &semMemFile); err != nil {
		return nil, fmt.Errorf("parse semantic memories: %w", err)
	}
	var semCaseFile struct {
		Cases []FixtureCase `json:"cases"`
	}
	if err := json.Unmarshal(semCaseBytes, &semCaseFile); err != nil {
		return nil, fmt.Errorf("parse semantic cases: %w", err)
	}
	allMem := append(append([]FixtureMemory{}, base.Memories...), semMemFile.Memories...)
	allCases := append(append([]FixtureCase{}, base.Cases...), semCaseFile.Cases...)
	return BuildLoaded(allMem, allCases)
}

// LoadSemanticOnly loads only semantic extension fixtures (hostile hybrid gate).
func LoadSemanticOnly(dir string) (*LoadedCorpus, error) {
	semMemPath := filepath.Join(dir, "semantic_memories.json")
	semCasePath := filepath.Join(dir, "semantic_cases.json")
	semMemBytes, err := os.ReadFile(semMemPath)
	if err != nil {
		return nil, fmt.Errorf("read semantic memories: %w", err)
	}
	semCaseBytes, err := os.ReadFile(semCasePath)
	if err != nil {
		return nil, fmt.Errorf("read semantic cases: %w", err)
	}
	var semMemFile struct {
		Memories []FixtureMemory `json:"memories"`
	}
	if err := json.Unmarshal(semMemBytes, &semMemFile); err != nil {
		return nil, fmt.Errorf("parse semantic memories: %w", err)
	}
	var semCaseFile struct {
		Cases []FixtureCase `json:"cases"`
	}
	if err := json.Unmarshal(semCaseBytes, &semCaseFile); err != nil {
		return nil, fmt.Errorf("parse semantic cases: %w", err)
	}
	return BuildLoaded(semMemFile.Memories, semCaseFile.Cases)
}

// BuildLoaded constructs runtime objects from fixture definitions.
func BuildLoaded(memories []FixtureMemory, cases []FixtureCase) (*LoadedCorpus, error) {
	lc := &LoadedCorpus{
		Corpus:      Corpus{Memories: memories, Cases: cases},
		LabelToID:   map[string]uuid.UUID{},
		IDToLabel:   map[uuid.UUID]string{},
		IDToDomain:  map[uuid.UUID]string{},
		IDToFixture: map[uuid.UUID]FixtureMemory{},
	}
	now := time.Now().UTC()
	for _, fm := range memories {
		if fm.Label == "" {
			return nil, fmt.Errorf("memory missing label")
		}
		if _, dup := lc.LabelToID[fm.Label]; dup {
			return nil, fmt.Errorf("duplicate label %q", fm.Label)
		}
		id := stableID(fm.Label)
		lc.LabelToID[fm.Label] = id
		lc.IDToLabel[id] = fm.Label
		lc.IDToDomain[id] = fm.Domain
		lc.IDToFixture[id] = fm
		status := strings.TrimSpace(fm.Status)
		if status == "" {
			status = string(api.StatusActive)
		}
		kind := api.MemoryKind(strings.TrimSpace(fm.Kind))
		app := api.Applicability(strings.TrimSpace(fm.Applicability))
		if app == "" {
			if fm.Authority >= 8 {
				app = api.ApplicabilityGoverning
			} else {
				app = api.ApplicabilityAdvisory
			}
		}
		tags := append([]string(nil), fm.Tags...)
		tags = append(tags, "bench:"+fm.Label, "domain:"+fm.Domain)
		createdAt := now
		if s := strings.TrimSpace(fm.CreatedAt); s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				createdAt = t.UTC()
			}
		}
		obj := memory.MemoryObject{
			ID:            id,
			Kind:          kind,
			Statement:     fm.Statement,
			Authority:     fm.Authority,
			Tags:          tags,
			Status:        api.Status(status),
			Applicability: app,
			CreatedAt:     createdAt,
			UpdatedAt:     now,
		}
		if s := strings.TrimSpace(fm.OccurredAt); s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				tt := t.UTC()
				obj.OccurredAt = &tt
			}
		}
		lc.Objects = append(lc.Objects, obj)
	}
	for _, c := range cases {
		if c.ID == "" {
			return nil, fmt.Errorf("case missing id")
		}
		if c.K <= 0 {
			return nil, fmt.Errorf("case %s: k must be > 0", c.ID)
		}
		for _, lbl := range append(append([]string{}, c.ExpectedLabels...), c.AcceptableLabels...) {
			if _, ok := lc.LabelToID[lbl]; !ok {
				return nil, fmt.Errorf("case %s: unknown label %q", c.ID, lbl)
			}
		}
		for _, lbl := range c.ForbiddenLabels {
			if _, ok := lc.LabelToID[lbl]; !ok {
				return nil, fmt.Errorf("case %s: unknown forbidden label %q", c.ID, lbl)
			}
		}
	}
	return lc, nil
}

func stableID(label string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("recall-benchmark:"+label))
}
