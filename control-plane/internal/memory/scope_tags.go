package memory

import "strings"

// DedupKey fingerprints canonical identity for uniqueness with kind + statement_key.
func DedupKey() string {
	return "shared"
}

// mergePersistTags persists only what the client put in tags (trimmed, deduped).
// Tags are explicit metadata only and never inferred.
func mergePersistTags(req CreateRequest) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, t := range req.Tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

func enrichFromTags(obj *MemoryObject, tags []string) {
	obj.Tags = tags
}
