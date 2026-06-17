# Public vs local documentation

Pluribus separates **what ships in the public repo** from **what stays on your machine** for editor workflows and internal audit history.

## Public (`docs/` — tracked in git)

Intended for clones, contributors, operators, and integrators.

| Category | Examples | Index |
|----------|----------|-------|
| Product doctrine | [memory-doctrine.md](memory-doctrine.md), [product-doctrine.md](product-doctrine.md) | [README.md](README.md) |
| Agent contract | [agent-facing-memory-contract.md](agent-facing-memory-contract.md), Phase 11 telemetry/utility docs | README § Phase 11 |
| Wire truth | [http-api-index.md](http-api-index.md), [rest-test-matrix.md](rest-test-matrix.md) | README § Public release |
| Operator | [pluribus-operational-guide.md](pluribus-operational-guide.md), [local-server-upgrade-runbook.md](local-server-upgrade-runbook.md) | README |
| Integrations | [integrations/](integrations/) — Cursor, Claude, generic MCP | [integrations/README.md](integrations/README.md) |
| Proof receipts | [pluribus-proof-index.md](pluribus-proof-index.md), [evaluation.md](evaluation.md) | Proof index |
| Historical product | [archive/](archive/) — bannered **ARCHIVED**, not active truth | README § Archived |

**Rule:** If a stranger clones the repo and needs it to run, adopt, or integrate Pluribus — it belongs here.

## Local-only (gitignored)

Not published; safe for personal workflow, Cursor commands, and phase audit noise.

| Path | Contents |
|------|----------|
| [local/](local/) | Editor rituals, benefit eval commands, work-order/constitution formats — see [local/README.md](local/README.md) |
| `reports/` | Phase prechange / implementation / sub-reports (Phases 1–12+) |

**Rule:** If it only makes sense with **your** `.cursor/`, **your** `archive/memory-bank/`, or **your** hostile-audit close-outs — keep it local.

## Also local (repo root, not under `docs/`)

These are outside this policy file but follow the same principle:

- `.cursor/` — rules, commands, MCP config (may contain API keys)
- `archive/memory-bank/` — historical sprint logs
- `.taskmaster/`, `active.md`, `decisions.md` — session state
- `.env`, `control-plane/configs/config.local.yaml` — secrets and local config

## Maintainer checklist

Before opening a public PR that touches docs:

1. New **product** or **operator** doc → top-level `docs/` or `docs/integrations/`, link from [README.md](README.md).
2. New **phase close-out** → `docs/reports/` (stays gitignored).
3. New **Cursor command / personal ritual** → `docs/local/`.
4. Do not link public docs to `docs/reports/` — link to proof commands, [evaluation.md](evaluation.md), or [http-api-index.md](http-api-index.md) instead.

## Gitignored local paths (repo root)

These stay on your machine and are listed in [`.gitignore`](../.gitignore):

| Path | Role |
|------|------|
| `.cursor/`, `.agents/` | Editor rules, commands, skills, MCP config |
| `active.md`, `decisions.md` | Session scratchpad |
| `adoption.md`, `future-todo.md`, `roadmap.md` | Personal planning notes |
| `archive/` | Historical memory-bank (not active truth) |
| `docs/reports/`, `docs/local/*` | Phase audits and editor workflows |
| `artifacts/*-benchmark.json`, `artifacts/agent-*.json`, … | Regenerated proof outputs (`make proof-*`) |
| `control-plane/controlplane`, `control-plane/pluribus-mcp` | Built binaries (`make build`) |
