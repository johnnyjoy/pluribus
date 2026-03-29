package curation

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
)

// Repo performs candidate event persistence.
type Repo struct {
	DB *sql.DB
}

// Create inserts a candidate event; returns it with ID and CreatedAt set.
func (r *Repo) Create(ctx context.Context, rawText string, salienceScore float64) (*CandidateEvent, error) {
	id := uuid.New()
	var c CandidateEvent
	err := r.DB.QueryRowContext(ctx,
		`INSERT INTO candidate_events (id, raw_text, salience_score, promotion_status)
		 VALUES ($1, $2, $3, 'pending')
		 RETURNING id, raw_text, salience_score, promotion_status, created_at`,
		id, rawText, salienceScore,
	).Scan(&c.ID, &c.RawText, &c.SalienceScore, &c.PromotionStatus, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// CreateDigest inserts a candidate with structured proposal_json.
func (r *Repo) CreateDigest(ctx context.Context, rawText string, salienceScore float64, proposalJSON []byte) (*CandidateEvent, error) {
	id := uuid.New()
	var c CandidateEvent
	var pj []byte
	err := r.DB.QueryRowContext(ctx,
		`INSERT INTO candidate_events (id, raw_text, salience_score, promotion_status, proposal_json)
		 VALUES ($1, $2, $3, 'pending', $4)
		 RETURNING id, raw_text, salience_score, promotion_status, created_at, proposal_json`,
		id, rawText, salienceScore, proposalJSON,
	).Scan(&c.ID, &c.RawText, &c.SalienceScore, &c.PromotionStatus, &c.CreatedAt, &pj)
	if err != nil {
		return nil, err
	}
	if len(pj) > 0 {
		c.ProposalJSON = pj
	}
	enrichPreview(&c)
	return &c, nil
}

func enrichPreview(c *CandidateEvent) {
	if len(c.ProposalJSON) == 0 {
		return
	}
	var p ProposalPayloadV1
	if err := json.Unmarshal(c.ProposalJSON, &p); err != nil {
		return
	}
	c.StructuredKind = string(p.Kind)
	if p.Statement != "" {
		s := p.Statement
		if len(s) > 200 {
			s = s[:200] + "…"
		}
		c.StatementPreview = s
	}
}

// ListPending returns candidate events with promotion_status = 'pending' (global queue).
func (r *Repo) ListPending(ctx context.Context) ([]CandidateEvent, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT id, raw_text, COALESCE(salience_score, 0), promotion_status, created_at, proposal_json
		 FROM candidate_events WHERE promotion_status = 'pending' ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CandidateEvent
	for rows.Next() {
		var c CandidateEvent
		var pj []byte
		if err := rows.Scan(&c.ID, &c.RawText, &c.SalienceScore, &c.PromotionStatus, &c.CreatedAt, &pj); err != nil {
			return nil, err
		}
		if len(pj) > 0 {
			c.ProposalJSON = pj
		}
		enrichPreview(&c)
		out = append(out, c)
	}
	return out, rows.Err()
}

// UpdatePromotionStatus sets promotion_status for the candidate (e.g. "promoted", "rejected").
func (r *Repo) UpdatePromotionStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE candidate_events SET promotion_status = $1 WHERE id = $2`, status, id)
	return err
}

// GetByID returns a candidate by ID, or nil if not found.
func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*CandidateEvent, error) {
	var c CandidateEvent
	var pj []byte
	err := r.DB.QueryRowContext(ctx,
		`SELECT id, raw_text, COALESCE(salience_score, 0), promotion_status, created_at, proposal_json
		 FROM candidate_events WHERE id = $1`,
		id,
	).Scan(&c.ID, &c.RawText, &c.SalienceScore, &c.PromotionStatus, &c.CreatedAt, &pj)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(pj) > 0 {
		c.ProposalJSON = pj
	}
	enrichPreview(&c)
	return &c, nil
}
