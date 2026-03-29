package recall

import (
	"strings"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

// BuildRecallCandidate computes transferability and contradiction status for RIU scoring.
func BuildRecallCandidate(obj memory.MemoryObject, domainTags, reqTags []string, unresolved map[uuid.UUID]bool) RecallCandidate {
	c := RecallCandidate{Object: obj}
	c.Transferable = transferableEligibility(obj.Tags, reqTags, domainTags)
	if unresolved != nil && unresolved[obj.ID] {
		c.ContradictionStatus = "unresolved"
	} else {
		c.ContradictionStatus = "none"
	}
	return c
}

func transferableEligibility(memTags, reqTags, domainTags []string) bool {
	return normalizedTagOverlap(memTags, reqTags) > 0 || domainTagOverlap(memTags, domainTags) > 0
}

// ApplicabilityEnumWeight maps applicability enum to 0..1 for RIU (deterministic).
func ApplicabilityEnumWeight(a api.Applicability) float64 {
	switch a {
	case api.ApplicabilityGoverning:
		return 1.0
	case api.ApplicabilityAdvisory:
		return 0.75
	case api.ApplicabilityAnalogical:
		return 0.5
	case api.ApplicabilityExperimental:
		return 0.25
	default:
		return 0.55
	}
}

// normalizedTagOverlap is 0..1: fraction of request tags present on memory; 0 when reqTags empty (RIU does not boost everything).
func normalizedTagOverlap(memTags, reqTags []string) float64 {
	if len(reqTags) == 0 {
		return 0
	}
	set := make(map[string]struct{}, len(memTags))
	for _, t := range memTags {
		set[strings.TrimSpace(t)] = struct{}{}
	}
	var overlap int
	for _, t := range reqTags {
		if _, ok := set[strings.TrimSpace(t)]; ok {
			overlap++
		}
	}
	return float64(overlap) / float64(len(reqTags))
}

// domainTagOverlap is 0..1: fraction of domain tags hit by memory tags.
func domainTagOverlap(memTags, domainTags []string) float64 {
	if len(domainTags) == 0 {
		return 0
	}
	set := make(map[string]struct{}, len(memTags))
	for _, t := range memTags {
		set[strings.TrimSpace(t)] = struct{}{}
	}
	var hit int
	for _, d := range domainTags {
		if _, ok := set[strings.TrimSpace(d)]; ok {
			hit++
		}
	}
	return float64(hit) / float64(len(domainTags))
}

// transferableScore combines tag and domain alignment (0..1).
func transferableScore(memTags, reqTags, domainTags []string) float64 {
	t := normalizedTagOverlap(memTags, reqTags)
	d := domainTagOverlap(memTags, domainTags)
	if t <= 0 && d <= 0 {
		if len(reqTags) == 0 && len(domainTags) == 0 {
			return 0.35
		}
		return 0
	}
	if t > 0 && d > 0 {
		return min(1.0, (t+d)/2)
	}
	if t > 0 {
		return t
	}
	return d
}

// applicabilityComponent is the weighted tag+enum applicability term for RIU (additive scale).
func applicabilityComponent(obj memory.MemoryObject, reqTags []string, w RIUWeights) float64 {
	// Align with tagMatchScore: empty request tags mean "no tag filter" — do not zero the tag
	// component (normalizedTagOverlap returns 0 when reqTags is empty, which underranked tagged
	// same-project memories on untagged GET /v1/recall/ vs tagged recall).
	var tagPart float64
	if len(reqTags) == 0 {
		tagPart = 1.0
	} else {
		tagPart = normalizedTagOverlap(obj.Tags, reqTags)
	}
	enumPart := ApplicabilityEnumWeight(obj.Applicability)
	// Blend: both tag relevance and declared applicability matter.
	blend := 0.55*tagPart + 0.45*enumPart
	return w.Applicability * blend
}
