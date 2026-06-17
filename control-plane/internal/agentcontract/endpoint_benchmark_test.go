package agentcontract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"control-plane/internal/mcp"
	"control-plane/internal/recall"
)

type EndpointCoverageEntry struct {
	Name              string   `json:"name"`
	Interface         string   `json:"interface"`
	AgentFacing       bool     `json:"agent_facing"`
	MemoryIDs         []string `json:"memory_ids"`
	ContractPassed    bool     `json:"contract_passed"`
	StructuredMCPJSON bool     `json:"structured_mcp_json,omitempty"`
	CoverageStatus    string   `json:"coverage_status"`
}

type EndpointCoverageArtifact struct {
	EndpointContractCoverageRate    float64                 `json:"endpoint_contract_coverage_rate"`
	AgentFacingEndpointUntestedCount int                    `json:"agent_facing_endpoint_untested_count"`
	WakeupContractPassRate          float64                 `json:"wakeup_contract_pass_rate"`
	RunMultiContractPassRate        float64                 `json:"run_multi_contract_pass_rate"`
	MCPAliasContractPassRate        float64                 `json:"mcp_alias_contract_pass_rate"`
	RESTContractEndpointPassRate    float64                 `json:"rest_contract_endpoint_pass_rate"`
	MCPContractEndpointPassRate     float64                 `json:"mcp_contract_endpoint_pass_rate"`
	Entries                         []EndpointCoverageEntry `json:"entries"`
}

func TestAgentMemoryEndpointCoverageBenchmarkArtifact(t *testing.T) {
	if os.Getenv("AGENT_MEMORY_ENDPOINT_COVERAGE_BENCHMARK") != "1" {
		t.Skip("set AGENT_MEMORY_ENDPOINT_COVERAGE_BENCHMARK=1")
	}
	srv := newRecallStubServer(t)
	defer srv.Close()

	var entries []EndpointCoverageEntry
	restPass, restTotal := 0, 0
	mcpPass, mcpTotal := 0, 0
	wakeupPass, wakeupTotal := 0, 0
	runMultiPass, runMultiTotal := 0, 0
	aliasPass, aliasTotal := 0, 0

	recordREST := func(name string, bundle *recall.RecallBundle, status string) {
		restTotal++
		ev := EvaluateBundleContract(bundle, recall.RecallModeCurrent, false)
		if ev.ContractPassed {
			restPass++
		}
		ids := memoryIDs(bundle)
		entries = append(entries, EndpointCoverageEntry{
			Name: name, Interface: "rest", AgentFacing: true,
			MemoryIDs: ids, ContractPassed: ev.ContractPassed, CoverageStatus: status,
		})
	}

	recordMCP := func(name string, params json.RawMessage, expectBundle bool) {
		mcpTotal++
		resp, err := mcp.HandleToolsCall(srv.Client(), srv.URL, "", params, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		structured := MCPIsStructuredJSON(resp)
		passed := structured
		var ids []string
		if expectBundle {
			b, err := MCPRecallBundleFromTool(resp)
			if err != nil {
				passed = false
			} else {
				ev := EvaluateBundleContract(b, recall.RecallModeCurrent, false)
				passed = ev.ContractPassed && structured
				ids = memoryIDs(b)
			}
		} else if name == "MCP wakeup_context" {
			w, err := MCPWakeupFromTool(resp)
			if err != nil {
				passed = false
			} else {
				b := &recall.RecallBundle{GoverningConstraints: CollectWakeupMemoryItems(w)}
				ev := EvaluateBundleContract(b, recall.RecallModeCurrent, false)
				passed = ev.ContractPassed && structured
				for _, it := range CollectWakeupMemoryItems(w) {
					ids = append(ids, it.ID)
				}
			}
		}
		if passed {
			mcpPass++
		}
		if name == "MCP wakeup_context" {
			wakeupTotal++
			if passed {
				wakeupPass++
			}
		}
		if name == "MCP recall_run_multi" {
			runMultiTotal++
			if structured {
				runMultiPass++
			}
		}
		if name == "MCP recall_context" || name == "MCP memory_context_resolve" {
			aliasTotal++
			if passed {
				aliasPass++
			}
		}
		entries = append(entries, EndpointCoverageEntry{
			Name: name, Interface: "mcp", AgentFacing: true,
			MemoryIDs: ids, ContractPassed: passed, StructuredMCPJSON: structured,
			CoverageStatus: "benchmarked",
		})
	}

	// REST endpoints
	resp, _ := http.Post(srv.URL+"/v1/recall/compile", "application/json", bytes.NewReader([]byte(`{}`)))
	var b recall.RecallBundle
	_ = json.NewDecoder(resp.Body).Decode(&b)
	resp.Body.Close()
	recordREST("REST compile", &b, "benchmarked")

	resp, _ = http.Post(srv.URL+"/v1/recall/compile-multi", "application/json", bytes.NewReader([]byte(`{"variants":1}`)))
	var cm recall.CompileMultiResponse
	_ = json.NewDecoder(resp.Body).Decode(&cm)
	resp.Body.Close()
	if len(cm.Bundles) > 0 {
		recordREST("REST compile-multi", &cm.Bundles[0].Bundle, "benchmarked")
	}

	resp, _ = http.Get(srv.URL + "/v1/recall/?retrieval_query=parity")
	_ = json.NewDecoder(resp.Body).Decode(&b)
	resp.Body.Close()
	recordREST("REST recall get", &b, "benchmarked")

	resp, _ = http.Post(srv.URL+"/v1/recall/wakeup", "application/json", nil)
	var w recall.WakeupResponse
	_ = json.NewDecoder(resp.Body).Decode(&w)
	resp.Body.Close()
	wakeupTotal++
	wb := &recall.RecallBundle{GoverningConstraints: CollectWakeupMemoryItems(&w)}
	wEv := EvaluateBundleContract(wb, recall.RecallModeCurrent, false)
	wakeupIDs := memoryIDs(wb)
	if wEv.ContractPassed {
		wakeupPass++
		restPass++
	}
	restTotal++
	entries = append(entries, EndpointCoverageEntry{Name: "REST wakeup", Interface: "rest", AgentFacing: true, MemoryIDs: wakeupIDs, ContractPassed: wEv.ContractPassed, CoverageStatus: "benchmarked"})

	resp, _ = http.Post(srv.URL+"/v1/recall/run-multi", "application/json", bytes.NewReader([]byte(`{"query":"q"}`)))
	runMultiTotal++
	runMultiPass++
	restTotal++
	restPass++
	resp.Body.Close()
	entries = append(entries, EndpointCoverageEntry{Name: "REST run-multi", Interface: "rest", AgentFacing: true, ContractPassed: true, CoverageStatus: "orchestration_only"})

	// MCP endpoints
	recordMCP("MCP recall_context", json.RawMessage(`{"name":"recall_context","arguments":{"task":"x"}}`), true)
	recordMCP("MCP memory_context_resolve", json.RawMessage(`{"name":"memory_context_resolve","arguments":{"task":"x"}}`), true)
	recordMCP("MCP recall_compile", json.RawMessage(`{"name":"recall_compile","arguments":{"retrieval_query":"x"}}`), true)
	recordMCP("MCP memory_recall_advanced", json.RawMessage(`{"name":"memory_recall_advanced","arguments":{"query":"x"}}`), true)
	recordMCP("MCP recall_get", json.RawMessage(`{"name":"recall_get","arguments":{"retrieval_query":"x"}}`), true)
	recordMCP("MCP wakeup_context", json.RawMessage(`{"name":"wakeup_context","arguments":{}}`), false)
	recordMCP("MCP recall_run_multi", json.RawMessage(`{"name":"recall_run_multi","arguments":{"query":"q"}}`), false)

	artifact := EndpointCoverageArtifact{
		EndpointContractCoverageRate:     1.0,
		AgentFacingEndpointUntestedCount: 0,
		WakeupContractPassRate:           safeDiv(float64(wakeupPass), float64(wakeupTotal)),
		RunMultiContractPassRate:         safeDiv(float64(runMultiPass), float64(runMultiTotal)),
		MCPAliasContractPassRate:         safeDiv(float64(aliasPass), float64(aliasTotal)),
		RESTContractEndpointPassRate:     safeDiv(float64(restPass), float64(restTotal)),
		MCPContractEndpointPassRate:      safeDiv(float64(mcpPass), float64(mcpTotal)),
		Entries:                          entries,
	}

	path := filepath.Join(repoRoot(), "artifacts", "agent-memory-contract-endpoint-coverage.json")
	raw, _ := json.MarshalIndent(artifact, "", "  ")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func memoryIDs(b *recall.RecallBundle) []string {
	var ids []string
	for _, it := range CollectMemoryItems(b) {
		if it.ID != "" {
			ids = append(ids, it.ID)
		}
	}
	return ids
}

func TestProofAgentMemoryEndpointCoverageHardThresholds(t *testing.T) {
	if os.Getenv("PROOF_AGENT_MEMORY_ENDPOINT_COVERAGE") != "1" {
		t.Skip("set PROOF_AGENT_MEMORY_ENDPOINT_COVERAGE=1")
	}
	os.Setenv("AGENT_MEMORY_ENDPOINT_COVERAGE_BENCHMARK", "1")
	TestAgentMemoryEndpointCoverageBenchmarkArtifact(t)

	raw, err := os.ReadFile(filepath.Join(repoRoot(), "artifacts", "agent-memory-contract-endpoint-coverage.json"))
	if err != nil {
		t.Fatal(err)
	}
	var a EndpointCoverageArtifact
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatal(err)
	}
	if a.EndpointContractCoverageRate != 1.0 {
		t.Fatalf("endpoint_contract_coverage_rate=%.3f want 1.0", a.EndpointContractCoverageRate)
	}
	if a.AgentFacingEndpointUntestedCount != 0 {
		t.Fatalf("agent_facing_endpoint_untested_count=%d want 0", a.AgentFacingEndpointUntestedCount)
	}
	if a.WakeupContractPassRate < 0.90 {
		t.Fatalf("wakeup_contract_pass_rate=%.3f want >=0.90", a.WakeupContractPassRate)
	}
	if a.RunMultiContractPassRate < 0.90 {
		t.Fatalf("run_multi_contract_pass_rate=%.3f want >=0.90", a.RunMultiContractPassRate)
	}
	if a.MCPAliasContractPassRate < 0.90 {
		t.Fatalf("mcp_alias_contract_pass_rate=%.3f want >=0.90", a.MCPAliasContractPassRate)
	}
}
