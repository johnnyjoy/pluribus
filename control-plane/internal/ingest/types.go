package ingest

import (
	"encoding/json"

	"github.com/google/uuid"
)

// CognitionRequest is the body for POST /v1/ingest/cognition (MCL).
//
// TempContributorID labels the sending integration for ingest trust/audit only (not durable memory partitioning).
type CognitionRequest struct {
	TempContributorID string          `json:"temp_contributor_id"`
	Query             string          `json:"query"`
	ReasoningTrace    []string        `json:"reasoning_trace"`
	ExtractedFacts    []ExtractedFact `json:"extracted_facts"`
	Confidence        float64         `json:"confidence"`
	SourceRefs        []string        `json:"source_refs,omitempty"`
	ContextWindowHash string          `json:"context_window_hash"`
	// ProposePromotion requests durable memory promotion when the server bridge is enabled (M7). Never implied by default.
	ProposePromotion bool `json:"propose_promotion,omitempty"`
}

// ExtractedFact is one client-supplied fact (pre-normalization in M1).
type ExtractedFact struct {
	Subject    string   `json:"subject"`
	Predicate  string   `json:"predicate"`
	Object     string   `json:"object"`
	Confidence *float64 `json:"confidence,omitempty"`
	Evidence   []string `json:"evidence,omitempty"`
}

// CognitionResponse is returned after ingest gateway processing.
type CognitionResponse struct {
	IngestionID    uuid.UUID         `json:"ingestion_id"`
	Status         string            `json:"status"` // accepted | rejected | processed
	CanonicalFacts []json.RawMessage `json:"canonical_facts"`
	Debug          IngestDebug       `json:"debug"`
}

// CommitResponse is returned by POST /v1/ingest/{id}/commit (M7 operator promotion).
type CommitResponse struct {
	IngestionID uuid.UUID            `json:"ingestion_id"`
	Promotion   IngestPromotionDebug `json:"promotion"`
}

// LineageEvent is queued for INSERT into canonical_fact_lineage (GMCL).
type LineageEvent struct {
	FactHash   string
	ParentHash string // empty => NULL in DB
	RootHash   string
	MergeType  string // reinforce | similar_unify
	Source     string // batch | db | promotion
	Meta       map[string]interface{}
}

// GMCLGlobalUnifyRowDebug documents one row's GlobalUnifyFromDB pass.
type GMCLGlobalUnifyRowDebug struct {
	SourceIndex        int     `json:"source_index"`
	CandidateCount     int     `json:"candidate_count"`
	SelectedAnchorHash *string `json:"selected_anchor_hash,omitempty"`
}

// IngestDebug is always JSON-shaped per MCL plan (extend in later phases).
type IngestDebug struct {
	RejectedReason         string                   `json:"rejected_reason"`
	MergeActions           []map[string]interface{} `json:"merge_actions"`
	ConflictsDetected      []map[string]interface{} `json:"conflicts_detected"`
	NormalizationWarnings  []string                 `json:"normalization_warnings"`
	NormalizationVersion   string                   `json:"normalization_version,omitempty"`
	TrustWeightApplied     float64                  `json:"trust_weight_applied"`
	PriorityFormulaVersion string                   `json:"priority_formula_version,omitempty"`
	PriorityWeights        map[string]float64       `json:"priority_weights,omitempty"`
	Promotion              IngestPromotionDebug     `json:"promotion"`
	// GMCL (global memory coherence layer)
	GMCLGlobalUnify        []GMCLGlobalUnifyRowDebug `json:"gmcl_global_unify,omitempty"`
	LineageWritten         int                       `json:"lineage_written"`
	ContradictionPersisted int                       `json:"contradiction_persisted"`
}

// IngestPromotionDebug documents optional M7 promotion bridge (gated; never silent auto-promote).
type IngestPromotionDebug struct {
	Attempted bool   `json:"attempted"`
	Reason    string `json:"reason"`
	// ServerAutoPromoteEnabled mirrors config ingest.auto_promote (default false).
	ServerAutoPromoteEnabled bool `json:"server_auto_promote_enabled"`
	// ClientProposePromotion is set when the client sent propose_promotion (inline path).
	ClientProposePromotion bool `json:"client_propose_promotion,omitempty"`
	// Mode is "inline" or "commit_operator" when promotion ran or was evaluated.
	Mode string `json:"mode,omitempty"`
	// MemoryIDs are promoted memories.id values (UUID strings) when successful.
	MemoryIDs []string `json:"memory_ids,omitempty"`
	Errors    []string `json:"errors,omitempty"`
}

func defaultDebug() IngestDebug {
	return IngestDebug{
		RejectedReason:        "",
		MergeActions:          []map[string]interface{}{},
		ConflictsDetected:     []map[string]interface{}{},
		NormalizationWarnings: []string{},
		TrustWeightApplied:    1.0,
		Promotion: IngestPromotionDebug{
			Attempted:                false,
			Reason:                   "promotion not evaluated (rejected or no canonical rows)",
			ServerAutoPromoteEnabled: false,
		},
	}
}
