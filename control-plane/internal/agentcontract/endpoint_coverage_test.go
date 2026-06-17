package agentcontract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"control-plane/internal/mcp"
	"control-plane/internal/agentcontract/parity"
	"control-plane/internal/recall"
)

type endpointSpec struct {
	Name              string
	Interface         string
	AgentFacing       bool
	ReturnsMemoryItem bool
	ReturnsRecallBundle bool
	FieldParityRequired bool
}

var agentFacingEndpoints = []endpointSpec{
	{Name: "REST compile", Interface: "rest", AgentFacing: true, ReturnsRecallBundle: true, FieldParityRequired: true},
	{Name: "REST compile-multi", Interface: "rest", AgentFacing: true, ReturnsRecallBundle: true, FieldParityRequired: true},
	{Name: "REST recall get", Interface: "rest", AgentFacing: true, ReturnsRecallBundle: true, FieldParityRequired: true},
	{Name: "REST wakeup", Interface: "rest", AgentFacing: true, ReturnsMemoryItem: true, FieldParityRequired: true},
	{Name: "REST run-multi", Interface: "rest", AgentFacing: true, FieldParityRequired: false},
	{Name: "MCP recall_context", Interface: "mcp", AgentFacing: true, ReturnsRecallBundle: true, FieldParityRequired: true},
	{Name: "MCP memory_context_resolve", Interface: "mcp", AgentFacing: true, ReturnsRecallBundle: true, FieldParityRequired: true},
	{Name: "MCP recall_compile", Interface: "mcp", AgentFacing: true, ReturnsRecallBundle: true, FieldParityRequired: true},
	{Name: "MCP memory_recall_advanced", Interface: "mcp", AgentFacing: true, ReturnsRecallBundle: true, FieldParityRequired: true},
	{Name: "MCP recall_get", Interface: "mcp", AgentFacing: true, ReturnsRecallBundle: true, FieldParityRequired: true},
	{Name: "MCP wakeup_context", Interface: "mcp", AgentFacing: true, ReturnsMemoryItem: true, FieldParityRequired: true},
	{Name: "MCP recall_run_multi", Interface: "mcp", AgentFacing: true, FieldParityRequired: false},
}

func newRecallStubServer(t *testing.T) *httptest.Server {
	t.Helper()
	bundle := CanonicalRecallBundle()
	wakeup := CanonicalWakeupResponse()
	compileMulti := CanonicalCompileMultiResponse()
	runMulti := recall.RunMultiResponse{
		Promoted:   false,
		Confidence: 0.5,
		Debug:      recall.RunMultiDebug{Orchestration: map[string]any{"variants": 1}},
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/recall/compile":
			_ = json.NewEncoder(w).Encode(bundle)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/recall/compile-multi":
			_ = json.NewEncoder(w).Encode(compileMulti)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/recall/":
			_ = json.NewEncoder(w).Encode(bundle)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/recall/wakeup":
			_ = json.NewEncoder(w).Encode(wakeup)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/recall/run-multi":
			_ = json.NewEncoder(w).Encode(runMulti)
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestAgentFacingEndpointInventory(t *testing.T) {
	if len(agentFacingEndpoints) < 12 {
		t.Fatalf("expected >=12 agent-facing endpoints, got %d", len(agentFacingEndpoints))
	}
}

func TestWakeupRESTMemoryContract(t *testing.T) {
	srv := newRecallStubServer(t)
	defer srv.Close()
	resp, err := http.Post(srv.URL+"/v1/recall/wakeup", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var w recall.WakeupResponse
	if err := json.NewDecoder(resp.Body).Decode(&w); err != nil {
		t.Fatal(err)
	}
	items := CollectWakeupMemoryItems(&w)
	b := &recall.RecallBundle{GoverningConstraints: items}
	ev := EvaluateBundleContract(b, recall.RecallModeCurrent, false)
	if !ev.ContractPassed {
		t.Fatalf("wakeup REST contract failed: %+v", ev)
	}
}

func TestWakeupMCPMemoryContract(t *testing.T) {
	srv := newRecallStubServer(t)
	defer srv.Close()
	params := json.RawMessage(`{"name":"wakeup_context","arguments":{}}`)
	toolResp, err := mcp.HandleToolsCall(srv.Client(), srv.URL, "", params, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !MCPIsStructuredJSON(toolResp) {
		t.Fatal("wakeup_context must return structured JSON")
	}
	w, err := MCPWakeupFromTool(toolResp)
	if err != nil {
		t.Fatal(err)
	}
	items := CollectWakeupMemoryItems(w)
	b := &recall.RecallBundle{GoverningConstraints: items}
	ev := EvaluateBundleContract(b, recall.RecallModeCurrent, false)
	if !ev.ContractPassed {
		t.Fatalf("wakeup MCP contract failed: %+v", ev)
	}
}

func TestWakeupContractParity(t *testing.T) {
	srv := newRecallStubServer(t)
	defer srv.Close()

	restResp, err := http.Post(srv.URL+"/v1/recall/wakeup", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	var restW recall.WakeupResponse
	_ = json.NewDecoder(restResp.Body).Decode(&restW)
	restResp.Body.Close()

	params := json.RawMessage(`{"name":"wakeup_context","arguments":{}}`)
	toolResp, err := mcp.HandleToolsCall(srv.Client(), srv.URL, "", params, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	mcpW, err := MCPWakeupFromTool(toolResp)
	if err != nil {
		t.Fatal(err)
	}
	pr := parity.CompareMemoryItems(CollectWakeupMemoryItems(&restW), CollectWakeupMemoryItems(mcpW))
	if !pr.ParityPassed || parity.FieldMismatchCount(pr) != 0 {
		t.Fatalf("wakeup parity failed: %+v", pr)
	}
}

func TestWakeupDoesNotReturnFlattenedTextOnlyMemory(t *testing.T) {
	srv := newRecallStubServer(t)
	defer srv.Close()
	params := json.RawMessage(`{"name":"wakeup_context","arguments":{}}`)
	toolResp, err := mcp.HandleToolsCall(srv.Client(), srv.URL, "", params, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !MCPIsStructuredJSON(toolResp) {
		t.Fatal("wakeup must not be text-only")
	}
}

func TestRunMultiRESTMemoryContract(t *testing.T) {
	srv := newRecallStubServer(t)
	defer srv.Close()
	body := []byte(`{"query":"parity test","merge":false}`)
	resp, err := http.Post(srv.URL+"/v1/recall/run-multi", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("run-multi status=%d", resp.StatusCode)
	}
	var rm recall.RunMultiResponse
	if err := json.NewDecoder(resp.Body).Decode(&rm); err != nil {
		t.Fatal(err)
	}
	if rm.Debug.Orchestration == nil {
		t.Fatal("run-multi debug.orchestration required")
	}
}

func TestRunMultiMCPStructuredJSON(t *testing.T) {
	srv := newRecallStubServer(t)
	defer srv.Close()
	params := json.RawMessage(`{"name":"recall_run_multi","arguments":{"query":"parity","merge":false}}`)
	toolResp, err := mcp.HandleToolsCall(srv.Client(), srv.URL, "", params, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !MCPIsStructuredJSON(toolResp) {
		t.Fatal("recall_run_multi must return structured JSON")
	}
}

func TestRunMultiPreservesContractFieldsAcrossVariants(t *testing.T) {
	srv := newRecallStubServer(t)
	defer srv.Close()
	resp, err := http.Post(srv.URL+"/v1/recall/compile-multi", "application/json", bytes.NewReader([]byte(`{"variants":1}`)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var cm recall.CompileMultiResponse
	if err := json.NewDecoder(resp.Body).Decode(&cm); err != nil {
		t.Fatal(err)
	}
	for _, vb := range cm.Bundles {
		ev := EvaluateBundleContract(&vb.Bundle, recall.RecallModeCurrent, false)
		if !ev.ContractPassed {
			t.Fatalf("compile-multi variant %s contract failed: %+v", vb.Variant, ev)
		}
	}
}

func TestMCPRecallContextAliasContractParity(t *testing.T) {
	srv := newRecallStubServer(t)
	defer srv.Close()
	for _, tool := range []string{"recall_context", "memory_context_resolve"} {
		t.Run(tool, func(t *testing.T) {
			params, _ := json.Marshal(map[string]any{
				"name": tool,
				"arguments": map[string]any{
					"task": "alias parity",
					"tags": []string{"project:parity"},
				},
			})
			toolResp, err := mcp.HandleToolsCall(srv.Client(), srv.URL, "", params, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			mcpBundle, err := MCPRecallBundleFromTool(toolResp)
			if err != nil {
				t.Fatal(err)
			}
			restBundle := CanonicalRecallBundle()
			pr := parity.CompareMemoryItems(CollectMemoryItems(restBundle), CollectMemoryItems(mcpBundle))
			if !pr.ParityPassed {
				t.Fatalf("%s parity failed: %+v", tool, pr)
			}
		})
	}
}

func TestMCPRecallCompileAliasContractParity(t *testing.T) {
	srv := newRecallStubServer(t)
	defer srv.Close()
	body, _ := json.Marshal(map[string]any{"retrieval_query": "parity"})
	params, _ := json.Marshal(map[string]any{"name": "recall_compile", "arguments": json.RawMessage(body)})
	toolResp, err := mcp.HandleToolsCall(srv.Client(), srv.URL, "", params, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	mcpBundle, err := MCPRecallBundleFromTool(toolResp)
	if err != nil {
		t.Fatal(err)
	}
	pr := parity.CompareMemoryItems(CollectMemoryItems(CanonicalRecallBundle()), CollectMemoryItems(mcpBundle))
	if !pr.ParityPassed {
		t.Fatalf("recall_compile parity failed: %+v", pr)
	}
}

func TestMCPRecallGetContractFields(t *testing.T) {
	srv := newRecallStubServer(t)
	defer srv.Close()
	params := json.RawMessage(`{"name":"recall_get","arguments":{"retrieval_query":"parity"}}`)
	toolResp, err := mcp.HandleToolsCall(srv.Client(), srv.URL, "", params, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := MCPRecallBundleFromTool(toolResp)
	if err != nil {
		t.Fatal(err)
	}
	ev := EvaluateBundleContract(bundle, recall.RecallModeCurrent, false)
	if !ev.ContractPassed {
		t.Fatalf("recall_get contract failed: %+v", ev)
	}
}

func TestMCPWakeupContextContractFields(t *testing.T) {
	TestWakeupMCPMemoryContract(t)
}

func TestMCPAliasesDoNotDropStructuredJSON(t *testing.T) {
	srv := newRecallStubServer(t)
	defer srv.Close()
	tools := []json.RawMessage{
		json.RawMessage(`{"name":"recall_compile","arguments":{"retrieval_query":"x"}}`),
		json.RawMessage(`{"name":"recall_get","arguments":{"retrieval_query":"x"}}`),
		json.RawMessage(`{"name":"memory_recall_advanced","arguments":{"query":"x"}}`),
		json.RawMessage(`{"name":"wakeup_context","arguments":{}}`),
		json.RawMessage(`{"name":"recall_context","arguments":{"task":"x"}}`),
	}
	for i, params := range tools {
		toolResp, err := mcp.HandleToolsCall(srv.Client(), srv.URL, "", params, nil, nil)
		if err != nil {
			t.Fatalf("tool %d: %v", i, err)
		}
		if !MCPIsStructuredJSON(toolResp) {
			t.Fatalf("tool %d returned text-only MCP response", i)
		}
	}
}

func TestRESTCompileMemoryContract(t *testing.T) {
	srv := newRecallStubServer(t)
	defer srv.Close()
	resp, err := http.Post(srv.URL+"/v1/recall/compile", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var b recall.RecallBundle
	if err := json.NewDecoder(resp.Body).Decode(&b); err != nil {
		t.Fatal(err)
	}
	ev := EvaluateBundleContract(&b, recall.RecallModeCurrent, false)
	if !ev.ContractPassed {
		t.Fatalf("REST compile contract failed: %+v", ev)
	}
}

func TestRESTMCPFieldParityCompile(t *testing.T) {
	srv := newRecallStubServer(t)
	defer srv.Close()
	restResp, err := http.Post(srv.URL+"/v1/recall/compile", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	var restBundle recall.RecallBundle
	_ = json.NewDecoder(restResp.Body).Decode(&restBundle)
	restResp.Body.Close()

	params := json.RawMessage(`{"name":"recall_compile","arguments":{"retrieval_query":"parity"}}`)
	toolResp, _ := mcp.HandleToolsCall(srv.Client(), srv.URL, "", params, nil, nil)
	mcpBundle, err := MCPRecallBundleFromTool(toolResp)
	if err != nil {
		t.Fatal(err)
	}
	pr := parity.CompareMemoryItems(CollectMemoryItems(&restBundle), CollectMemoryItems(mcpBundle))
	if !pr.ParityPassed || parity.FieldMismatchCount(pr) != 0 {
		t.Fatalf("compile parity failed: %+v", pr)
	}
}

func TestAllAgentFacingEndpointsCovered(t *testing.T) {
	srv := newRecallStubServer(t)
	defer srv.Close()
	covered := 0
	for _, ep := range agentFacingEndpoints {
		if !ep.AgentFacing {
			continue
		}
		switch ep.Name {
		case "REST compile", "REST compile-multi", "REST recall get", "REST wakeup", "REST run-multi":
			covered++
		case "MCP recall_context", "MCP memory_context_resolve", "MCP recall_compile",
			"MCP memory_recall_advanced", "MCP recall_get", "MCP wakeup_context", "MCP recall_run_multi":
			covered++
		default:
			t.Fatalf("untested agent-facing endpoint: %s", ep.Name)
		}
	}
	if covered != len(agentFacingEndpoints) {
		t.Fatalf("endpoint coverage %d != %d", covered, len(agentFacingEndpoints))
	}
}
