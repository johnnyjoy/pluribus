package recall_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"control-plane/internal/memory"
	"control-plane/internal/recall"
	"control-plane/internal/recall/benchmark"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

func TestPhase4_emptyTagPluribusBeatsDornan(t *testing.T) {
	now := time.Now()
	req := recall.ScoreRequest{
		SituationQuery: "Pluribus MCP interface recall with empty tags caused bad matches",
	}
	w := recall.DefaultRankingWeights()
	relevant := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindFailure, Authority: 7,
		Statement: "Empty MCP tags on recall requests caused over-broad lexical matches",
		Tags:      []string{"mcp", "tags", "failure", "recall", "pluribus"},
		UpdatedAt: now,
	}
	irrelevant := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindPattern, Authority: 9,
		Statement: "Dornan Pro enterprise marketing positions fractional CTO services",
		Tags:      []string{"marketing", "enterprise", "consulting"},
		UpdatedAt: now,
	}
	out := recall.ScoreAndSortWithReason([]memory.MemoryObject{irrelevant, relevant}, req, w, 0)
	if out[0].Object.ID != relevant.ID {
		t.Fatalf("expected relevant pluribus memory first, got %q", out[0].Object.Statement)
	}
}

func TestPhase4_highAuthorityIrrelevantLoses(t *testing.T) {
	now := time.Now()
	req := recall.ScoreRequest{
		SituationQuery: "Primary MCP interface authority for Pluribus agent memory",
		Tags:           []string{"mcp", "interface"},
	}
	w := recall.DefaultRankingWeights()
	relevant := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindConstraint, Authority: 7,
		Statement: "Pluribus MCP is the primary agent interface for recall and record",
		Tags:      []string{"mcp", "interface", "pluribus"},
		UpdatedAt: now,
	}
	irrelevant := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindPattern, Authority: 9,
		Statement: "Dornan Pro enterprise marketing interface for executive positioning",
		Tags:      []string{"marketing", "enterprise"},
		UpdatedAt: now,
	}
	out := recall.ScoreAndSortWithReason([]memory.MemoryObject{irrelevant, relevant}, req, w, 0)
	if out[0].Object.ID != relevant.ID {
		t.Fatalf("expected lower-authority relevant memory first")
	}
}

func TestPhase4_lexicalOverlapOnguardDoesNotBeatPluribus(t *testing.T) {
	now := time.Now()
	req := recall.ScoreRequest{
		SituationQuery: "Pluribus MCP integration interface for agent recall binding",
		Tags:           []string{"mcp", "interface", "integration"},
	}
	w := recall.DefaultRankingWeights()
	pluribus := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindConstraint, Authority: 9,
		Statement: "Pluribus MCP is the primary agent interface; agents must use recall_context",
		Tags:      []string{"mcp", "interface", "agent-loop"},
		UpdatedAt: now,
	}
	onguard := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindPattern, Authority: 7,
		Statement: "OnGuard exposes a REST integration interface for housing systems",
		Tags:      []string{"integration", "interface", "api", "housing"},
		UpdatedAt: now,
	}
	out := recall.ScoreAndSortWithReason([]memory.MemoryObject{onguard, pluribus}, req, w, 0)
	if out[0].Object.ID != pluribus.ID {
		t.Fatalf("expected pluribus MCP memory over onguard integration trap")
	}
}

func TestPhase4_repoHintBiasesPluribus(t *testing.T) {
	now := time.Now()
	req := recall.ScoreRequest{
		SituationQuery: "integration interface for recall and credential binding",
		Tags:           []string{"integration", "interface"},
		RepoRoot:       "/projects/pluribus",
	}
	w := recall.DefaultRankingWeights()
	pluribus := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindConstraint, Authority: 9,
		Statement: "Pluribus MCP is the primary agent interface for recall binding",
		Tags:      []string{"mcp", "interface", "recall"},
		UpdatedAt: now,
	}
	onguard := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindPattern, Authority: 7,
		Statement: "OnGuard exposes a REST integration interface for housing credential sync",
		Tags:      []string{"integration", "interface", "housing"},
		UpdatedAt: now,
	}
	out := recall.ScoreAndSortWithReason([]memory.MemoryObject{onguard, pluribus}, req, w, 0)
	if out[0].Object.ID != pluribus.ID {
		t.Fatalf("expected repo-hint pluribus memory first")
	}
}

func TestPhase4_noBenchmarkLabelHardcoding(t *testing.T) {
	now := time.Now()
	req := recall.ScoreRequest{SituationQuery: "totally unknown xyzzy project query tokens"}
	w := recall.DefaultRankingWeights()
	a := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 5,
		Statement: "xyzzy project uses custom recall heuristics",
		Tags:      []string{"xyzzy", "custom"},
		UpdatedAt: now,
	}
	b := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 5,
		Statement: "unrelated decision about something else entirely",
		Tags:      []string{"other"},
		UpdatedAt: now,
	}
	out := recall.ScoreAndSortWithReason([]memory.MemoryObject{a, b}, req, w, 0)
	if out[0].Object.ID != a.ID {
		t.Fatal("expected statement overlap to rank xyzzy memory first without benchmark labels")
	}
}

func TestPhase4_tagRichPrimaryInBundle(t *testing.T) {
	lc, err := benchmark.Load(benchmark.DefaultDir())
	if err != nil {
		t.Fatal(err)
	}
	var c benchmark.FixtureCase
	for _, x := range lc.Cases {
		if x.ID == "pluribus_tag_rich_recall" {
			c = x
			break
		}
	}
	w := recall.DefaultRankingWeights()
	compiler := &recall.Compiler{
		Memory:   &benchmark.MemoryStub{All: lc.Objects},
		Ranking:  &w,
		Semantic: &recall.SemanticRecallConfig{Enabled: false},
	}
	bundle, err := compiler.Compile(context.Background(), recall.CompileRequest{
		RetrievalQuery: c.Query,
		Tags:           c.Tags,
		RepoRoot:       c.RepoRootHint,
		MaxPerKind:     10,
		MaxTotal:       40,
	})
	if err != nil {
		t.Fatal(err)
	}
	id := lc.LabelToID["pluribus_mcp_primary_interface_constraint"].String()
	var primaryScore float64
	for _, it := range bundle.GoverningConstraints {
		if it.ID == id && it.Justification != nil {
			primaryScore = it.Justification.Score
		}
	}
	if primaryScore <= 0 {
		t.Fatalf("primary constraint score=%v (expected >0)", primaryScore)
	}
	flat := recall.FlattenBundleByScore(bundle)
	for _, it := range flat {
		if it.ID == id {
			return
		}
	}
	t.Fatalf("primary constraint score=%.3f present in bundle but filtered from flat (top flat score=%.3f)", primaryScore, flat[0].Justification.Score)
}

func TestPhase4_vagueQuerySuppressesProductLeak(t *testing.T) {
	now := time.Now()
	req := recall.ScoreRequest{
		SituationQuery: "help me integrate the system interface properly",
	}
	w := recall.DefaultRankingWeights()
	generic := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindPattern, Authority: 3,
		Statement: "Systems should expose a clean interface for integration with external agents and services through standard binding layers.",
		Tags:      []string{"interface", "integration", "binding", "agent"},
		UpdatedAt: now,
	}
	pluribus := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindConstraint, Authority: 9,
		Statement: "Pluribus MCP is the primary agent interface; agents must use recall_context and record_experience through MCP tools.",
		Tags:      []string{"mcp", "interface", "agent-loop"},
		UpdatedAt: now,
	}
	out := recall.ScoreAndSortWithReason([]memory.MemoryObject{pluribus, generic}, req, w, 0)
	if out[0].Object.ID != generic.ID {
		t.Fatalf("expected generic interface noise over product leak on vague query")
	}
	if out[0].Score <= out[1].Score {
		t.Fatalf("expected generic to outrank product memory")
	}
}

func TestPhase4_symbolEchoPenalty(t *testing.T) {
	now := time.Now()
	req := recall.ScoreRequest{
		SituationQuery: "billing webhook handler timeout",
		Symbols:        []string{"pkg/billing.Handler"},
	}
	w := recall.DefaultRankingWeights()
	noise := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindPattern, Authority: 8,
		Statement: "billing webhook handler timeout without symbols payload",
		UpdatedAt: now, Payload: []byte(`{}`),
	}
	var p memory.PatternPayload
	p.Symbols = []string{"pkg/billing.Handler"}
	pl, _ := json.Marshal(p)
	targeted := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindPattern, Authority: 8,
		Statement: "handler stability", UpdatedAt: now, Payload: pl,
	}
	out := recall.ScoreAndSortWithReason([]memory.MemoryObject{noise, targeted}, req, w, 0)
	if out[0].Object.ID != targeted.ID {
		t.Fatalf("symbol overlap should beat query-echo pattern")
	}
}
