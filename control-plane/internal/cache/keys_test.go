package cache

import (
	"testing"
)

func TestRecallBundleKey_differsWhenSymbolsDiffer(t *testing.T) {
	k1 := RecallBundleKey(nil, 5, 0, 0, "", "", nil, "", "", 0, 0, "", EvidenceBundleKey{}, false)
	k2 := RecallBundleKey(nil, 5, 0, 0, "", "", []string{"A", "B"}, "", "", 0, 0, "", EvidenceBundleKey{}, false)
	k3 := RecallBundleKey(nil, 5, 0, 0, "", "", []string{"B", "A"}, "", "", 0, 0, "", EvidenceBundleKey{}, false)
	if k1 == k2 {
		t.Fatal("expected different keys when symbols differ")
	}
	if k2 != k3 {
		t.Fatal("symbol order should not change key (sorted in hash)")
	}
}

func TestRecallBundleKey_differsWhenLSPFocusPathDiffers(t *testing.T) {
	k1 := RecallBundleKey(nil, 5, 0, 0, "", "", nil, "/r", "a.go", 0, 0, "", EvidenceBundleKey{}, false)
	k2 := RecallBundleKey(nil, 5, 0, 0, "", "", nil, "/r", "b.go", 0, 0, "", EvidenceBundleKey{}, false)
	if k1 == k2 {
		t.Fatal("expected different keys for different lsp_focus_path")
	}
}

func TestRecallBundleKey_differsWhenEvidenceInBundleEnabled(t *testing.T) {
	off := EvidenceBundleKey{}
	on := EvidenceBundleKey{Enabled: true, MaxPerMemory: 2, MaxPerBundle: 12, SummaryMaxChars: 256}
	k1 := RecallBundleKey(nil, 5, 0, 0, "", "", nil, "", "", 0, 0, "", off, false)
	k2 := RecallBundleKey(nil, 5, 0, 0, "", "", nil, "", "", 0, 0, "", on, false)
	if k1 == k2 {
		t.Fatal("expected different keys when evidence_in_bundle is enabled")
	}
}

func TestRecallBundleKey_differsWhenCorrelationIDDiffers(t *testing.T) {
	k1 := RecallBundleKey(nil, 5, 0, 0, "", "", nil, "", "", 0, 0, "sess-a", EvidenceBundleKey{}, false)
	k2 := RecallBundleKey(nil, 5, 0, 0, "", "", nil, "", "", 0, 0, "sess-b", EvidenceBundleKey{}, false)
	if k1 == k2 {
		t.Fatal("expected different keys when correlation_id differs")
	}
}

func TestRecallBundleKey_differsWhenSkipExperienceHydrationDiffers(t *testing.T) {
	ev := EvidenceBundleKey{}
	k1 := RecallBundleKey(nil, 5, 0, 0, "", "", nil, "", "", 0, 0, "", ev, false)
	k2 := RecallBundleKey(nil, 5, 0, 0, "", "", nil, "", "", 0, 0, "", ev, true)
	if k1 == k2 {
		t.Fatal("expected different keys when skip_experience_hydration differs")
	}
}
