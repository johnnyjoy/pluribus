package formation

import (
	"encoding/json"
	"strings"

	"control-plane/internal/formationquality"
	"control-plane/pkg/api"
)

// QualityInputFromCreate builds formation quality input from create input.
func QualityInputFromCreate(in *CreateInput) formationquality.Input {
	if in == nil {
		return formationquality.Input{}
	}
	qi := formationquality.Input{
		Path:          qualityPathForEval(in.Path),
		Kind:          string(in.Kind),
		Statement:     in.Statement,
		Authority:     in.Authority,
		Applicability: string(in.Applicability),
		Status:        string(in.Status),
		Tags:          append([]string(nil), in.Tags...),
	}
	if len(in.Payload) > 0 {
		var m map[string]any
		if json.Unmarshal(in.Payload, &m) == nil {
			qi.SchemaType = qualityStrField(m, "schema_type", "memory_schema_type")
			qi.Scope = qualityStrField(m, "scope")
			qi.UseInstruction = qualityStrField(m, "use_instruction")
			qi.MisuseWarning = qualityStrField(m, "misuse_warning")
			qi.SourceType = qualityStrField(m, "source_type")
			qi.AuthorityBasis = qualityStrField(m, "authority_basis")
			qi.LifecycleRole = qualityStrField(m, "lifecycle_role")
			qi.OccurredAt = qualityStrField(m, "occurred_at")
			qi.Reason = qualityStrField(m, "reason")
			qi.NegativeScope = qualityStrSlice(m, "negative_scope")
			qi.RetrievalCues = qualityStrSlice(m, "retrieval_cues", "retrieval_terms")
		}
	}
	if len(in.Payload) > 0 {
		raw := json.RawMessage(in.Payload)
		prov := ExtractProvenance(&raw, in.Tags)
		qi.ProvenanceFields = countQualityProv(prov)
	} else {
		qi.ProvenanceFields = countQualityProv(ExtractProvenance(nil, in.Tags))
	}
	return qi
}

func qualityPathForEval(p Path) string {
	switch p {
	case PathProbationaryIngest:
		return "record_experience"
	default:
		return string(p)
	}
}

func qualityStrField(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func qualityStrSlice(m map[string]any, keys ...string) []string {
	for _, k := range keys {
		v, ok := m[k]
		if !ok {
			continue
		}
		switch arr := v.(type) {
		case []any:
			var out []string
			for _, item := range arr {
				if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
					out = append(out, strings.TrimSpace(s))
				}
			}
			return out
		case []string:
			return arr
		}
	}
	return nil
}

func countQualityProv(p Provenance) int {
	n := 0
	if p.Source != "" {
		n++
	}
	if p.SourceTool != "" {
		n++
	}
	if p.SourceSession != "" {
		n++
	}
	if p.ClientName != "" {
		n++
	}
	if p.CorrelationID != "" {
		n++
	}
	if p.EvidenceID != "" {
		n++
	}
	if p.Reason != "" {
		n++
	}
	if p.RepoRoot != "" {
		n++
	}
	return n
}

func applyQualityMutations(in *CreateInput, qr formationquality.Result, hiveActive bool) {
	if in == nil {
		return
	}
	if qr.SuggestedStatus == "pending" && !hiveActive {
		in.Status = api.StatusPending
	}
	if qr.SuggestedApplicability != "" {
		in.Applicability = api.Applicability(qr.SuggestedApplicability)
	}
	for _, d := range qr.Defects {
		if d.Code == formationquality.DefectAgentInferredHighAuthority && in.Authority > 2 {
			in.Authority = 2
		}
	}
}
