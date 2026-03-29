package ingest

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

// expectGlobalUnifyCandidatesEmpty runs GMCL's two candidate queries (subject+predicate, then subject-wide).
func expectGlobalUnifyCandidatesEmpty(mock sqlmock.Sqlmock, subjectNorm, predicateNorm string) {
	mock.ExpectQuery(`SELECT predicate_norm, object_norm, normalized_hash FROM canonical_fact_extractions WHERE subject_norm = \$1 AND predicate_norm = \$2 ORDER BY normalized_hash ASC LIMIT \$3`).
		WithArgs(subjectNorm, predicateNorm, 500).
		WillReturnRows(sqlmock.NewRows([]string{"predicate_norm", "object_norm", "normalized_hash"}))
	mock.ExpectQuery(`SELECT predicate_norm, object_norm, normalized_hash FROM canonical_fact_extractions WHERE subject_norm = \$1 ORDER BY normalized_hash ASC LIMIT \$2`).
		WithArgs(subjectNorm, 500).
		WillReturnRows(sqlmock.NewRows([]string{"predicate_norm", "object_norm", "normalized_hash"}))
}

func expectLineageInsert(mock sqlmock.Sqlmock, ingID uuid.UUID) {
	mock.ExpectExec(`INSERT INTO canonical_fact_lineage`).
		WithArgs(ingID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
}

func expectContradictionInsert(mock sqlmock.Sqlmock, ingID uuid.UUID) {
	mock.ExpectExec(`INSERT INTO canonical_fact_contradictions`).
		WithArgs(ingID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
}

func validCognitionRequest() CognitionRequest {
	return CognitionRequest{
		TempContributorID: "client-1",
		Query:             "analyze dependency",
		ReasoningTrace: []string{
			"Reviewed go.mod and verified dependency declarations for the service",
			"Confirmed pg driver appears in module dependencies as required",
		},
		ExtractedFacts:    []ExtractedFact{{Subject: "app", Predicate: "depends_on", Object: "postgres"}},
		Confidence:        0.75,
		ContextWindowHash: "hashctx-v1",
	}
}

func expectTrustWeight(mock sqlmock.Sqlmock, tempContributorID string, weight float64) {
	mock.ExpectQuery(`SELECT trust_weight FROM temp_contributor_profiles WHERE temp_contributor_id = \$1`).
		WithArgs(tempContributorID).
		WillReturnRows(sqlmock.NewRows([]string{"trust_weight"}).AddRow(weight))
}

func expectMaxCreatedAtNull(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(`SELECT MAX\(created_at\) FROM canonical_fact_extractions WHERE normalized_hash = \$1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(nil))
}

func TestService_IngestCognition_accepted(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ingID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	expectTrustWeight(mock, "client-1", 1.0)
	mock.ExpectBegin()
	payload, _ := json.Marshal(validCognitionRequest())
	mock.ExpectQuery(`INSERT INTO ingestion_records`).
		WithArgs("client-1", "accepted", nil, payload, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(ingID))

	expectGlobalUnifyCandidatesEmpty(mock, NormalizeFactToken("app"), NormalizeFactToken("depends_on"))
	mock.ExpectQuery(`SELECT COALESCE\(MAX\(confidence\), 0\), COUNT\(\*\) FROM canonical_fact_extractions WHERE normalized_hash = \$1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"max", "count"}).AddRow(0.0, int64(0)))

	expectMaxCreatedAtNull(mock)
	mock.ExpectExec(`INSERT INTO canonical_fact_extractions`).
		WithArgs(ingID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 0, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	svc := NewService(&Repo{DB: db})
	resp, err := svc.IngestCognition(context.Background(), validCognitionRequest())
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "accepted" {
		t.Fatalf("status %q", resp.Status)
	}
	if resp.IngestionID != ingID {
		t.Fatalf("ingestion id %v", resp.IngestionID)
	}
	if len(resp.CanonicalFacts) != 1 {
		t.Fatalf("expected 1 canonical fact, got %d", len(resp.CanonicalFacts))
	}
	var cf CanonicalFactJSON
	if err := json.Unmarshal(resp.CanonicalFacts[0], &cf); err != nil {
		t.Fatal(err)
	}
	if cf.Subject != "app" || cf.Predicate != "depends_on" || cf.Object != "postgres" {
		t.Fatalf("unexpected canonical: %+v", cf)
	}
	if cf.NormalizeVersion != NormalizePipelineVersion {
		t.Fatalf("version %q", cf.NormalizeVersion)
	}
	if resp.Debug.NormalizationVersion != NormalizePipelineVersion {
		t.Fatal("debug missing normalization version")
	}
	if resp.Debug.PriorityFormulaVersion != PriorityFormulaVersion {
		t.Fatalf("priority formula: %q", resp.Debug.PriorityFormulaVersion)
	}
	if resp.Debug.Promotion.Attempted {
		t.Fatal("promotion must not be attempted")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestService_IngestCognition_reinforcesWhenHashExistsInDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ingID := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	expectTrustWeight(mock, "client-1", 1.0)
	mock.ExpectBegin()
	payload, _ := json.Marshal(validCognitionRequest())
	mock.ExpectQuery(`INSERT INTO ingestion_records`).
		WithArgs("client-1", "accepted", nil, payload, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(ingID))

	expectGlobalUnifyCandidatesEmpty(mock, NormalizeFactToken("app"), NormalizeFactToken("depends_on"))
	mock.ExpectQuery(`SELECT COALESCE\(MAX\(confidence\), 0\), COUNT\(\*\) FROM canonical_fact_extractions WHERE normalized_hash = \$1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"max", "count"}).AddRow(0.75, int64(1)))

	expectMaxCreatedAtNull(mock)
	wantConf := ApplyReinforce(0.75, 0.75)
	mock.ExpectExec(`INSERT INTO canonical_fact_extractions`).
		WithArgs(ingID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), wantConf, sqlmock.AnyArg(), sqlmock.AnyArg(), 0, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectLineageInsert(mock, ingID)

	mock.ExpectCommit()

	svc := NewService(&Repo{DB: db})
	resp, err := svc.IngestCognition(context.Background(), validCognitionRequest())
	if err != nil {
		t.Fatal(err)
	}
	var cf CanonicalFactJSON
	if err := json.Unmarshal(resp.CanonicalFacts[0], &cf); err != nil {
		t.Fatal(err)
	}
	if cf.Confidence != wantConf {
		t.Fatalf("reinforced confidence: got %v want %v", cf.Confidence, wantConf)
	}
	if len(resp.Debug.MergeActions) != 1 {
		t.Fatalf("merge_actions: got %d want 1", len(resp.Debug.MergeActions))
	}
	if resp.Debug.MergeActions[0]["action"] != "reinforce" {
		t.Fatalf("merge action: %+v", resp.Debug.MergeActions[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestService_IngestCognition_reinforcesDuplicateInSameBatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ingID := uuid.MustParse("55555555-5555-5555-5555-555555555555")

	req := validCognitionRequest()
	req.ExtractedFacts = []ExtractedFact{
		{Subject: "app", Predicate: "depends_on", Object: "postgres"},
		{Subject: "app", Predicate: "depends_on", Object: "postgres"},
	}

	expectTrustWeight(mock, "client-1", 1.0)
	mock.ExpectBegin()
	payload, _ := json.Marshal(req)
	mock.ExpectQuery(`INSERT INTO ingestion_records`).
		WithArgs("client-1", "accepted", nil, payload, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(ingID))

	expectGlobalUnifyCandidatesEmpty(mock, NormalizeFactToken("app"), NormalizeFactToken("depends_on"))
	expectGlobalUnifyCandidatesEmpty(mock, NormalizeFactToken("app"), NormalizeFactToken("depends_on"))

	secondConf := ApplyReinforce(0.75, 0.75)
	mock.ExpectQuery(`SELECT COALESCE\(MAX\(confidence\), 0\), COUNT\(\*\) FROM canonical_fact_extractions WHERE normalized_hash = \$1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"max", "count"}).AddRow(0.0, int64(0)))
	expectMaxCreatedAtNull(mock)
	mock.ExpectExec(`INSERT INTO canonical_fact_extractions`).
		WithArgs(ingID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 0.75, sqlmock.AnyArg(), sqlmock.AnyArg(), 0, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery(`SELECT COALESCE\(MAX\(confidence\), 0\), COUNT\(\*\) FROM canonical_fact_extractions WHERE normalized_hash = \$1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"max", "count"}).AddRow(0.75, int64(1)))
	expectMaxCreatedAtNull(mock)
	mock.ExpectExec(`INSERT INTO canonical_fact_extractions`).
		WithArgs(ingID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), secondConf, sqlmock.AnyArg(), sqlmock.AnyArg(), 1, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectLineageInsert(mock, ingID)

	mock.ExpectCommit()

	svc := NewService(&Repo{DB: db})
	resp, err := svc.IngestCognition(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.CanonicalFacts) != 2 {
		t.Fatalf("canonical facts: %d", len(resp.CanonicalFacts))
	}
	var cf0, cf1 CanonicalFactJSON
	_ = json.Unmarshal(resp.CanonicalFacts[0], &cf0)
	_ = json.Unmarshal(resp.CanonicalFacts[1], &cf1)
	if cf0.Confidence != 0.75 || cf1.Confidence != secondConf {
		t.Fatalf("confidences: %+v %+v", cf0, cf1)
	}
	if len(resp.Debug.MergeActions) != 1 {
		t.Fatalf("merge_actions: %d", len(resp.Debug.MergeActions))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestService_IngestCognition_conflictReportedWithoutReject(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ingID := uuid.MustParse("66666666-6666-6666-6666-666666666666")

	req := validCognitionRequest()
	req.ExtractedFacts = []ExtractedFact{
		{Subject: "app", Predicate: "uses", Object: "postgres"},
		{Subject: "app", Predicate: "uses", Object: "mysql"},
	}

	expectTrustWeight(mock, "client-1", 1.0)
	mock.ExpectBegin()
	payload, _ := json.Marshal(req)
	mock.ExpectQuery(`INSERT INTO ingestion_records`).
		WithArgs("client-1", "accepted", nil, payload, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(ingID))

	expectGlobalUnifyCandidatesEmpty(mock, NormalizeFactToken("app"), NormalizeFactToken("uses"))
	expectGlobalUnifyCandidatesEmpty(mock, NormalizeFactToken("app"), NormalizeFactToken("uses"))

	mock.ExpectQuery(`SELECT COALESCE\(MAX\(confidence\), 0\), COUNT\(\*\) FROM canonical_fact_extractions WHERE normalized_hash = \$1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"max", "count"}).AddRow(0.0, int64(0)))
	expectMaxCreatedAtNull(mock)
	mock.ExpectExec(`INSERT INTO canonical_fact_extractions`).
		WithArgs(ingID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 0, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery(`SELECT COALESCE\(MAX\(confidence\), 0\), COUNT\(\*\) FROM canonical_fact_extractions WHERE normalized_hash = \$1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"max", "count"}).AddRow(0.0, int64(0)))
	expectMaxCreatedAtNull(mock)
	mock.ExpectExec(`INSERT INTO canonical_fact_extractions`).
		WithArgs(ingID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 1, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectContradictionInsert(mock, ingID)

	mock.ExpectCommit()

	svc := NewService(&Repo{DB: db})
	resp, err := svc.IngestCognition(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "accepted" {
		t.Fatalf("status %q", resp.Status)
	}
	if len(resp.Debug.ConflictsDetected) != 1 {
		t.Fatalf("conflicts: %v", resp.Debug.ConflictsDetected)
	}
	if resp.Debug.ConflictsDetected[0]["code"] != "same_predicate_divergent_object" {
		t.Fatalf("code: %v", resp.Debug.ConflictsDetected[0]["code"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestService_IngestCognition_similarUnifyThenReinforceSecondRow(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ingID := uuid.MustParse("77777777-7777-7777-7777-777777777777")

	req := validCognitionRequest()
	req.ExtractedFacts = []ExtractedFact{
		{Subject: "app", Predicate: "depends_on", Object: "alpha beta gamma"},
		{Subject: "app", Predicate: "depends_on", Object: "alpha beta gamma delta"},
	}

	expectTrustWeight(mock, "client-1", 1.0)
	mock.ExpectBegin()
	payload, _ := json.Marshal(req)
	mock.ExpectQuery(`INSERT INTO ingestion_records`).
		WithArgs("client-1", "accepted", nil, payload, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(ingID))

	secondConf := ApplyReinforce(0.75, 0.75)
	expectGlobalUnifyCandidatesEmpty(mock, NormalizeFactToken("app"), NormalizeFactToken("depends_on"))
	expectGlobalUnifyCandidatesEmpty(mock, NormalizeFactToken("app"), NormalizeFactToken("depends_on"))

	mock.ExpectQuery(`SELECT COALESCE\(MAX\(confidence\), 0\), COUNT\(\*\) FROM canonical_fact_extractions WHERE normalized_hash = \$1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"max", "count"}).AddRow(0.0, int64(0)))
	expectMaxCreatedAtNull(mock)
	mock.ExpectExec(`INSERT INTO canonical_fact_extractions`).
		WithArgs(ingID, sqlmock.AnyArg(), sqlmock.AnyArg(), "alpha beta gamma", 0.75, sqlmock.AnyArg(), sqlmock.AnyArg(), 0, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery(`SELECT COALESCE\(MAX\(confidence\), 0\), COUNT\(\*\) FROM canonical_fact_extractions WHERE normalized_hash = \$1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"max", "count"}).AddRow(0.75, int64(1)))
	expectMaxCreatedAtNull(mock)
	mock.ExpectExec(`INSERT INTO canonical_fact_extractions`).
		WithArgs(ingID, sqlmock.AnyArg(), sqlmock.AnyArg(), "alpha beta gamma", secondConf, sqlmock.AnyArg(), sqlmock.AnyArg(), 1, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectLineageInsert(mock, ingID)
	expectLineageInsert(mock, ingID)

	mock.ExpectCommit()

	svc := NewService(&Repo{DB: db})
	resp, err := svc.IngestCognition(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Debug.MergeActions) != 2 {
		t.Fatalf("merge_actions want 2 similar+reinforce, got %v", resp.Debug.MergeActions)
	}
	if resp.Debug.MergeActions[0]["action"] != "similar_unify" {
		t.Fatalf("first action: %+v", resp.Debug.MergeActions[0])
	}
	if resp.Debug.MergeActions[1]["action"] != "reinforce" {
		t.Fatalf("second action: %+v", resp.Debug.MergeActions[1])
	}
	var cf1 CanonicalFactJSON
	_ = json.Unmarshal(resp.CanonicalFacts[1], &cf1)
	if cf1.Object != "alpha beta gamma" || cf1.Confidence != secondConf {
		t.Fatalf("row1: %+v", cf1)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestService_IngestCognition_rejectedNoiseLowConfidence(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ingID := uuid.MustParse("88888888-8888-8888-8888-888888888888")

	req := validCognitionRequest()
	req.ExtractedFacts[0].Confidence = ptrFloat(0.01)

	expectTrustWeight(mock, "client-1", 1.0)
	payload, _ := json.Marshal(req)
	reason := "noise: extracted_facts[0] confidence*trust_weight=0.01 below minimum 0.15"

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO ingestion_records`).
		WithArgs("client-1", "rejected", &reason, payload, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(ingID))
	mock.ExpectCommit()

	svc := NewService(&Repo{DB: db})
	resp, err := svc.IngestCognition(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "rejected" || !strings.HasPrefix(resp.Debug.RejectedReason, "noise:") {
		t.Fatalf("got status %q reason %q", resp.Status, resp.Debug.RejectedReason)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestService_IngestCognition_highTrustAcceptsMarginalConfidence(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ingID := uuid.MustParse("99999999-9999-9999-9999-999999999999")

	req := validCognitionRequest()
	req.ExtractedFacts[0].Confidence = ptrFloat(0.06)

	expectTrustWeight(mock, "client-1", 3.0)
	mock.ExpectBegin()
	payload, _ := json.Marshal(req)
	mock.ExpectQuery(`INSERT INTO ingestion_records`).
		WithArgs("client-1", "accepted", nil, payload, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(ingID))

	expectGlobalUnifyCandidatesEmpty(mock, NormalizeFactToken("app"), NormalizeFactToken("depends_on"))
	mock.ExpectQuery(`SELECT COALESCE\(MAX\(confidence\), 0\), COUNT\(\*\) FROM canonical_fact_extractions WHERE normalized_hash = \$1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"max", "count"}).AddRow(0.0, int64(0)))
	expectMaxCreatedAtNull(mock)
	mock.ExpectExec(`INSERT INTO canonical_fact_extractions`).
		WithArgs(ingID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 0, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	svc := NewService(&Repo{DB: db})
	resp, err := svc.IngestCognition(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "accepted" {
		t.Fatalf("status %q", resp.Status)
	}
	if resp.Debug.TrustWeightApplied != 3.0 {
		t.Fatalf("trust %v", resp.Debug.TrustWeightApplied)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestService_IngestCognition_rejectedStillPersists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ingID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	req := validCognitionRequest()
	req.ContextWindowHash = ""
	payload, _ := json.Marshal(req)
	reason := "context_window_hash: required"

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO ingestion_records`).
		WithArgs("client-1", "rejected", &reason, payload, nil).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(ingID))
	mock.ExpectCommit()

	svc := NewService(&Repo{DB: db})
	resp, err := svc.IngestCognition(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "rejected" {
		t.Fatalf("status %q", resp.Status)
	}
	if len(resp.CanonicalFacts) != 0 {
		t.Fatalf("rejected should have no canonical facts")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
