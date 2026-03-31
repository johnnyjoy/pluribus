package memory

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"control-plane/internal/memorynorm"
	"control-plane/pkg/api"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Repo performs memory persistence and search against memories / memories_tags.
type Repo struct {
	DB *sql.DB
}

// CountActiveFailuresWithStatementKey returns how many active or pending failure rows share the statement key.
func (r *Repo) CountActiveFailuresWithStatementKey(ctx context.Context, statementKey string) (int, error) {
	if statementKey == "" {
		return 0, nil
	}
	var n int
	err := r.DB.QueryRowContext(ctx,
		`SELECT COUNT(*)::int FROM memories
		 WHERE kind = 'failure' AND statement_key = $1 AND status IN ('active','pending')`,
		statementKey,
	).Scan(&n)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// FindActiveDuplicate returns an existing memory id when an active or pending row matches
// the dedup key (shared key + statement_key). Returns nil, nil when none.
func (r *Repo) FindActiveDuplicate(ctx context.Context, kind api.MemoryKind, statementKey string) (*uuid.UUID, error) {
	dedup := DedupKey()
	var id uuid.UUID
	err := r.DB.QueryRowContext(ctx,
		`SELECT id FROM memories
		 WHERE kind = $1 AND dedup_key = $2 AND statement_key = $3
		 AND status IN ('active', 'pending')
		 LIMIT 1`,
		string(kind), dedup, statementKey,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// Create inserts a memory row and its tags; returns the object with ID and timestamps.
func (r *Repo) Create(ctx context.Context, req CreateRequest) (*MemoryObject, error) {
	id := uuid.New()
	applicability := string(req.Applicability)
	if applicability == "" {
		applicability = "governing"
	}
	status := string(api.StatusActive)
	if req.Status != "" {
		status = string(req.Status)
	}
	var ttl *int
	if req.TTLSeconds > 0 {
		ttl = &req.TTLSeconds
	}
	var payloadArg interface{}
	if req.Payload != nil && len(*req.Payload) > 0 {
		payloadArg = []byte(*req.Payload)
	}
	dedup := DedupKey()
	canon := req.StatementCanonical
	if canon == "" && strings.TrimSpace(req.Statement) != "" {
		canon = memorynorm.StatementCanonical(req.Statement)
	}
	stmtKey := req.StatementKey
	if stmtKey == "" && canon != "" {
		stmtKey = memorynorm.StatementKey(req.Statement)
	}
	var occurredArg interface{}
	if req.OccurredAt != nil {
		occurredArg = *req.OccurredAt
	}
	var obj MemoryObject
	var deprecatedAt sql.NullTime
	var ttlReturn sql.NullInt64
	var payloadReturn []byte
	var occurredReturn sql.NullTime
	var err error
	if len(req.Embedding) > 0 {
		vec := FormatVectorLiteral(req.Embedding)
		err = r.DB.QueryRowContext(ctx,
			`INSERT INTO memories (id, kind, statement, statement_canonical, statement_key, dedup_key, authority, applicability, status, ttl_seconds, payload, occurred_at, embedding)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13::vector)
			 RETURNING id, kind, statement, statement_canonical, statement_key, authority, applicability, status, deprecated_at, ttl_seconds, payload, created_at, updated_at, occurred_at`,
			id, string(req.Kind), req.Statement, canon, stmtKey, dedup, req.Authority, applicability, status, ttl, payloadArg, occurredArg, vec,
		).Scan(&obj.ID, &obj.Kind, &obj.Statement, &obj.StatementCanonical, &obj.StatementKey, &obj.Authority, &obj.Applicability, &obj.Status, &deprecatedAt, &ttlReturn, &payloadReturn, &obj.CreatedAt, &obj.UpdatedAt, &occurredReturn)
	} else {
		err = r.DB.QueryRowContext(ctx,
			`INSERT INTO memories (id, kind, statement, statement_canonical, statement_key, dedup_key, authority, applicability, status, ttl_seconds, payload, occurred_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			 RETURNING id, kind, statement, statement_canonical, statement_key, authority, applicability, status, deprecated_at, ttl_seconds, payload, created_at, updated_at, occurred_at`,
			id, string(req.Kind), req.Statement, canon, stmtKey, dedup, req.Authority, applicability, status, ttl, payloadArg, occurredArg,
		).Scan(&obj.ID, &obj.Kind, &obj.Statement, &obj.StatementCanonical, &obj.StatementKey, &obj.Authority, &obj.Applicability, &obj.Status, &deprecatedAt, &ttlReturn, &payloadReturn, &obj.CreatedAt, &obj.UpdatedAt, &occurredReturn)
	}
	if err != nil {
		if isPGUniqueViolation(err) {
			dupID, findErr := r.FindActiveDuplicate(ctx, req.Kind, stmtKey)
			if findErr != nil {
				return nil, findErr
			}
			if dupID != nil {
				return nil, &ErrDuplicateMemory{ExistingID: *dupID}
			}
		}
		return nil, err
	}
	if deprecatedAt.Valid {
		obj.DeprecatedAt = &deprecatedAt.Time
	}
	if ttlReturn.Valid {
		t := int(ttlReturn.Int64)
		obj.TTLSeconds = &t
	}
	if len(payloadReturn) > 0 {
		obj.Payload = payloadReturn
	}
	applyOccurredAt(occurredReturn, &obj)
	tags := mergePersistTags(req)
	for _, tag := range tags {
		if _, err = r.DB.ExecContext(ctx, `INSERT INTO memories_tags (memory_id, tag) VALUES ($1, $2)`, id, tag); err != nil {
			return nil, err
		}
	}
	obj.Tags = tags
	enrichFromTags(&obj, tags)
	return &obj, nil
}

// GetByID returns a memory by ID, or nil if not found.
func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*MemoryObject, error) {
	var obj MemoryObject
	var deprecatedAt sql.NullTime
	var ttlReturn sql.NullInt64
	var payloadReturn []byte
	var occurredAt sql.NullTime
	err := r.DB.QueryRowContext(ctx,
		`SELECT id, kind, statement, statement_canonical, statement_key, authority, applicability, status, deprecated_at, ttl_seconds, payload, created_at, updated_at, occurred_at
		 FROM memories WHERE id = $1`,
		id,
	).Scan(&obj.ID, &obj.Kind, &obj.Statement, &obj.StatementCanonical, &obj.StatementKey, &obj.Authority, &obj.Applicability, &obj.Status, &deprecatedAt, &ttlReturn, &payloadReturn, &obj.CreatedAt, &obj.UpdatedAt, &occurredAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if deprecatedAt.Valid {
		obj.DeprecatedAt = &deprecatedAt.Time
	}
	if ttlReturn.Valid {
		t := int(ttlReturn.Int64)
		obj.TTLSeconds = &t
	}
	if len(payloadReturn) > 0 {
		obj.Payload = payloadReturn
	}
	applyOccurredAt(occurredAt, &obj)
	obj.Tags, _ = r.tagsForMemory(ctx, obj.ID)
	enrichFromTags(&obj, obj.Tags)
	return &obj, nil
}

// UpdateAuthority sets the authority field and updated_at for a memory.
func (r *Repo) UpdateAuthority(ctx context.Context, id uuid.UUID, authority int) error {
	_, err := r.DB.ExecContext(ctx,
		`UPDATE memories SET authority = $1, updated_at = now() WHERE id = $2`,
		authority, id)
	return err
}

// UpdatePayload sets payload JSON and updated_at for a memory.
func (r *Repo) UpdatePayload(ctx context.Context, id uuid.UUID, payload []byte) error {
	_, err := r.DB.ExecContext(ctx,
		`UPDATE memories SET payload = $1, updated_at = now() WHERE id = $2`,
		payload, id)
	return err
}

// MergeTagsIntoMemory inserts tags not already present (case-insensitive dedupe vs existing).
func (r *Repo) MergeTagsIntoMemory(ctx context.Context, memoryID uuid.UUID, newTags []string) error {
	if r == nil || r.DB == nil {
		return errors.New("memory repo: not configured")
	}
	existing, err := r.tagsForMemory(ctx, memoryID)
	if err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(existing))
	for _, t := range existing {
		seen[strings.ToLower(strings.TrimSpace(t))] = struct{}{}
	}
	for _, t := range newTags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		key := strings.ToLower(t)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if _, err := r.DB.ExecContext(ctx,
			`INSERT INTO memories_tags (memory_id, tag) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			memoryID, t,
		); err != nil {
			return err
		}
	}
	return nil
}

// MarkSuperseded sets status to superseded and deprecated_at to the given time (Task 75).
func (r *Repo) MarkSuperseded(ctx context.Context, id uuid.UUID, deprecatedAt time.Time) error {
	_, err := r.DB.ExecContext(ctx,
		`UPDATE memories SET status = 'superseded', deprecated_at = $1, updated_at = now() WHERE id = $2`,
		deprecatedAt, id)
	return err
}

// ListExpiredCandidates returns active memories with TTL set that have expired and authority below threshold (Task 75).
func (r *Repo) ListExpiredCandidates(ctx context.Context, authorityThreshold int, asOf time.Time) ([]MemoryObject, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT id, kind, statement, statement_canonical, statement_key, authority, applicability, status, deprecated_at, ttl_seconds, payload, created_at, updated_at, occurred_at
		 FROM memories
		 WHERE status = 'active' AND authority < $1 AND ttl_seconds IS NOT NULL AND ttl_seconds > 0
		   AND created_at + (ttl_seconds * interval '1 second') < $2`,
		authorityThreshold, asOf)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []MemoryObject
	for rows.Next() {
		var obj MemoryObject
		var deprecatedAt sql.NullTime
		var ttlReturn sql.NullInt64
		var payloadReturn []byte
		var occurredAt sql.NullTime
		if err := rows.Scan(&obj.ID, &obj.Kind, &obj.Statement, &obj.StatementCanonical, &obj.StatementKey, &obj.Authority, &obj.Applicability, &obj.Status, &deprecatedAt, &ttlReturn, &payloadReturn, &obj.CreatedAt, &obj.UpdatedAt, &occurredAt); err != nil {
			return nil, err
		}
		if deprecatedAt.Valid {
			obj.DeprecatedAt = &deprecatedAt.Time
		}
		if ttlReturn.Valid {
			t := int(ttlReturn.Int64)
			obj.TTLSeconds = &t
		}
		if len(payloadReturn) > 0 {
			obj.Payload = payloadReturn
		}
		applyOccurredAt(occurredAt, &obj)
		obj.Tags, _ = r.tagsForMemory(ctx, obj.ID)
		enrichFromTags(&obj, obj.Tags)
		list = append(list, obj)
	}
	return list, rows.Err()
}

// UpdateStatus sets status and updated_at for a memory.
func (r *Repo) UpdateStatus(ctx context.Context, id uuid.UUID, status api.Status) error {
	_, err := r.DB.ExecContext(ctx,
		`UPDATE memories SET status = $1, updated_at = now() WHERE id = $2`,
		status, id)
	return err
}

// Search lists memories from the shared pool; Tags, Status, Max, and optional Kinds apply.
func (r *Repo) Search(ctx context.Context, req SearchRequest) ([]MemoryObject, error) {
	q := req
	if q.Status == "" {
		q.Status = "active"
	}
	if q.Max <= 0 {
		q.Max = 20
	}
	return r.SearchUnscoped(ctx, q)
}

// SearchUnscoped lists memories without project/global SQL filtering (tag filter optional).
func (r *Repo) SearchUnscoped(ctx context.Context, req SearchRequest) ([]MemoryObject, error) {
	status := req.Status
	if status == "" {
		status = "active"
	}
	max := req.Max
	if max <= 0 {
		max = 20
	}
	tagList := filterEmptyStringSlice(req.Tags)
	var kindStrs []string
	for _, k := range req.Kinds {
		if k != "" {
			kindStrs = append(kindStrs, string(k))
		}
	}
	var rows *sql.Rows
	var err error
	switch {
	case len(tagList) > 0 && len(kindStrs) > 0:
		rows, err = r.DB.QueryContext(ctx,
			`SELECT m.id, m.kind, m.statement, m.statement_canonical, m.statement_key, m.authority, m.applicability, m.status, m.payload, m.created_at, m.updated_at, m.occurred_at
			 FROM memories m
			 WHERE m.status = $1
			   AND m.kind = ANY($2)
			   AND EXISTS (SELECT 1 FROM memories_tags t WHERE t.memory_id = m.id AND t.tag = ANY($3))
			 ORDER BY m.authority DESC, m.updated_at DESC, m.id
			 LIMIT $4`,
			status, pq.Array(kindStrs), pq.Array(tagList), max)
	case len(tagList) > 0:
		rows, err = r.DB.QueryContext(ctx,
			`SELECT m.id, m.kind, m.statement, m.statement_canonical, m.statement_key, m.authority, m.applicability, m.status, m.payload, m.created_at, m.updated_at, m.occurred_at
			 FROM memories m
			 WHERE m.status = $1
			   AND EXISTS (SELECT 1 FROM memories_tags t WHERE t.memory_id = m.id AND t.tag = ANY($2))
			 ORDER BY m.authority DESC, m.updated_at DESC, m.id
			 LIMIT $3`,
			status, pq.Array(tagList), max)
	case len(kindStrs) > 0:
		rows, err = r.DB.QueryContext(ctx,
			`SELECT id, kind, statement, statement_canonical, statement_key, authority, applicability, status, payload, created_at, updated_at, occurred_at
			 FROM memories
			 WHERE status = $1 AND kind = ANY($2)
			 ORDER BY authority DESC, updated_at DESC, id
			 LIMIT $3`,
			status, pq.Array(kindStrs), max)
	default:
		rows, err = r.DB.QueryContext(ctx,
			`SELECT id, kind, statement, statement_canonical, statement_key, authority, applicability, status, payload, created_at, updated_at, occurred_at
			 FROM memories
			 WHERE status = $1
			 ORDER BY authority DESC, updated_at DESC, id
			 LIMIT $2`,
			status, max)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []MemoryObject
	var ids []uuid.UUID
	for rows.Next() {
		var obj MemoryObject
		var payloadReturn []byte
		var occurredAt sql.NullTime
		if err := rows.Scan(&obj.ID, &obj.Kind, &obj.Statement, &obj.StatementCanonical, &obj.StatementKey, &obj.Authority, &obj.Applicability, &obj.Status, &payloadReturn, &obj.CreatedAt, &obj.UpdatedAt, &occurredAt); err != nil {
			return nil, err
		}
		if len(payloadReturn) > 0 {
			obj.Payload = payloadReturn
		}
		applyOccurredAt(occurredAt, &obj)
		ids = append(ids, obj.ID)
		list = append(list, obj)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	tagMap, err := r.tagsForMemories(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range list {
		list[i].Tags = tagMap[list[i].ID]
		enrichFromTags(&list[i], list[i].Tags)
	}
	return list, nil
}

// SearchSimilar returns active memories with non-null embeddings, ordered by cosine distance to queryEmbedding.
// maxCosineDistance is the maximum allowed pgvector cosine distance (<=>); rows with distance above this are excluded.
// The returned map gives approximate cosine similarity 1 - distance per id (for hybrid ranking).
func (r *Repo) SearchSimilar(ctx context.Context, queryEmbedding []float32, req SearchRequest, limit int, maxCosineDistance float64) ([]MemoryObject, map[uuid.UUID]float64, error) {
	if r == nil || r.DB == nil {
		return nil, nil, errors.New("memory repo not configured")
	}
	if len(queryEmbedding) == 0 {
		return nil, nil, errors.New("empty query embedding")
	}
	if limit <= 0 {
		limit = 20
	}
	if maxCosineDistance <= 0 {
		maxCosineDistance = 0.65
	}
	status := req.Status
	if status == "" {
		status = "active"
	}
	vec := FormatVectorLiteral(queryEmbedding)
	tagList := filterEmptyStringSlice(req.Tags)
	var kindStrs []string
	for _, k := range req.Kinds {
		if k != "" {
			kindStrs = append(kindStrs, string(k))
		}
	}
	var rows *sql.Rows
	var err error
	switch {
	case len(tagList) > 0 && len(kindStrs) > 0:
		rows, err = r.DB.QueryContext(ctx,
			`SELECT m.id, m.kind, m.statement, m.statement_canonical, m.statement_key, m.authority, m.applicability, m.status, m.payload, m.created_at, m.updated_at, m.occurred_at,
				(m.embedding <=> $1::vector) AS vec_dist
			 FROM memories m
			 WHERE m.status = $2
			   AND m.embedding IS NOT NULL
			   AND m.kind = ANY($3)
			   AND (m.embedding <=> $1::vector) <= $4
			   AND EXISTS (SELECT 1 FROM memories_tags t WHERE t.memory_id = m.id AND t.tag = ANY($5))
			 ORDER BY vec_dist ASC, m.id
			 LIMIT $6`,
			vec, status, pq.Array(kindStrs), maxCosineDistance, pq.Array(tagList), limit)
	case len(tagList) > 0:
		rows, err = r.DB.QueryContext(ctx,
			`SELECT m.id, m.kind, m.statement, m.statement_canonical, m.statement_key, m.authority, m.applicability, m.status, m.payload, m.created_at, m.updated_at, m.occurred_at,
				(m.embedding <=> $1::vector) AS vec_dist
			 FROM memories m
			 WHERE m.status = $2
			   AND m.embedding IS NOT NULL
			   AND (m.embedding <=> $1::vector) <= $3
			   AND EXISTS (SELECT 1 FROM memories_tags t WHERE t.memory_id = m.id AND t.tag = ANY($4))
			 ORDER BY vec_dist ASC, m.id
			 LIMIT $5`,
			vec, status, maxCosineDistance, pq.Array(tagList), limit)
	case len(kindStrs) > 0:
		rows, err = r.DB.QueryContext(ctx,
			`SELECT id, kind, statement, statement_canonical, statement_key, authority, applicability, status, payload, created_at, updated_at, occurred_at,
				(embedding <=> $1::vector) AS vec_dist
			 FROM memories
			 WHERE status = $2
			   AND embedding IS NOT NULL
			   AND kind = ANY($3)
			   AND (embedding <=> $1::vector) <= $4
			 ORDER BY vec_dist ASC, id
			 LIMIT $5`,
			vec, status, pq.Array(kindStrs), maxCosineDistance, limit)
	default:
		rows, err = r.DB.QueryContext(ctx,
			`SELECT id, kind, statement, statement_canonical, statement_key, authority, applicability, status, payload, created_at, updated_at, occurred_at,
				(embedding <=> $1::vector) AS vec_dist
			 FROM memories
			 WHERE status = $2
			   AND embedding IS NOT NULL
			   AND (embedding <=> $1::vector) <= $3
			 ORDER BY vec_dist ASC, id
			 LIMIT $4`,
			vec, status, maxCosineDistance, limit)
	}
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var list []MemoryObject
	var ids []uuid.UUID
	simByID := make(map[uuid.UUID]float64)
	for rows.Next() {
		var obj MemoryObject
		var payloadReturn []byte
		var occurredAt sql.NullTime
		var vecDist float64
		if err := rows.Scan(&obj.ID, &obj.Kind, &obj.Statement, &obj.StatementCanonical, &obj.StatementKey, &obj.Authority, &obj.Applicability, &obj.Status, &payloadReturn, &obj.CreatedAt, &obj.UpdatedAt, &occurredAt, &vecDist); err != nil {
			return nil, nil, err
		}
		if len(payloadReturn) > 0 {
			obj.Payload = payloadReturn
		}
		applyOccurredAt(occurredAt, &obj)
		sim := 1.0 - vecDist
		if sim < 0 {
			sim = 0
		}
		if sim > 1 {
			sim = 1
		}
		simByID[obj.ID] = sim
		ids = append(ids, obj.ID)
		list = append(list, obj)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	tagMap, err := r.tagsForMemories(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	for i := range list {
		list[i].Tags = tagMap[list[i].ID]
		enrichFromTags(&list[i], list[i].Tags)
	}
	return list, simByID, nil
}

// SearchTagOnly searches by optional text query and tags (no project/global filter). Used by POST /v1/memories/search.
func (r *Repo) SearchTagOnly(ctx context.Context, query string, tags []string, status string, max int) ([]MemoryObject, error) {
	if status == "" {
		status = "active"
	}
	if max <= 0 {
		max = 20
	}
	if max > 200 {
		max = 200
	}
	q := strings.TrimSpace(query)
	tagList := filterEmptyStringSlice(tags)

	var rows *sql.Rows
	var err error
	switch {
	case len(tagList) > 0 && q != "":
		rows, err = r.DB.QueryContext(ctx,
			`SELECT m.id, m.kind, m.statement, m.statement_canonical, m.statement_key, m.authority, m.applicability, m.status, m.payload, m.created_at, m.updated_at, m.occurred_at
			 FROM memories m
			 WHERE m.status = $1
			   AND m.statement ILIKE '%' || $2 || '%'
			   AND EXISTS (SELECT 1 FROM memories_tags t WHERE t.memory_id = m.id AND t.tag = ANY($3))
			 ORDER BY m.authority DESC, m.updated_at DESC, m.id
			 LIMIT $4`,
			status, q, pq.Array(tagList), max)
	case len(tagList) > 0:
		rows, err = r.DB.QueryContext(ctx,
			`SELECT m.id, m.kind, m.statement, m.statement_canonical, m.statement_key, m.authority, m.applicability, m.status, m.payload, m.created_at, m.updated_at, m.occurred_at
			 FROM memories m
			 WHERE m.status = $1
			   AND EXISTS (SELECT 1 FROM memories_tags t WHERE t.memory_id = m.id AND t.tag = ANY($2))
			 ORDER BY m.authority DESC, m.updated_at DESC, m.id
			 LIMIT $3`,
			status, pq.Array(tagList), max)
	case q != "":
		rows, err = r.DB.QueryContext(ctx,
			`SELECT id, kind, statement, statement_canonical, statement_key, authority, applicability, status, payload, created_at, updated_at, occurred_at
			 FROM memories
			 WHERE status = $1 AND statement ILIKE '%' || $2 || '%'
			 ORDER BY authority DESC, updated_at DESC, id
			 LIMIT $3`,
			status, q, max)
	default:
		rows, err = r.DB.QueryContext(ctx,
			`SELECT id, kind, statement, statement_canonical, statement_key, authority, applicability, status, payload, created_at, updated_at, occurred_at
			 FROM memories
			 WHERE status = $1
			 ORDER BY authority DESC, updated_at DESC, id
			 LIMIT $2`,
			status, max)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []MemoryObject
	var ids []uuid.UUID
	for rows.Next() {
		var obj MemoryObject
		var payloadReturn []byte
		var occurredAt sql.NullTime
		if err := rows.Scan(&obj.ID, &obj.Kind, &obj.Statement, &obj.StatementCanonical, &obj.StatementKey, &obj.Authority, &obj.Applicability, &obj.Status, &payloadReturn, &obj.CreatedAt, &obj.UpdatedAt, &occurredAt); err != nil {
			return nil, err
		}
		if len(payloadReturn) > 0 {
			obj.Payload = payloadReturn
		}
		applyOccurredAt(occurredAt, &obj)
		ids = append(ids, obj.ID)
		list = append(list, obj)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	tagMap, err := r.tagsForMemories(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range list {
		list[i].Tags = tagMap[list[i].ID]
		enrichFromTags(&list[i], list[i].Tags)
	}
	return list, nil
}

func filterEmptyStringSlice(tags []string) []string {
	var out []string
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// ListBindingMemory returns active memories that may bind for pre-change enforcement.
// Memory-first rule: binding lookup is not project-partitioned.
func (r *Repo) ListBindingMemory(ctx context.Context, req ListBindingRequest) ([]MemoryObject, error) {
	kinds := req.Kinds
	if len(kinds) == 0 {
		kinds = []api.MemoryKind{
			api.MemoryKindConstraint,
			api.MemoryKindDecision,
			api.MemoryKindFailure,
			api.MemoryKindPattern,
		}
	}
	kindStrs := make([]string, len(kinds))
	for i, k := range kinds {
		kindStrs[i] = string(k)
	}
	max := req.Max
	if max <= 0 {
		max = 120
	}
	rows, err := r.DB.QueryContext(ctx,
		`SELECT m.id, m.kind, m.statement, m.statement_canonical, m.statement_key, m.authority, m.applicability, m.status, m.payload, m.created_at, m.updated_at, m.occurred_at
		 FROM memories m
		 WHERE m.status = 'active'
		   AND m.authority >= $1
		   AND m.kind = ANY($2)
		   AND m.applicability != 'advisory'
		 ORDER BY m.authority DESC
		 LIMIT $3`,
		req.MinAuthority, pq.Array(kindStrs), max)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []MemoryObject
	var ids []uuid.UUID
	for rows.Next() {
		var obj MemoryObject
		var payloadReturn []byte
		var occurredAt sql.NullTime
		if err := rows.Scan(&obj.ID, &obj.Kind, &obj.Statement, &obj.StatementCanonical, &obj.StatementKey, &obj.Authority, &obj.Applicability, &obj.Status, &payloadReturn, &obj.CreatedAt, &obj.UpdatedAt, &occurredAt); err != nil {
			return nil, err
		}
		if len(payloadReturn) > 0 {
			obj.Payload = payloadReturn
		}
		applyOccurredAt(occurredAt, &obj)
		ids = append(ids, obj.ID)
		list = append(list, obj)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	tagMap, err := r.tagsForMemories(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range list {
		list[i].Tags = tagMap[list[i].ID]
		enrichFromTags(&list[i], list[i].Tags)
	}
	return list, nil
}

func (r *Repo) tagsForMemories(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID][]string, error) {
	out := make(map[uuid.UUID][]string, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.DB.QueryContext(ctx,
		`SELECT memory_id, tag FROM memories_tags WHERE memory_id = ANY($1) ORDER BY memory_id, tag`,
		pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var mid uuid.UUID
		var tag string
		if err := rows.Scan(&mid, &tag); err != nil {
			return nil, err
		}
		out[mid] = append(out[mid], tag)
	}
	return out, rows.Err()
}

func (r *Repo) tagsForMemory(ctx context.Context, memoryID uuid.UUID) ([]string, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT tag FROM memories_tags WHERE memory_id = $1 ORDER BY tag`, memoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

// GetAttributes returns attribute key-value pairs for a memory (Task 78: conflict detection).
func (r *Repo) GetAttributes(ctx context.Context, memoryID uuid.UUID) (map[string]string, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT attr_key, attr_value FROM memory_attributes WHERE memory_id = $1`, memoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		m[k] = v
	}
	return m, rows.Err()
}

// SetAttribute upserts a single attribute for a memory.
func (r *Repo) SetAttribute(ctx context.Context, memoryID uuid.UUID, key, value string) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO memory_attributes (memory_id, attr_key, attr_value) VALUES ($1, $2, $3)
		 ON CONFLICT (memory_id, attr_key) DO UPDATE SET attr_value = $3`,
		memoryID, key, value)
	return err
}

// ReplaceAttributes replaces all attributes for a memory (deletes existing, inserts attrs). Empty map clears.
func (r *Repo) ReplaceAttributes(ctx context.Context, memoryID uuid.UUID, attrs map[string]string) error {
	if _, err := r.DB.ExecContext(ctx, `DELETE FROM memory_attributes WHERE memory_id = $1`, memoryID); err != nil {
		return err
	}
	for k, v := range attrs {
		if err := r.SetAttribute(ctx, memoryID, k, v); err != nil {
			return err
		}
	}
	return nil
}

func applyOccurredAt(nt sql.NullTime, obj *MemoryObject) {
	if nt.Valid {
		t := nt.Time
		obj.OccurredAt = &t
	}
}

func isPGUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return false
	}
	return pqErr.Code == "23505"
}
