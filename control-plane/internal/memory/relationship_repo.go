package memory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// RelationshipType is a typed edge between two memories (global pool; not a container).
type RelationshipType string

const (
	RelSupports           RelationshipType = "supports"
	RelContradicts        RelationshipType = "contradicts"
	RelSupersedes         RelationshipType = "supersedes"
	RelSamePatternFamily  RelationshipType = "same_pattern_family"
	RelDerivedFrom        RelationshipType = "derived_from"
)

var allowedRelationshipTypes = map[RelationshipType]struct{}{
	RelSupports:          {},
	RelContradicts:       {},
	RelSupersedes:        {},
	RelSamePatternFamily: {},
	RelDerivedFrom:       {},
}

// MemoryRelationship is one durable row in memory_relationships.
type MemoryRelationship struct {
	ID                uuid.UUID        `json:"id"`
	FromMemoryID      uuid.UUID        `json:"from_memory_id"`
	ToMemoryID        uuid.UUID        `json:"to_memory_id"`
	RelationshipType  RelationshipType `json:"relationship_type"`
	Reason            string           `json:"reason,omitempty"`
	Source            string           `json:"source,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
}

// RelationshipRepo persists typed memory-to-memory edges.
type RelationshipRepo struct {
	DB *sql.DB
}

// CreateRelationship inserts an edge (idempotent on duplicate natural key).
func (r *RelationshipRepo) CreateRelationship(ctx context.Context, from, to uuid.UUID, typ RelationshipType, reason, source string) (*MemoryRelationship, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("memory relationships: repo not configured")
	}
	if from == to {
		return nil, fmt.Errorf("memory relationships: from and to must differ")
	}
	if _, ok := allowedRelationshipTypes[typ]; !ok {
		return nil, fmt.Errorf("memory relationships: invalid relationship_type %q", typ)
	}
	var n int
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*)::int FROM memories WHERE id IN ($1, $2)`, from, to).Scan(&n); err != nil {
		return nil, err
	}
	if n != 2 {
		return nil, fmt.Errorf("memory relationships: from or to memory not found")
	}
	reason = strings.TrimSpace(reason)
	source = strings.TrimSpace(source)

	var out MemoryRelationship
	err := r.DB.QueryRowContext(ctx, `
		INSERT INTO memory_relationships (from_memory_id, to_memory_id, relationship_type, reason, source)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''))
		ON CONFLICT (from_memory_id, to_memory_id, relationship_type) DO UPDATE SET
			reason = COALESCE(EXCLUDED.reason, memory_relationships.reason),
			source = COALESCE(EXCLUDED.source, memory_relationships.source)
		RETURNING id, from_memory_id, to_memory_id, relationship_type, COALESCE(reason, ''), COALESCE(source, ''), created_at`,
		from, to, string(typ), reason, source,
	).Scan(&out.ID, &out.FromMemoryID, &out.ToMemoryID, &out.RelationshipType, &out.Reason, &out.Source, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListForMemory returns outbound and inbound edges for a memory id.
func (r *RelationshipRepo) ListForMemory(ctx context.Context, memoryID uuid.UUID) (outbound, inbound []MemoryRelationship, err error) {
	if r == nil || r.DB == nil {
		return nil, nil, errors.New("memory relationships: repo not configured")
	}
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, from_memory_id, to_memory_id, relationship_type, COALESCE(reason, ''), COALESCE(source, ''), created_at
		FROM memory_relationships WHERE from_memory_id = $1 ORDER BY created_at ASC`, memoryID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var m MemoryRelationship
		if err := rows.Scan(&m.ID, &m.FromMemoryID, &m.ToMemoryID, &m.RelationshipType, &m.Reason, &m.Source, &m.CreatedAt); err != nil {
			return nil, nil, err
		}
		outbound = append(outbound, m)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	rows2, err := r.DB.QueryContext(ctx, `
		SELECT id, from_memory_id, to_memory_id, relationship_type, COALESCE(reason, ''), COALESCE(source, ''), created_at
		FROM memory_relationships WHERE to_memory_id = $1 ORDER BY created_at ASC`, memoryID)
	if err != nil {
		return nil, nil, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var m MemoryRelationship
		if err := rows2.Scan(&m.ID, &m.FromMemoryID, &m.ToMemoryID, &m.RelationshipType, &m.Reason, &m.Source, &m.CreatedAt); err != nil {
			return nil, nil, err
		}
		inbound = append(inbound, m)
	}
	if err := rows2.Err(); err != nil {
		return nil, nil, err
	}
	return outbound, inbound, nil
}

// LoadSupersedesMap returns superseded_memory_id -> superseding_memory_id for rows whose
// superseded id appears in ids (relationship_type = supersedes: from supersedes to).
func (r *RelationshipRepo) LoadSupersedesMap(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	if r == nil || r.DB == nil || len(ids) == 0 {
		return map[uuid.UUID]uuid.UUID{}, nil
	}
	rows, err := r.DB.QueryContext(ctx, `
		SELECT to_memory_id, from_memory_id FROM memory_relationships
		WHERE relationship_type = 'supersedes' AND to_memory_id = ANY($1)`, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[uuid.UUID]uuid.UUID)
	for rows.Next() {
		var oldID, newID uuid.UUID
		if err := rows.Scan(&oldID, &newID); err != nil {
			return nil, err
		}
		out[oldID] = newID
	}
	return out, rows.Err()
}
