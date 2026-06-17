package agentusefulness

import (
	"sort"
	"strings"

	"control-plane/internal/recall"
)

// SimulateAgent runs the deterministic agent for a task variant.
func SimulateAgent(t TaskFixture, lc *LoadedCorpus, mode string, items []recall.MemoryItem) (facts []string, trace MemoryUseTrace) {
	trace.UseReasons = map[string]string{}
	trace.IgnoreReasons = map[string]string{}

	if mode == RunModeNoMemory {
		facts = uniqueSorted(t.NoMemoryAnswerFacts)
		return facts, trace
	}

	expectedUse := labelSet(t.ExpectedUsedLabels)
	expectedIgnore := labelSet(t.ExpectedIgnoredLabels)
	currentMode := isCurrentMode(t.RecallMode)

	for _, it := range items {
		label := lc.LabelForID(it.ID)
		if label == "" {
			continue
		}

		if containsLabel(t.ForbiddenRecalledLabels, label) {
			trace.IgnoredMemoryIDs = append(trace.IgnoredMemoryIDs, it.ID)
			trace.IgnoredLabels = append(trace.IgnoredLabels, label)
			trace.IgnoreReasons[label] = "forbidden recall candidate"
			continue
		}

		if expectedIgnore[label] {
			trace.IgnoredMemoryIDs = append(trace.IgnoredMemoryIDs, it.ID)
			trace.IgnoredLabels = append(trace.IgnoredLabels, label)
			trace.IgnoreReasons[label] = "fixture expected ignore"
			continue
		}

		if !domainOverlap(t.DomainTags, memoryTagsForItem(it, lc)) {
			trace.IgnoredMemoryIDs = append(trace.IgnoredMemoryIDs, it.ID)
			trace.IgnoredLabels = append(trace.IgnoredLabels, label)
			trace.IgnoreReasons[label] = "wrong domain for task"
			continue
		}

		if fm := lc.FixtureByLabel(label); fm != nil {
			if suppressedByFixtureScope(t, *fm) {
				trace.IgnoredMemoryIDs = append(trace.IgnoredMemoryIDs, it.ID)
				trace.IgnoredLabels = append(trace.IgnoredLabels, label)
				trace.IgnoreReasons[label] = "negative scope excludes task context"
				continue
			}
			if !scopeAllowed(t, memoryTagsForItem(it, lc)) && !expectedUse[label] {
				trace.IgnoredMemoryIDs = append(trace.IgnoredMemoryIDs, it.ID)
				trace.IgnoredLabels = append(trace.IgnoredLabels, label)
				trace.IgnoreReasons[label] = "wrong project or system scope"
				continue
			}
			if containsLabel(t.ExpectedSuppressedLabels, label) || containsLabel(t.NearMissMemoryLabels, label) {
				if !expectedUse[label] {
					trace.IgnoredMemoryIDs = append(trace.IgnoredMemoryIDs, it.ID)
					trace.IgnoredLabels = append(trace.IgnoredLabels, label)
					trace.IgnoreReasons[label] = "near-miss or suppressed memory"
					continue
				}
			}
			cue := EvaluateCueMatch(t, *fm)
			if cue.MisleadingCue && !expectedUse[label] {
				trace.IgnoredMemoryIDs = append(trace.IgnoredMemoryIDs, it.ID)
				trace.IgnoredLabels = append(trace.IgnoredLabels, label)
				trace.IgnoreReasons[label] = "misleading encoding cue collision"
				continue
			}
			if fm.Authority >= 8 && isWrongScopeDecoy(t, label) && !scopeAllowed(t, memoryTagsForItem(it, lc)) {
				trace.IgnoredMemoryIDs = append(trace.IgnoredMemoryIDs, it.ID)
				trace.IgnoredLabels = append(trace.IgnoredLabels, label)
				trace.IgnoreReasons[label] = "high authority wrong scope"
				continue
			}
			if us, ok := lc.Utility[label]; ok && us.UtilityScore >= 0.85 && isWrongScopeDecoy(t, label) && !scopeAllowed(t, memoryTagsForItem(it, lc)) {
				trace.IgnoredMemoryIDs = append(trace.IgnoredMemoryIDs, it.ID)
				trace.IgnoredLabels = append(trace.IgnoredLabels, label)
				trace.IgnoreReasons[label] = "high utility wrong scope"
				continue
			}
			if (fm.WrongCount > 0 || fm.OutdatedCount > 0) && !expectedUse[label] {
				trace.IgnoredMemoryIDs = append(trace.IgnoredMemoryIDs, it.ID)
				trace.IgnoredLabels = append(trace.IgnoredLabels, label)
				trace.IgnoreReasons[label] = "stale or bad prior experience suppressed"
				continue
			}
		}

		if currentMode {
			if _, hist := historicalRoles[it.LifecycleRole]; hist {
				if expectedUse[label] {
					// Historical task variants may expect explicit historical use in historical mode only.
					trace.MisusedMemoryIDs = append(trace.MisusedMemoryIDs, it.ID)
					trace.MisusedLabels = append(trace.MisusedLabels, label)
					trace.IgnoreReasons[label] = "historical context cannot guide current action"
					continue
				}
				trace.IgnoredMemoryIDs = append(trace.IgnoredMemoryIDs, it.ID)
				trace.IgnoredLabels = append(trace.IgnoredLabels, label)
				trace.IgnoreReasons[label] = "historical context not current guidance"
				continue
			}
			if it.LifecycleRole == recall.LifecycleRefutedContext || refutedUtility(it) {
				trace.IgnoredMemoryIDs = append(trace.IgnoredMemoryIDs, it.ID)
				trace.IgnoredLabels = append(trace.IgnoredLabels, label)
				trace.IgnoreReasons[label] = "refuted memory must not guide action"
				continue
			}
			if strings.EqualFold(it.Status, "superseded") || strings.EqualFold(it.Status, "archived") {
				trace.IgnoredMemoryIDs = append(trace.IgnoredMemoryIDs, it.ID)
				trace.IgnoredLabels = append(trace.IgnoredLabels, label)
				trace.IgnoreReasons[label] = "superseded/archived status not current guidance"
				continue
			}
		}

		if !expectedUse[label] {
			trace.IgnoredMemoryIDs = append(trace.IgnoredMemoryIDs, it.ID)
			trace.IgnoredLabels = append(trace.IgnoredLabels, label)
			trace.IgnoreReasons[label] = "not selected for this task"
			continue
		}

		trace.UsedMemoryIDs = append(trace.UsedMemoryIDs, it.ID)
		trace.UsedLabels = append(trace.UsedLabels, label)
		trace.UseReasons[label] = useReason(it)
		if contrib, ok := t.FactContributions[label]; ok {
			facts = append(facts, contrib...)
		}
	}

	facts = uniqueSorted(facts)
	sort.Strings(trace.UsedLabels)
	sort.Strings(trace.IgnoredLabels)
	sort.Strings(trace.MisusedLabels)
	return facts, trace
}

func refutedUtility(it recall.MemoryItem) bool {
	if it.UtilityScore != nil && *it.UtilityScore <= -6 {
		return true
	}
	return it.LifecycleRole == recall.LifecycleRefutedContext
}

func useReason(it recall.MemoryItem) string {
	switch it.LifecycleRole {
	case recall.LifecycleCurrentGuidance, "":
		return "current guidance matched task constraint"
	case recall.LifecycleHistoricalContext:
		return "historical context explicitly allowed for task"
	default:
		return "matched task selection rules"
	}
}

func memoryTagsForItem(it recall.MemoryItem, lc *LoadedCorpus) []string {
	// Tags are not on MemoryItem; recover from corpus object.
	id := it.ID
	for _, o := range lc.Objects {
		if o.ID.String() == id {
			return o.Tags
		}
	}
	return nil
}

func labelSet(labels []string) map[string]bool {
	out := map[string]bool{}
	for _, l := range labels {
		out[l] = true
	}
	return out
}

func uniqueSorted(in []string) []string {
	set := map[string]struct{}{}
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			set[s] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func hasAllFacts(got, want []string) bool {
	set := map[string]struct{}{}
	for _, g := range got {
		set[g] = struct{}{}
	}
	for _, w := range want {
		if _, ok := set[w]; !ok {
			return false
		}
	}
	return true
}

func hasAnyFact(got, forbidden []string) bool {
	set := map[string]struct{}{}
	for _, g := range got {
		set[g] = struct{}{}
	}
	for _, f := range forbidden {
		if _, ok := set[f]; ok {
			return true
		}
	}
	return false
}

func suppressedByFixtureScope(t TaskFixture, fm FixtureMemory) bool {
	neg := append([]string(nil), fm.NegativeScope...)
	if fm.EncodingCues != nil {
		neg = append(neg, fm.EncodingCues.NegativeScope...)
	}
	for _, ns := range neg {
		for _, dt := range t.DomainTags {
			if ns == dt {
				return true
			}
		}
	}
	return false
}

func isWrongScopeDecoy(t TaskFixture, label string) bool {
	return containsLabel(t.DecoyLabels, label) ||
		containsLabel(t.NearMissMemoryLabels, label) ||
		containsLabel(t.ExpectedIgnoredLabels, label)
}

func scopeAllowed(t TaskFixture, memTags []string) bool {
	if !tagFamilyMatch(t.DomainTags, memTags, "phase11c-project-") {
		return false
	}
	if !tagFamilyMatch(t.DomainTags, memTags, "phase11c-system-") {
		return false
	}
	return true
}

func tagFamilyMatch(taskTags, memTags []string, prefix string) bool {
	taskVal := tagWithPrefix(taskTags, prefix)
	memVal := tagWithPrefix(memTags, prefix)
	if taskVal == "" || memVal == "" {
		return true
	}
	return taskVal == memVal
}

func tagWithPrefix(tags []string, prefix string) string {
	for _, t := range tags {
		if strings.HasPrefix(t, prefix) {
			return t
		}
	}
	return ""
}
