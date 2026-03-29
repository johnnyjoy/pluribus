package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// CanonicalFactRow is one persisted normalized extraction (M2).
type CanonicalFactRow struct {
	IngestionID    uuid.UUID
	SubjectNorm    string
	PredicateNorm  string
	ObjectNorm     string
	Confidence     float64
	Provenance     json.RawMessage
	NormalizedHash string
	SourceIndex    int
	PriorityScore  float64
}

// ProvenanceDoc links extraction to client payload (M2).
type ProvenanceDoc struct {
	SourceRefs []string `json:"source_refs,omitempty"`
	Evidence   []string `json:"evidence,omitempty"`
	QueryHash  string   `json:"context_window_hash,omitempty"`
}

// BuildCanonicalRows normalizes extracted_facts and computes hashes.
func BuildCanonicalRows(ingestionID uuid.UUID, req CognitionRequest) ([]CanonicalFactRow, []string) {
	var warnings []string
	var rows []CanonicalFactRow
	ch := strings.TrimSpace(req.ContextWindowHash)
	for i, f := range req.ExtractedFacts {
		sub := NormalizeFactToken(f.Subject)
		pred := NormalizeFactToken(f.Predicate)
		obj := NormalizeFactToken(f.Object)
		if sub == "" || pred == "" || obj == "" {
			warnings = append(warnings, "extracted_facts["+strconv.Itoa(i)+"]: normalized to empty component")
			continue
		}
		conf := req.Confidence
		if f.Confidence != nil {
			conf = *f.Confidence
		}
		prov := ProvenanceDoc{
			SourceRefs: append([]string(nil), req.SourceRefs...),
			Evidence:   append([]string(nil), f.Evidence...),
			QueryHash:  ch,
		}
		pb, _ := json.Marshal(prov)
		h := normalizedFactHash(sub, pred, obj)
		rows = append(rows, CanonicalFactRow{
			IngestionID:    ingestionID,
			SubjectNorm:    sub,
			PredicateNorm:  pred,
			ObjectNorm:     obj,
			Confidence:     conf,
			Provenance:     pb,
			NormalizedHash: h,
			SourceIndex:    i,
		})
	}
	return rows, warnings
}

// CanonicalFactJSON is the API shape for canonical_facts[] in CognitionResponse.
type CanonicalFactJSON struct {
	Subject          string  `json:"subject"`
	Predicate        string  `json:"predicate"`
	Object           string  `json:"object"`
	Confidence       float64 `json:"confidence"`
	NormalizedHash   string  `json:"normalized_hash"`
	SourceIndex      int     `json:"source_index"`
	NormalizeVersion string  `json:"normalize_version"`
	PriorityScore    float64 `json:"priority_score"`
}

// RecomputeCanonicalHash updates NormalizedHash from current normalized fields.
func RecomputeCanonicalHash(r *CanonicalFactRow) {
	r.NormalizedHash = normalizedFactHash(r.SubjectNorm, r.PredicateNorm, r.ObjectNorm)
}

func rowsToCanonicalJSON(rows []CanonicalFactRow) ([]json.RawMessage, error) {
	out := make([]json.RawMessage, 0, len(rows))
	for _, r := range rows {
		c := CanonicalFactJSON{
			Subject:          r.SubjectNorm,
			Predicate:        r.PredicateNorm,
			Object:           r.ObjectNorm,
			Confidence:       r.Confidence,
			NormalizedHash:   r.NormalizedHash,
			SourceIndex:      r.SourceIndex,
			NormalizeVersion: NormalizePipelineVersion,
			PriorityScore:    r.PriorityScore,
		}
		b, err := json.Marshal(c)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

func normalizedFactHash(sub, pred, obj string) string {
	s := sub + "|" + pred + "|" + obj
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
