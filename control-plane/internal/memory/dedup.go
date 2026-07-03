package memory

// DedupConfig controls exact canonical duplicate rejection on POST /v1/memory (Phase C).
// Nil Enabled means on (same default pattern as enforcement.enabled).
type DedupConfig struct {
	Enabled *bool
	// SemanticConsolidateThreshold: when > 0 and semantic retrieval is on, create path
	// reinforces an existing same-kind row whose embedding cosine ≥ threshold instead of
	// inserting a near-duplicate. 0 = off (default).
	SemanticConsolidateThreshold float64
}

// IsEnabled returns whether duplicate detection runs before insert. Omitted or nil Enabled → true.
func (d *DedupConfig) IsEnabled() bool {
	if d == nil {
		return true
	}
	if d.Enabled == nil {
		return true
	}
	return *d.Enabled
}
