package agentobedience

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"control-plane/internal/agentcontract"
	"control-plane/internal/mcp"
	"control-plane/internal/recall"
)

// RunCase executes one obedience fixture end-to-end.
func RunCase(c ObedienceCase, iface string) (MemoryUseTelemetry, ObedienceEvaluation) {
	bundle := BundleFromCase(c)
	tel := RunScriptedAgent(c, bundle, iface)
	ev := EvaluateObedience(c, bundle, tel)
	return tel, ev
}

func TestHostileObedienceCases(t *testing.T) {
	cases, err := LoadCases("")
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) < 24 {
		t.Fatalf("need >=24 cases, got %d", len(cases))
	}
	for _, c := range cases {
		t.Run(c.ID, func(t *testing.T) {
			ifaces := []string{c.Interface}
			if c.Interface == "both" {
				ifaces = []string{InterfaceREST, InterfaceMCP}
			}
			for _, iface := range ifaces {
				if iface == "both" {
					continue
				}
				_, ev := RunCase(c, iface)
				if !DetectionPassed(c, ev) {
					t.Fatalf("interface=%s detection failed: passed=%v violations=%v expected=%v mode=%s",
						iface, ev.ObediencePassed, ev.Violations, c.ExpectedViolationCodes, c.AgentMode)
				}
			}
		})
	}
}

func TestMCPRESTAgentFlow(t *testing.T) {
	cases, err := LoadCases("")
	if err != nil {
		t.Fatal(err)
	}
	var parityCase *ObedienceCase
	for i := range cases {
		if cases[i].ID == "mcp_rest_obedience_parity_same_task" {
			parityCase = &cases[i]
			break
		}
	}
	if parityCase == nil {
		t.Fatal("missing parity case")
	}
	bundle := BundleFromCase(*parityCase)
	raw, _ := json.Marshal(bundle)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	}))
	defer srv.Close()

	// REST flow
	restResp, err := http.Post(srv.URL+"/v1/recall/compile", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	var restBundle recall.RecallBundle
	if err := json.NewDecoder(restResp.Body).Decode(&restBundle); err != nil {
		t.Fatal(err)
	}
	restResp.Body.Close()
	restTel := RunScriptedAgent(*parityCase, &restBundle, InterfaceREST)
	restEv := EvaluateObedience(*parityCase, &restBundle, restTel)

	// MCP flow
	params := json.RawMessage(`{"name":"recall_compile","arguments":{"retrieval_query":"parity"}}`)
	toolResp, err := mcp.HandleToolsCall(srv.Client(), srv.URL, "", params, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	mcpBundle, err := agentcontract.MCPRecallBundleFromTool(toolResp)
	if err != nil {
		t.Fatal(err)
	}
	mcpTel := RunScriptedAgent(*parityCase, mcpBundle, InterfaceMCP)
	mcpEv := EvaluateObedience(*parityCase, mcpBundle, mcpTel)

	if !restEv.ObediencePassed || !mcpEv.ObediencePassed {
		t.Fatalf("parity flow failed rest=%v mcp=%v", restEv.ObediencePassed, mcpEv.ObediencePassed)
	}
	if len(restTel.UsedMemoryIDs) != len(mcpTel.UsedMemoryIDs) {
		t.Fatalf("used memory mismatch rest=%v mcp=%v", restTel.UsedMemoryIDs, mcpTel.UsedMemoryIDs)
	}
}

func TestScriptedAgentModesExist(t *testing.T) {
	for _, mode := range []string{AgentObedient, AgentSloppy, AgentBroken} {
		c := ObedienceCase{
			ID: "smoke", TaskID: "t", Interface: InterfaceREST, AgentMode: mode,
			TaskTags: []string{"project:payments"},
			InputMemories: []CaseMemory{{
				MemoryID: "m1", Statement: "s", SchemaType: "constraint",
				LifecycleRole: recall.LifecycleCurrentGuidance, Status: "active",
				Scope: "project:payments", NegativeScope: []string{"project:wrong"},
				UseInstruction: "u", SourceType: "formationquality", AuthorityBasis: "a",
				QualityState: "accept_active",
			}},
		}
		u := 0.7
		q := 0.8
		c.InputMemories[0].UtilityScore = &u
		c.InputMemories[0].QualityScore = &q
		tel := RunScriptedAgent(c, BundleFromCase(c), InterfaceREST)
		if tel.AgentKind != mode {
			t.Fatalf("mode %s got %s", mode, tel.AgentKind)
		}
	}
}
