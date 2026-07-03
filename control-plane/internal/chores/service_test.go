package chores

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"control-plane/internal/contradiction"
	"control-plane/internal/memory"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

var (
	choreID  = uuid.MustParse("c0000000-0000-4000-8000-000000000001")
	memSubj  = uuid.MustParse("a0000000-0000-4000-8000-000000000001")
	memRel   = uuid.MustParse("a0000000-0000-4000-8000-000000000002")
	contraID = uuid.MustParse("b0000000-0000-4000-8000-000000000001")
)

func newTestService(t *testing.T) (*Service, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	svc := &Service{
		Repo: &Repo{DB: db},
		Memory: &memory.Service{
			Repo:          &memory.Repo{DB: db},
			Relationships: &memory.RelationshipRepo{DB: db},
		},
		Contradictions: &contradiction.Repo{DB: db},
	}
	return svc, mock, func() { db.Close() }
}

func choreRows(id uuid.UUID, choreType string, subject uuid.UUID, related *uuid.UUID, evidence any) *sqlmock.Rows {
	var rel any
	if related != nil {
		rel = *related
	}
	var ev []byte
	if evidence != nil {
		ev, _ = json.Marshal(evidence)
	}
	return sqlmock.NewRows([]string{
		"id", "chore_type", "subject_memory_id", "related_memory_id", "evidence",
		"state", "resolution_action", "created_at", "resolved_at",
	}).AddRow(id, choreType, subject, rel, ev, StateOpen, "", time.Now(), nil)
}

func memoryRows(id uuid.UUID, statement, status, agentID string, authority int, createdAt time.Time) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "kind", "statement", "statement_canonical", "statement_key", "authority",
		"applicability", "status", "deprecated_at", "ttl_seconds", "payload",
		"created_at", "updated_at", "occurred_at", "agent_id",
	}).AddRow(id, "constraint", statement, "", "", authority, "advisory", status, nil, nil, nil, createdAt, createdAt, nil, agentID)
}

// expectGetMemory mocks memory.Repo.GetByID (row fetch + tags fetch).
func expectGetMemory(mock sqlmock.Sqlmock, id uuid.UUID, statement, status, agentID string, authority int, createdAt time.Time) {
	mock.ExpectQuery(`FROM memories WHERE id`).WithArgs(id).
		WillReturnRows(memoryRows(id, statement, status, agentID, authority, createdAt))
	mock.ExpectQuery(`FROM memories_tags WHERE memory_id`).WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"tag"}))
}

func TestResolve_requiresAgentID(t *testing.T) {
	svc, _, done := newTestService(t)
	defer done()
	_, err := svc.Resolve(context.Background(), choreID, ResolveRequest{Action: ActionCoexist})
	if err == nil || err.Error() != "agent_id is required" {
		t.Fatalf("expected agent_id error, got %v", err)
	}
}

func TestResolve_invalidActionForType(t *testing.T) {
	svc, mock, done := newTestService(t)
	defer done()
	rel := memRel
	mock.ExpectQuery(`FROM curation_chores c WHERE`).WithArgs(choreID).
		WillReturnRows(choreRows(choreID, TypeDuplicatePair, memSubj, &rel, nil))

	_, err := svc.Resolve(context.Background(), choreID, ResolveRequest{AgentID: "agent-x", Action: ActionRelease})
	if err == nil {
		t.Fatal("expected action validation error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestResolve_singleVoteDoesNotApply(t *testing.T) {
	svc, mock, done := newTestService(t)
	defer done()
	rel := memRel
	now := time.Now()
	mock.ExpectQuery(`FROM curation_chores c WHERE`).WithArgs(choreID).
		WillReturnRows(choreRows(choreID, TypeDuplicatePair, memSubj, &rel, duplicateEvidence{CosineSimilarity: 0.95}))
	expectGetMemory(mock, memSubj, "use pg 16", "active", "author-a", 5, now.Add(-2*time.Hour))
	expectGetMemory(mock, memRel, "use postgres 16", "active", "author-b", 3, now)
	mock.ExpectExec(`INSERT INTO curation_chore_votes`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`FROM curation_chore_votes`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	resp, err := svc.Resolve(context.Background(), choreID, ResolveRequest{AgentID: "agent-x", Action: ActionConsolidate})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Applied || resp.State != StateOpen || resp.VotesForAction != 1 || !resp.Recorded || !resp.Counted {
		t.Fatalf("unexpected response: %+v", resp)
	}
	// ExpectationsWereMet proves no UPDATE memories / chores fired.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestResolve_secondDistinctAgentApplies_consolidate(t *testing.T) {
	svc, mock, done := newTestService(t)
	defer done()
	rel := memRel
	now := time.Now()
	mock.ExpectQuery(`FROM curation_chores c WHERE`).WithArgs(choreID).
		WillReturnRows(choreRows(choreID, TypeDuplicatePair, memSubj, &rel, duplicateEvidence{CosineSimilarity: 0.95}))
	// subject: authority 5 → survivor; related: authority 3 → superseded.
	expectGetMemory(mock, memSubj, "use pg 16", "active", "author-a", 5, now.Add(-2*time.Hour))
	expectGetMemory(mock, memRel, "use postgres 16", "active", "author-b", 3, now)
	mock.ExpectExec(`INSERT INTO curation_chore_votes`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`FROM curation_chore_votes`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	// apply: SupersedeMemory reloads the loser, marks it superseded, records the edge.
	expectGetMemory(mock, memRel, "use postgres 16", "active", "author-b", 3, now)
	mock.ExpectExec(`UPDATE memories SET status = 'superseded'`).
		WithArgs(sqlmock.AnyArg(), memRel).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`FROM memories WHERE id IN`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(`INSERT INTO memory_relationships`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "from_memory_id", "to_memory_id", "relationship_type", "reason", "source", "created_at"}).
			AddRow(uuid.New(), memSubj, memRel, "supersedes", "r", choreSource, time.Now()))
	mock.ExpectExec(`UPDATE curation_chores SET state`).
		WithArgs(StateResolved, ActionConsolidate, sqlmock.AnyArg(), choreID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	resp, err := svc.Resolve(context.Background(), choreID, ResolveRequest{AgentID: "agent-y", Action: ActionConsolidate})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Applied || resp.State != StateResolved {
		t.Fatalf("expected applied+resolved, got %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestResolve_sameAgentDoubleVoteRejected(t *testing.T) {
	svc, mock, done := newTestService(t)
	defer done()
	rel := memRel
	now := time.Now()
	mock.ExpectQuery(`FROM curation_chores c WHERE`).WithArgs(choreID).
		WillReturnRows(choreRows(choreID, TypeDuplicatePair, memSubj, &rel, nil))
	expectGetMemory(mock, memSubj, "s", "active", "author-a", 5, now)
	expectGetMemory(mock, memRel, "r", "active", "author-b", 3, now)
	// ON CONFLICT DO NOTHING → zero rows affected.
	mock.ExpectExec(`INSERT INTO curation_chore_votes`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`FROM curation_chore_votes`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	resp, err := svc.Resolve(context.Background(), choreID, ResolveRequest{AgentID: "agent-x", Action: ActionConsolidate})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Recorded || resp.Counted || resp.Applied {
		t.Fatalf("duplicate vote must not record, count, or apply: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestResolve_authorVoteNeverCounts(t *testing.T) {
	svc, mock, done := newTestService(t)
	defer done()
	svc.MinResolvers = 1 // even a loosened hive cannot self-curate
	now := time.Now()
	mock.ExpectQuery(`FROM curation_chores c WHERE`).WithArgs(choreID).
		WillReturnRows(choreRows(choreID, TypeQuarantineReview, memSubj, nil, quarantineEvidence{Reason: "harmful_advice"}))
	expectGetMemory(mock, memSubj, "rm -rf is fine", "quarantined", "author-a", 1, now)
	mock.ExpectExec(`INSERT INTO curation_chore_votes`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	// Count excludes the author hash → 0 counted votes.
	mock.ExpectQuery(`FROM curation_chore_votes`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	resp, err := svc.Resolve(context.Background(), choreID, ResolveRequest{AgentID: "author-a", Action: ActionRelease})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Counted || resp.Applied {
		t.Fatalf("author vote must not count or apply: %+v", resp)
	}
	if !resp.Recorded {
		t.Fatalf("author vote should still be recorded for audit: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestResolve_contradictionKeepSubject(t *testing.T) {
	svc, mock, done := newTestService(t)
	defer done()
	rel := memRel
	now := time.Now()
	mock.ExpectQuery(`FROM curation_chores c WHERE`).WithArgs(choreID).
		WillReturnRows(choreRows(choreID, TypeContradiction, memSubj, &rel,
			contradictionEvidence{ContradictionRecordID: contraID}))
	expectGetMemory(mock, memSubj, "use sqlite", "pending", "author-a", 4, now)
	expectGetMemory(mock, memRel, "never use sqlite", "active", "author-b", 4, now.Add(-time.Hour))
	mock.ExpectExec(`INSERT INTO curation_chore_votes`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`FROM curation_chore_votes`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	// keep_subject → related is superseded.
	expectGetMemory(mock, memRel, "never use sqlite", "active", "author-b", 4, now.Add(-time.Hour))
	mock.ExpectExec(`UPDATE memories SET status = 'superseded'`).
		WithArgs(sqlmock.AnyArg(), memRel).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`FROM memories WHERE id IN`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(`INSERT INTO memory_relationships`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "from_memory_id", "to_memory_id", "relationship_type", "reason", "source", "created_at"}).
			AddRow(uuid.New(), memSubj, memRel, "supersedes", "r", choreSource, time.Now()))
	mock.ExpectExec(`UPDATE contradiction_records SET resolution_state`).
		WithArgs(contradiction.ResolutionDeprecated, contraID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE curation_chores SET state`).
		WithArgs(StateResolved, ActionKeepSubject, sqlmock.AnyArg(), choreID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	resp, err := svc.Resolve(context.Background(), choreID, ResolveRequest{AgentID: "agent-y", Action: ActionKeepSubject})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Applied || resp.State != StateResolved {
		t.Fatalf("expected applied, got %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestResolve_duplicateDistinctDismisses(t *testing.T) {
	svc, mock, done := newTestService(t)
	defer done()
	rel := memRel
	now := time.Now()
	mock.ExpectQuery(`FROM curation_chores c WHERE`).WithArgs(choreID).
		WillReturnRows(choreRows(choreID, TypeDuplicatePair, memSubj, &rel, nil))
	expectGetMemory(mock, memSubj, "s", "active", "author-a", 5, now)
	expectGetMemory(mock, memRel, "r", "active", "author-b", 3, now)
	mock.ExpectExec(`INSERT INTO curation_chore_votes`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`FROM curation_chore_votes`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	// distinct → dismissed, no memory mutation at all.
	mock.ExpectExec(`UPDATE curation_chores SET state`).
		WithArgs(StateDismissed, ActionDistinct, sqlmock.AnyArg(), choreID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	resp, err := svc.Resolve(context.Background(), choreID, ResolveRequest{AgentID: "agent-y", Action: ActionDistinct})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Applied || resp.State != StateDismissed {
		t.Fatalf("expected dismissed, got %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestSyncReviewChores_opensChores(t *testing.T) {
	svc, mock, done := newTestService(t)
	defer done()
	mock.ExpectQuery(`FROM contradiction_records`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "memory_id", "conflict_with_id"}).
			AddRow(contraID, memSubj, memRel))
	mock.ExpectExec(`INSERT INTO curation_chores`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`FROM memories`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status_reason"}).
			AddRow(memSubj, "harmful_advice"))
	mock.ExpectExec(`INSERT INTO curation_chores`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	opened, err := svc.SyncReviewChores(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if opened != 2 {
		t.Fatalf("expected 2 chores opened, got %d", opened)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestScanNearDuplicates_opensPairChore(t *testing.T) {
	svc, mock, done := newTestService(t)
	defer done()
	mock.ExpectQuery(`SELECT LEAST`).
		WillReturnRows(sqlmock.NewRows([]string{"least", "greatest", "similarity"}).
			AddRow(memSubj, memRel, 0.95))
	mock.ExpectExec(`INSERT INTO curation_chores`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	opened, err := svc.ScanNearDuplicates(context.Background(), 0.92, 14, 20)
	if err != nil {
		t.Fatal(err)
	}
	if opened != 1 {
		t.Fatalf("expected 1 chore, got %d", opened)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestEnsureChore_pairAlreadyKnownIsNotReopened(t *testing.T) {
	svc, mock, done := newTestService(t)
	defer done()
	// WHERE NOT EXISTS filtered the insert → zero rows affected.
	mock.ExpectExec(`INSERT INTO curation_chores`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	rel := memRel
	created, err := svc.Repo.EnsureChore(context.Background(), TypeDuplicatePair, memSubj, &rel, nil)
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Fatal("existing pair must not open a second chore")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestHousekeepingLine_formats(t *testing.T) {
	rel := memRel
	ch := &Chore{ID: choreID, Type: TypeDuplicatePair, SubjectMemoryID: memSubj, RelatedMemoryID: &rel,
		SubjectStatement: "always run migrations before deploy", RelatedStatement: "run db migrations prior to deploying"}
	line := HousekeepingLine(ch)
	if line == "" {
		t.Fatal("expected a housekeeping line")
	}
	for _, want := range []string{choreID.String(), "resolve_chore", "consolidate", "distinct"} {
		if !strings.Contains(line, want) {
			t.Fatalf("line missing %q: %s", want, line)
		}
	}
	if HousekeepingLine(nil) != "" {
		t.Fatal("nil chore must produce no line")
	}
}
