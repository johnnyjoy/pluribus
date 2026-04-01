# OpenClaw — deep adoption (real integrations only)

OpenClaw does **not** expose a stable, documented “lifecycle hook” API in this repository. **Deep integration** here means:

1. **MCP** — Register Pluribus (`HTTP POST /v1/mcp` or stdio `pluribus-mcp`) so the agent can call **`recall_context`** and **`record_experience`**.
2. **Policy** — Pin **[`policy.template.md`](policy.template.md)** + **[`snippets/context-prime.txt`](snippets/context-prime.txt)** in the gateway **system / policy** field so every run is primed with the mandatory loop.
3. **Skill** — Optional **[`skill.md`](skill.md)** for the step table inline.

That combination is what makes Pluribus part of the **execution loop**: the model sees the loop in policy, and the tools exist to run it.

## What we do *not* ship

- **No fake plugin** that claims to intercept OpenClaw’s internal task scheduler without a real vendor API.
- **No invented JSON** for “before every task” / “after every task” unless OpenClaw documents it for your version.

If your OpenClaw build adds **documented** hooks (e.g. pre/post shell, webhook, or plugin entry points), wrap **`curl`** calls to **`POST /v1/recall/compile`** and **`POST /v1/advisory-episodes`** the same way you would for any HTTP client—still backed by this control plane.

## Verification

- **`openclaw mcp`** (or your version’s equivalent) lists Pluribus with the URL you configured.
- Agent transcripts or logs show **`recall_context`** (or **`memory_context_resolve`**) before substantive work and **`record_experience`** after meaningful outcomes.
- Optional: **`GET /v1/curation/pending`** shows candidates when distillation/promotion is in play.

Canonical behavior: **[`../pluribus-instructions.md`](../pluribus-instructions.md)** · Hub: **[../../docs/integrations/openclaw.md](../../docs/integrations/openclaw.md)**
