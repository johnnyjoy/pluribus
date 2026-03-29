package curation

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"control-plane/internal/memory"
	"control-plane/pkg/api"
	"github.com/DATA-DOG/go-sqlmock"

	"github.com/google/uuid"
)

func TestService_Evaluate_setsShouldReviewAndShouldPromote(t *testing.T) {
	// CandidateThreshold 1.0 so we don't create (avoids nil repo); low review/promote thresholds
	cfg := &SalienceConfig{
		CandidateThreshold: 1.0, // no candidate created
		ReviewThreshold:    0.2,
		PromoteThreshold:   0.5,
	}
	svc := &Service{Repo: nil, Config: cfg}
	ctx := context.Background()

	// Text with "always" and "must" should score high; should_review and possibly should_promote
	res, err := svc.Evaluate(ctx, EvaluateRequest{Text: "We must always use POST for create."})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if res.SalienceScore <= 0 {
		t.Errorf("expected positive salience: %f", res.SalienceScore)
	}
	if !res.ShouldReview {
		t.Errorf("expected should_review true with review_threshold=0.2, score=%f", res.SalienceScore)
	}
	if res.SalienceScore >= 0.5 && !res.ShouldPromote {
		t.Errorf("expected should_promote when score >= 0.5: score=%f", res.SalienceScore)
	}
	if res.Created {
		t.Error("expected no candidate created (CandidateThreshold=1.0)")
	}
}

// fakeMemoryCreator implements MemoryCreator for tests (Task 99).
type fakeMemoryCreator struct {
	create func(context.Context, memory.CreateRequest) (*memory.MemoryObject, error)
}

func (f *fakeMemoryCreator) Create(ctx context.Context, req memory.CreateRequest) (*memory.MemoryObject, error) {
	if f.create != nil {
		return f.create(ctx, req)
	}
	return &memory.MemoryObject{ID: uuid.New(), Kind: req.Kind, Statement: req.Statement}, nil
}

func TestService_PromoteToPattern_success(t *testing.T) {
	ctx := context.Background()
	candidateID := uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	payload := &memory.PatternPayload{
		Polarity:   "negative",
		Experience: "Deployed without tests",
		Decision:   "Require tests",
		Outcome:    "Regression",
		Impact:     memory.PatternImpact{Severity: "high"},
		Directive:  "Never skip tests.",
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := &Repo{DB: db}
	mock.ExpectQuery(`SELECT id, raw_text`).WithArgs(candidateID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "raw_text", "salience_score", "promotion_status", "created_at", "proposal_json"}).
			AddRow(candidateID, "Raw candidate text", 0.8, "pending", time.Now(), nil))
	mock.ExpectExec(`UPDATE candidate_events SET promotion_status`).WithArgs("promoted", candidateID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	var createReq memory.CreateRequest
	fakeMem := &fakeMemoryCreator{
		create: func(_ context.Context, req memory.CreateRequest) (*memory.MemoryObject, error) {
			createReq = req
			return &memory.MemoryObject{
				ID: uuid.New(), Kind: api.MemoryKindPattern,
				Statement: req.Statement,
			}, nil
		},
	}
	svc := &Service{Repo: repo, Memory: fakeMem}

	obj, err := svc.PromoteToPattern(ctx, candidateID, payload)
	if err != nil {
		t.Fatalf("PromoteToPattern: %v", err)
	}
	if obj == nil || obj.Kind != api.MemoryKindPattern {
		t.Errorf("expected pattern memory: %+v", obj)
	}
	if createReq.Kind != api.MemoryKindPattern {
		t.Errorf("CreateRequest: %+v", createReq)
	}
	if createReq.Statement != "Never skip tests." {
		t.Errorf("Statement = %q, want directive", createReq.Statement)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestService_PromoteToPattern_candidateNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := &Repo{DB: db}
	mock.ExpectQuery(`SELECT id, raw_text`).WillReturnError(sql.ErrNoRows)

	payload := &memory.PatternPayload{
		Polarity: "negative", Experience: "x", Decision: "d", Outcome: "o",
		Impact: memory.PatternImpact{Severity: "high"}, Directive: "dir",
	}
	svc := &Service{Repo: repo, Memory: &fakeMemoryCreator{}}
	_, err = svc.PromoteToPattern(context.Background(), uuid.New(), payload)
	if err == nil || err.Error() != "candidate not found" {
		t.Errorf("expected candidate not found: %v", err)
	}
}

func TestService_PromoteToPattern_invalidPayload(t *testing.T) {
	payload := &memory.PatternPayload{Polarity: "invalid"}
	svc := &Service{Repo: nil, Memory: &fakeMemoryCreator{}}
	_, err := svc.PromoteToPattern(context.Background(), uuid.New(), payload)
	if err == nil {
		t.Fatal("expected validation error")
	}
}
