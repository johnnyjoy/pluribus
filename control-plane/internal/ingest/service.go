package ingest

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrIngestionNotFound is returned when an ingestion id does not exist.
var ErrIngestionNotFound = errors.New("ingest: ingestion not found")

// ErrIngestionNotPromotable is returned when commit/promotion is not allowed for this ingestion row.
var ErrIngestionNotPromotable = errors.New("ingest: ingestion is not in accepted state or has no canonical data")

// Service runs the ingest gateway (M1: validate + persist audit row; M2: canonical facts).
type Service struct {
	Repo                     *Repo
	Limits                   Limits
	RequireContextWindowHash bool
	// M4: similar merge + conflict hints (thresholds; zero = defaults).
	SimilarJaccardMin        float64
	ConflictObjectMaxJaccard float64
	// M5: noise gates (zero = defaults).
	MinConfidenceTrustProduct float64
	MinTraceTotalChars        int
	// M6: priority score (zero weights = defaults).
	PriorityWeights PriorityWeights
	// M7: optional promotion bridge (default off; requires client propose_promotion or operator commit).
	AutoPromote bool
	Promoter    MemoryPromoter
}

// NewService returns a gateway with defaults.
func NewService(repo *Repo) *Service {
	return &Service{
		Repo:                      repo,
		Limits:                    DefaultLimits(),
		RequireContextWindowHash:  true,
		SimilarJaccardMin:         DefaultSimilarJaccardMin,
		ConflictObjectMaxJaccard:  DefaultConflictObjectMaxJaccard,
		MinConfidenceTrustProduct: DefaultMinConfidenceTrustProduct,
		MinTraceTotalChars:        DefaultMinTraceTotalChars,
		PriorityWeights:           PriorityWeights{},
	}
}

// IngestCognition validates, persists ingestion_records, returns response.
func (s *Service) IngestCognition(ctx context.Context, req CognitionRequest) (*CognitionResponse, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	dbg := defaultDebug()

	reason := Validate(req, s.Limits, s.RequireContextWindowHash)
	tempContributorID := strings.TrimSpace(req.TempContributorID)
	if reason == "" {
		trust, err := s.Repo.TrustWeightByTempContributorID(ctx, tempContributorID)
		if err != nil {
			return nil, err
		}
		dbg.TrustWeightApplied = trust
		if nr := NoiseRejectReason(req, trust, s.MinConfidenceTrustProduct, s.MinTraceTotalChars); nr != "" {
			reason = nr
		}
	} else {
		dbg.TrustWeightApplied = 1.0
	}

	status := "accepted"
	var rejectedPtr *string
	if reason != "" {
		status = "rejected"
		rejectedPtr = &reason
		dbg.RejectedReason = reason
	}
	ch := strings.TrimSpace(req.ContextWindowHash)
	var chPtr *string
	if ch != "" {
		chPtr = &ch
	}

	tx, err := s.Repo.DB.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	id, err := insertIngestion(ctx, tx, tempContributorID, status, rejectedPtr, payload, chPtr)
	if err != nil {
		return nil, err
	}

	canonJSON := []json.RawMessage{}
	var rows []CanonicalFactRow
	if status == "accepted" {
		dbg.NormalizationVersion = NormalizePipelineVersion
		dbg.PriorityFormulaVersion = PriorityFormulaVersion
		dbg.PriorityWeights = priorityWeightsDebugMap(s.PriorityWeights)
		var warnings []string
		rows, warnings = BuildCanonicalRows(id, req)
		dbg.NormalizationWarnings = append(dbg.NormalizationWarnings, warnings...)
		dbg.ConflictsDetected = append(dbg.ConflictsDetected, DetectConflictsAmongRows(rows, s.ConflictObjectMaxJaccard)...)
		var lineageEvents []LineageEvent
		UnifySimilarWithinBatch(rows, s.SimilarJaccardMin, &dbg.MergeActions, &lineageEvents)
		if err := GlobalUnifyFromDB(ctx, tx, rows, s.SimilarJaccardMin, &dbg.MergeActions, &lineageEvents, &dbg); err != nil {
			return nil, err
		}
		for i := range rows {
			row := &rows[i]
			priorPeak, priorCount, err := SelectCanonicalConfidenceStats(ctx, tx, row.NormalizedHash)
			if err != nil {
				return nil, err
			}
			incoming := row.Confidence
			if priorCount > 0 {
				row.Confidence = ApplyReinforce(priorPeak, incoming)
				dbg.MergeActions = append(dbg.MergeActions, reinforceMergeAction(
					row.NormalizedHash, row.SourceIndex, priorPeak, incoming, row.Confidence, priorCount,
				))
				lineageEvents = append(lineageEvents, LineageEvent{
					FactHash:   row.NormalizedHash,
					ParentHash: "",
					RootHash:   row.NormalizedHash,
					MergeType:  "reinforce",
					Source:     "db",
					Meta: map[string]interface{}{
						"source_index": row.SourceIndex,
						"prior_peak":   priorPeak,
						"incoming":     incoming,
						"prior_count":  priorCount,
					},
				})
			}
			lastSeen, err := SelectMaxCreatedAtForHash(ctx, tx, row.NormalizedHash)
			if err != nil {
				return nil, err
			}
			now := time.Now().UTC()
			row.PriorityScore = ComputePriorityScore(row.Confidence, priorCount, lastSeen, now, dbg.TrustWeightApplied, s.PriorityWeights)
			if err := insertCanonicalFact(ctx, tx, *row); err != nil {
				return nil, err
			}
		}
		if err := insertCanonicalFactLineageBatch(ctx, tx, id, lineageEvents); err != nil {
			return nil, err
		}
		dbg.LineageWritten = len(lineageEvents)
		cn, err := persistCanonicalContradictions(ctx, tx, id, rows, s.ConflictObjectMaxJaccard)
		if err != nil {
			return nil, err
		}
		dbg.ContradictionPersisted = int(cn)
		canonJSON, err = rowsToCanonicalJSON(rows)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	if status == "accepted" {
		dbg.Promotion = applyPromotions(ctx, s.Promoter, s.AutoPromote, req.ProposePromotion, rows, id)
	} else {
		dbg.Promotion = IngestPromotionDebug{
			Attempted:                false,
			ServerAutoPromoteEnabled: s.AutoPromote,
			ClientProposePromotion:   req.ProposePromotion,
			Reason:                   "ingest rejected; canonical rows not persisted",
		}
	}

	return &CognitionResponse{
		IngestionID:    id,
		Status:         status,
		CanonicalFacts: canonJSON,
		Debug:          dbg,
	}, nil
}

// CommitIngestion runs operator promotion for a prior accepted ingest (M7). Requires ingest.auto_promote and Promoter.
func (s *Service) CommitIngestion(ctx context.Context, ingestionID uuid.UUID) (*CommitResponse, error) {
	if s.Repo == nil {
		return nil, errors.New("ingest: repo not configured")
	}
	st, err := s.Repo.GetIngestionStatus(ctx, ingestionID)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, ErrIngestionNotFound
	}
	if st.Status != "accepted" {
		return nil, ErrIngestionNotPromotable
	}
	rows, err := s.Repo.ListCanonicalRowsByIngestionID(ctx, ingestionID)
	if err != nil {
		return nil, err
	}
	promo := applyCommitPromotions(ctx, s.Promoter, s.AutoPromote, rows, ingestionID)
	return &CommitResponse{IngestionID: ingestionID, Promotion: promo}, nil
}
