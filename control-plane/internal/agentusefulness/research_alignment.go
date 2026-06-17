package agentusefulness

// ResearchSource documents a cited research anchor for cognitive memory engineering.
type ResearchSource struct {
	ID          string   `json:"id"`
	Authors     string   `json:"authors"`
	Year        int      `json:"year,omitempty"`
	Title       string   `json:"title"`
	URL         string   `json:"url,omitempty"`
	Principles  []string `json:"principles"`
	Engineering string   `json:"engineering_interpretation"`
}

// Research principle identifiers used in fixtures and metrics.
const (
	PrincipleEncodingSpecificity   = "encoding_specificity"
	PrincipleSchemaMemory          = "schema_memory"
	PrincipleRetrievalPractice     = "retrieval_practice"
	PrincipleContextDependent      = "context_dependent_recall"
	PrincipleInterferenceControl   = "interference_control"
	PrincipleReconstructiveRisk    = "reconstructive_memory_risk"
	PrincipleExperienceFollowing   = "experience_following_risk"
	PrincipleLifecycleGovernance   = "memory_lifecycle_governance"
)

// DocumentedResearchSources returns the Phase 11C research anchors with citations.
// Engineering interpretations are Pluribus-specific; research claims are source-backed.
func DocumentedResearchSources() []ResearchSource {
	return []ResearchSource{
		{
			ID: "tulving_thomson_1973", Authors: "Tulving & Thomson", Year: 1973,
			Title: "Encoding specificity and retrieval processes in episodic memory",
			URL:   "https://doi.org/10.1037/h0020071",
			Principles: []string{PrincipleEncodingSpecificity, PrincipleContextDependent},
			Engineering: "Retrieval cues must overlap encoding cues; Pluribus memories need explicit retrieval_terms, scope, and domain tags.",
		},
		{
			ID: "roediger_karpicke_2006", Authors: "Roediger & Karpicke", Year: 2006,
			Title: "Test-enhanced learning: Taking memory tests improves long-term retention",
			URL:   "https://doi.org/10.1111/j.1467-9280.2006.01693.x",
			Principles: []string{PrincipleRetrievalPractice},
			Engineering: "Recall alone is insufficient; harness distinguishes recalled, used, helpful, harmful, and ignored.",
		},
		{
			ID: "bartlett_1932", Authors: "Bartlett", Year: 1932,
			Title: "Remembering: A Study in Experimental and Social Psychology (schema theory)",
			URL:   "https://en.wikipedia.org/wiki/Schema_(psychology)",
			Principles: []string{PrincipleSchemaMemory, PrincipleReconstructiveRisk},
			Engineering: "Memories need schema types (constraint, failure, procedure); agents can misapply recalled text without metadata.",
		},
		{
			ID: "xiong_et_al_2025", Authors: "Xiong et al.", Year: 2025,
			Title: "How Memory Management Impacts LLM Agents: Experience-Following Behavior",
			URL:   "https://arxiv.org/abs/2505.16067",
			Principles: []string{PrincipleExperienceFollowing, PrincipleInterferenceControl},
			Engineering: "Similar retrieved experiences can replay errors; Pluribus must suppress bad/stale near-miss memories.",
		},
		{
			ID: "zhang_et_al_2024", Authors: "Zhang et al.", Year: 2024,
			Title: "A Survey on the Memory Mechanism of Large Language Model based Agents",
			URL:   "https://arxiv.org/abs/2404.13534",
			Principles: []string{PrincipleLifecycleGovernance, PrincipleSchemaMemory},
			Engineering: "Agent memory needs lifecycle governance: write, store, retrieve, use, demote, recover.",
		},
	}
}

// RequiredResearchPrinciples returns principles that must have fixture coverage.
func RequiredResearchPrinciples() []string {
	return []string{
		PrincipleEncodingSpecificity,
		PrincipleSchemaMemory,
		PrincipleRetrievalPractice,
		PrincipleContextDependent,
		PrincipleInterferenceControl,
		PrincipleReconstructiveRisk,
		PrincipleExperienceFollowing,
		PrincipleLifecycleGovernance,
	}
}

// PrincipleForTask returns the research principle tag on a task fixture.
func PrincipleForTask(t TaskFixture) string {
	return t.ResearchPrinciple
}

// CoveredPrinciples returns unique principles present in task fixtures.
func CoveredPrinciples(tasks []TaskFixture) map[string]int {
	out := map[string]int{}
	for _, t := range tasks {
		if p := stringsTrim(t.ResearchPrinciple); p != "" {
			out[p]++
		}
	}
	return out
}

func stringsTrim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
