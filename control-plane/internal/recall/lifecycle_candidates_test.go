package recall

import (
	"testing"

	"control-plane/internal/memory"
)

func TestIncludeSupersededCandidates(t *testing.T) {
	if !includeSupersededCandidates("Is SQLite still the Pluribus memory store?") {
		t.Fatal("expected lifecycle query to include superseded")
	}
	if includeSupersededCandidates("Harden the MCP interface") {
		t.Fatal("expected normal query to skip superseded merge")
	}
}

func TestRepoRootAffinityBoost(t *testing.T) {
	boost := repoRootAffinityBoost("/projects/pluribus", "Pluribus MCP is the primary interface", nil)
	if boost <= 0 {
		t.Fatalf("expected repo basename boost, got %v", boost)
	}
}

func TestFilterKeywordBridgeCandidates(t *testing.T) {
	t.Run("noise_suppression", func(t *testing.T) {
		reqTags := []string{"proof:signals:run1"}
		keywords := []string{"proof", "signal", "unique", "hub"}
		noise := memory.MemoryObject{
			Statement: "zzz noise proof filler one",
			Tags:      []string{"proof:noisejunk:run1"},
		}
		signal := memory.MemoryObject{
			Statement: "PROOF SIGNAL HUB UNIQUE run1",
			Tags:      []string{"proof:signals:run1"},
		}
		filtered := filterKeywordBridgeCandidates(reqTags, "proof", keywords, []memory.MemoryObject{noise, signal})
		if len(filtered) != 1 || filtered[0].Statement != signal.Statement {
			t.Fatalf("expected only signal, got %+v", filtered)
		}
	})
	t.Run("cross_tag_mobile_credential", func(t *testing.T) {
		reqTags := []string{"housing", "badge", "stanford"}
		keywords := []string{"onguard", "stanford", "housing", "badge", "integration", "mobile", "credential"}
		mobile := memory.MemoryObject{
			Statement: "Mobile credential rollout chose BLE-first tap with NFC fallback",
			Tags:      []string{"mobile", "credential"},
		}
		filtered := filterKeywordBridgeCandidates(reqTags, "mobile", keywords, []memory.MemoryObject{mobile})
		if len(filtered) != 1 {
			t.Fatalf("expected mobile credential kept via distinctive keyword bridge, got %d", len(filtered))
		}
	})
}
