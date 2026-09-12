# Agent Memory Contract Endpoint Coverage

Phase 11G explicitly benchmarks every agent-facing recall surface.

## Agent-facing inventory

| Surface | Type | Agent-Facing? | Returns MemoryItem? | Returns RecallBundle? | Contract Evaluated? | Field Parity Required? | Current Gap |
|---------|------|---------------|---------------------|------------------------|---------------------|------------------------|-------------|
| POST /v1/recall/compile | REST | yes | via buckets | yes | yes | yes | none |
| POST /v1/recall/compile-multi | REST | yes | via variant bundles | yes | yes | yes | none |
| GET /v1/recall/ | REST | yes | via buckets | yes | yes | yes | none |
| POST /v1/recall/wakeup | REST | yes | yes (identity + governing) | no | yes | yes | none |
| POST /v1/recall/run-multi | REST | yes | indirect via compile-multi | no | orchestration + compile-multi substrate | no (orchestration) | none |
| recall_context | MCP | yes | via bundle | yes | yes | yes | none |
| memory_context_resolve | MCP alias | yes | via bundle | yes | yes | yes | none |
| recall_compile | MCP | yes | via bundle | yes | yes | yes | none |
| memory_recall_advanced | MCP | yes | via bundle | yes | yes | yes | none |
| recall_get | MCP | yes | via bundle | yes | yes | yes | none |
| wakeup_context | MCP | yes | yes | no | yes | yes | none |
| recall_run_multi | MCP | yes | indirect | no | structured JSON + compile-multi substrate | no (orchestration) | none |

## Non-agent-facing exclusions

- `POST /v1/recall/preflight` — risk probe, no memory returned.
- Curation, enforcement, compliance, memory CRUD tools — not recall surfaces.

## MCP compile-multi

No dedicated MCP tool maps to `POST /v1/recall/compile-multi`. Agents use REST compile-multi or `recall_run_multi` orchestration. Documented exemption.

## Artifacts

- `artifacts/agent-memory-contract-endpoint-coverage.json`
- `artifacts/agent-memory-contract-parity-benchmark.json`

## Make targets

```bash
make test-agent-memory-endpoint-coverage
make agent-memory-endpoint-coverage-benchmark
make proof-agent-memory-endpoint-coverage
```
