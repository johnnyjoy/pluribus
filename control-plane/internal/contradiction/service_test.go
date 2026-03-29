package contradiction

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"

	"control-plane/internal/memory"
	"control-plane/pkg/api"
)

type fakeMemoryRepo struct {
	byID       map[uuid.UUID]*memory.MemoryObject
	attributes map[uuid.UUID]map[string]string
}

func (f *fakeMemoryRepo) GetByID(ctx context.Context, id uuid.UUID) (*memory.MemoryObject, error) {
	return f.byID[id], nil
}

func (f *fakeMemoryRepo) GetAttributes(ctx context.Context, memoryID uuid.UUID) (map[string]string, error) {
	return f.attributes[memoryID], nil
}

func TestService_Create_rejectsSelfContradiction(t *testing.T) {
	svc := &Service{Repo: nil, MemoryRepo: nil} // Repo not used for this test
	ctx := context.Background()
	id := uuid.New()
	_, err := svc.Create(ctx, CreateRequest{MemoryID: id, ConflictWithID: id})
	if err != ErrSelfContradiction {
		t.Errorf("Create(same id): want ErrSelfContradiction, got %v", err)
	}
}

func TestService_DetectConflict_noOverlap(t *testing.T) {
	mem1 := uuid.New()
	mem2 := uuid.New()
	fake := &fakeMemoryRepo{
		byID: map[uuid.UUID]*memory.MemoryObject{
			mem1: {ID: mem1, Status: api.StatusActive},
			mem2: {ID: mem2, Status: api.StatusActive},
		},
		attributes: map[uuid.UUID]map[string]string{
			mem1: {"lang": "go"},
			mem2: {"lang": "go"},
		},
	}
	svc := &Service{MemoryRepo: fake}
	ctx := context.Background()
	ok, err := svc.DetectConflict(ctx, mem1, mem2)
	if err != nil {
		t.Fatalf("DetectConflict: %v", err)
	}
	if ok {
		t.Error("DetectConflict: same attribute value should not conflict")
	}
}

func TestService_DetectConflict_overlapDifferentValues(t *testing.T) {
	mem1 := uuid.New()
	mem2 := uuid.New()
	fake := &fakeMemoryRepo{
		byID: map[uuid.UUID]*memory.MemoryObject{
			mem1: {ID: mem1, Status: api.StatusActive},
			mem2: {ID: mem2, Status: api.StatusActive},
		},
		attributes: map[uuid.UUID]map[string]string{
			mem1: {"max_connections": "10"},
			mem2: {"max_connections": "100"},
		},
	}
	svc := &Service{MemoryRepo: fake}
	ctx := context.Background()
	ok, err := svc.DetectConflict(ctx, mem1, mem2)
	if err != nil {
		t.Fatalf("DetectConflict: %v", err)
	}
	if !ok {
		t.Error("DetectConflict: same key different value should conflict")
	}
}

func TestService_DetectConflict_noAttributes(t *testing.T) {
	mem1 := uuid.New()
	mem2 := uuid.New()
	fake := &fakeMemoryRepo{
		byID: map[uuid.UUID]*memory.MemoryObject{
			mem1: {ID: mem1, Status: api.StatusActive},
			mem2: {ID: mem2, Status: api.StatusActive},
		},
		attributes: map[uuid.UUID]map[string]string{
			mem1: {"a": "1"},
			mem2: {}, // no overlap
		},
	}
	svc := &Service{MemoryRepo: fake}
	ctx := context.Background()
	ok, err := svc.DetectConflict(ctx, mem1, mem2)
	if err != nil {
		t.Fatalf("DetectConflict: %v", err)
	}
	if ok {
		t.Error("DetectConflict: no overlapping keys should not conflict")
	}
}

func TestService_DetectConflict_inactiveMemory(t *testing.T) {
	mem1 := uuid.New()
	mem2 := uuid.New()
	fake := &fakeMemoryRepo{
		byID: map[uuid.UUID]*memory.MemoryObject{
			mem1: {ID: mem1, Status: api.StatusActive},
			mem2: {ID: mem2, Status: api.StatusSuperseded},
		},
		attributes: map[uuid.UUID]map[string]string{
			mem1: {"x": "1"},
			mem2: {"x": "2"},
		},
	}
	svc := &Service{MemoryRepo: fake}
	ctx := context.Background()
	ok, err := svc.DetectConflict(ctx, mem1, mem2)
	if err != nil {
		t.Fatalf("DetectConflict: %v", err)
	}
	if ok {
		t.Error("DetectConflict: inactive memory should not participate in conflict")
	}
}

func TestService_Create_and_ListMemoryIDsInUnresolved(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mem1 := uuid.MustParse("a1000000-0000-0000-0000-000000000001")
	mem2 := uuid.MustParse("a1000000-0000-0000-0000-000000000002")
	recID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`INSERT INTO contradiction_records`).
		WithArgs(sqlmock.AnyArg(), mem1, mem2, "unresolved").
		WillReturnRows(sqlmock.NewRows([]string{"id", "memory_id", "conflict_with_id", "resolution_state", "created_at", "updated_at"}).
			AddRow(recID, mem1, mem2, ResolutionUnresolved, now, now))
	// ListMemoryIDsInUnresolved runs one UNION query returning one column
	mock.ExpectQuery(`SELECT memory_id FROM contradiction_records`).
		WillReturnRows(sqlmock.NewRows([]string{"memory_id"}).AddRow(mem1).AddRow(mem2))

	repo := &Repo{DB: db}
	svc := &Service{Repo: repo, MemoryRepo: &fakeMemoryRepo{}}
	ctx := context.Background()

	rec, err := svc.Create(ctx, CreateRequest{MemoryID: mem1, ConflictWithID: mem2})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if rec.ID != recID {
		t.Errorf("Create: id = %v", rec.ID)
	}

	ids, err := svc.ListMemoryIDsInUnresolved(ctx)
	if err != nil {
		t.Fatalf("ListMemoryIDsInUnresolved: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("ListMemoryIDsInUnresolved: len = %d, want 2", len(ids))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestValidResolutionState(t *testing.T) {
	if !validResolutionState(ResolutionUnresolved) || !validResolutionState(ResolutionOverride) {
		t.Error("validResolutionState: unresolved/override should be valid")
	}
	if validResolutionState("invalid") {
		t.Error("validResolutionState: invalid should be false")
	}
}
