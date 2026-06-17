package formation

import (
	"encoding/json"
	"fmt"
	"strings"

	"control-plane/internal/formationquality"
	"control-plane/pkg/api"
)

// Gate applies memory formation quality rules.
type Gate struct {
	cfg Config
}

// NewGate returns a gate with normalized config (nil-safe defaults).
func NewGate(cfg *Config) *Gate {
	c := DefaultConfig()
	if cfg != nil {
		c = *cfg
	}
	c.Normalize()
	return &Gate{cfg: c}
}

// Config returns the effective configuration.
func (g *Gate) Config() Config {
	if g == nil {
		return DefaultConfig()
	}
	return g.cfg
}

// EvaluateDirectCreate applies guardrails for POST /v1/memory and MCP memory_create.
func (g *Gate) EvaluateDirectCreate(path Path, kind api.MemoryKind, applicability api.Applicability, authority int, status api.Status, statement string, prov Provenance) Decision {
	if g == nil {
		return Decision{Outcome: OutcomeAllow}
	}
	dc := g.cfg.DirectCreate
	if !dc.Enabled {
		return Decision{Outcome: OutcomeReject, Reason: "direct memory create is disabled by policy"}
	}
	if !ValidDirectCreateKind(kind) {
		return Decision{Outcome: OutcomeReject, Reason: "invalid kind for direct create (experience and candidate are not allowed)"}
	}
	if dc.RejectJunk && IsJunkStatement(statement) {
		return Decision{Outcome: OutcomeReject, Reason: "statement rejected as junk memory"}
	}
	if applicability == "" {
		applicability = api.ApplicabilityAdvisory
	}
	if status == "" {
		status = api.StatusActive
	}

	highRisk := IsHighRiskWrite(kind, applicability, authority, dc.HighRiskAuthorityThreshold)
	governing := applicability == api.ApplicabilityGoverning || (kind == api.MemoryKindConstraint && applicability == api.ApplicabilityGoverning)

	// Authority cap for non-admin direct create
	if authority > dc.MaxClientAuthority {
		if highRisk || governing {
			return Decision{Outcome: OutcomePending, Reason: "authority exceeds client cap; requires review", CapAuthority: dc.MaxClientAuthority, ForcePending: true}
		}
		return Decision{Outcome: OutcomeAllow, Reason: "authority capped", CapAuthority: dc.MaxClientAuthority}
	}

	// Governing constraints cannot become active immediately
	if kind == api.MemoryKindConstraint && (governing || applicability == api.ApplicabilityGoverning) {
		if dc.RequireAdminForGoverning && status == api.StatusActive {
			return Decision{Outcome: OutcomePending, Reason: "governing constraint requires review before active recall", ForcePending: true}
		}
	}
	if applicability == api.ApplicabilityGoverning && status == api.StatusActive && dc.GoverningDefaultStatus == "pending" {
		return Decision{Outcome: OutcomePending, Reason: "governing memory default status is pending", ForcePending: true}
	}

	// Provenance for high-risk / governing
	needProv := (dc.RequireProvenanceForGoverning && (governing || kind == api.MemoryKindConstraint)) ||
		(authority >= dc.RequireProvenanceAuthorityGE)
	if needProv && !prov.HasMinimumForHighRisk() {
		if status == api.StatusActive {
			return Decision{Outcome: OutcomePending, Reason: "high-risk write missing provenance; pending review", ForcePending: true}
		}
	}

	// Block active high-authority governing without review
	if status == api.StatusActive && highRisk && (governing || authority >= dc.HighRiskAuthorityThreshold) {
		if !dc.AllowActiveHighRiskGoverning {
			return Decision{Outcome: OutcomePending, Reason: "high-risk active memory requires review", ForcePending: true}
		}
	}

	return Decision{Outcome: OutcomeAllow}
}

// EvaluatePromote applies guardrails for POST /v1/memory/promote.
func (g *Gate) EvaluatePromote(kind api.MemoryKind, authority int, requireReview bool, statement string, source string) Decision {
	prov := Provenance{Source: source}
	app := api.ApplicabilityAdvisory
	if kind == api.MemoryKindConstraint {
		app = api.ApplicabilityGoverning
	}
	status := api.StatusActive
	if requireReview {
		status = api.StatusPending
	}
	d := g.EvaluateDirectCreate(PathPromote, kind, app, authority, status, statement, prov)
	if d.Outcome == OutcomeAllow && kind == api.MemoryKindConstraint && !requireReview {
		if !g.cfg.DirectCreate.AllowActivePromotedGoverning {
			// Promote path sets governing on constraints — force pending unless already reviewed
			d = Decision{Outcome: OutcomePending, Reason: "promoted governing constraint requires review", ForcePending: true}
		}
	}
	return d
}

// EvaluateProbationaryCreate gates vet inline memory formation from record_experience.
func (g *Gate) EvaluateProbationaryCreate(kind api.MemoryKind, authority int, statement string) Decision {
	if g == nil {
		return Decision{Outcome: OutcomeAllow}
	}
	re := g.cfg.RecordExperience
	if re.RejectJunk && IsWeakRecordExperienceSummary(statement, re.MinActionableWords) {
		return Decision{Outcome: OutcomeReject, Reason: "summary rejected as junk or too vague for probationary memory"}
	}
	capAuth := authority
	if re.MaxProbationaryAuthority > 0 && capAuth > re.MaxProbationaryAuthority {
		capAuth = re.MaxProbationaryAuthority
	}
	d := Decision{Outcome: OutcomeAllow, CapAuthority: capAuth}
	if re.RiskyConstraintPending && kind == api.MemoryKindConstraint {
		d.ForcePending = true
		d.Outcome = OutcomePending
		d.Reason = "constraint from record_experience requires review before active recall"
		d.ForceAdvisory = true
	}
	return d
}

// RejectRecordExperienceSummary returns reject reason if summary should not form memory.
func (g *Gate) RejectRecordExperienceSummary(summary string) (reject bool, reason string) {
	if g == nil {
		return false, ""
	}
	if !g.cfg.RecordExperience.RejectJunk {
		return false, ""
	}
	if IsWeakRecordExperienceSummary(summary, g.cfg.RecordExperience.MinActionableWords) {
		return true, "junk_or_vague_summary"
	}
	return false, ""
}

// ApplyDecision mutates a create request according to the gate decision.
func ApplyDecision(req *CreateInput, d Decision) error {
	if req == nil {
		return nil
	}
	switch d.Outcome {
	case OutcomeReject:
		return &ErrRejected{Reason: d.Reason, Path: req.Path}
	case OutcomePending, OutcomeAllow:
		if d.CapAuthority > 0 && req.Authority > d.CapAuthority {
			req.Authority = d.CapAuthority
		}
		if d.ForcePending {
			req.Status = api.StatusPending
		}
		if d.ForceAdvisory {
			req.Applicability = api.ApplicabilityAdvisory
		}
	}
	return nil
}

// CreateInput is the subset of memory.CreateRequest used by the formation gate.
type CreateInput struct {
	Path          Path
	Kind          api.MemoryKind
	Authority     int
	Applicability api.Applicability
	Status        api.Status
	Statement     string
	Tags          []string
	Payload       []byte // optional JSON
}

// EvaluateCreateInput is the shared entry for memory service.
func (g *Gate) EvaluateCreateInput(in *CreateInput) (Decision, error) {
	if in == nil {
		return Decision{Outcome: OutcomeAllow}, nil
	}
	var prov Provenance
	if len(in.Payload) > 0 {
		raw := json.RawMessage(in.Payload)
		prov = ExtractProvenance(&raw, in.Tags)
	} else {
		prov = ExtractProvenance(nil, in.Tags)
	}
	switch in.Path {
	case PathProbationaryIngest:
		d := g.EvaluateProbationaryCreate(in.Kind, in.Authority, in.Statement)
		if err := g.applyQualityLayer(in, &d); err != nil {
			return d, err
		}
		if err := ApplyDecision(in, d); err != nil {
			return d, err
		}
		return d, nil
	case PathPromote:
		src := prov.Source
		d := g.EvaluatePromote(in.Kind, in.Authority, in.Status == api.StatusPending, in.Statement, src)
		if err := g.applyQualityLayer(in, &d); err != nil {
			return d, err
		}
		if err := ApplyDecision(in, d); err != nil {
			return d, err
		}
		return d, nil
	default:
		d := g.EvaluateDirectCreate(in.Path, in.Kind, in.Applicability, in.Authority, in.Status, in.Statement, prov)
		if err := g.applyQualityLayer(in, &d); err != nil {
			return d, err
		}
		if err := ApplyDecision(in, d); err != nil {
			return d, err
		}
		return d, nil
	}
}

// applyQualityLayer runs Phase 11D formation quality evaluation.
func (g *Gate) applyQualityLayer(in *CreateInput, d *Decision) error {
	if g == nil || in == nil || d == nil {
		return nil
	}
	if d.Outcome == OutcomeReject {
		return nil
	}
	preserveCap := d.CapAuthority
	forceAdvisory := d.ForceAdvisory
	eval := formationquality.NewEvaluator()
	qr := eval.Evaluate(QualityInputFromCreate(in))
	qrCopy := qr
	d.Quality = &qrCopy

	// Persist deterministic formation-quality results into the create payload so
	// recall can expose an agent-facing memory contract without re-running LLMs.
	persistQualityIntoPayload(in, qrCopy)

	switch qr.Decision {
	case formationquality.DecisionRejectGarbage:
		return &ErrRejected{Reason: string(qr.Decision), Path: in.Path}
	case formationquality.DecisionRejectDangerous:
		return &ErrRejected{Reason: string(qr.Decision), Path: in.Path}
	case formationquality.DecisionAcceptPending, formationquality.DecisionNeedsCuration:
		if g.cfg.DirectCreate.AllowActiveDespiteQualityReview {
			return nil
		}
		if g.cfg.DirectCreate.AllowActiveWithQualityWarnings &&
			qr.Decision == formationquality.DecisionAcceptPending &&
			!formationquality.HasHardDefects(qr) {
			return nil
		}
		applyQualityMutations(in, qr)
		quality := d.Quality
		*d = Decision{
			Outcome:       OutcomePending,
			Reason:        "formation quality: " + string(qr.Decision),
			ForcePending:  true,
			ForceAdvisory: forceAdvisory || qr.SuggestedApplicability == "advisory",
			CapAuthority:  preserveCap,
			Quality:       quality,
		}
	}
	return nil
}

func persistQualityIntoPayload(in *CreateInput, qr formationquality.Result) {
	if in == nil {
		return
	}
	// Always store under shared keys so recall can extract contract fields
	// deterministically. If payload is absent or non-object, replace it with an
	// object to avoid breaking downstream JSON parsing.
	var m map[string]any
	if len(in.Payload) > 0 {
		if err := json.Unmarshal(in.Payload, &m); err != nil || m == nil {
			m = map[string]any{}
		}
	} else {
		m = map[string]any{}
	}
	if m == nil {
		m = map[string]any{}
	}

	// Quality state/score are required for Phase 11F agent-contract enforcement.
	m["quality_score"] = qr.QualityScore
	m["quality_state"] = string(qr.Decision)
	m["quality_passed"] = qr.Passed
	m["safe_for_active_recall"] = qr.SafeForActiveRecall
	if m["schema_type"] == nil || strings.TrimSpace(asString(m["schema_type"])) == "" {
		// formationquality always has a schema_type via its evaluator.
		m["schema_type"] = string(qr.SchemaType)
	}

	defects := make([]map[string]any, 0, len(qr.Defects))
	for _, d := range qr.Defects {
		defects = append(defects, map[string]any{
			"code":     string(d.Code),
			"severity": string(d.Severity),
		})
	}
	warnings := make([]map[string]any, 0, len(qr.Warnings))
	for _, w := range qr.Warnings {
		warnings = append(warnings, map[string]any{
			"code":     string(w.Code),
			"severity": string(w.Severity),
		})
	}
	m["quality_defects"] = defects
	m["quality_warnings"] = warnings

	raw, err := json.Marshal(m)
	if err == nil {
		in.Payload = raw
	}
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

// ValidateAdvisorySummaryShared is used by MCP and REST advisory paths for parity.
func ValidateAdvisorySummaryShared(summary string, minRunes int, gate *Gate) error {
	s := strings.TrimSpace(summary)
	if len([]rune(s)) < minRunes {
		return fmt.Errorf("summary must be at least %d characters", minRunes)
	}
	if gate != nil {
		if reject, reason := gate.RejectRecordExperienceSummary(s); reject {
			return fmt.Errorf("summary rejected: %s", reason)
		}
	}
	return nil
}
