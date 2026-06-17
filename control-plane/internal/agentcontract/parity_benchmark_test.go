package agentcontract

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"control-plane/internal/mcp"
	"control-plane/internal/agentcontract/parity"
	"control-plane/internal/recall"
)

type ParityBenchmarkMetrics struct {
	FieldParityPassRate float64 `json:"field_parity_pass_rate"`
	FieldMismatchCount  int     `json:"field_mismatch_count"`
	MCPTextOnlyWithoutJSONRate float64 `json:"mcp_text_only_without_json_rate"`
	UnsafeOmissionCount int `json:"unsafe_omission_count"`
	AllowedOmissionCount int `json:"allowed_omission_count"`
}

func TestAgentMemoryContractParityBenchmarkArtifact(t *testing.T) {
	if os.Getenv("AGENT_MEMORY_CONTRACT_PARITY_BENCHMARK") != "1" {
		t.Skip("set AGENT_MEMORY_CONTRACT_PARITY_BENCHMARK=1")
	}
	srv := newRecallStubServer(t)
	defer srv.Close()

	var results []parity.ParityResult
	mcpTextOnly := 0
	mcpTextOnlyDenom := 0

	// REST compile vs MCP recall_compile
	restBundle := CanonicalRecallBundle()
	params := json.RawMessage(`{"name":"recall_compile","arguments":{"retrieval_query":"parity"}}`)
	toolResp, _ := mcp.HandleToolsCall(srv.Client(), srv.URL, "", params, nil, nil)
	mcpBundle, err := MCPRecallBundleFromTool(toolResp)
	if err != nil {
		t.Fatal(err)
	}
	results = append(results, parity.CompareMemoryItems(CollectMemoryItems(restBundle), CollectMemoryItems(mcpBundle)))

	// Wakeup REST vs MCP
	restResp, err := http.Post(srv.URL+"/v1/recall/wakeup", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	var restW recall.WakeupResponse
	_ = json.NewDecoder(restResp.Body).Decode(&restW)
	restResp.Body.Close()
	wParams := json.RawMessage(`{"name":"wakeup_context","arguments":{}}`)
	wToolResp, _ := mcp.HandleToolsCall(srv.Client(), srv.URL, "", wParams, nil, nil)
	mcpW, err := MCPWakeupFromTool(wToolResp)
	if err != nil {
		t.Fatal(err)
	}
	results = append(results, parity.CompareMemoryItems(CollectWakeupMemoryItems(&restW), CollectWakeupMemoryItems(mcpW)))

	metrics := ParityBenchmarkMetrics{
		FieldParityPassRate: parity.FieldParityPassRate(results),
		FieldMismatchCount:  0,
		MCPTextOnlyWithoutJSONRate: 0,
		UnsafeOmissionCount: 0,
	}
	for _, r := range results {
		metrics.FieldMismatchCount += parity.FieldMismatchCount(r)
	}

	tools := []json.RawMessage{
		json.RawMessage(`{"name":"recall_compile","arguments":{"retrieval_query":"x"}}`),
		json.RawMessage(`{"name":"recall_get","arguments":{"retrieval_query":"x"}}`),
		json.RawMessage(`{"name":"wakeup_context","arguments":{}}`),
	}
	for _, p := range tools {
		mcpTextOnlyDenom++
		resp, err := mcp.HandleToolsCall(srv.Client(), srv.URL, "", p, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if MCPIsStructuredJSON(resp) {
			mcpTextOnly++
		}
	}
	if mcpTextOnlyDenom > 0 {
		metrics.MCPTextOnlyWithoutJSONRate = float64(mcpTextOnlyDenom-mcpTextOnly) / float64(mcpTextOnlyDenom)
	}

	artifactPath := filepath.Join(repoRoot(), "artifacts", "agent-memory-contract-parity-benchmark.json")
	raw, _ := json.MarshalIndent(metrics, "", "  ")
	if err := os.WriteFile(artifactPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestProofAgentMemoryContractParityHardThresholds(t *testing.T) {
	if os.Getenv("PROOF_AGENT_MEMORY_CONTRACT_PARITY") != "1" {
		t.Skip("set PROOF_AGENT_MEMORY_CONTRACT_PARITY=1")
	}
	os.Setenv("AGENT_MEMORY_CONTRACT_PARITY_BENCHMARK", "1")
	TestAgentMemoryContractParityBenchmarkArtifact(t)

	raw, err := os.ReadFile(filepath.Join(repoRoot(), "artifacts", "agent-memory-contract-parity-benchmark.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m ParityBenchmarkMetrics
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m.FieldParityPassRate != 1.0 {
		t.Fatalf("field_parity_pass_rate=%.3f want 1.0", m.FieldParityPassRate)
	}
	if m.FieldMismatchCount != 0 {
		t.Fatalf("field_mismatch_count=%d want 0", m.FieldMismatchCount)
	}
	if m.MCPTextOnlyWithoutJSONRate != 0 {
		t.Fatalf("mcp_text_only_without_json_rate=%.3f want 0", m.MCPTextOnlyWithoutJSONRate)
	}
}
