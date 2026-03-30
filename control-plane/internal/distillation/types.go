package distillation

// DistillRequest is POST /v1/episodes/distill. Provide episode_id (loads advisory row) or summary (inline text).
type DistillRequest struct {
	EpisodeID string   `json:"episode_id,omitempty"`
	Summary   string   `json:"summary,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Entities  []string `json:"entities,omitempty"`
	// OriginDistill is set only by in-process callers (e.g. advisory ingest). HTTP clients should omit; handler forces "manual".
	OriginDistill string `json:"-"`
}

// DistillResponse lists created candidate_events rows (pending curation; not canonical memory).
type DistillResponse struct {
	Candidates []DistillCandidateOut `json:"candidates"`
}

// DistillCandidateOut is one row written or merged in candidate_events with structured proposal_json.
type DistillCandidateOut struct {
	CandidateID               string   `json:"candidate_id"`
	Kind                      string   `json:"kind"`
	Statement                 string   `json:"statement"`
	Reason                    string   `json:"reason"`
	Tags                      []string `json:"tags,omitempty"`
	SourceAdvisoryEpisodeID   string   `json:"source_advisory_episode_id,omitempty"`
	SourceAdvisoryEpisodeIDs  []string `json:"source_advisory_episode_ids,omitempty"`
	SalienceScore             float64  `json:"salience_score"`
	DistillSupportCount       int      `json:"distill_support_count,omitempty"`
	Merged                    bool     `json:"merged,omitempty"`
}
