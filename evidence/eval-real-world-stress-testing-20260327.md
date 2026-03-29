# PLURIBUS SPRINT — REAL-WORLD STRESS TESTING (2026-03-27)

## Scope
- Goal: validate behavior under long/messy workflows without tuning the system mid-test.
- Mode: additive extension to `internal/eval` (legacy scenarios preserved).

## Scenarios Added
1. `stress-long-workflow-001` (`long_workflow`)
2. `stress-resume-001` (`resume`)
3. `stress-mixed-task-001` (`mixed_task`)
4. `stress-multi-agent-001` (`multi_agent`)
5. `stress-failure-injection-001` (`failure_injection`)

## Harness Extensions (Build Artifacts)
- Optional stress schema and step model in `internal/eval/types.go`.
- Stepped scenario execution + drift classification in `internal/eval/stress.go`.
- Runner/report integration with required logs:
  - `[WORKFLOW STEP]`
  - `[RESUME]`
  - `[DRIFT]`
  - scenario-level behavior summary (`continuity/failure/pattern/drift/issues`)

## Gate Results
- `cd control-plane && go test ./internal/eval -v` ✅
- `cd control-plane && go test ./...` ✅
- `make regression` ✅

## Scenario Summary

### stress-long-workflow-001
- continuity maintained: **yes**
- failure avoided: **yes**
- pattern reused: **yes**
- drift detected: **no**
- issues: none

### stress-resume-001
- continuity maintained: **yes**
- failure avoided: **yes**
- pattern reused: **yes**
- drift detected: **no**
- issues: none

### stress-mixed-task-001
- continuity maintained: **yes**
- failure avoided: **yes**
- pattern reused: **yes**
- drift detected: **no**
- issues: none

### stress-multi-agent-001
- continuity maintained: **yes**
- failure avoided: **yes**
- pattern reused: **yes**
- drift detected: **no**
- issues: none

### stress-failure-injection-001
- continuity maintained: **yes**
- failure avoided: **yes**
- pattern reused: **yes**
- drift detected: **no**
- issues: none

## Representative Log Blocks

```text
[WORKFLOW STEP]
task: deploy
action: inject conflicting proposal and noisy memory
recall_used: yes
```

```text
[RESUME]
restored_state: 2
restored_constraints: 2
```

```text
[DRIFT]
type: reintroduced_failure
cause: did not avoid failure action: <term>
```

> Note: the final passing run emitted no active drift events for the five added stress scenarios.

## Failure Mode Analysis
- Observed hard failures in final run: **none**.
- During implementation, one transient false-positive drift signal appeared in mixed-task scenario design (term overlap in `must_avoid` phrase). This was corrected at scenario definition level, not by changing runtime behavior.
- Failure attribution map (current evidence): no confirmed runtime regressions in:
  - retrieval
  - authority/ranking
  - constraint enforcement
  - pattern reuse
  - validation flow

## Residual Risks / Recommendations
1. Current stress runner uses deterministic in-memory search pool; it does not yet emulate noisy retrieval ranking variance over persisted long histories.
2. Add one future scenario with intentionally expected drift (`must_detect`) to continuously verify drift telemetry paths.
3. Add agent/task tag filters to step checks if stricter isolation guarantees are required.
