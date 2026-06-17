package agenttelemetry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"control-plane/internal/agentobedience"

	"github.com/google/uuid"
)

var (
	ErrUnknownSession      = errors.New("unknown session")
	ErrUnknownRecall       = errors.New("unknown recall event")
	ErrUnknownOutput       = errors.New("unknown output event")
	ErrMalformedPayload    = errors.New("malformed payload")
	ErrSelfReportBypass    = errors.New("self-reported obedience rejected; evaluator required")
	ErrWrongSessionRecall  = errors.New("recall_event_id does not belong to session")
)

// Service persists and evaluates agent memory-use telemetry.
type Service struct {
	Repo *Repo
	mem  *memStore
}

// NewService creates telemetry service (in-memory when Repo.DB is nil).
func NewService() *Service {
	return &Service{mem: newMemStore()}
}

// NewServiceWithRepo wires Postgres persistence when db is non-nil.
func NewServiceWithRepo(repo *Repo) *Service {
	s := NewService()
	s.Repo = repo
	return s
}

func parseUUID(s string) (uuid.UUID, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return uuid.Nil, fmt.Errorf("empty id")
	}
	return uuid.Parse(s)
}

// StartSession begins a telemetry session.
func (s *Service) StartSession(ctx context.Context, req StartSessionRequest) (*TelemetrySession, error) {
	iface := strings.TrimSpace(req.Interface)
	if iface != agentobedience.InterfaceREST && iface != agentobedience.InterfaceMCP {
		return nil, fmt.Errorf("%w: interface must be rest or mcp", ErrMalformedPayload)
	}
	var id uuid.UUID
	if req.SessionID != "" {
		var err error
		id, err = parseUUID(req.SessionID)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrMalformedPayload, err)
		}
	} else {
		id = uuid.New()
	}
	now := time.Now().UTC()
	sess := TelemetrySession{
		ID:         id,
		StartedAt:  now,
		Interface:  iface,
		AgentID:    req.AgentID,
		ClientName: req.ClientName,
		Tags:       append([]string(nil), req.Tags...),
		Metadata:   req.Metadata,
	}
	if err := s.insertSession(ctx, sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

// RecordRecall persists a recall exposure event.
func (s *Service) RecordRecall(ctx context.Context, req RecordRecallRequest) (*RecallEvent, error) {
	sid, err := parseUUID(req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("%w: session_id", ErrMalformedPayload)
	}
	if _, ok := s.getSession(ctx, sid); !ok {
		return nil, ErrUnknownSession
	}
	if len(req.RecalledMemoryIDs) == 0 {
		return nil, fmt.Errorf("%w: recalled_memory_ids required", ErrMalformedPayload)
	}
	for _, mid := range req.RecalledMemoryIDs {
		if err := validateMemoryID(mid); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrMalformedPayload, err)
		}
	}
	iface := strings.TrimSpace(req.Interface)
	if iface == "" {
		iface = agentobedience.InterfaceREST
	}
	mode := req.RecallMode
	if mode == "" {
		mode = "current"
	}
	ev := RecallEvent{
		ID:                uuid.New(),
		SessionID:         sid,
		TaskID:            req.TaskID,
		Interface:         iface,
		RecallRequestJSON: req.RecallRequest,
		RecallBundleID:    req.RecallBundleID,
		RecalledMemoryIDs: append([]string(nil), req.RecalledMemoryIDs...),
		RecallBundleJSON:  req.RecallBundle,
		RecallMode:        mode,
		CreatedAt:         time.Now().UTC(),
	}
	if err := s.insertRecall(ctx, ev); err != nil {
		return nil, err
	}
	return &ev, nil
}

// RecordDecision persists memory use decisions.
func (s *Service) RecordDecision(ctx context.Context, req RecordDecisionRequest) ([]MemoryDecisionRow, error) {
	sid, err := parseUUID(req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("%w: session_id", ErrMalformedPayload)
	}
	rid, err := parseUUID(req.RecallEventID)
	if err != nil {
		return nil, fmt.Errorf("%w: recall_event_id", ErrMalformedPayload)
	}
	recall, ok := s.getRecall(ctx, rid)
	if !ok {
		return nil, ErrUnknownRecall
	}
	if recall.SessionID != sid {
		return nil, ErrWrongSessionRecall
	}
	if len(req.Decisions) == 0 {
		return nil, fmt.Errorf("%w: decisions required", ErrMalformedPayload)
	}
	now := time.Now().UTC()
	var rows []MemoryDecisionRow
	for _, d := range req.Decisions {
		if err := validateMemoryID(d.MemoryID); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrMalformedPayload, err)
		}
		if d.Decision == "used" && len(d.ContractFieldsCited) == 0 {
			return nil, fmt.Errorf("%w: contract_fields_cited required for used", ErrMalformedPayload)
		}
		rows = append(rows, MemoryDecisionRow{
			ID:                   uuid.New(),
			RecallEventID:        rid,
			MemoryID:             d.MemoryID,
			Decision:             d.Decision,
			Reason:               d.Reason,
			ContractFieldsCited:  append([]string(nil), d.ContractFieldsCited...),
			OutputFactsSupported: append([]string(nil), d.OutputFactsSupported...),
			CreatedAt:            now,
		})
	}
	// subset check — unrecalled use recorded as violation code on decision
	usedIDs := []string{}
	for _, d := range rows {
		if d.Decision == "used" {
			usedIDs = append(usedIDs, d.MemoryID)
		}
	}
	if !validateUsedSubset(recall.RecalledMemoryIDs, usedIDs) {
		for i := range rows {
			if rows[i].Decision == "used" {
				rows[i].ViolationCodes = append(rows[i].ViolationCodes, agentobedience.ViolationUsedUnrecalledMemory)
			}
		}
	}
	if err := s.insertDecisions(ctx, rid, rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// RecordOutput persists agent output.
func (s *Service) RecordOutput(ctx context.Context, req RecordOutputRequest) (*OutputEvent, error) {
	sid, err := parseUUID(req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("%w: session_id", ErrMalformedPayload)
	}
	if _, ok := s.getSession(ctx, sid); !ok {
		return nil, ErrUnknownSession
	}
	var rid uuid.UUID
	if req.RecallEventID != "" {
		rid, err = parseUUID(req.RecallEventID)
		if err != nil {
			return nil, fmt.Errorf("%w: recall_event_id", ErrMalformedPayload)
		}
		rec, ok := s.getRecall(ctx, rid)
		if !ok {
			return nil, ErrUnknownRecall
		}
		if rec.SessionID != sid {
			return nil, ErrWrongSessionRecall
		}
	}
	if len(req.OutputFacts) == 0 && len(req.OutputActions) == 0 {
		return nil, fmt.Errorf("%w: output_facts or output_actions required", ErrMalformedPayload)
	}
	for _, mid := range req.MemoryCitations {
		if err := validateMemoryID(mid); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrMalformedPayload, err)
		}
	}
	out := OutputEvent{
		ID:              uuid.New(),
		SessionID:       sid,
		TaskID:          req.TaskID,
		RecallEventID:   rid,
		OutputFacts:     append([]string(nil), req.OutputFacts...),
		OutputActions:   append([]string(nil), req.OutputActions...),
		MemoryCitations: append([]string(nil), req.MemoryCitations...),
		CreatedAt:       time.Now().UTC(),
	}
	if err := s.insertOutput(ctx, out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Evaluate runs deterministic obedience evaluator and persists results.
func (s *Service) Evaluate(ctx context.Context, req EvaluateRequest) (*EvaluateResponse, error) {
	sid, err := parseUUID(req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("%w: session_id", ErrMalformedPayload)
	}
	sess, ok := s.getSession(ctx, sid)
	if !ok {
		return nil, ErrUnknownSession
	}
	rid, err := parseUUID(req.RecallEventID)
	if err != nil {
		return nil, fmt.Errorf("%w: recall_event_id", ErrMalformedPayload)
	}
	recall, ok := s.getRecall(ctx, rid)
	if !ok {
		return nil, ErrUnknownRecall
	}
	if recall.SessionID != sid {
		return nil, ErrWrongSessionRecall
	}
	decisions := s.listDecisionsByRecall(ctx, rid)
	if len(decisions) == 0 {
		return nil, fmt.Errorf("%w: decisions required before evaluate", ErrMalformedPayload)
	}
	var output *OutputEvent
	if req.OutputID != "" {
		oid, err := parseUUID(req.OutputID)
		if err != nil {
			return nil, fmt.Errorf("%w: output_id", ErrMalformedPayload)
		}
		output, ok = s.getOutput(ctx, oid)
		if !ok {
			return nil, ErrUnknownOutput
		}
	} else {
		outs := s.listOutputsBySession(ctx, sid)
		for i := range outs {
			if outs[i].RecallEventID == rid || req.TaskID == "" || outs[i].TaskID == req.TaskID {
				output = &outs[i]
				break
			}
		}
	}
	if output == nil {
		return nil, fmt.Errorf("%w: output required before evaluate", ErrMalformedPayload)
	}

	tel := buildTelemetryFromPersisted(*sess, *recall, decisions, output)
	tel.TelemetryComplete = true

	bundleMemories := bundleFromRecallJSON(recall.RecallBundleJSON)
	agentMode := strings.TrimSpace(req.AgentMode)
	if agentMode == "" {
		agentMode = agentobedience.AgentObedient
	}
	oc := agentobedience.ObedienceCase{
		ID:                         "telemetry-" + rid.String(),
		TaskID:                     req.TaskID,
		Interface:                  recall.Interface,
		AgentMode:                  agentMode,
		ViolationBehaviors:         append([]string(nil), req.ViolationBehaviors...),
		InputMemories:              bundleMemories,
		TaskTags:                   taskTagsFromRequest(req),
		ExpectedOutputFacts:        req.ExpectedFacts,
		ForbiddenOutputFacts:       req.ForbiddenFacts,
		RequiredConstraintMemoryID: "",
	}
	rb := agentobedience.BundleFromCase(oc)
	tel.RubricResult = agentobedience.EvaluateRubric(tel.FinalOutput.Facts, req.ExpectedFacts, req.ForbiddenFacts)
	eval := agentobedience.EvaluateObedience(oc, rb, tel)

	rejectedSelf := false
	if req.ObediencePassed != nil && *req.ObediencePassed && !eval.ObediencePassed {
		rejectedSelf = true
	}

	now := time.Now().UTC()
	evalRow := ObedienceEvaluationRow{
		ID:               uuid.New(),
		SessionID:        sid,
		TaskID:           req.TaskID,
		RecallEventID:    rid,
		OutputID:         output.ID,
		ObediencePassed:  eval.ObediencePassed,
		ObedienceScore:   eval.ObedienceScore,
		Violations:       append([]string(nil), eval.Violations...),
		EvaluatorVersion: EvaluatorVersion,
		CreatedAt:        now,
	}
	vrows := violationsToRows(evalRow.ID.String(), eval.Violations, tel)
	for i := range vrows {
		vrows[i].ID = uuid.New()
		vrows[i].CreatedAt = now
	}
	cands := generateUtilityCandidates(evalRow.ID.String(), eval, tel)
	for i := range cands {
		cands[i].ID = uuid.New()
		cands[i].CreatedAt = now
	}
	if s.usePostgres() {
		if err := s.Repo.EvaluateTransactional(ctx, evalRow, vrows, cands); err != nil {
			return nil, err
		}
	} else {
		if err := s.insertEval(ctx, evalRow); err != nil {
			return nil, err
		}
		if err := s.insertViolations(ctx, evalRow.ID, vrows); err != nil {
			return nil, err
		}
		if err := s.insertCandidates(ctx, evalRow.ID, cands); err != nil {
			return nil, err
		}
	}

	return &EvaluateResponse{
		Evaluation:                  evalRow,
		Violations:                  vrows,
		UtilityCandidates:           cands,
		EvaluatorRejectedSelfReport: rejectedSelf,
	}, nil
}

func constraintFromMemories(bundle []agentobedience.CaseMemory) string {
	for _, m := range bundle {
		if m.MisuseWarning != "" || m.UseInstruction != "" {
			return m.MemoryID
		}
	}
	if len(bundle) > 0 {
		return bundle[0].MemoryID
	}
	return ""
}

func taskTagsFromRequest(req EvaluateRequest) []string {
	return append([]string(nil), req.TaskTags...)
}

// GetSessionSummary returns full session telemetry.
func (s *Service) GetSessionSummary(ctx context.Context, sessionID string) (*SessionSummary, error) {
	sid, err := parseUUID(sessionID)
	if err != nil {
		return nil, fmt.Errorf("%w: session_id", ErrMalformedPayload)
	}
	sum := s.sessionSummary(ctx, sid)
	if sum == nil {
		return nil, ErrUnknownSession
	}
	return sum, nil
}

// GetMemorySummary returns per-memory aggregates.
func (s *Service) GetMemorySummary(ctx context.Context, memoryID string) (*MemorySummary, error) {
	if err := validateMemoryID(memoryID); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMalformedPayload, err)
	}
	sum := s.memorySummary(ctx, memoryID)
	return &sum, nil
}

// ListViolations returns violations with optional filters.
func (s *Service) ListViolations(ctx context.Context, memoryID, code string) ([]ViolationRow, error) {
	all := s.listAllViolations(ctx)
	var out []ViolationRow
	for _, v := range all {
		if memoryID != "" && v.MemoryID != memoryID {
			continue
		}
		if code != "" && v.ViolationCode != code {
			continue
		}
		out = append(out, v)
	}
	return out, nil
}

// ListUtilityCandidates returns utility candidates with optional memory filter.
func (s *Service) ListUtilityCandidates(ctx context.Context, memoryID string) ([]UtilityCandidate, error) {
	all := s.listAllCandidates(ctx)
	var out []UtilityCandidate
	for _, c := range all {
		if memoryID != "" && c.MemoryID != memoryID {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

// HasSession reports whether session exists (for tests).
func (s *Service) HasSession(ctx context.Context, sessionID string) bool {
	sid, err := parseUUID(sessionID)
	if err != nil {
		return false
	}
	_, ok := s.getSession(ctx, sid)
	return ok
}
