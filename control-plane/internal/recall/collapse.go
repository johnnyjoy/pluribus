package recall

import (
	"control-plane/internal/memory"
	"control-plane/internal/memorynorm"
	"control-plane/internal/similarity"
	"control-plane/pkg/api"
)

// collapseKindState tracks accepted rows per memory kind for exact-key and near-dup suppression.
type collapseKindState struct {
	seenKey map[string]struct{}
	canons  []string // statement_canonical text of accepted rows (for Jaccard)
}

func statementKeyForCollapse(o memory.MemoryObject) string {
	if o.StatementKey != "" {
		return o.StatementKey
	}
	return memorynorm.StatementKey(o.Statement)
}

func canonicalForCollapse(o memory.MemoryObject) string {
	if o.StatementCanonical != "" {
		return o.StatementCanonical
	}
	return memorynorm.StatementCanonical(o.Statement)
}

// collapseScoredForRecall drops redundant rows per kind after ranking (Phase F):
// (1) same statement_key as an already accepted row in that kind;
// (2) if nearDupJaccardThreshold > 0, Jaccard on canonical text ≥ threshold vs an accepted row.
// Order follows the incoming scored slice (authority-first stable sort).
func collapseScoredForRecall(scored []ScoredMemory, nearDupJaccardThreshold float64) []ScoredMemory {
	if len(scored) == 0 {
		return scored
	}
	byKind := make(map[api.MemoryKind]*collapseKindState)
	out := make([]ScoredMemory, 0, len(scored))
	for _, s := range scored {
		kind := s.Object.Kind
		st := byKind[kind]
		if st == nil {
			st = &collapseKindState{seenKey: make(map[string]struct{})}
			byKind[kind] = st
		}
		key := statementKeyForCollapse(s.Object)
		if key != "" {
			if _, dup := st.seenKey[key]; dup {
				continue
			}
		}
		canon := canonicalForCollapse(s.Object)
		if nearDupJaccardThreshold > 0 && len(st.canons) > 0 {
			skip := false
			for _, prev := range st.canons {
				if similarity.CanonicalTokenJaccard(canon, prev) >= nearDupJaccardThreshold {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
		}
		out = append(out, s)
		if key != "" {
			st.seenKey[key] = struct{}{}
		}
		st.canons = append(st.canons, canon)
	}
	return out
}
