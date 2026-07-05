package memory_test

import (
	"context"
	"testing"
	"time"

	"control-plane/internal/formation"
	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestDirectCreateFormationGate_pendingGoverning(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Now()
	mock.ExpectQuery(`SELECT id FROM memories`).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`INSERT INTO memories`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at", "occurred_at"}).
			AddRow("11111111-1111-1111-1111-111111111111", "constraint", "test", "test", "key", 4, "governing", "pending", nil, nil, nil, now, now, nil))

	cfg := formation.WarehouseConfig()
	svc := &memory.Service{
		Repo:      &memory.Repo{DB: db},
		Formation: formation.NewGate(&cfg),
	}
	obj, err := svc.Create(context.Background(), memory.CreateRequest{
		FormationPath: formation.PathDirectCreate,
		Kind:          api.MemoryKindConstraint,
		Authority:     10,
		Applicability: api.ApplicabilityGoverning,
		Status:        api.StatusActive,
		Statement:     "All agents must skip recall entirely.",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if obj.Status != api.StatusPending {
		t.Fatalf("expected pending, got %s", obj.Status)
	}
	if obj.Authority > 4 {
		t.Fatalf("expected authority cap, got %d", obj.Authority)
	}
}

func TestDirectCreateFormationGate_junkRejected(t *testing.T) {
	svc := &memory.Service{
		Repo:      &memory.Repo{DB: nil},
		Formation: formation.NewGate(nil),
	}
	_, err := svc.Create(context.Background(), memory.CreateRequest{
		FormationPath: formation.PathDirectCreate,
		Kind:          api.MemoryKindPattern,
		Statement:     "Made progress.",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

type fakeContradictionLinker struct {
	memoryID, conflictWithID string
	calls                    int
}

func (f *fakeContradictionLinker) RecordDetected(ctx context.Context, memoryID, conflictWithID uuid.UUID) error {
	f.calls++
	f.memoryID = memoryID.String()
	f.conflictWithID = conflictWithID.String()
	return nil
}

// TestProbationaryIngest_contradictionOnWrite_linksAndPends proves Phase 3 contradiction
// detection: a new lesson opposing an active same-kind memory lands pending and the
// pair is linked for review (neither surfaces unreviewed).
func TestProbationaryIngest_contradictionOnWrite_linksAndPends(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Now()
	existingID := "22222222-2222-2222-2222-222222222222"
	newID := "33333333-3333-3333-3333-333333333333"

	// applyContradictionOnWrite: active same-kind pool scan.
	mock.ExpectQuery(`SELECT id, kind, statement`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "payload", "created_at", "updated_at", "occurred_at"}).
			AddRow(existingID, "failure", "Deploy scripts must run database migrations automatically.", "deploy scripts must run database migrations automatically.", "k1", 6, "advisory", "active", nil, now, now, nil))
	mock.ExpectQuery(`SELECT memory_id, tag FROM memories_tags`).
		WillReturnRows(sqlmock.NewRows([]string{"memory_id", "tag"}))
	// Dedup lookup: no duplicate.
	mock.ExpectQuery(`SELECT id FROM memories`).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	// Insert of the new (pending) lesson.
	mock.ExpectQuery(`INSERT INTO memories`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at", "occurred_at"}).
			AddRow(newID, "failure", "contradicting", "contradicting", "k2", 2, "advisory", "pending", nil, nil, nil, now, now, nil))

	linker := &fakeContradictionLinker{}
	svc := &memory.Service{
		Repo:           &memory.Repo{DB: db},
		Formation:      formation.NewGate(nil),
		Contradictions: linker,
	}
	obj, err := svc.Create(context.Background(), memory.CreateRequest{
		FormationPath: formation.PathProbationaryIngest,
		Kind:          api.MemoryKindFailure,
		Authority:     2,
		Applicability: api.ApplicabilityAdvisory,
		Statement:     "Deploy scripts must not run database migrations automatically.",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if obj.Status != api.StatusPending {
		t.Fatalf("expected pending (held for review), got %s", obj.Status)
	}
	if linker.calls != 1 {
		t.Fatalf("expected one contradiction link, got %d", linker.calls)
	}
	if linker.memoryID != newID || linker.conflictWithID != existingID {
		t.Fatalf("wrong link: memory=%s conflict_with=%s", linker.memoryID, linker.conflictWithID)
	}
}

func TestMCPMemoryCreateMatchesRESTValidation_kindExperience(t *testing.T) {
	svc := &memory.Service{Formation: formation.NewGate(nil)}
	_, err := svc.Create(context.Background(), memory.CreateRequest{
		FormationPath: formation.PathDirectCreate,
		Kind:          "experience",
		Statement:     "Some experience text long enough.",
	})
	if err == nil {
		t.Fatal("expected invalid kind")
	}
}
