package curation

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

type stubFailureCounter struct{ n int }

func (s *stubFailureCounter) CountActiveFailuresWithStatementKey(_ context.Context, _ string) (int, error) {
	return s.n, nil
}

func TestDigest_repeatedFailurePromotesConstraint(t *testing.T) {
	svc := &Service{
		Config:         &SalienceConfig{CandidateThreshold: 0.5, ReviewThreshold: 0.7, PromoteThreshold: 0.85},
		DigestLimits:   defaultDigestLimits(),
		FailureCounter: &stubFailureCounter{n: 1},
	}
	res, err := svc.Digest(context.Background(), DigestRequest{
		WorkSummary: "Shipped digest API with tests and documentation.",
		CurationAnswers: &DigestCurationAnswers{
			Failure: "Bad deploy broke staging rollback path",
		},
		Options: &DigestOptions{DryRun: true},
	})
	if err != nil {
		t.Fatalf("Digest: %v", err)
	}
	var found bool
	for _, p := range res.Proposals {
		if p.Kind == api.MemoryKindConstraint && strings.Contains(p.Reason, "repeated failure") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected promoted: repeated failure constraint, got %+v", res.Proposals)
	}
}

func TestDigest_dryRun(t *testing.T) {
	svc := &Service{
		Repo:         nil,
		Config:       &SalienceConfig{CandidateThreshold: 0.5, ReviewThreshold: 0.7, PromoteThreshold: 0.85},
		DigestLimits: defaultDigestLimits(),
	}
	res, err := svc.Digest(context.Background(), DigestRequest{
		WorkSummary: "Shipped digest API with tests and documentation.",
		CurationAnswers: &DigestCurationAnswers{
			Decision: "Prefer dry_run for previews",
		},
		Options: &DigestOptions{DryRun: true},
	})
	if err != nil {
		t.Fatalf("Digest: %v", err)
	}
	if len(res.Proposals) == 0 {
		t.Fatal("expected proposals in dry_run")
	}
	if res.Proposals[0].CandidateID != "" {
		t.Errorf("dry_run should not set candidate_id: %+v", res.Proposals[0])
	}
}

func TestDigest_rejectedNoSignal(t *testing.T) {
	svc := &Service{DigestLimits: defaultDigestLimits()}
	res, err := svc.Digest(context.Background(), DigestRequest{
		WorkSummary: "short", // too short for fallback + no answers
	})
	if err != nil {
		t.Fatalf("Digest: %v", err)
	}
	if len(res.Rejected) == 0 || res.Rejected[0].Reason != "no_proposals" {
		t.Fatalf("expected no_proposals rejected: %+v", res)
	}
}

func TestDigest_doesNotInferProjectOrGlobalTags(t *testing.T) {
	svc := &Service{
		Config:       &SalienceConfig{CandidateThreshold: 0.5, ReviewThreshold: 0.7, PromoteThreshold: 0.85},
		DigestLimits: defaultDigestLimits(),
	}
	res, err := svc.Digest(context.Background(), DigestRequest{
		WorkSummary: "Implemented digest parsing and materialization with tests.",
		CurationAnswers: &DigestCurationAnswers{
			Decision: "Keep memory identity independent from project metadata.",
		},
		Options: &DigestOptions{DryRun: true},
	})
	if err != nil {
		t.Fatalf("Digest: %v", err)
	}
	if len(res.Proposals) == 0 {
		t.Fatal("expected at least one proposal")
	}
	for _, p := range res.Proposals {
		for _, tag := range p.Tags {
			if strings.HasPrefix(tag, "project:") || tag == "global" || tag == "scope:global" {
				t.Fatalf("unexpected inferred tag %q in proposal tags %v", tag, p.Tags)
			}
		}
	}
}

func TestMaterialize_success(t *testing.T) {
	ctx := context.Background()
	candidateID := uuid.MustParse("b0000000-0000-0000-0000-000000000001")

	p := ProposalPayloadV1{
		V:                 1,
		Kind:              api.MemoryKindConstraint,
		Statement:         "Use migrations before deploy",
		Reason:            "test",
		ProposedAuthority: 7,
	}
	pj, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := &Repo{DB: db}
	mock.ExpectQuery(`SELECT id, raw_text`).WithArgs(candidateID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "raw_text", "salience_score", "promotion_status", "created_at", "proposal_json"}).
			AddRow(candidateID, "raw", 0.5, "pending", time.Now(), pj))
	mock.ExpectExec(`UPDATE candidate_events SET promotion_status`).WithArgs("promoted", candidateID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	var gotStmt string
	var gotTags []string
	var gotPayload []byte
	mem := &fakeMemoryCreator{
		create: func(_ context.Context, req memory.CreateRequest) (*memory.MemoryObject, error) {
			gotStmt = req.Statement
			gotTags = append([]string(nil), req.Tags...)
			if req.Payload != nil {
				gotPayload = append([]byte(nil), *req.Payload...)
			}
			return &memory.MemoryObject{ID: uuid.New(), Kind: req.Kind, Statement: req.Statement}, nil
		},
	}
	svc := &Service{Repo: repo, Memory: mem, Promotion: &PromotionDigestConfig{}}

	obj, err := svc.Materialize(ctx, candidateID)
	if err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	if obj == nil || gotStmt != "Use migrations before deploy" {
		t.Fatalf("unexpected: obj=%v stmt=%q", obj, gotStmt)
	}
	if len(gotTags) != 0 {
		t.Fatalf("materialize inferred unexpected tags: %v", gotTags)
	}
	if !strings.Contains(string(gotPayload), "pluribus_promotion") || !strings.Contains(string(gotPayload), candidateID.String()) {
		t.Fatalf("expected materialize payload with candidate id, got %q", string(gotPayload))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestMaterialize_requireEvidence(t *testing.T) {
	ctx := context.Background()
	candidateID := uuid.MustParse("c0000000-0000-0000-0000-000000000001")
	p := ProposalPayloadV1{V: 1, Kind: api.MemoryKindDecision, Statement: "this decision text is long enough for validation sixteen"}
	pj, _ := json.Marshal(p)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := &Repo{DB: db}
	mock.ExpectQuery(`SELECT id, raw_text`).WithArgs(candidateID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "raw_text", "salience_score", "promotion_status", "created_at", "proposal_json"}).
			AddRow(candidateID, "raw", 0.5, "pending", time.Now(), pj))

	svc := &Service{
		Repo:      repo,
		Memory:    &fakeMemoryCreator{},
		Promotion: &PromotionDigestConfig{RequireEvidence: true},
	}
	_, err = svc.Materialize(ctx, candidateID)
	if err == nil {
		t.Fatal("expected error when require_evidence and no evidence ids")
	}
}
