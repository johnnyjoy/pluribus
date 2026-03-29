package recall

import (
	"control-plane/internal/memory"

	"github.com/google/uuid"
)

// NormalizeRIUWeights applies default weights for any zero component (same rule as ranking config).
func NormalizeRIUWeights(w RIUWeights) RIUWeights {
	return RIUWeightsFromConfig(w.Applicability, w.Transferable, w.LineageProxy, w.ContradictionPenalty)
}

// RIUWeightsFromConfig builds RIUWeights from YAML fields; zero uses defaults per field.
func RIUWeightsFromConfig(applicability, transferable, lineageProxy, contradictionPenalty float64) RIUWeights {
	d := DefaultRIUWeights()
	w := RIUWeights{
		Applicability:        applicability,
		Transferable:         transferable,
		LineageProxy:         lineageProxy,
		ContradictionPenalty: contradictionPenalty,
	}
	if w.Applicability == 0 {
		w.Applicability = d.Applicability
	}
	if w.Transferable == 0 {
		w.Transferable = d.Transferable
	}
	if w.LineageProxy == 0 {
		w.LineageProxy = d.LineageProxy
	}
	if w.ContradictionPenalty == 0 {
		w.ContradictionPenalty = d.ContradictionPenalty
	}
	return w
}

// ScoreAndSortWithRIU extends ranking-only ScoreAt with RIU components and deterministic stable ordering.
func ScoreAndSortWithRIU(
	objs []memory.MemoryObject,
	req ScoreRequest,
	weights RankingWeights,
	riu RIUWeights,
	domainTags []string,
	reqTags []string,
	unresolved map[uuid.UUID]bool,
	maxAuthority int,
) []ScoredMemory {
	if len(objs) == 0 {
		return nil
	}
	if maxAuthority <= 0 {
		for _, o := range objs {
			if o.Authority > maxAuthority {
				maxAuthority = o.Authority
			}
		}
		if maxAuthority <= 0 {
			maxAuthority = 10
		}
	}
	refTime := RefTimeForRanking(objs)
	out := make([]ScoredMemory, len(objs))
	for i, o := range objs {
		ranking := scoreAt(o, req, weights, maxAuthority, refTime)
		cand := BuildRecallCandidate(o, domainTags, reqTags, unresolved)
		app := applicabilityComponent(o, reqTags, riu)
		tf := 0.0
		if cand.Transferable {
			tf = riu.Transferable * transferableScore(o.Tags, reqTags, domainTags)
		}
		// Lineage proxy: authority-normalized until lineage tables are wired (Pluribus RIU plan §5.3).
		authNorm := float64(o.Authority) / float64(maxAuthority)
		if authNorm > 1 {
			authNorm = 1
		}
		lineage := riu.LineageProxy * authNorm
		pen := 0.0
		if unresolved != nil && unresolved[o.ID] {
			pen = riu.ContradictionPenalty
		}
		total := ranking + app + tf + lineage - pen
		br := &RIUScoreBreakdown{
			RankingScore:         ranking,
			ApplicabilityScore:   app,
			TransferableScore:    tf,
			LineageProxyScore:    lineage,
			ContradictionPenalty: pen,
			TotalScore:           total,
			Transferable:         cand.Transferable,
			ContradictionStatus:  cand.ContradictionStatus,
		}
		out[i] = ScoredMemory{
			Object: o,
			Score:  total,
			Reason: dominantReasonAt(o, req, weights, maxAuthority, refTime),
			RIU:    br,
		}
	}
	sortScoredMemoriesStable(out)
	return out
}
