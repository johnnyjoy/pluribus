package drift

import (
	"context"
	"encoding/json"
	"testing"

	"control-plane/internal/memory"
	"control-plane/internal/tooling"
	"control-plane/pkg/api"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestService_Check_no_memory_returns_passed(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectExec(`INSERT INTO drift_checks`).
		WithArgs(sqlmock.AnyArg(), true, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &Service{Repo: &Repo{DB: db}, Memory: nil}
	result, err := svc.Check(context.Background(), CheckRequest{
		Proposal: "any text",
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !result.Passed {
		t.Error("expected Passed true when no constraints/failures")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestService_Check_structural_risk_escalation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectExec(`INSERT INTO drift_checks`).
		WithArgs(sqlmock.AnyArg(), true, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &Service{Repo: &Repo{DB: db}, Memory: nil}
	result, err := svc.Check(context.Background(), CheckRequest{
		Proposal: "any text",
		StructuralSignals: &StructuralSignals{ChangeCount: 4, BoundaryViolationCount: 0},
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if result.RiskLevel != RiskHigh || !result.BlockExecution {
		t.Errorf("expected risk_level=high, block_execution=true; got risk_level=%q block=%v", result.RiskLevel, result.BlockExecution)
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warnings for high risk")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestService_Check_slowPathSetsRequiresFollowupCheck(t *testing.T) {
	svc := &Service{Repo: nil, Memory: nil, RequireSecondDriftCheck: true}
	result, err := svc.Check(context.Background(), CheckRequest{
		Proposal:         "any text",
		SlowPathRequired: true,
		IsFollowupCheck:  false,
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !result.RequiresFollowupCheck {
		t.Error("RequiresFollowupCheck = false, want true")
	}
	if result.FollowupReason == "" {
		t.Error("FollowupReason empty, want set")
	}
}

func TestService_Check_slowPathSecondCallClearsFollowup(t *testing.T) {
	svc := &Service{Repo: nil, Memory: nil, RequireSecondDriftCheck: true}
	result, err := svc.Check(context.Background(), CheckRequest{
		Proposal:         "any text",
		SlowPathRequired: true,
		IsFollowupCheck:  true,
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if result.RequiresFollowupCheck {
		t.Error("RequiresFollowupCheck = true on second call, want false")
	}
}

func TestService_Check_slowPathDisabledByConfig(t *testing.T) {
	svc := &Service{Repo: nil, Memory: nil, RequireSecondDriftCheck: false}
	result, err := svc.Check(context.Background(), CheckRequest{
		Proposal:         "any text",
		SlowPathRequired: true,
		IsFollowupCheck:  false,
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if result.RequiresFollowupCheck {
		t.Error("RequiresFollowupCheck = true when config disabled, want false")
	}
}

// TestService_Check_lspHighReferenceEscalatesRisk verifies that when LSP is configured and a touched symbol
// has reference count above LSPHighRiskReferenceThreshold, risk is escalated to high and execution blocked (Task 100).
func TestService_Check_lspHighReferenceEscalatesRisk(t *testing.T) {
	// Fake LSP returns 15 references; threshold 10 => 15 > 10 so we escalate
	fake := &tooling.FakeLSPClient{
		References: make([]tooling.Reference, 15),
	}
	svc := &Service{
		Repo:                          nil,
		Memory:                        nil,
		LSP:                           fake,
		LSPHighRiskReferenceThreshold: 10,
	}
	result, err := svc.Check(context.Background(), CheckRequest{
		Proposal: "change to Foo",
		RepoRoot:  "/repo",
		TouchedSymbols: []SymbolPosition{
			{Path: "pkg/foo.go", Line: 10, Column: 5},
		},
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if result.RiskLevel != RiskHigh {
		t.Errorf("RiskLevel = %q, want high (LSP ref count above threshold)", result.RiskLevel)
	}
	if !result.BlockExecution {
		t.Error("BlockExecution = false, want true when LSP high reference risk")
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warning about high reference count")
	}
}

// fakeMemorySearcher returns fixed memories for drift tests.
type fakeMemorySearcher struct {
	objs []memory.MemoryObject
}

func (f *fakeMemorySearcher) Search(ctx context.Context, req memory.SearchRequest) ([]memory.MemoryObject, error) {
	return f.objs, nil
}

// TestService_Check_negativePatternViolation verifies overlap with negative pattern payload.
func TestService_Check_negativePatternViolation(t *testing.T) {
	payload := memory.PatternPayload{
		Polarity:   "negative",
		Experience: "Duplicate handlers caused bugs",
		Decision:   "Single owner",
		Outcome:    "Regression",
		Impact:     memory.PatternImpact{Severity: "high"},
		Directive:  "duplicate handlers",
	}
	raw, _ := json.Marshal(payload)
	fakeMem := &fakeMemorySearcher{
		objs: []memory.MemoryObject{{
			ID:        uuid.New(),
			Kind:      api.MemoryKindPattern,
			Statement: "Avoid duplicate handlers",
			Payload:   raw,
		}},
	}
	svc := &Service{Repo: nil, Memory: fakeMem, PatternHighBlocks: true}
	result, err := svc.Check(context.Background(), CheckRequest{
		Proposal: "we will add duplicate handlers for alternate path",
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	var found bool
	for _, v := range result.Violations {
		if v.Code == "negative_pattern" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected one violation with code negative_pattern; got violations %+v", result.Violations)
	}
	if result.Passed {
		t.Error("Passed = true, want false when negative pattern matches (high severity)")
	}
	if result.RiskLevel != RiskHigh {
		t.Errorf("RiskLevel = %q, want high (negative pattern high severity)", result.RiskLevel)
	}
	if !result.BlockExecution {
		t.Error("BlockExecution = false, want true for high-severity pattern match")
	}
	if result.PatternViolationCount == 0 {
		t.Error("PatternViolationCount = 0, want > 0")
	}
}

func TestService_Check_negativePatternWarningNoBlock(t *testing.T) {
	payload := memory.PatternPayload{
		Polarity:   "negative",
		Experience: "Avoid temporary hacks",
		Impact:     memory.PatternImpact{Severity: "low"},
		Directive:  "temporary hack",
	}
	raw, _ := json.Marshal(payload)
	fakeMem := &fakeMemorySearcher{
		objs: []memory.MemoryObject{{
			ID:        uuid.New(),
			Kind:      api.MemoryKindPattern,
			Statement: "Avoid temporary hacks",
			Payload:   raw,
		}},
	}
	svc := &Service{Repo: nil, Memory: fakeMem, PatternHighBlocks: true}
	result, err := svc.Check(context.Background(), CheckRequest{
		Proposal: "we can ship a temporary hack first",
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if result.Passed != true {
		t.Error("Passed = false, want true for low-severity warning path")
	}
	if result.BlockExecution {
		t.Error("BlockExecution = true, want false for low-severity warning path")
	}
	if result.PatternWarningCount == 0 {
		t.Error("PatternWarningCount = 0, want > 0")
	}
}
