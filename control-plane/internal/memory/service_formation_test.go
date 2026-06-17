package memory_test

import (
	"context"
	"testing"
	"time"

	"control-plane/internal/formation"
	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/DATA-DOG/go-sqlmock"
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

	svc := &memory.Service{
		Repo:      &memory.Repo{DB: db},
		Formation: formation.NewGate(nil),
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
