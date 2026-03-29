# PLURIBUS SPRINT — REMOVE DEPENDENCE ON DISCIPLINE (2026-03-27)

## Enforcement Gaps Found (Baseline)
1. `run-multi` did not auto-recover when explicit triggered recall was omitted for risky/decision proposals.
2. Validation context compile in `run-multi` did not use a query derived from the active prompt by default.
3. No dedicated intervention signal log for discipline-recovery actions.

## Intervention Logic Added
- **Auto recall recovery in run-multi** (`internal/recall/service.go`)
  - Detects risky/decision boundaries via existing `DetectTriggers`.
  - Injects retrieval query automatically when:
    - trigger system is enabled,
    - no retrieval query already set,
    - risk/decision trigger matched.
  - Intervention events:
    - `missed_recall -> recall_injected`
    - `risk_detected -> revise`
- **Validation context hardening**
  - Behavior-validation compile now uses:
    - `in.RetrievalQuery` when available, else
    - current `req.Query`.
- **Explicit intervention logging**
  - Added minimal logs:
    - `[INTERVENTION] type=... action=... reason=...`
  - Added debug payload key:
    - `debug.filter_reasons.interventions`.

## Files Changed
- `control-plane/internal/recall/service.go`
- `control-plane/internal/recall/service_runmulti_test.go`
- `control-plane/internal/eval/scenarios/discipline-skip-recall-001.json`
- `control-plane/internal/eval/scenarios/discipline-weak-proposal-001.json`

## Tests Added
- `TestService_RunMulti_autoInjectsRecallOnRiskWhenTriggeredNotExplicit`
- `TestService_RunMulti_doesNotInjectRecallForLowSignalShortQuery`
- Eval scenarios:
  - `discipline-skip-recall-001`
  - `discipline-weak-proposal-001`

## Before / After Behavior Examples

### Example A — Missed recall on risky action
- **Before:** `run-multi` with risky prompt and no `enable_triggered_recall` could run without retrieval query enrichment.
- **After:** system auto-injects retrieval query via existing trigger fragments and logs intervention.

### Example B — Weak/incomplete decision proposal
- **Before:** validation context could be less aligned with prompt intent when retrieval query omitted.
- **After:** validation context compile uses prompt-derived query fallback, improving deterministic guard checks.

## Eval Results
- `cd control-plane && go test ./internal/eval -v` ✅
- `cd control-plane && go test ./...` ✅
- `make regression` ✅

## Remaining Gaps
1. Intervention currently targets `run-multi` path; explicit compile path remains opt-in (`enable_triggered_recall`).
2. Risk classification still heuristic-token based; robust but intentionally lightweight.
3. Constraint-block semantics remain enforced through validation outcomes and blocked flags, but no new hard transport-level status code change was introduced in this sprint.
