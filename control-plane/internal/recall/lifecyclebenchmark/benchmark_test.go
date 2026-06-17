package lifecyclebenchmark

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"control-plane/internal/memory"
	"control-plane/internal/mcp"
	"control-plane/internal/recall"
	"control-plane/internal/utility"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

type fixtureCase struct {
	ID                      string   `json:"id"`
	Category                string   `json:"category"`
	RecallMode              string   `json:"recall_mode"`
	Query                   string   `json:"query"`
	Tags                    []string `json:"tags"`
	MustContainStatement    string   `json:"must_contain_statement"`
	MustNotContainStatement string   `json:"must_not_contain_statement"`
	ExpectLifecycleRole     string   `json:"expect_lifecycle_role"`
	ForbidLifecycleRoles    []string `json:"forbid_lifecycle_roles"`
	ExpectStatusInBundle    []string `json:"expect_status_in_bundle"`
	ForbidStatusInBundle    []string `json:"forbid_status_in_bundle"`
	ExpectCompileError      bool     `json:"expect_compile_error"`
	CheckMCPBody            bool     `json:"check_mcp_body"`
}

type fixtureFile struct {
	Cases []fixtureCase `json:"cases"`
}

func loadCases(t *testing.T) fixtureFile {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "..", "testdata", "lifecycle_recall", "cases.json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read cases: %v", err)
	}
	var f fixtureFile
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatalf("parse cases: %v", err)
	}
	return f
}

func TestLifecycleRecallBenchmarkCases(t *testing.T) {
	f := loadCases(t)
	compiler := newLifecycleCompiler()
	for _, c := range f.Cases {
		t.Run(c.ID, func(t *testing.T) {
			if err := evalCase(compiler, c); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLifecycleRecallBenchmarkGate(t *testing.T) {
	f := loadCases(t)
	compiler := newLifecycleCompiler()
	var failed int
	for _, c := range f.Cases {
		if err := evalCase(compiler, c); err != nil {
			failed++
			t.Errorf("%s: %v", c.ID, err)
		}
	}
	if failed > 0 {
		t.Fatalf("lifecycle recall gate: %d/%d failed", failed, len(f.Cases))
	}
}

func evalCase(compiler *recall.Compiler, c fixtureCase) error {
	if c.CheckMCPBody {
		body, _, err := mcp.BuildMemoryContextResolveCompileBodyForTest(json.RawMessage(fmt.Sprintf(`{"task":%q,"recall_mode":%q,"tags":["phase8-lifecycle"]}`, c.Query, c.RecallMode)))
		if err != nil {
			return err
		}
		var m map[string]any
		if err := json.Unmarshal(body, &m); err != nil {
			return err
		}
		if m["recall_mode"] != c.RecallMode {
			return fmt.Errorf("mcp body recall_mode=%v want %q", m["recall_mode"], c.RecallMode)
		}
	}

	req := recall.CompileRequest{
		RetrievalQuery: c.Query,
		Tags:           c.Tags,
		MaxPerKind:     10,
	}
	if c.RecallMode != "" {
		req.RecallMode = c.RecallMode
	}

	bundle, err := compiler.Compile(context.Background(), req)
	if c.ExpectCompileError {
		if err == nil {
			return fmt.Errorf("expected compile error")
		}
		return nil
	}
	if err != nil {
		return err
	}
	items := collectItems(bundle)
	if c.MustContainStatement != "" {
		if !containsStatement(items, c.MustContainStatement) {
			return fmt.Errorf("missing statement %q", c.MustContainStatement)
		}
	}
	if c.MustNotContainStatement != "" {
		if containsStatement(items, c.MustNotContainStatement) {
			return fmt.Errorf("forbidden statement %q present", c.MustNotContainStatement)
		}
	}
	if c.ExpectLifecycleRole != "" && c.MustContainStatement != "" {
		if !hasRoleForStatement(items, c.MustContainStatement, c.ExpectLifecycleRole) {
			return fmt.Errorf("expected lifecycle_role %q for %q", c.ExpectLifecycleRole, c.MustContainStatement)
		}
	}
	for _, role := range c.ForbidLifecycleRoles {
		if c.MustContainStatement != "" {
			if hasRoleForStatement(items, c.MustContainStatement, role) {
				return fmt.Errorf("forbidden lifecycle_role %q on %q", role, c.MustContainStatement)
			}
			continue
		}
		if hasLifecycleRole(items, role) {
			return fmt.Errorf("forbidden lifecycle_role %q present", role)
		}
	}
	for _, st := range c.ForbidStatusInBundle {
		if hasStatus(items, st) {
			return fmt.Errorf("forbidden status %q in bundle", st)
		}
	}
	if c.RecallMode == "" || c.RecallMode == "current" {
		if bundle.LifecycleRecall == nil || bundle.LifecycleRecall.RecallMode != "current" {
			return fmt.Errorf("expected default lifecycle recall_mode=current")
		}
	}
	if c.RecallMode == "historical" {
		if bundle.LifecycleRecall == nil || !bundle.LifecycleRecall.HistoricalMode {
			return fmt.Errorf("expected historical_mode=true")
		}
	}
	return nil
}

func collectItems(b *recall.RecallBundle) []recall.MemoryItem {
	if b == nil {
		return nil
	}
	var out []recall.MemoryItem
	for _, s := range [][]recall.MemoryItem{
		b.GoverningConstraints, b.Decisions, b.KnownFailures, b.ApplicablePatterns,
		b.Continuity, b.Constraints, b.Experience,
	} {
		out = append(out, s...)
	}
	return out
}

func containsStatement(items []recall.MemoryItem, frag string) bool {
	for _, it := range items {
		if strings.Contains(it.Statement, frag) {
			return true
		}
	}
	return false
}

func hasRoleForStatement(items []recall.MemoryItem, frag, role string) bool {
	for _, it := range items {
		if strings.Contains(it.Statement, frag) && it.LifecycleRole == role {
			return true
		}
	}
	return false
}

func hasLifecycleRole(items []recall.MemoryItem, role string) bool {
	for _, it := range items {
		if it.LifecycleRole == role {
			return true
		}
	}
	return false
}

func hasStatus(items []recall.MemoryItem, status string) bool {
	for _, it := range items {
		if it.Status == status {
			return true
		}
	}
	return false
}

func newLifecycleCompiler() *recall.Compiler {
	tag := "phase8-lifecycle"
	activeID := uuid.New()
	supID := uuid.New()
	archID := uuid.New()
	pendID := uuid.New()
	rejID := uuid.New()
	outdatedID := uuid.New()
	refutedID := uuid.New()
	newID := uuid.New()

	objs := []memory.MemoryObject{
		{ID: activeID, Kind: api.MemoryKindConstraint, Statement: "PHASE8_ACTIVE_CONSTRAINT governs current phase8 lifecycle recall", Authority: 8, Status: api.StatusActive, Tags: []string{tag}},
		{ID: supID, Kind: api.MemoryKindConstraint, Statement: "PHASE8_SUPERSEDED_CONSTRAINT was replaced by newer guidance", Authority: 7, Status: api.StatusSuperseded, Tags: []string{tag}},
		{ID: archID, Kind: api.MemoryKindPattern, Statement: "PHASE8_ARCHIVED_PATTERN legacy archived phase8 pattern", Authority: 5, Status: api.StatusArchived, Tags: []string{tag}},
		{ID: pendID, Kind: api.MemoryKindConstraint, Statement: "PHASE8_PENDING_SECRET unapproved candidate", Authority: 9, Status: api.StatusPending, Tags: []string{tag}},
		{ID: rejID, Kind: api.MemoryKindFailure, Statement: "PHASE8_REJECTED_JUNK rejected formation", Authority: 3, Status: api.StatusRejected, Tags: []string{tag}},
		{ID: outdatedID, Kind: api.MemoryKindPattern, Statement: "PHASE8_OUTDATED_PATTERN once useful outdated phase8 utility", Authority: 6, Status: api.StatusActive, Tags: []string{tag}},
		{ID: refutedID, Kind: api.MemoryKindFailure, Statement: "PHASE8_REFUTED_FAILURE refuted wrong phase8 belief audit", Authority: 4, Status: api.StatusActive, Tags: []string{tag}},
		{ID: newID, Kind: api.MemoryKindConstraint, Statement: "PHASE8_ACTIVE_CONSTRAINT successor after supersession", Authority: 8, Status: api.StatusActive, Tags: []string{tag}},
	}

	w := recall.DefaultRankingWeights()
	return &recall.Compiler{
		Memory:  &statusFilteringSearcher{all: objs},
		Ranking: &w,
		Utility: &fakeUtilityProvider{
			scores: map[uuid.UUID]float64{
				outdatedID: -4,
				refutedID:  -8,
			},
			summaries: map[uuid.UUID]utility.Score{
				outdatedID: {MemoryID: outdatedID, UtilityScore: -4, OutdatedCount: 2},
				refutedID:  {MemoryID: refutedID, UtilityScore: -8, WrongCount: 1, RefutedCount: 1},
			},
		},
		UtilityWeight: 0.12,
	}
}

type statusFilteringSearcher struct {
	all []memory.MemoryObject
}

func (s *statusFilteringSearcher) Search(ctx context.Context, req memory.SearchRequest) ([]memory.MemoryObject, error) {
	st := req.Status
	if st == "" {
		st = "active"
	}
	var out []memory.MemoryObject
	for _, o := range s.all {
		os := string(o.Status)
		if os == "" {
			os = "active"
		}
		if len(req.Tags) > 0 && !tagOverlap(o.Tags, req.Tags) {
			continue
		}
		if os == st {
			out = append(out, o)
		}
	}
	return out, nil
}

func (s *statusFilteringSearcher) SearchMemories(ctx context.Context, req memory.MemoriesSearchRequest) ([]memory.MemoryObject, error) {
	st := req.Status
	if st == "" {
		st = "active"
	}
	var out []memory.MemoryObject
	q := strings.ToLower(req.Query)
	for _, o := range s.all {
		os := string(o.Status)
		if os == "" {
			os = "active"
		}
		if os != st {
			continue
		}
		if len(req.Tags) > 0 && !tagOverlap(o.Tags, req.Tags) {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(o.Statement), q) {
			continue
		}
		out = append(out, o)
	}
	return out, nil
}

func tagOverlap(memTags, reqTags []string) bool {
	set := map[string]struct{}{}
	for _, t := range memTags {
		set[t] = struct{}{}
	}
	for _, t := range reqTags {
		if _, ok := set[t]; ok {
			return true
		}
	}
	return false
}

type fakeUtilityProvider struct {
	scores    map[uuid.UUID]float64
	summaries map[uuid.UUID]utility.Score
}

func (f *fakeUtilityProvider) GetScoresForMemories(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]float64, error) {
	out := map[uuid.UUID]float64{}
	for _, id := range ids {
		if v, ok := f.scores[id]; ok {
			out[id] = v
		}
	}
	return out, nil
}

func (f *fakeUtilityProvider) GetUtilitySummaries(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]utility.Score, error) {
	out := map[uuid.UUID]utility.Score{}
	for _, id := range ids {
		if v, ok := f.summaries[id]; ok {
			out[id] = v
		}
	}
	return out, nil
}
