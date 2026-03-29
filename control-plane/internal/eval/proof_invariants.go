package eval

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"control-plane/internal/recall"
)

// Executable invariant identifiers (system contract at REST boundary).
const (
	InvNoContainerDependency  = "no_container_dependency"
	InvParaphraseInvariance   = "paraphrase_invariance"
	InvFailureDominance       = "failure_dominance"
	InvConstraintEnforcement  = "constraint_enforcement"
	InvPatternDominance       = "pattern_dominance"
	InvNoiseSuppression       = "noise_suppression"
	InvAuthorityConvergence   = "authority_convergence"
	InvCrossContextReuse      = "cross_context_reuse"
)

// ProofLog emits mandatory structured proof lines.
func ProofLog(invariant string, pass bool, details string) {
	status := "fail"
	if pass {
		status = "pass"
	}
	d := strings.TrimSpace(details)
	if d == "" {
		d = "-"
	}
	log.Printf("[PROOF] invariant=%s status=%s details=%s", invariant, status, d)
}

func recallBundleMemoryIDs(body []byte) map[string]struct{} {
	var rb recall.RecallBundle
	if err := json.Unmarshal(body, &rb); err != nil {
		return nil
	}
	m := make(map[string]struct{})
	add := func(items []recall.MemoryItem) {
		for _, it := range items {
			if it.ID != "" {
				m[it.ID] = struct{}{}
			}
		}
	}
	add(rb.Continuity)
	add(rb.Constraints)
	add(rb.Experience)
	add(rb.GoverningConstraints)
	add(rb.Decisions)
	add(rb.KnownFailures)
	add(rb.ApplicablePatterns)
	return m
}

// checkParaphraseOverlap requires stored bundle_a and bundle_b with overlapping recalled memory IDs.
func checkParaphraseOverlap(ctx *ProofRunContext) (bool, string) {
	ba, okA := ctx.Stored["bundle_a"]
	bb, okB := ctx.Stored["bundle_b"]
	if !okA || !okB {
		return false, "missing bundle_a or bundle_b in store"
	}
	ma := recallBundleMemoryIDs(ba)
	mb := recallBundleMemoryIDs(bb)
	if len(ma) == 0 || len(mb) == 0 {
		return false, fmt.Sprintf("empty recall id sets (a=%d b=%d)", len(ma), len(mb))
	}
	var shared []string
	for id := range ma {
		if _, ok := mb[id]; ok {
			shared = append(shared, id)
		}
	}
	if len(shared) == 0 {
		return false, "no shared memory ids between paraphrase compiles"
	}
	return true, fmt.Sprintf("shared_ids=%v", shared)
}

// checkCrossContextOverlap requires bundle_ctx_a and bundle_ctx_b to share at least one memory id (same pattern, different tag filters).
func checkCrossContextOverlap(ctx *ProofRunContext) (bool, string) {
	ba, okA := ctx.Stored["bundle_ctx_a"]
	bb, okB := ctx.Stored["bundle_ctx_b"]
	if !okA || !okB {
		return false, "missing bundle_ctx_a or bundle_ctx_b"
	}
	ma := recallBundleMemoryIDs(ba)
	mb := recallBundleMemoryIDs(bb)
	for id := range ma {
		if _, ok := mb[id]; ok {
			return true, fmt.Sprintf("shared_id=%s", id)
		}
	}
	return false, "no shared memory across tag contexts"
}

// checkFailureDominance requires bundle_fail_dom: known failure row outranks weak constraint on authority when both surface.
func checkFailureDominance(ctx *ProofRunContext) (bool, string) {
	b, ok := ctx.Stored["bundle_fail_dom"]
	if !ok {
		return false, "missing bundle_fail_dom"
	}
	var rb recall.RecallBundle
	if err := json.Unmarshal(b, &rb); err != nil {
		return false, err.Error()
	}
	failMark := "PROOF_FAILURE_DOMINANT"
	weakMark := "PROOF_WEAK_CONSTRAINT_ONLY"
	var failAuth int
	var sawFail bool
	for _, f := range rb.KnownFailures {
		if strings.Contains(f.Statement, failMark) {
			sawFail = true
			if f.Authority > failAuth {
				failAuth = f.Authority
			}
		}
	}
	if !sawFail {
		return false, "dominant failure not present in known_failures"
	}
	for _, c := range rb.GoverningConstraints {
		if strings.Contains(c.Statement, weakMark) && c.Authority >= failAuth {
			return false, fmt.Sprintf("weak constraint authority %d should be below failure %d", c.Authority, failAuth)
		}
	}
	return true, fmt.Sprintf("failure auth %d over weak constraint bucket", failAuth)
}

// checkPatternDominance requires merged experience slices to list higher-authority pattern before lower (same retrieval situation).
func checkPatternDominance(ctx *ProofRunContext) (bool, string) {
	b, ok := ctx.Stored["bundle_pat_dom"]
	if !ok {
		return false, "missing bundle_pat_dom"
	}
	var rb recall.RecallBundle
	if err := json.Unmarshal(b, &rb); err != nil {
		return false, err.Error()
	}
	items := append(append([]recall.MemoryItem{}, rb.Experience...), rb.ApplicablePatterns...)
	idxHigh, idxLow := -1, -1
	for i, m := range items {
		if strings.Contains(m.Statement, "PROOF_PATTERN_HIGH_AUTH") && idxHigh < 0 {
			idxHigh = i
		}
		if strings.Contains(m.Statement, "PROOF_PATTERN_LOW_AUTH") && idxLow < 0 {
			idxLow = i
		}
	}
	if idxHigh < 0 || idxLow < 0 {
		return false, "missing high or low pattern marker in experience slices"
	}
	if idxHigh < idxLow {
		return true, fmt.Sprintf("high at %d before low at %d", idxHigh, idxLow)
	}
	return false, fmt.Sprintf("ordering high=%d low=%d (want high first)", idxHigh, idxLow)
}

func checkNoiseSuppression(ctx *ProofRunContext) (bool, string) {
	b, ok := ctx.Stored["bundle_noise"]
	if !ok {
		return false, "missing bundle_noise"
	}
	var rb recall.RecallBundle
	if err := json.Unmarshal(b, &rb); err != nil {
		return false, err.Error()
	}
	noisePrefix := "zzz noise proof filler"
	var sawSignal bool
	var noiseLeaked int
	scan := func(items []recall.MemoryItem) {
		for _, m := range items {
			if strings.Contains(m.Statement, "PROOF SIGNAL HUB UNIQUE") {
				sawSignal = true
			}
			if strings.Contains(m.Statement, noisePrefix) {
				noiseLeaked++
			}
		}
	}
	scan(rb.Continuity)
	scan(rb.Constraints)
	scan(rb.Experience)
	scan(rb.GoverningConstraints)
	scan(rb.Decisions)
	scan(rb.KnownFailures)
	scan(rb.ApplicablePatterns)
	if !sawSignal {
		return false, "signal memory not present in any recall slice"
	}
	if noiseLeaked > 0 {
		return false, fmt.Sprintf("low-value noise rows in bundle slices: %d", noiseLeaked)
	}
	return true, "signal present; noise fillers absent from compiled slices"
}

// checkDriftMultiMemory requires bundle_drift to contain two unique markers from seeded memories.
func checkDriftMultiMemory(ctx *ProofRunContext) (bool, string) {
	b, ok := ctx.Stored["bundle_drift"]
	if !ok {
		return false, "missing bundle_drift"
	}
	body := string(b)
	m1 := "PROOF_DRIFT_STEP_ONE"
	m2 := "PROOF_DRIFT_STEP_TWO"
	if strings.Contains(body, m1) && strings.Contains(body, m2) {
		return true, "both step memories surfaced"
	}
	return false, fmt.Sprintf("expected both markers in bundle (m1=%v m2=%v)", strings.Contains(body, m1), strings.Contains(body, m2))
}

var proofInvariantChecks = map[string]func(*ProofRunContext) (bool, string){
	InvParaphraseInvariance:  checkParaphraseOverlap,
	InvCrossContextReuse:     checkCrossContextOverlap,
	InvFailureDominance:      checkFailureDominance,
	InvPatternDominance:      checkPatternDominance,
	InvNoiseSuppression:      checkNoiseSuppression,
	// Drift suite maps to multi-step recall surfacing (no separate constant required — use after_invariants name drift_multi_surfaced).
	"drift_multi_surfaced": checkDriftMultiMemory,
}

// RunAfterInvariants executes named checks against ctx.Stored / ctx.Vars.
func RunAfterInvariants(ctx *ProofRunContext, names []string) []ProofInvariantResult {
	var out []ProofInvariantResult
	for _, name := range names {
		fn, ok := proofInvariantChecks[name]
		if !ok {
			r := ProofInvariantResult{Name: name, Pass: false, Detail: "unknown invariant key"}
			ProofLog(name, false, r.Detail)
			out = append(out, r)
			continue
		}
		pass, detail := fn(ctx)
		ProofLog(name, pass, detail)
		out = append(out, ProofInvariantResult{Name: name, Pass: pass, Detail: detail})
	}
	return out
}
