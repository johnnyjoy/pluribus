package memory

// DedupConfig controls exact canonical duplicate rejection on POST /v1/memory (Phase C).
// Nil Enabled means on (same default pattern as enforcement.enabled).
type DedupConfig struct {
	Enabled *bool
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
