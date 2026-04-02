# Pluribus benefit evaluation (engineering proof)

This repo’s **honest benefit proof sprint** for Cursor + **pluribus-mcp** + control-plane lives in **archived** planning notes:

| Doc | Purpose |
|-----|---------|
| [plan-pluribus-benefit-proof.md](../archive/memory-bank/plans/plan-pluribus-benefit-proof.md) | Full eval spec: systems A/B/C, metrics, phases |
| [pluribus-benefit-eval-automation.md](../archive/memory-bank/plans/pluribus-benefit-eval-automation.md) | **Scripts + Cursor commands** (automation layers) |
| [pluribus-benefit-eval-seeds.md](../archive/memory-bank/plans/pluribus-benefit-eval-seeds.md) | `memory_create` seed payloads |
| [pluribus-benefit-eval-execution-notes.md](../archive/memory-bank/plans/pluribus-benefit-eval-execution-notes.md) | Environment + protocol checklist |
| [pluribus-benefit-eval-results-template.md](../archive/memory-bank/plans/pluribus-benefit-eval-results-template.md) | Scoring tables + conclusion template |

**Cursor commands (Agent-driven):**

| Command | System |
|---------|--------|
| **`/pluribus-benefit-eval-baseline`** | **A** — no MCP; same prompts as C |
| **`/pluribus-benefit-eval`** | **C** — seed script + `recall_get` + answers |

**Shell:**

| Script | Role |
|--------|------|
| [pluribus-benefit-eval-seed](../scripts/pluribus-benefit-eval-seed) | HTTP bootstrap + seeds from [fixtures/benefit-eval-seeds.json](../scripts/fixtures/benefit-eval-seeds.json) (wire shapes per [http-api-index.md](http-api-index.md) / `CreateRequest`) |
| [pluribus-benefit-eval-check](../scripts/pluribus-benefit-eval-check) | Mechanical `GET /v1/recall/` check (e.g. PostgreSQL in bundle) |

**POC verification** (single-path smoke) remains **[cursor-verify-recall.md](cursor-verify-recall.md)** / **`/ask-cursor-verify-recall`**.

**Curation MCP (digest / materialize)** — benefit tiers live in [curation-loop.md](curation-loop.md) § *Proving curation MCP benefit*.

The benefit sprint is **broader**: baselines, multiple dimensions, explicit failure logging — **run by Cursor** following the commands above, not only by hand.

## Agent discipline (anti-evasion)

Commands are written so **the Agent executes** MCP tools and answers—not the human. Rule: [pluribus-eval-execute.mdc](../.cursor/rules/pluribus-eval-execute.mdc). If the model only *describes* `recall_get` or deflects to shell scripts, the run is **invalid** until it actually calls tools (unless MCP is unavailable).
