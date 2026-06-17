package formation

import (
	"encoding/json"
	"strings"

	"control-plane/pkg/api"
)

// Provenance captures minimum attribution for high-risk memory writes.
type Provenance struct {
	Source        string
	SourceTool    string
	SourceSession string
	ClientName    string
	CorrelationID string
	EvidenceID    string
	Reason        string
	RepoRoot      string
}

// ExtractProvenance reads provenance from payload JSON and tags.
func ExtractProvenance(payload *json.RawMessage, tags []string) Provenance {
	var p Provenance
	if payload != nil && len(*payload) > 0 {
		var m map[string]any
		if json.Unmarshal(*payload, &m) == nil {
			p.Source = stringField(m, "source")
			p.SourceTool = stringField(m, "source_tool")
			p.SourceSession = stringField(m, "source_session", "session_id")
			p.ClientName = stringField(m, "client_name")
			p.CorrelationID = stringField(m, "correlation_id")
			p.EvidenceID = stringField(m, "evidence_id")
			p.Reason = stringField(m, "reason")
			p.RepoRoot = stringField(m, "repo_root", "workspace")
			if prov, ok := m["provenance"].(map[string]any); ok {
				if p.Source == "" {
					p.Source = stringField(prov, "source")
				}
				if p.SourceTool == "" {
					p.SourceTool = stringField(prov, "source_tool")
				}
				if p.SourceSession == "" {
					p.SourceSession = stringField(prov, "source_session", "session_id")
				}
			}
			if ing, ok := m["pluribus_ingest"].(map[string]any); ok {
				if p.Source == "" {
					p.Source = "advisory_ingest"
				}
				if p.SourceSession == "" {
					p.SourceSession = stringField(ing, "advisory_experience_id")
				}
			}
		}
	}
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		lower := strings.ToLower(t)
		switch {
		case strings.HasPrefix(lower, "source:"):
			if p.Source == "" {
				p.Source = strings.TrimSpace(t[len("source:"):])
			}
		case strings.HasPrefix(lower, "session:"):
			if p.SourceSession == "" {
				p.SourceSession = strings.TrimSpace(t[len("session:"):])
			}
		case strings.HasPrefix(lower, "client:"):
			if p.ClientName == "" {
				p.ClientName = strings.TrimSpace(t[len("client:"):])
			}
		case strings.HasPrefix(lower, "correlation:"):
			if p.CorrelationID == "" {
				p.CorrelationID = strings.TrimSpace(t[len("correlation:"):])
			}
		case strings.HasPrefix(lower, "evidence:"):
			if p.EvidenceID == "" {
				p.EvidenceID = strings.TrimSpace(t[len("evidence:"):])
			}
		}
	}
	return p
}

func stringField(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					return s
				}
			}
		}
	}
	return ""
}

// HasMinimumForHighRisk returns true when enough provenance exists for governing/high-authority writes.
func (p Provenance) HasMinimumForHighRisk() bool {
	fields := 0
	if p.Source != "" {
		fields++
	}
	if p.SourceTool != "" {
		fields++
	}
	if p.SourceSession != "" {
		fields++
	}
	if p.ClientName != "" {
		fields++
	}
	if p.CorrelationID != "" {
		fields++
	}
	if p.EvidenceID != "" {
		fields++
	}
	if p.Reason != "" {
		fields++
	}
	if p.RepoRoot != "" {
		fields++
	}
	return fields >= 2
}

// IsHighRiskWrite reports whether the create request is high-risk for provenance/review rules.
func IsHighRiskWrite(kind api.MemoryKind, applicability api.Applicability, authority int, threshold int) bool {
	if threshold <= 0 {
		threshold = 8
	}
	if applicability == api.ApplicabilityGoverning {
		return true
	}
	if kind == api.MemoryKindConstraint && authority >= 5 {
		return true
	}
	if authority >= threshold {
		return true
	}
	return false
}
