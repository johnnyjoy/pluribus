package ingest

// Predicate negation pairs (normalized lowercase tokens). Debug-only conflict hints (M4).
var oppositePredicate = map[string]string{
	"is":               "is_not",
	"is_not":           "is",
	"equals":           "not_equals",
	"not_equals":       "equals",
	"must":             "must_not",
	"must_not":         "must",
	"supports":         "does_not_support",
	"does_not_support": "supports",
}

// OppositePredicateNorm returns the registered opposite normalized predicate for a, if any.
func OppositePredicateNorm(a string) (string, bool) {
	if a == "" {
		return "", false
	}
	if o, ok := oppositePredicate[a]; ok {
		return o, true
	}
	return "", false
}

func predicatesOpposite(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if oppositePredicate[a] == b {
		return true
	}
	return oppositePredicate[b] == a
}

// DetectConflictsAmongRows compares all pairs with equal subject; does not mutate rows.
// Same predicate + dissimilar objects → conflict; opposite predicates → conflict.
func DetectConflictsAmongRows(rows []CanonicalFactRow, conflictObjectMaxJaccard float64) []map[string]interface{} {
	if conflictObjectMaxJaccard <= 0 {
		conflictObjectMaxJaccard = DefaultConflictObjectMaxJaccard
	}
	var out []map[string]interface{}
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if rows[i].SubjectNorm != rows[j].SubjectNorm {
				continue
			}
			a, b := rows[i], rows[j]
			if predicatesOpposite(a.PredicateNorm, b.PredicateNorm) {
				out = append(out, conflictEntry("opposite_predicate", i, j, a, b,
					"normalized predicates are registered opposites"))
				continue
			}
			if a.PredicateNorm == b.PredicateNorm {
				if a.ObjectNorm == b.ObjectNorm {
					continue
				}
				jacc := TokenJaccard(a.ObjectNorm, b.ObjectNorm)
				if jacc < conflictObjectMaxJaccard {
					out = append(out, conflictEntry("same_predicate_divergent_object", i, j, a, b,
						"object token overlap below threshold"))
				}
			}
		}
	}
	return out
}

func conflictEntry(code string, i, j int, a, b CanonicalFactRow, detail string) map[string]interface{} {
	return map[string]interface{}{
		"code":              code,
		"detail":            detail,
		"subject":           a.SubjectNorm,
		"source_index_a":    a.SourceIndex,
		"source_index_b":    b.SourceIndex,
		"predicate_a":       a.PredicateNorm,
		"predicate_b":       b.PredicateNorm,
		"object_a":          a.ObjectNorm,
		"object_b":          b.ObjectNorm,
		"normalize_version": NormalizePipelineVersion,
	}
}
