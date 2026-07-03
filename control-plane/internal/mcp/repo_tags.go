package mcp

// TagsFromRepoRoot derives soft-scope tags from an absolute workspace path.
// Doctrine: tags only (project:* / repo:*), never SQL partitions.
func TagsFromRepoRoot(repoRoot string) []string {
	base := repoBasename(repoRoot)
	if base == "" || base == "unspecified" {
		return nil
	}
	return []string{"project:" + base, "repo:" + base}
}

func appendRepoRootTags(tags []string, repoRoot string) []string {
	derived := TagsFromRepoRoot(repoRoot)
	if len(derived) == 0 {
		return tags
	}
	return dedupeStrings(append(tags, derived...))
}
