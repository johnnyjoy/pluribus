package memory

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"testing"
	"time"

	"control-plane/internal/cache"
	"control-plane/internal/memorynorm"
	"control-plane/pkg/api"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

// Ensure fakeCache implements cache.Store.
var _ cache.Store = (*fakeCache)(nil)

// fakeCache implements cache.Store for tests (tag-index cache behavior).
type fakeCache struct {
	mu    sync.Mutex
	store map[string][]byte
	ttl   time.Duration
}

func (f *fakeCache) Get(ctx context.Context, key string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.store[key], nil
}

func (f *fakeCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.store == nil {
		f.store = make(map[string][]byte)
	}
	f.store[key] = value
	return nil
}

func (f *fakeCache) DeleteByPrefix(ctx context.Context, prefix string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for k := range f.store {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(f.store, k)
		}
	}
	return nil
}

func TestService_Search_cacheHit(t *testing.T) {
	ctx := context.Background()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// One Search query: no tags, status active, max 20; Task 97: payload column
	mock.ExpectQuery(`SELECT id, kind, statement`).
		WithArgs("active", 20).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "payload", "created_at", "updated_at"}).
			AddRow(uuid.New(), api.MemoryKindDecision, "Use POST for create", memorynorm.StatementCanonical("Use POST for create"), memorynorm.StatementKey("Use POST for create"), 8, "governing", "active", nil, time.Now(), time.Now()))
	mock.ExpectQuery(`SELECT memory_id, tag FROM memories_tags WHERE memory_id = ANY`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"memory_id", "tag"}))

	repo := &Repo{DB: db}
	fake := &fakeCache{store: make(map[string][]byte)}
	svc := &Service{Repo: repo, Cache: fake, CacheTTL: time.Minute}

	// First Search: cache miss, hits DB, caches result.
	list1, err := svc.Search(ctx, SearchRequest{})
	if err != nil {
		t.Fatalf("first Search: %v", err)
	}
	if len(list1) != 1 {
		t.Fatalf("first Search: got %d items", len(list1))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations after first Search: %v", err)
	}

	// Second Search: cache hit, no DB call.
	list2, err := svc.Search(ctx, SearchRequest{})
	if err != nil {
		t.Fatalf("second Search: %v", err)
	}
	if len(list2) != 1 {
		t.Fatalf("second Search: got %d items", len(list2))
	}
	// Cached result should match (same statement).
	if list2[0].Statement != list1[0].Statement {
		t.Errorf("cached result mismatch: %q vs %q", list2[0].Statement, list1[0].Statement)
	}
}

func TestService_Create_validation(t *testing.T) {
	svc := &Service{Repo: nil}
	ctx := context.Background()

	_, err := svc.Create(ctx, CreateRequest{Kind: api.MemoryKindDecision, Statement: ""})
	if err == nil || err.Error() != "statement is required" {
		t.Errorf("expected statement required: %v", err)
	}

	_, err = svc.Create(ctx, CreateRequest{Kind: "invalid", Statement: "x"})
	if err == nil || err.Error() != `invalid kind "invalid"` {
		t.Errorf("expected invalid kind: %v", err)
	}

	_, err = svc.Create(ctx, CreateRequest{Kind: api.MemoryKindDecision, Statement: "x", Status: api.StatusArchived})
	if err == nil || !strings.Contains(err.Error(), "invalid status") {
		t.Errorf("expected invalid status on create: %v", err)
	}

	_, err = svc.Create(ctx, CreateRequest{Kind: "candidate", Authority: 5, Statement: "x"})
	if err == nil || !strings.Contains(err.Error(), `invalid kind "candidate"`) {
		t.Errorf("expected invalid kind for non-behavior kind: %v", err)
	}
}

func TestService_Search_validation(t *testing.T) {
	svc := &Service{Repo: nil}
	ctx := context.Background()

	_, err := svc.Search(ctx, SearchRequest{})
	if err == nil || err.Error() != "memory repo not configured" {
		t.Errorf("expected repo error: %v", err)
	}
}

func TestService_ApplyAuthorityEvent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	id := uuid.MustParse("a0000000-0000-0000-0000-000000000001")

	sk := memorynorm.StatementKey("A decision")
	mock.ExpectQuery(`SELECT id, kind, statement`).WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at"}).
			AddRow(id, api.MemoryKindDecision, "A decision", memorynorm.StatementCanonical("A decision"), sk, 5, "governing", "active", nil, nil, nil, time.Now(), time.Now()))
	mock.ExpectQuery(`SELECT tag FROM memories_tags`).WithArgs(id).WillReturnRows(
		sqlmock.NewRows([]string{"tag"}).AddRow("api").AddRow("decision"))
	mock.ExpectExec(`UPDATE memories SET authority`).WithArgs(4, id).WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &Service{
		Repo:      &Repo{DB: db},
		Lifecycle: &LifecycleConfig{AuthorityPositiveDelta: 0.1, AuthorityNegativeDelta: 0.2},
	}
	obj, err := svc.ApplyAuthorityEvent(ctx, id, "contradiction")
	if err != nil {
		t.Fatalf("ApplyAuthorityEvent: %v", err)
	}
	if obj.Authority != 4 {
		t.Errorf("ApplyAuthorityEvent: authority = %d, want 4", obj.Authority)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestService_ApplyAuthorityEvent_noLifecycleConfig(t *testing.T) {
	svc := &Service{Repo: &Repo{}, Lifecycle: nil}
	_, err := svc.ApplyAuthorityEvent(context.Background(), uuid.New(), "validation")
	if err == nil || err.Error() != "memory lifecycle not configured" {
		t.Errorf("expected lifecycle not configured: %v", err)
	}
}

func TestService_ApplyAuthorityEvent_notFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	id := uuid.New()
	mock.ExpectQuery(`SELECT id, kind, statement`).WithArgs(id).WillReturnError(sql.ErrNoRows)

	svc := &Service{Repo: &Repo{DB: db}, Lifecycle: &LifecycleConfig{AuthorityPositiveDelta: 0.1, AuthorityNegativeDelta: 0.2}}
	_, err = svc.ApplyAuthorityEvent(context.Background(), id, "validation")
	if err == nil || err.Error() != "memory not found" {
		t.Errorf("expected memory not found: %v", err)
	}
}

// Authority changes apply uniformly regardless of tags.
func TestService_ApplyAuthorityEvent_taggedGlobal_isDowngraded(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectQuery(`SELECT id, kind, statement`).WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at"}).
			AddRow(id, api.MemoryKindDecision, "Global rule", memorynorm.StatementCanonical("Global rule"), memorynorm.StatementKey("Global rule"), 5, "governing", "active", nil, nil, nil, time.Now(), time.Now()))
	mock.ExpectQuery(`SELECT tag FROM memories_tags`).WithArgs(id).WillReturnRows(
		sqlmock.NewRows([]string{"tag"}).AddRow("global"))
	mock.ExpectExec(`UPDATE memories SET authority`).WithArgs(4, id).WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &Service{Repo: &Repo{DB: db}, Lifecycle: &LifecycleConfig{AuthorityPositiveDelta: 0.1, AuthorityNegativeDelta: 0.2}}
	obj, err := svc.ApplyAuthorityEvent(ctx, id, "contradiction")
	if err != nil {
		t.Fatalf("ApplyAuthorityEvent: %v", err)
	}
	if obj.Authority != 4 {
		t.Errorf("authority should downgrade uniformly, got %d", obj.Authority)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestService_Create_doesNotInferProjectOrGlobalTags(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()

	repo := &Repo{DB: db}
	dedupOff := false
	svc := &Service{Repo: repo, Dedup: &DedupConfig{Enabled: &dedupOff}}

	statement := "Memory-first create must not infer tags"
	canon := memorynorm.StatementCanonical(statement)
	sk := memorynorm.StatementKey(statement)
	dedup := DedupKey()
	memID := uuid.New()

	mock.ExpectQuery(`INSERT INTO memories`).
		WithArgs(sqlmock.AnyArg(), api.MemoryKindDecision, statement, canon, sk, dedup, 7, api.ApplicabilityGoverning, "active", nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at",
		}).AddRow(memID, api.MemoryKindDecision, statement, canon, sk, 7, api.ApplicabilityGoverning, "active", nil, nil, nil, time.Now(), time.Now()))
	// No memories_tags inserts are expected: create path must not infer project:* or global tags.

	obj, err := svc.Create(ctx, CreateRequest{
		Kind:          api.MemoryKindDecision,
		Authority:     7,
		Applicability: api.ApplicabilityGoverning,
		Statement:     statement,
		Tags:          nil,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(obj.Tags) != 0 {
		t.Fatalf("expected no inferred tags, got %v", obj.Tags)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestService_ReinforceRecallUsage_incrementsAuthority(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

	mock.ExpectQuery(`SELECT id, kind, statement`).WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at"}).
			AddRow(id, api.MemoryKindPattern, "p", memorynorm.StatementCanonical("p"), memorynorm.StatementKey("p"), 5, "advisory", "active", nil, nil, nil, time.Now(), time.Now()))
	mock.ExpectQuery(`SELECT tag FROM memories_tags`).WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"tag"}).AddRow("api"))
	mock.ExpectExec(`UPDATE memories SET authority`).WithArgs(6, id).WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &Service{Repo: &Repo{DB: db}}
	if err := svc.ReinforceRecallUsage(ctx, []uuid.UUID{id, id, uuid.Nil}); err != nil {
		t.Fatalf("ReinforceRecallUsage: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestService_ReinforceRecallUsageWithMeta_highImpactUsesLargerDelta(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	id := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	mock.ExpectQuery(`SELECT id, kind, statement`).WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at"}).
			AddRow(id, api.MemoryKindPattern, "p", memorynorm.StatementCanonical("p"), memorynorm.StatementKey("p"), 5, "advisory", "active", nil, nil, nil, time.Now(), time.Now()))
	mock.ExpectQuery(`SELECT tag FROM memories_tags`).WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"tag"}))
	mock.ExpectExec(`UPDATE memories SET authority`).WithArgs(7, id).WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &Service{Repo: &Repo{DB: db}}
	if err := svc.ReinforceRecallUsageWithMeta(ctx, []uuid.UUID{id}, ReinforceMeta{Impact: "high", SignalStrength: 2}); err != nil {
		t.Fatalf("ReinforceRecallUsageWithMeta: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestService_ReinforceRecallUsageWithMeta_lowImpactSkipsBelowSignalThreshold(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	id := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")

	// No DB calls expected: low-impact with signal below threshold should no-op.
	svc := &Service{
		Repo: &Repo{DB: db},
		Reinforcement: &RecallReinforcementConfig{
			ImpactLowDelta:          1,
			MinSignalStrengthForLow: 2,
		},
	}
	if err := svc.ReinforceRecallUsageWithMeta(ctx, []uuid.UUID{id}, ReinforceMeta{Impact: "low", SignalStrength: 1}); err != nil {
		t.Fatalf("ReinforceRecallUsageWithMeta: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
