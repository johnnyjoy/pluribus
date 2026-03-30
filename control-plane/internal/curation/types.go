package curation

import (
	"encoding/json"
	"time"

	"control-plane/pkg/api"

	"github.com/google/uuid"
)

// CandidateEvent is a possible future memory (pending promotion/rejection).
type CandidateEvent struct {
	ID              uuid.UUID       `json:"id"`
	RawText         string          `json:"raw_text"`
	SalienceScore   float64         `json:"salience_score,omitempty"`
	PromotionStatus string          `json:"promotion_status"` // "pending", "promoted", "rejected"
	CreatedAt       time.Time       `json:"created_at"`
	ProposalJSON    json.RawMessage `json:"proposal_json,omitempty"`
	// StructuredKind and StatementPreview are set when proposal_json is present (for list UX).
	StructuredKind   string `json:"structured_kind,omitempty"`
	StatementPreview string `json:"statement_preview,omitempty"`
	// PromotionReadiness / ReadinessReason are derived for review (deterministic; not auto-promotion).
	PromotionReadiness string `json:"promotion_readiness,omitempty"`
	ReadinessReason    string `json:"readiness_reason,omitempty"`
}

// EvaluateRequest is the payload for POST /curation/evaluate.
type EvaluateRequest struct {
	Text string `json:"text"`
}

// EvaluateResult is the response from Evaluate (salience and threshold flags).
type EvaluateResult struct {
	SalienceScore float64 `json:"salience_score"`
	ShouldReview  bool    `json:"should_review"`  // score >= review_threshold
	ShouldPromote bool    `json:"should_promote"` // score >= promote_threshold
	Created       bool    `json:"created"`        // candidate created (score >= candidate_threshold)
	CandidateID   string  `json:"candidate_id,omitempty"`
}

// SalienceConfig holds thresholds for candidate/review/promote (from app config).
type SalienceConfig struct {
	CandidateThreshold float64 // min score to create a candidate
	ReviewThreshold    float64 // min score for should_review
	PromoteThreshold   float64 // min score for should_promote
}

// DigestCurationAnswers optional structured answers for POST /v1/curation/digest.
type DigestCurationAnswers struct {
	WhatChanged string `json:"what_changed,omitempty"`
	WhatLearned string `json:"what_learned,omitempty"`
	Decision    string `json:"decision,omitempty"`
	Constraint  string `json:"constraint,omitempty"`
	Failure     string `json:"failure,omitempty"`
	Pattern     string `json:"pattern,omitempty"`
	NeverAgain  string `json:"never_again,omitempty"`
}

// ArtifactRef is an opaque reference stored in proposal_json (v1: not resolved server-side).
type ArtifactRef struct {
	Kind string `json:"kind"`
	Ref  string `json:"ref"`
}

// DigestOptions tunes digest behavior.
type DigestOptions struct {
	MaxProposals int  `json:"max_proposals,omitempty"`
	DryRun       bool `json:"dry_run,omitempty"`
}

// DigestRequest is the payload for POST /v1/curation/digest.
type DigestRequest struct {
	WorkSummary     string                 `json:"work_summary"`
	Signals         []string               `json:"signals,omitempty"`
	CurationAnswers *DigestCurationAnswers `json:"curation_answers,omitempty"`
	EvidenceIDs     []uuid.UUID            `json:"evidence_ids,omitempty"`
	ArtifactRefs    []ArtifactRef          `json:"artifact_refs,omitempty"`
	Options         *DigestOptions         `json:"options,omitempty"`
}

// DigestProposal is one structured proposal returned from digest (and stored per candidate row when not dry_run).
type DigestProposal struct {
	ProposalID        string         `json:"proposal_id"`
	Kind              api.MemoryKind `json:"kind"`
	Statement         string         `json:"statement"`
	Reason            string         `json:"reason"`
	Tags              []string       `json:"tags,omitempty"`
	ProposedAuthority int            `json:"proposed_authority"`
	EvidenceIDs       []uuid.UUID    `json:"evidence_ids,omitempty"`
	CandidateID       string         `json:"candidate_id,omitempty"`
	SalienceScore     float64        `json:"salience_score"`
}

// DigestRejected records a guardrail skip.
type DigestRejected struct {
	Reason string `json:"reason"`
	Detail string `json:"detail,omitempty"`
}

// DigestResult is the response from POST /v1/curation/digest.
type DigestResult struct {
	Proposals []DigestProposal `json:"proposals"`
	Rejected  []DigestRejected `json:"rejected,omitempty"`
	Truncated bool             `json:"truncated"`
}

// ProposalPayloadV1 is persisted in candidate_events.proposal_json (v: 1).
type ProposalPayloadV1 struct {
	V                 int            `json:"v"`
	Kind              api.MemoryKind `json:"kind"`
	Statement         string         `json:"statement"`
	Reason            string         `json:"reason"`
	Tags              []string       `json:"tags,omitempty"`
	ProposedAuthority int            `json:"proposed_authority"`
	EvidenceIDs       []uuid.UUID    `json:"evidence_ids,omitempty"`
	ArtifactRefs      []ArtifactRef  `json:"artifact_refs,omitempty"`
	WorkSummary       string         `json:"work_summary,omitempty"`
	ProposalID        string         `json:"proposal_id,omitempty"`
	// SourceAdvisoryEpisodeID is set when the candidate was distilled from an advisory_episodes row (not canonical).
	SourceAdvisoryEpisodeID string `json:"source_advisory_episode_id,omitempty"`
	// DistillStatementKey is memorynorm.StatementKey(Statement) for pending dedup (distilled candidates only).
	DistillStatementKey string `json:"distill_statement_key,omitempty"`
	// SourceAdvisoryEpisodeIDs lists all advisory episodes that contributed (merge traceability).
	SourceAdvisoryEpisodeIDs []string `json:"source_advisory_episode_ids,omitempty"`
	// DistillSupportCount is how many distill operations merged into this row (repetition strengthening).
	DistillSupportCount int `json:"distill_support_count,omitempty"`
	// PluribusDistillOrigin is how the row was produced: "manual" (explicit POST /v1/episodes/distill), "auto" (post-advisory ingest), or "mixed" (merged from both). Omitted on legacy rows (treated as manual).
	PluribusDistillOrigin string `json:"pluribus_distill_origin,omitempty"`
	// SupersedesMemoryID when set to an existing memory UUID, materialize creates the new row with CreateRequest.supersedes_id (marks prior row superseded — additive evolution, not archive).
	SupersedesMemoryID string `json:"supersedes_memory_id,omitempty"`
	// PluribusEvolution optional additive relationships stored on the created memory payload (pluribus_evolution).
	PluribusEvolution *PluribusEvolutionV1 `json:"pluribus_evolution,omitempty"`
}

// PluribusEvolutionV1 is stored under memory payload key pluribus_evolution (additive, auditable).
type PluribusEvolutionV1 struct {
	SupersededBy  string   `json:"superseded_by,omitempty"`
	Contradicts   []string `json:"contradicts,omitempty"`
	InvalidatedBy string   `json:"invalidated_by,omitempty"`
}

// DigestLimits holds byte and count limits (from config).
type DigestLimits struct {
	MaxProposals        int
	WorkSummaryMaxBytes int
	StatementMaxBytes   int
	ReasonMaxBytes      int
}

// SupportingEpisodeSummary is a short view of an advisory episode for candidate review (not full payload).
type SupportingEpisodeSummary struct {
	EpisodeID string `json:"episode_id"`
	Summary   string `json:"summary"`
}

// TagsGrouped splits tags for readability at review time (entity:* vs other domain/context tags).
type TagsGrouped struct {
	Entities []string `json:"entities,omitempty"`
	Domain   []string `json:"domain,omitempty"`
}

// PromotionPreview is a read-only projection of what materialize would apply (no writes).
type PromotionPreview struct {
	Kind              string   `json:"kind"`
	Statement         string   `json:"statement"`
	Tags              []string `json:"tags,omitempty"`
	ProposedAuthority int      `json:"proposed_authority"`
	Applicability     string   `json:"applicability"` // "advisory" | "governing"
	MemoryStatusNote  string   `json:"memory_status_note,omitempty"`
}

// Promotion validation result from ValidatePromotionCandidate (manual or auto materialize).
type PromotionValidationResult struct {
	Allow  bool   `json:"allow"`
	Reason string `json:"reason"`
}

// AutoPromoteResponse is returned from POST /v1/curation/auto-promote.
type AutoPromoteResponse struct {
	Results []AutoPromoteResultRow `json:"results"`
}

// AutoPromoteResultRow is one candidate outcome from the auto-promote batch.
type AutoPromoteResultRow struct {
	CandidateID string `json:"candidate_id"`
	MemoryID    string `json:"memory_id,omitempty"`
	Status      string `json:"status"` // promoted | skipped | error
	Detail      string `json:"detail,omitempty"`
}

// CandidateReviewResponse is GET /v1/curation/candidates/{id}/review — assistance for humans only.
type CandidateReviewResponse struct {
	CandidateID          string                     `json:"candidate_id"`
	PromotionStatus      string                     `json:"promotion_status"`
	PromotionReadiness   string                     `json:"promotion_readiness,omitempty"`
	ReadinessReason      string                     `json:"readiness_reason,omitempty"`
	Explanation          string                     `json:"explanation"`
	SupportingEpisodes   []SupportingEpisodeSummary `json:"supporting_episodes,omitempty"`
	SignalStrength       string                     `json:"signal_strength"` // "low" | "moderate" | "strong"
	SignalDetail         string                     `json:"signal_detail"`
	TagsGrouped          TagsGrouped                `json:"tags_grouped"`
	EntitiesDisplay      []string                   `json:"entities_display,omitempty"`
	PromotionPreview     *PromotionPreview          `json:"promotion_preview,omitempty"`
}
