# Agent Contract Obedience

Phase 11H defines what it means for an agent to **obey** the Pluribus memory contract — distinct from merely receiving structured recall.

## Core principle

**Recall is not use.** An agent that receives lifecycle metadata but treats historical context as current guidance has failed obedience even if the contract was complete.

**Self-report is not proof.** Telemetry must tie recalled memory IDs, used memory IDs, contract-field citations, and machine-checkable output facts.

## Obedience states

| State | Meaning |
|-------|---------|
| `used_correctly` | Current guidance used with scope match and citations |
| `ignored_correctly` | Memory correctly excluded (wrong scope, refuted, etc.) |
| `historical_only_correctly` | Historical context acknowledged but not used as guidance |
| `misused` | Memory used against contract discipline |
| `unsafe_use` | Missing scope or other unsafe application |
| `unsupported_claim` | Output fact not supported by used memories |
| `uncited_memory_use` | Used memory not cited in output |
| `missing_telemetry` | Required memory-use telemetry absent |

## What counts as use

An agent **uses** a memory when it:
- Includes the memory ID in `used_memory_ids`
- Cites the memory in `final_output.memory_citations`
- Emits `memory_decisions` with `decision=used` and contract-field citations
- Supports output facts listed in `output_facts_supported`

## What counts as misuse

- Historical/refuted/superseded/archived memory guiding current action
- Wrong-scope or negative-scope violation
- Unrecalled memory used or cited
- Unsupported output claims
- Missing telemetry or missing contract-field citations

## Scripted harness (not LLM)

Three deterministic agent modes prove the evaluator:

| Mode | Purpose |
|------|---------|
| `obedient_agent` | Follows `DecideUseDiscipline`; full telemetry |
| `sloppy_agent` | Injects sloppy behaviors; must be **detected** |
| `malicious_or_broken_agent` | Injects hostile behaviors; must be **detected** |

No LLM judge. No external AI scoring.

## Memory-use telemetry

Schema: `control-plane/internal/agentobedience/types.go`

Required linkage:
- `recalled_memory_ids` ← from recall bundle
- `used_memory_ids` ⊆ recalled (except detected violations)
- `memory_decisions[].contract_fields_cited` for each use
- `final_output.facts` ← rubric-checked against expected/forbidden facts

## MCP and REST/API

Both interfaces tested via in-process httptest + MCP `HandleToolsCall` (current checkout only).

## Make targets

```bash
make test-agent-contract-obedience
make agent-contract-obedience-benchmark
make proof-agent-contract-obedience
```

Artifact: `artifacts/agent-contract-obedience-benchmark.json`

## Honest limits

- Proves deterministic scripted obedience, **not** production LLM obedience
- Optional real-agent smoke is intentionally **not** implemented (non-gating)
- Telemetry is benchmark-scoped; production persistence is a future gap
