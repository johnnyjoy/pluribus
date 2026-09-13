package mcp

import (
	"encoding/json"
	"strings"
	"testing"

	"control-plane/internal/formation"
)

func TestInferContextStrategy_lastTimeAndClock(t *testing.T) {
	st, mode, _ := inferContextStrategy("how did we do the restore last time?", "")
	if st != ctxStrategyEpisodic || mode != "thread" {
		t.Fatalf("last-time strategy %q mode %q", st, mode)
	}
	st, mode, _ = inferContextStrategy("what were you doing last friday at 3pm?", "")
	if st != ctxStrategyEpisodic || mode != "thread" {
		t.Fatalf("clock phrasing strategy %q mode %q", st, mode)
	}
}

func TestWantsActivityLane_boundsOrEpisodic(t *testing.T) {
	if !wantsActivityLane(ctxStrategyContinuity, "2026-09-01T00:00:00Z", "") {
		t.Fatal("occurred_after must open activity lane")
	}
	if !wantsActivityLane(ctxStrategyEpisodic, "", "") {
		t.Fatal("episodic strategy must open activity lane")
	}
	if wantsActivityLane(ctxStrategyContinuity, "", "") {
		t.Fatal("plain continuity must not open activity lane")
	}
}

func TestFormatActivityLane_noAdvisoryLabel(t *testing.T) {
	payload := []byte(`{"advisory_similar_cases":[{"id":"11111111-1111-1111-1111-111111111111","summary":"Restored hive from Cursor dumps.","occurred_at":"2026-09-13T15:00:00Z"}]}`)
	got := formatActivityLane(payload)
	if !strings.Contains(got, "Activity:") || !strings.Contains(got, "Restored hive") {
		t.Fatalf("activity text: %q", got)
	}
	for _, bad := range []string{"advisory", "non_canonical", "subordinate"} {
		if strings.Contains(strings.ToLower(got), bad) {
			t.Fatalf("activity text must not say %q: %q", bad, got)
		}
	}
}

func TestBuildAdvisoryEpisodeMCPBody_forwardsOccurredAt(t *testing.T) {
	args := json.RawMessage(`{"summary":"Wrote occurred_at on MCP ingest for clock recall.","occurred_at":"2026-09-13T15:00:00Z"}`)
	out, err := buildAdvisoryEpisodeMCPBody(args, DefaultMemoryFormationPolicy(), formation.NewGate(nil))
	if err != nil {
		t.Fatal(err)
	}
	if out["occurred_at"] != "2026-09-13T15:00:00Z" {
		t.Fatalf("occurred_at not forwarded: %+v", out)
	}
}

func TestPaddedSalesmanSummaryRejected(t *testing.T) {
	essay := "Happy to help! Here is a comprehensive overview of everything we accomplished. Please note that it is important to note our next steps: feel free to ask if you want more detail about the work."
	if err := ValidateMcpEpisodeSummaryWithGate(essay, DefaultMemoryFormationPolicy(), formation.NewGate(nil)); err == nil {
		t.Fatal("padded salesman summary must be rejected")
	}
	dense := "Forwarded occurred_at from MCP record_experience so clock recall can list the episode."
	if err := ValidateMcpEpisodeSummaryWithGate(dense, DefaultMemoryFormationPolicy(), formation.NewGate(nil)); err != nil {
		t.Fatalf("dense claim rejected: %v", err)
	}
}

func TestEnsureDefaultAgentID(t *testing.T) {
	got := ensureDefaultAgentID("record_experience", json.RawMessage(`{"summary":"x"}`), "cursor-agent")
	var m map[string]any
	if err := json.Unmarshal(got, &m); err != nil {
		t.Fatal(err)
	}
	if m["agent_id"] != "cursor-agent" {
		t.Fatalf("agent_id %v", m["agent_id"])
	}
	kept := ensureDefaultAgentID("record_experience", json.RawMessage(`{"summary":"x","agent_id":"explicit"}`), "cursor-agent")
	_ = json.Unmarshal(kept, &m)
	if m["agent_id"] != "explicit" {
		t.Fatalf("must keep explicit agent_id, got %v", m["agent_id"])
	}
}
