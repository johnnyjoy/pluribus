package chores

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Repo persists chores and votes.
type Repo struct {
	DB *sql.DB
}

const choreColumns = `c.id, c.chore_type, c.subject_memory_id, c.related_memory_id, c.evidence,
	c.state, COALESCE(c.resolution_action, ''), c.created_at, c.resolved_at`

func scanChore(scan func(dest ...any) error) (*Chore, error) {
	var ch Chore
	var related uuid.NullUUID
	var evidence []byte
	var resolvedAt sql.NullTime
	if err := scan(&ch.ID, &ch.Type, &ch.SubjectMemoryID, &related, &evidence,
		&ch.State, &ch.ResolutionAction, &ch.CreatedAt, &resolvedAt); err != nil {
		return nil, err
	}
	if related.Valid {
		ch.RelatedMemoryID = &related.UUID
	}
	if len(evidence) > 0 {
		ch.Evidence = json.RawMessage(evidence)
	}
	if resolvedAt.Valid {
		t := resolvedAt.Time
		ch.ResolvedAt = &t
	}
	ch.Actions = AllowedActions[ch.Type]
	return &ch, nil
}

// EnsureChore inserts an open chore unless one already exists for the same
// (type, subject, related) triple in ANY state — resolved/dismissed pairs are
// never re-offered. Returns true when a new chore was created.
func (r *Repo) EnsureChore(ctx context.Context, choreType string, subject uuid.UUID, related *uuid.UUID, evidence any) (bool, error) {
	var evJSON []byte
	if evidence != nil {
		b, err := json.Marshal(evidence)
		if err != nil {
			return false, err
		}
		evJSON = b
	}
	var relatedVal any
	relatedKey := uuid.Nil
	if related != nil {
		relatedVal = *related
		relatedKey = *related
	}
	res, err := r.DB.ExecContext(ctx,
		`INSERT INTO curation_chores (id, chore_type, subject_memory_id, related_memory_id, evidence)
		 SELECT $1, $2, $3, $4, $5
		 WHERE NOT EXISTS (
		   SELECT 1 FROM curation_chores
		   WHERE chore_type = $2 AND subject_memory_id = $3
		     AND COALESCE(related_memory_id, '00000000-0000-0000-0000-000000000000'::uuid) = $6
		 )`,
		uuid.New(), choreType, subject, relatedVal, evJSON, relatedKey)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Get returns a chore by id, or nil when absent.
func (r *Repo) Get(ctx context.Context, id uuid.UUID) (*Chore, error) {
	row := r.DB.QueryRowContext(ctx,
		`SELECT `+choreColumns+` FROM curation_chores c WHERE c.id = $1`, id)
	ch, err := scanChore(row.Scan)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return ch, nil
}

// ListOpen returns open chores oldest-first with statement snippets and vote counts.
func (r *Repo) ListOpen(ctx context.Context, limit int) ([]Chore, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.DB.QueryContext(ctx,
		`SELECT `+choreColumns+`,
			COALESCE(ms.statement, ''), COALESCE(mr.statement, ''),
			(SELECT COUNT(*) FROM curation_chore_votes v WHERE v.chore_id = c.id)
		 FROM curation_chores c
		 JOIN memories ms ON ms.id = c.subject_memory_id
		 LEFT JOIN memories mr ON mr.id = c.related_memory_id
		 WHERE c.state = 'open'
		 ORDER BY c.created_at ASC, c.id
		 LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Chore
	for rows.Next() {
		var ch Chore
		var related uuid.NullUUID
		var evidence []byte
		var resolvedAt sql.NullTime
		if err := rows.Scan(&ch.ID, &ch.Type, &ch.SubjectMemoryID, &related, &evidence,
			&ch.State, &ch.ResolutionAction, &ch.CreatedAt, &resolvedAt,
			&ch.SubjectStatement, &ch.RelatedStatement, &ch.VoteCount); err != nil {
			return nil, err
		}
		if related.Valid {
			ch.RelatedMemoryID = &related.UUID
		}
		if len(evidence) > 0 {
			ch.Evidence = json.RawMessage(evidence)
		}
		if resolvedAt.Valid {
			t := resolvedAt.Time
			ch.ResolvedAt = &t
		}
		ch.Actions = AllowedActions[ch.Type]
		out = append(out, ch)
	}
	return out, rows.Err()
}

// InsertVote records a vote. Returns false when this agent already voted on
// the chore (one vote per agent hash; votes are immutable for auditability).
func (r *Repo) InsertVote(ctx context.Context, choreID uuid.UUID, agentID, agentHash, action, reason string) (bool, error) {
	res, err := r.DB.ExecContext(ctx,
		`INSERT INTO curation_chore_votes (id, chore_id, agent_id, agent_hash, action, reason)
		 VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''))
		 ON CONFLICT (chore_id, agent_hash) DO NOTHING`,
		uuid.New(), choreID, agentID, agentHash, action, reason)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// CountVotesForAction counts distinct agent hashes that voted for action,
// excluding the given hashes (memory authors never count toward the threshold).
func (r *Repo) CountVotesForAction(ctx context.Context, choreID uuid.UUID, action string, excludeHashes []string) (int, error) {
	if excludeHashes == nil {
		excludeHashes = []string{}
	}
	var n int
	err := r.DB.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT agent_hash) FROM curation_chore_votes
		 WHERE chore_id = $1 AND action = $2 AND NOT (agent_hash = ANY($3))`,
		choreID, action, pq.Array(excludeHashes)).Scan(&n)
	return n, err
}

// MarkResolved closes a chore with the winning action and final state.
func (r *Repo) MarkResolved(ctx context.Context, choreID uuid.UUID, action, state string, at time.Time) error {
	_, err := r.DB.ExecContext(ctx,
		`UPDATE curation_chores SET state = $1, resolution_action = $2, resolved_at = $3
		 WHERE id = $4 AND state = 'open'`,
		state, action, at, choreID)
	return err
}
