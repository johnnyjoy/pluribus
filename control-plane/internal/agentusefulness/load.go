package agentusefulness

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"control-plane/internal/memory"
	"control-plane/internal/utility"

	"github.com/google/uuid"
)

// DefaultDir returns testdata/agent_memory_usefulness.
func DefaultDir() string {
	if dir := strings.TrimSpace(os.Getenv("AGENT_MEMORY_USEFULNESS_DIR")); dir != "" {
		return dir
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join("control-plane", "testdata", "agent_memory_usefulness")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "agent_memory_usefulness")
}

// Load reads Phase 11B fixtures only (tasks.json + memories.json).
func Load(dir string) (*LoadedCorpus, error) {
	return loadFromDir(dir, false)
}

// LoadCognitive reads Phase 11B + Phase 11C merged fixtures.
func LoadCognitive(dir string) (*LoadedCorpus, error) {
	return loadFromDir(dir, true)
}

func loadFromDir(dir string, includeCognitive bool) (*LoadedCorpus, error) {
	memBytes, err := os.ReadFile(filepath.Join(dir, "memories.json"))
	if err != nil {
		return nil, fmt.Errorf("read memories: %w", err)
	}
	taskBytes, err := os.ReadFile(filepath.Join(dir, "tasks.json"))
	if err != nil {
		return nil, fmt.Errorf("read tasks: %w", err)
	}
	var memFile struct {
		Memories []FixtureMemory `json:"memories"`
	}
	if err := json.Unmarshal(memBytes, &memFile); err != nil {
		return nil, fmt.Errorf("parse memories: %w", err)
	}
	var taskFile struct {
		Tasks []TaskFixture `json:"tasks"`
	}
	if err := json.Unmarshal(taskBytes, &taskFile); err != nil {
		return nil, fmt.Errorf("parse tasks: %w", err)
	}

	if includeCognitive {
		if b, err := os.ReadFile(filepath.Join(dir, "cognitive_memories.json")); err == nil {
			var extra struct {
				Memories []FixtureMemory `json:"memories"`
			}
			if err := json.Unmarshal(b, &extra); err != nil {
				return nil, fmt.Errorf("parse cognitive memories: %w", err)
			}
			memFile.Memories = append(memFile.Memories, extra.Memories...)
		}
		if b, err := os.ReadFile(filepath.Join(dir, "cognitive_tasks.json")); err == nil {
			var extra struct {
				Tasks []TaskFixture `json:"tasks"`
			}
			if err := json.Unmarshal(b, &extra); err != nil {
				return nil, fmt.Errorf("parse cognitive tasks: %w", err)
			}
			taskFile.Tasks = append(taskFile.Tasks, extra.Tasks...)
		}
	}
	return BuildLoaded(memFile.Memories, taskFile.Tasks)
}

// BuildLoaded constructs runtime objects from fixtures.
func BuildLoaded(memories []FixtureMemory, tasks []TaskFixture) (*LoadedCorpus, error) {
	lc := &LoadedCorpus{
		LabelToID:      map[string]string{},
		IDToLabel:      map[string]string{},
		Utility:        map[string]utility.Score{},
		Tasks:          tasks,
		MemoryFixtures: append([]FixtureMemory(nil), memories...),
	}
	now := time.Now().UTC()
	labelToUUID := map[string]uuid.UUID{}

	for _, fm := range memories {
		if fm.Label == "" {
			return nil, fmt.Errorf("memory missing label")
		}
		if _, dup := lc.LabelToID[fm.Label]; dup {
			return nil, fmt.Errorf("duplicate label %q", fm.Label)
		}
		id := stableID("phase11b:" + fm.Label)
		labelToUUID[fm.Label] = id
		lc.LabelToID[fm.Label] = id.String()
		lc.IDToLabel[id.String()] = fm.Label
	}

	for i, fm := range memories {
		id := labelToUUID[fm.Label]
		createdAt := now
		if t := parseRFC3339(fm.CreatedAt); t != nil {
			createdAt = *t
		}
		tags := append([]string(nil), fm.Tags...)
		if fm.Scope != "" {
			tags = append(tags, fm.Scope)
		}
		if fm.EncodingCues != nil && fm.EncodingCues.Scope != "" {
			tags = append(tags, fm.EncodingCues.Scope)
		}
		obj := memory.MemoryObject{
			ID:            id,
			Kind:          parseKind(fm.Kind),
			Statement:     fm.Statement,
			Authority:     fm.Authority,
			Tags:          tags,
			Status:        parseStatus(fm.Status),
			Applicability: parseApplicability(fm.Applicability, fm.Authority),
			CreatedAt:     createdAt,
			UpdatedAt:     now,
		}
		if t := parseRFC3339(fm.OccurredAt); t != nil {
			obj.OccurredAt = t
		}
		lc.Objects = append(lc.Objects, obj)
		memories[i] = fm

		if fm.UtilityScore != 0 || fm.RefutedCount > 0 || fm.WrongCount > 0 || fm.OutdatedCount > 0 {
			lc.Utility[fm.Label] = utility.Score{
				MemoryID:      id,
				UtilityScore:  fm.UtilityScore,
				RefutedCount:  fm.RefutedCount,
				WrongCount:    fm.WrongCount,
				OutdatedCount: fm.OutdatedCount,
			}
		}
	}
	lc.MemoryFixtures = memories

	if err := validateTasks(lc, tasks); err != nil {
		return nil, err
	}
	return lc, nil
}

func validateTasks(lc *LoadedCorpus, tasks []TaskFixture) error {
	for _, t := range tasks {
		if t.ID == "" {
			return fmt.Errorf("task missing id")
		}
		allLabels := collectTaskLabels(t)
		for _, lbl := range allLabels {
			if lbl == "" {
				continue
			}
			if _, ok := lc.LabelToID[lbl]; !ok {
				return fmt.Errorf("task %s: unknown label %q", t.ID, lbl)
			}
		}
		for lbl := range t.FactContributions {
			if _, ok := lc.LabelToID[lbl]; !ok {
				return fmt.Errorf("task %s: unknown fact contribution label %q", t.ID, lbl)
			}
		}
	}
	return nil
}

func collectTaskLabels(t TaskFixture) []string {
	var out []string
	out = append(out, t.MemoryLabels...)
	out = append(out, t.DecoyLabels...)
	out = append(out, t.ExpectedRecalledLabels...)
	out = append(out, t.ForbiddenRecalledLabels...)
	out = append(out, t.ExpectedUsedLabels...)
	out = append(out, t.ExpectedIgnoredLabels...)
	out = append(out, t.NearMissMemoryLabels...)
	out = append(out, t.ExpectedSuppressedLabels...)
	return out
}

func stableID(label string) uuid.UUID {
	h := sha1.Sum([]byte(label))
	var u uuid.UUID
	copy(u[:], h[:16])
	u[6] = (u[6] & 0x0f) | 0x50
	u[8] = (u[8] & 0x3f) | 0x80
	return u
}

// ObjectsForTask returns memory objects for a task's pool (memory + decoy labels).
func (lc *LoadedCorpus) ObjectsForTask(t TaskFixture) []memory.MemoryObject {
	want := map[string]struct{}{}
	for _, lbl := range collectTaskLabels(t) {
		if lbl != "" {
			want[lbl] = struct{}{}
		}
	}
	var out []memory.MemoryObject
	for _, o := range lc.Objects {
		lbl := lc.IDToLabel[o.ID.String()]
		if _, ok := want[lbl]; ok {
			out = append(out, o)
		}
	}
	return out
}

// CognitiveTasks returns tasks with a research_principle set.
func (lc *LoadedCorpus) CognitiveTasks() []TaskFixture {
	var out []TaskFixture
	for _, t := range lc.Tasks {
		if strings.TrimSpace(t.ResearchPrinciple) != "" {
			out = append(out, t)
		}
	}
	return out
}

// Phase11BTasks returns tasks without research_principle (legacy harness).
func (lc *LoadedCorpus) Phase11BTasks() []TaskFixture {
	var out []TaskFixture
	for _, t := range lc.Tasks {
		if strings.TrimSpace(t.ResearchPrinciple) == "" {
			out = append(out, t)
		}
	}
	return out
}

// LabelForID returns fixture label for memory UUID string.
func (lc *LoadedCorpus) LabelForID(id string) string {
	return lc.IDToLabel[id]
}

// IDForLabel returns UUID string for label.
func (lc *LoadedCorpus) IDForLabel(label string) string {
	return lc.LabelToID[label]
}

// FixtureByLabel returns the fixture memory definition.
func (lc *LoadedCorpus) FixtureByLabel(label string) *FixtureMemory {
	for i := range lc.MemoryFixtures {
		if lc.MemoryFixtures[i].Label == label {
			cp := lc.MemoryFixtures[i]
			return &cp
		}
	}
	return nil
}
