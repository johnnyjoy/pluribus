ARCHIVED — NOT ACTIVE SYSTEM TRUTH

---

# Foundational beta — standing proof-of-benefit baseline

This repo’s **authoritative** product is the **control-plane** HTTP service (`docker compose` → `controlplane` on `:8123`).

This document is the **minimal standing baseline** for answering:

> Does using structured recall/memory improve outcomes versus weaker context?

It is **not** a full research platform. It is a **repeatable procedure** you can run before/after changes that touch recall, memory, or agent workflows.

## What to run (lean path)

1. **Service correctness + regression (merge gate)**  
   From repo root:
   ```bash
   make regression
   ```
   Same command runs in CI (control-plane, real Postgres in Docker, integration-tagged tests).

2. **Benefit-oriented A vs C eval (human-in-the-loop + MCP)**  
   Follow **[pluribus-benefit-eval.md](pluribus-benefit-eval.md)**:
   - `/pluribus-benefit-eval-baseline` (A)
   - Seed: `scripts/pluribus-benefit-eval-seed.sh`
   - `/pluribus-benefit-eval` (C)
   - Check: `scripts/pluribus-benefit-eval-check.sh`  
   Record results in **`memory-bank/plans/pluribus-benefit-eval-results-latest.md`** (or the template there).

3. **Optional in-session sanity (not CI)**  
   **`/ask-cursor-verify-recall`** — see [cursor-verify-recall.md](cursor-verify-recall.md).

## Benefit dimensions to keep in mind

When interpreting runs, prefer concrete signals over vibes:

- **Constraint obedience** — violations decrease when recall is applied.
- **Continuity** — prior decisions/constraints show up when they should.
- **Drift** — fewer repeated mistakes / contradictions on follow-on tasks.
- **Rediscovery** — less re-deriving facts already captured in memory.

## Scope guardrail

Foundational beta hardening **does not** expand the memory model or add new product surfaces. This baseline is for **measuring** the existing system, not for shipping new features under the guise of “eval.”

## Related artifacts

- Hardening / execution plans: `memory-bank/plans/plan-pluribus-foundational-beta-hardening-20260323.md`, `memory-bank/plans/plan-pluribus-foundational-beta-execution-20260323.md`
- Benefit spec: `memory-bank/plans/plan-pluribus-benefit-proof.md`
- Deploy truth: [deployment-poc.md](deployment-poc.md), [INSTALL.md](../INSTALL.md)
