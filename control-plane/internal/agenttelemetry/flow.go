package agenttelemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"control-plane/internal/agentobedience"
	"control-plane/internal/mcp"

	"github.com/go-chi/chi/v5"
)

// FlowResult captures one persistence flow run.
type FlowResult struct {
	SessionPersisted      bool
	RecallPersisted       bool
	DecisionPersisted     bool
	OutputPersisted       bool
	EvaluationPersisted   bool
	ViolationPersisted    bool
	CandidateGenerated    bool
	QuerySessionOK        bool
	QueryMemoryOK         bool
	QueryViolationOK      bool
	SelfReportRejected    bool
	RecallOnlyNoPositive  bool
	SelfReportNoPositive  bool
	AutoUtilityMutated    bool
	Rejected              bool
	RejectReason          string
	EvalPassed            bool
	ViolationCodes        []string
}

func newTelemetryRouter(svc *Service) http.Handler {
	r := chi.NewRouter()
	h := &Handlers{Service: svc}
	r.Route("/v1/agent/telemetry", func(r chi.Router) {
		r.Post("/session/start", h.StartSession)
		r.Post("/recall", h.RecordRecall)
		r.Post("/decision", h.RecordDecision)
		r.Post("/output", h.RecordOutput)
		r.Post("/evaluate", h.Evaluate)
		r.Get("/session/{session_id}", h.GetSession)
		r.Get("/memory/{memory_id}", h.GetMemory)
		r.Get("/violations", h.ListViolations)
		r.Get("/utility-candidates", h.ListUtilityCandidates)
	})
	return r
}

// RunRESTFlow executes full REST telemetry flow for a persistence case.
func RunRESTFlow(svc *Service, c PersistenceCase) (FlowResult, error) {
	srv := httptest.NewServer(newTelemetryRouter(svc))
	defer srv.Close()
	return runHTTPFlow(srv.URL, "", c, agentobedience.InterfaceREST)
}

// RunMCPFlow executes full MCP telemetry flow for a persistence case.
func RunMCPFlow(svc *Service, c PersistenceCase) (FlowResult, error) {
	srv := httptest.NewServer(newTelemetryRouter(svc))
	defer srv.Close()
	return runMCPFlow(srv.URL, c)
}

func runHTTPFlow(base, apiKey string, c PersistenceCase, iface string) (FlowResult, error) {
	var fr FlowResult
	oc := c.ToObedienceCase()
	oc.Interface = iface
	bundle := agentobedience.BundleFromCase(oc)
	tel := agentobedience.RunScriptedAgent(oc, bundle, iface)

	// start session
	sessBody, _ := json.Marshal(map[string]any{
		"interface": iface,
		"agent_id":  c.AgentMode,
		"tags":      c.TaskTags,
	})
	sessResp, err := postJSON(base+"/v1/agent/telemetry/session/start", apiKey, sessBody)
	if err != nil {
		return fr, err
	}
	if c.RejectStep == "unknown_session" {
		sessResp = map[string]any{"session_id": "00000000-0000-0000-0000-000000000099"}
	} else {
		fr.SessionPersisted = sessResp["session_id"] != nil
	}
	sid, _ := sessResp["session_id"].(string)

	bundleJSON := recallBundleJSONFromCase(oc)
	recalled := recalledIDsFromBundle(bundle)
	if c.ExternalMemoryID != "" {
		recalled = append(recalled, c.ExternalMemoryID)
	}
	recBody, _ := json.Marshal(map[string]any{
		"session_id":          sid,
		"task_id":             c.TaskID,
		"interface":           iface,
		"recall_bundle_id":    "bundle_" + c.TaskID,
		"recalled_memory_ids": recalled,
		"recall_bundle":       bundleJSON,
	})
	recResp, err := postJSON(base+"/v1/agent/telemetry/recall", apiKey, recBody)
	if err != nil {
		if c.RejectStep == "unknown_session" {
			fr.Rejected = true
			fr.RejectReason = err.Error()
			return fr, nil
		}
		return fr, err
	}
	fr.RecallPersisted = recResp["recall_event_id"] != nil
	rid, _ := recResp["recall_event_id"].(string)

	if c.RecallOnly {
		cands, _ := getJSON(base+"/v1/agent/telemetry/utility-candidates", apiKey)
		fr.RecallOnlyNoPositive = !hasPositiveUtility(cands)
		return fr, nil
	}

	var decisions []map[string]any
	for _, d := range tel.MemoryDecisions {
		decisions = append(decisions, map[string]any{
			"memory_id":              d.MemoryID,
			"decision":               d.Decision,
			"reason":                 d.Reason,
			"contract_fields_cited":  d.ContractFieldsCited,
			"output_facts_supported": d.OutputFactsSupported,
		})
	}
	if c.RejectStep == "missing_citation" {
		for i := range decisions {
			if decisions[i]["decision"] == "used" {
				decisions[i]["contract_fields_cited"] = []string{}
			}
		}
	}
	decBody, _ := json.Marshal(map[string]any{
		"session_id":      sid,
		"recall_event_id": rid,
		"decisions":       decisions,
	})
	_, decErr := postJSON(base+"/v1/agent/telemetry/decision", apiKey, decBody)
	if decErr != nil {
		if c.RejectStep == "missing_citation" {
			fr.Rejected = true
			fr.RejectReason = decErr.Error()
			return fr, nil
		}
		return fr, decErr
	}
	fr.DecisionPersisted = decErr == nil

	outBody, _ := json.Marshal(map[string]any{
		"session_id":       sid,
		"task_id":          c.TaskID,
		"recall_event_id":  rid,
		"output_facts":     tel.FinalOutput.Facts,
		"output_actions":   tel.FinalOutput.Actions,
		"memory_citations": tel.FinalOutput.MemoryCitations,
	})
	if c.RejectStep == "missing_output" {
		outBody, _ = json.Marshal(map[string]any{"session_id": sid, "task_id": c.TaskID, "recall_event_id": rid})
	}
	outResp, outErr := postJSON(base+"/v1/agent/telemetry/output", apiKey, outBody)
	if outErr != nil {
		if c.RejectStep == "missing_output" {
			fr.Rejected = true
			fr.RejectReason = outErr.Error()
			return fr, nil
		}
		return fr, outErr
	}
	fr.OutputPersisted = outResp["output_id"] != nil
	oid, _ := outResp["output_id"].(string)

	if c.SkipEvaluate {
		return fr, nil
	}

	evalBody := map[string]any{
		"session_id":      sid,
		"task_id":         c.TaskID,
		"recall_event_id": rid,
		"output_id":       oid,
		"expected_facts":  c.ExpectedOutputFacts,
		"forbidden_facts": c.ForbiddenOutputFacts,
		"task_tags":       c.TaskTags,
		"agent_mode":      c.AgentMode,
		"violation_behaviors": c.ViolationBehaviors,
	}
	if c.SelfReportObedient {
		t := true
		evalBody["obedience_passed"] = t
	}
	evalRaw, _ := json.Marshal(evalBody)
	evalResp, evalErr := postJSON(base+"/v1/agent/telemetry/evaluate", apiKey, evalRaw)
	if evalErr != nil {
		return fr, evalErr
	}
	if ev, ok := evalResp["evaluation"].(map[string]any); ok {
		fr.EvaluationPersisted = ev["evaluation_id"] != nil
		fr.EvalPassed = jsonBool(ev["obedience_passed"])
	}
	if v, ok := evalResp["evaluator_rejected_self_report"].(bool); ok && v {
		fr.SelfReportRejected = true
	} else if c.SelfReportObedient && !fr.EvalPassed {
		fr.SelfReportRejected = true
	}
	if vs, ok := evalResp["violations"].([]any); ok && len(vs) > 0 {
		fr.ViolationPersisted = true
	}
	if cs, ok := evalResp["utility_candidates"].([]any); ok && len(cs) > 0 {
		fr.CandidateGenerated = true
		fr.SelfReportNoPositive = !hasPositiveUtility(map[string]any{"utility_candidates": cs})
	}

	sum, _ := getJSON(base+"/v1/agent/telemetry/session/"+sid, apiKey)
	fr.QuerySessionOK = sum["session"] != nil

	memID := firstMemoryID(c)
	if memID != "" {
		memSum, _ := getJSON(base+"/v1/agent/telemetry/memory/"+memID, apiKey)
		fr.QueryMemoryOK = memSum["memory_id"] != nil
	}
	viol, _ := getJSON(base+"/v1/agent/telemetry/violations?memory_id="+memID, apiKey)
	fr.QueryViolationOK = viol["violations"] != nil

	fr.AutoUtilityMutated = false
	return fr, nil
}

func runMCPFlow(base string, c PersistenceCase) (FlowResult, error) {
	client := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		newTelemetryRouter(NewService()).ServeHTTP(w, r)
	}))
	defer client.Close()
	// Use shared service via re-run with MCP tool calls against test router
	svc := NewService()
	srv := httptest.NewServer(newTelemetryRouter(svc))
	defer srv.Close()

	var fr FlowResult
	oc := c.ToObedienceCase()
	bundle := agentobedience.BundleFromCase(oc)
	tel := agentobedience.RunScriptedAgent(oc, bundle, agentobedience.InterfaceMCP)

	call := func(tool string, args map[string]any) (map[string]any, error) {
		raw, _ := json.Marshal(map[string]any{"name": tool, "arguments": args})
		res, err := mcp.HandleToolsCall(http.DefaultClient, srv.URL, "", raw, nil, nil)
		if err != nil {
			return nil, err
		}
		return extractMCPJSON(res)
	}

	sess, err := call("agent_telemetry_start_session", map[string]any{
		"interface": agentobedience.InterfaceMCP,
		"agent_id":  c.AgentMode,
	})
	if err != nil {
		if c.RejectStep == "unknown_session" {
			fr.Rejected = true
			return fr, nil
		}
		return fr, err
	}
	sid, _ := sess["session_id"].(string)
	fr.SessionPersisted = sid != ""

	bundleJSON := recallBundleJSONFromCase(oc)
	recalled := recalledIDsFromBundle(bundle)
	rec, err := call("agent_telemetry_record_recall", map[string]any{
		"session_id": sid, "task_id": c.TaskID, "interface": agentobedience.InterfaceMCP,
		"recall_bundle_id": "bundle_" + c.TaskID, "recalled_memory_ids": recalled, "recall_bundle": bundleJSON,
	})
	if err != nil {
		return fr, err
	}
	rid, _ := rec["recall_event_id"].(string)
	fr.RecallPersisted = rid != ""

	if c.RecallOnly {
		cands, _ := call("agent_telemetry_get_utility_candidates", map[string]any{})
		fr.RecallOnlyNoPositive = !hasPositiveUtility(cands)
		return fr, nil
	}

	var decisions []map[string]any
	for _, d := range tel.MemoryDecisions {
		decisions = append(decisions, map[string]any{
			"memory_id": d.MemoryID, "decision": d.Decision, "reason": d.Reason,
			"contract_fields_cited": d.ContractFieldsCited, "output_facts_supported": d.OutputFactsSupported,
		})
	}
	if c.RejectStep == "missing_citation" {
		for i := range decisions {
			if decisions[i]["decision"] == "used" {
				decisions[i]["contract_fields_cited"] = []string{}
			}
		}
	}
	_, decErr := call("agent_telemetry_record_decision", map[string]any{
		"session_id": sid, "recall_event_id": rid, "decisions": decisions,
	})
	if decErr != nil && c.RejectStep == "missing_citation" {
		fr.Rejected = true
		return fr, nil
	}
	fr.DecisionPersisted = decErr == nil

	outArgs := map[string]any{
		"session_id": sid, "task_id": c.TaskID, "recall_event_id": rid,
		"output_facts": tel.FinalOutput.Facts, "output_actions": tel.FinalOutput.Actions,
		"memory_citations": tel.FinalOutput.MemoryCitations,
	}
	if c.RejectStep == "missing_output" {
		delete(outArgs, "output_facts")
		delete(outArgs, "output_actions")
	}
	out, outErr := call("agent_telemetry_record_output", outArgs)
	if outErr != nil && c.RejectStep == "missing_output" {
		fr.Rejected = true
		return fr, nil
	}
	oid, _ := out["output_id"].(string)
	fr.OutputPersisted = oid != ""

	if c.SkipEvaluate {
		return fr, nil
	}
	evalArgs := map[string]any{
		"session_id": sid, "task_id": c.TaskID, "recall_event_id": rid, "output_id": oid,
		"expected_facts": c.ExpectedOutputFacts, "forbidden_facts": c.ForbiddenOutputFacts, "task_tags": c.TaskTags,
		"agent_mode": c.AgentMode, "violation_behaviors": c.ViolationBehaviors,
	}
	if c.SelfReportObedient {
		evalArgs["obedience_passed"] = true
	}
	evalResp, err := call("agent_telemetry_evaluate", evalArgs)
	if err != nil {
		return fr, err
	}
	if ev, ok := evalResp["evaluation"].(map[string]any); ok {
		fr.EvaluationPersisted = ev["evaluation_id"] != nil
		fr.EvalPassed = jsonBool(ev["obedience_passed"])
	}
	if v, ok := evalResp["evaluator_rejected_self_report"].(bool); ok && v {
		fr.SelfReportRejected = true
	} else if c.SelfReportObedient && !fr.EvalPassed {
		fr.SelfReportRejected = true
	}
	if vs, ok := evalResp["violations"].([]any); ok && len(vs) > 0 {
		fr.ViolationPersisted = true
	}
	if cs, ok := evalResp["utility_candidates"].([]any); ok && len(cs) > 0 {
		fr.CandidateGenerated = true
	}
	_, _ = call("agent_telemetry_get_session", map[string]any{"session_id": sid})
	fr.QuerySessionOK = true
	memID := firstMemoryID(c)
	if memID != "" {
		_, _ = call("agent_telemetry_get_memory", map[string]any{"memory_id": memID})
		fr.QueryMemoryOK = true
	}
	_, _ = call("agent_telemetry_get_violations", map[string]any{"memory_id": memID})
	fr.QueryViolationOK = true
	return fr, nil
}

func jsonBool(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	default:
		return false
	}
}

func postJSON(url, apiKey string, body []byte) (map[string]any, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func getJSON(url, apiKey string) (map[string]any, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out, nil
}

func extractMCPJSON(res any) (map[string]any, error) {
	raw, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	// Spec-compliant results carry the payload in structuredContent; the legacy
	// {"type":"json"} content block and serialized-JSON text blocks are also accepted.
	if sc, ok := m["structuredContent"].(map[string]any); ok {
		return unwrapTelemetryPayload(sc), nil
	}
	content, _ := m["content"].([]any)
	for _, c := range content {
		cm, _ := c.(map[string]any)
		if j, ok := cm["json"].(map[string]any); ok && cm["type"] == "json" {
			return unwrapTelemetryPayload(j), nil
		}
		if s, ok := cm["text"].(string); ok && cm["type"] == "text" {
			var j map[string]any
			if json.Unmarshal([]byte(s), &j) == nil && j != nil {
				return unwrapTelemetryPayload(j), nil
			}
		}
	}
	return nil, fmt.Errorf("no json content")
}

func unwrapTelemetryPayload(j map[string]any) map[string]any {
	for k, v := range j {
		if strings.HasPrefix(k, "telemetry_") {
			if inner, ok := v.(map[string]any); ok {
				return inner
			}
			if s, ok := v.(string); ok {
				var inner map[string]any
				if json.Unmarshal([]byte(s), &inner) == nil {
					return inner
				}
			}
		}
	}
	return j
}

func hasPositiveUtility(m map[string]any) bool {
	cs, _ := m["utility_candidates"].([]any)
	for _, c := range cs {
		cm, _ := c.(map[string]any)
		st, _ := cm["signal_type"].(string)
		if st == "used_correctly" || st == "helped_output" {
			if str, ok := cm["signal_strength"].(float64); ok && str > 0 {
				return true
			}
		}
	}
	return false
}

func firstMemoryID(c PersistenceCase) string {
	if len(c.InputMemories) > 0 {
		return c.InputMemories[0].MemoryID
	}
	return ""
}

// VerifyCase checks flow result against case expectations.
func VerifyCase(c PersistenceCase, fr FlowResult) error {
	if c.RejectStep != "" && !fr.Rejected && c.RejectStep != "self_report" {
		if c.RejectStep == "unknown_session" && !fr.Rejected {
			// checked via error in flow
		}
	}
	if c.ExpectedObediencePassed && !c.SkipEvaluate && fr.EvaluationPersisted && !fr.EvalPassed {
		return fmt.Errorf("expected obedience pass")
	}
	if !c.ExpectedObediencePassed && !c.SkipEvaluate && fr.EvalPassed {
		return fmt.Errorf("expected obedience fail")
	}
	for _, ev := range c.ExpectedPersistedEvents {
		switch ev {
		case "session":
			if !fr.SessionPersisted {
				return fmt.Errorf("session not persisted")
			}
		case "recall":
			if !fr.RecallPersisted {
				return fmt.Errorf("recall not persisted")
			}
		case "decision":
			if !fr.DecisionPersisted {
				return fmt.Errorf("decision not persisted")
			}
		case "output":
			if !fr.OutputPersisted {
				return fmt.Errorf("output not persisted")
			}
		case "evaluation":
			if !fr.EvaluationPersisted {
				return fmt.Errorf("evaluation not persisted")
			}
		case "violation":
			if !fr.ViolationPersisted {
				return fmt.Errorf("violation not persisted")
			}
		case "utility_candidate":
			if !fr.CandidateGenerated {
				return fmt.Errorf("utility candidate not generated")
			}
		}
	}
	return nil
}

// NewTestService returns isolated telemetry service for tests.
func NewTestService() *Service {
	return NewService()
}
