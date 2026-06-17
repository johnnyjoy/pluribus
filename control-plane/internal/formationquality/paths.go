package formationquality

import "strings"

// IsDirectLikePath is true for paths that may create active guidance (direct create, promote).
func IsDirectLikePath(path string) bool {
	switch strings.TrimSpace(path) {
	case "direct_create", "memories_create", "promote":
		return true
	default:
		return false
	}
}

// IsProbationaryPath is true for advisory/probationary ingest paths.
func IsProbationaryPath(path string) bool {
	switch strings.TrimSpace(path) {
	case "record_experience", "probationary_ingest":
		return true
	default:
		return false
	}
}
