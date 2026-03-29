package ingest

import (
	"context"
	"sort"
)

// UnifySimilarWithinBatch aligns later facts to earlier ones when subject matches and
// predicate/object are similar (token Jaccard). Lower source index wins (deterministic).
// When lineage is non-nil, each merge appends a similar_unify event (source=batch).
func UnifySimilarWithinBatch(rows []CanonicalFactRow, minJaccard float64, merge *[]map[string]interface{}, lineage *[]LineageEvent) {
	if minJaccard <= 0 {
		minJaccard = DefaultSimilarJaccardMin
	}
	if merge == nil {
		return
	}
	for j := 1; j < len(rows); j++ {
		for i := 0; i < j; i++ {
			if rows[j].NormalizedHash == rows[i].NormalizedHash {
				continue
			}
			if !SimilarForUnify(
				rows[j].SubjectNorm, rows[j].PredicateNorm, rows[j].ObjectNorm,
				rows[i].SubjectNorm, rows[i].PredicateNorm, rows[i].ObjectNorm,
				minJaccard,
			) {
				continue
			}
			oldHash := rows[j].NormalizedHash
			anchorHash := rows[i].NormalizedHash
			rows[j].PredicateNorm = rows[i].PredicateNorm
			rows[j].ObjectNorm = rows[i].ObjectNorm
			rows[j].SubjectNorm = rows[i].SubjectNorm
			RecomputeCanonicalHash(&rows[j])
			newHash := rows[j].NormalizedHash
			*merge = append(*merge, similarUnifyMergeAction(rows[j].SourceIndex, oldHash, newHash, rows[i].SourceIndex))
			if lineage != nil {
				root := minLexHash(oldHash, anchorHash)
				meta := map[string]interface{}{
					"anchor_source_index": rows[i].SourceIndex,
					"source_index":        rows[j].SourceIndex,
					"prior_hash":          oldHash,
					"anchor_hash":         anchorHash,
					"resulting_hash":      newHash,
				}
				*lineage = append(*lineage, LineageEvent{
					FactHash:   newHash,
					ParentHash: oldHash,
					RootHash:   root,
					MergeType:  "similar_unify",
					Source:     "batch",
					Meta:       meta,
				})
			}
			break
		}
	}
}

func similarUnifyMergeAction(sourceIndex int, priorHash, resultingHash string, anchorSourceIndex int) map[string]interface{} {
	return map[string]interface{}{
		"action":              "similar_unify",
		"source_index":        sourceIndex,
		"anchor_source_index": anchorSourceIndex,
		"prior_hash":          priorHash,
		"resulting_hash":      resultingHash,
	}
}

func minLexHash(a, b string) string {
	if a < b {
		return a
	}
	return b
}

// GlobalUnifyFromDB aligns each row to prior extractions (DB + earlier rows in this ingest)
// when subject matches and predicate/object are similar. Deterministic: smallest matching
// normalized_hash wins; row only changes when that winner is lexicographically smaller than
// the row's current hash (same rule as unifyRowFromDB).
func GlobalUnifyFromDB(ctx context.Context, q rowsQuerier, rows []CanonicalFactRow, minJaccard float64, merge *[]map[string]interface{}, lineage *[]LineageEvent, dbg *IngestDebug) error {
	if merge == nil {
		return nil
	}
	if minJaccard <= 0 {
		minJaccard = DefaultSimilarJaccardMin
	}
	for i := range rows {
		row := &rows[i]
		dbCands, err := selectCanonicalCandidates(ctx, q, row.SubjectNorm, row.PredicateNorm, 500)
		if err != nil {
			return err
		}
		merged := collectSimilarCandidates(row, i, rows, dbCands, minJaccard)
		candCount := len(merged)
		var best *canonicalTriple
		for _, c := range merged {
			if c.NormalizedHash >= row.NormalizedHash {
				continue
			}
			if best == nil || c.NormalizedHash < best.NormalizedHash {
				cc := c
				best = &cc
			}
		}
		if dbg != nil {
			rowDbg := GMCLGlobalUnifyRowDebug{SourceIndex: row.SourceIndex, CandidateCount: candCount}
			if best != nil {
				h := best.NormalizedHash
				rowDbg.SelectedAnchorHash = &h
			}
			dbg.GMCLGlobalUnify = append(dbg.GMCLGlobalUnify, rowDbg)
		}
		if best == nil {
			continue
		}
		oldHash := row.NormalizedHash
		row.PredicateNorm = best.PredicateNorm
		row.ObjectNorm = best.ObjectNorm
		RecomputeCanonicalHash(row)
		newHash := row.NormalizedHash
		root := minLexHash(oldHash, best.NormalizedHash)
		*merge = append(*merge, map[string]interface{}{
			"action":           "similar_unify",
			"source":           "prior_extraction",
			"source_index":     row.SourceIndex,
			"prior_hash":       oldHash,
			"anchor_hash":      best.NormalizedHash,
			"resulting_hash":   newHash,
			"anchor_predicate": best.PredicateNorm,
			"anchor_object":    best.ObjectNorm,
		})
		if lineage != nil {
			meta := map[string]interface{}{
				"source_index":   row.SourceIndex,
				"prior_hash":     oldHash,
				"anchor_hash":    best.NormalizedHash,
				"resulting_hash": newHash,
			}
			*lineage = append(*lineage, LineageEvent{
				FactHash:   newHash,
				ParentHash: oldHash,
				RootHash:   root,
				MergeType:  "similar_unify",
				Source:     "db",
				Meta:       meta,
			})
		}
	}
	return nil
}

func collectSimilarCandidates(row *CanonicalFactRow, idx int, rows []CanonicalFactRow, db []canonicalTriple, minJaccard float64) []canonicalTriple {
	seen := make(map[string]struct{})
	var out []canonicalTriple
	add := func(c canonicalTriple) {
		if _, ok := seen[c.NormalizedHash]; ok {
			return
		}
		if c.NormalizedHash == row.NormalizedHash {
			return
		}
		if !SimilarForUnify(row.SubjectNorm, row.PredicateNorm, row.ObjectNorm,
			row.SubjectNorm, c.PredicateNorm, c.ObjectNorm, minJaccard) {
			return
		}
		seen[c.NormalizedHash] = struct{}{}
		out = append(out, c)
	}
	for i := range db {
		add(db[i])
	}
	for j := 0; j < idx; j++ {
		o := rows[j]
		if o.SubjectNorm != row.SubjectNorm {
			continue
		}
		add(canonicalTriple{PredicateNorm: o.PredicateNorm, ObjectNorm: o.ObjectNorm, NormalizedHash: o.NormalizedHash})
	}
	sort.Slice(out, func(a, b int) bool {
		return out[a].NormalizedHash < out[b].NormalizedHash
	})
	return out
}
