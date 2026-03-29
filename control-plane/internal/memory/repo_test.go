package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"control-plane/internal/memorynorm"
	"control-plane/pkg/api"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestRepo_Create_withTags(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &Repo{DB: db}
	dedup := DedupKey()
	canon := memorynorm.StatementCanonical("Do not duplicate")
	sk := memorynorm.StatementKey("Do not duplicate")

	mock.ExpectQuery(`INSERT INTO memories`).
		WithArgs(sqlmock.AnyArg(), "constraint", "Do not duplicate", canon, sk, dedup, 7, "governing", "active", nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at"}).
			AddRow(uuid.MustParse("22222222-2222-2222-2222-222222222222"), api.MemoryKindConstraint, "Do not duplicate", canon, sk, 7, "governing", "active", nil, nil, nil, time.Now(), time.Now()))
	for _, tag := range []string{"api", "rest"} {
		mock.ExpectExec(`INSERT INTO memories_tags`).WithArgs(sqlmock.AnyArg(), tag).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}

	obj, err := repo.Create(ctx, CreateRequest{
		Kind:      api.MemoryKindConstraint,
		Authority: 7,
		Statement: "Do not duplicate",
		Tags:      []string{"api", "rest"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if obj.Statement != "Do not duplicate" || len(obj.Tags) != 2 {
		t.Errorf("got %+v len(tags)=%d", obj, len(obj.Tags))
	}
	if obj.StatementCanonical != canon {
		t.Errorf("StatementCanonical = %q", obj.StatementCanonical)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestRepo_Search_authorityOrdering(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &Repo{DB: db}
	id1 := uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	id2 := uuid.MustParse("a0000000-0000-0000-0000-000000000002")
	mock.ExpectQuery(`SELECT m.id, m.kind`).
		WithArgs("active", sqlmock.AnyArg(), 10).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "payload", "created_at", "updated_at"}).
			AddRow(id1, api.MemoryKindConstraint, "High authority", memorynorm.StatementCanonical("High authority"), memorynorm.StatementKey("High authority"), 9, "governing", "active", nil, time.Now(), time.Now()).
			AddRow(id2, api.MemoryKindDecision, "Lower authority", memorynorm.StatementCanonical("Lower authority"), memorynorm.StatementKey("Lower authority"), 5, "governing", "active", nil, time.Now(), time.Now()))
	mock.ExpectQuery(`SELECT memory_id, tag FROM memories_tags WHERE memory_id = ANY`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"memory_id", "tag"}).
			AddRow(id1, "api").
			AddRow(id2, "api"))

	list, err := repo.Search(ctx, SearchRequest{Tags: []string{"api"}, Max: 10})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 results, got %d", len(list))
	}
	if list[0].Authority != 9 || list[1].Authority != 5 {
		t.Errorf("expected authority order 9 then 5, got %d then %d", list[0].Authority, list[1].Authority)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestRepo_Search_kindsFilter(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &Repo{DB: db}
	id1 := uuid.MustParse("c0000000-0000-0000-0000-000000000001")
	mock.ExpectQuery(`SELECT id, kind`).
		WithArgs("active", sqlmock.AnyArg(), 5).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "payload", "created_at", "updated_at"}).
			AddRow(id1, api.MemoryKindPattern, "Always validate input", memorynorm.StatementCanonical("Always validate input"), memorynorm.StatementKey("Always validate input"), 6, "advisory", "active", nil, time.Now(), time.Now()))
	mock.ExpectQuery(`SELECT memory_id, tag FROM memories_tags WHERE memory_id = ANY`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"memory_id", "tag"}))

	list, err := repo.Search(ctx, SearchRequest{
		Max:   5,
		Kinds: []api.MemoryKind{api.MemoryKindPattern},
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(list) != 1 || list[0].Kind != api.MemoryKindPattern {
		t.Fatalf("got %+v", list)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestRepo_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	repo := &Repo{DB: db}
	id := uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	canon := memorynorm.StatementCanonical("A decision")
	sk := memorynorm.StatementKey("A decision")

	mock.ExpectQuery(`SELECT id, kind, statement`).WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at"}).
			AddRow(id, api.MemoryKindDecision, "A decision", canon, sk, 5, "governing", "active", nil, nil, nil, time.Now(), time.Now()))
	mock.ExpectQuery(`SELECT tag FROM memories_tags`).WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"tag"}).AddRow("topic:decisions").AddRow("api"))

	obj, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if obj == nil || obj.Statement != "A decision" || obj.Authority != 5 {
		t.Errorf("GetByID: got %+v", obj)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

// TestRepo_Create_patternPayload_roundTrip verifies payload is persisted and retrieved.
func TestRepo_Create_patternPayload_roundTrip(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	repo := &Repo{DB: db}
	dedup := DedupKey()
	payload := PatternPayload{
		Polarity:   "negative",
		Experience: "Deployed without tests.",
		Decision:   "Require tests",
		Outcome:    "Fewer regressions",
		Impact:     PatternImpact{Severity: "high"},
		Directive:  "Never skip tests.",
	}
	payloadBytes, _ := json.Marshal(payload)
	memID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	canon := memorynorm.StatementCanonical("Never skip tests.")
	sk := memorynorm.StatementKey("Never skip tests.")
	insertRows := sqlmock.NewRows([]string{"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at"}).
		AddRow(memID, api.MemoryKindPattern, "Never skip tests.", canon, sk, 5, "governing", "active", nil, nil, payloadBytes, time.Now(), time.Now())
	mock.ExpectQuery(`INSERT INTO memories`).
		WithArgs(sqlmock.AnyArg(), "pattern", "Never skip tests.", canon, sk, dedup, 5, "governing", "active", nil, payloadBytes).
		WillReturnRows(insertRows)
	raw := json.RawMessage(payloadBytes)
	obj, err := repo.Create(ctx, CreateRequest{
		Kind:         api.MemoryKindPattern,
		Authority:    5,
		Statement:    "Never skip tests.",
		StatementKey: sk,
		Payload:      &raw,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if obj == nil || obj.Kind != api.MemoryKindPattern {
		t.Fatalf("Create: got %+v", obj)
	}
	if len(obj.Payload) == 0 {
		t.Fatal("Create: expected payload in returned object")
	}
	var got PatternPayload
	if err := json.Unmarshal(obj.Payload, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if got.Polarity != payload.Polarity || got.Directive != payload.Directive {
		t.Errorf("payload round-trip: got %+v", got)
	}

	getByIDRows := sqlmock.NewRows([]string{"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at"}).
		AddRow(memID, api.MemoryKindPattern, "Never skip tests.", canon, sk, 5, "governing", "active", nil, nil, payloadBytes, time.Now(), time.Now())
	mock.ExpectQuery(`SELECT id, kind, statement`).WithArgs(memID).
		WillReturnRows(getByIDRows)
	mock.ExpectQuery(`SELECT tag FROM memories_tags`).WithArgs(memID).
		WillReturnRows(sqlmock.NewRows([]string{"tag"}))

	gotObj, err := repo.GetByID(ctx, memID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if gotObj == nil || len(gotObj.Payload) == 0 {
		t.Fatal("GetByID: expected payload")
	}
	if err := json.Unmarshal(gotObj.Payload, &got); err != nil {
		t.Fatalf("GetByID unmarshal payload: %v", err)
	}
	if got.Decision != payload.Decision || got.Impact.Severity != payload.Impact.Severity {
		t.Errorf("GetByID payload: got %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestRepo_GetByID_notFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := &Repo{DB: db}
	id := uuid.MustParse("a0000000-0000-0000-0000-000000000099")
	mock.ExpectQuery(`SELECT id, kind, statement`).WithArgs(id).WillReturnError(sql.ErrNoRows)

	obj, err := repo.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if obj != nil {
		t.Errorf("GetByID: expected nil, got %+v", obj)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestRepo_UpdateAuthority(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	id := uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	mock.ExpectExec(`UPDATE memories SET authority`).WithArgs(4, id).WillReturnResult(sqlmock.NewResult(0, 1))

	err = (&Repo{DB: db}).UpdateAuthority(context.Background(), id, 4)
	if err != nil {
		t.Fatalf("UpdateAuthority: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}
