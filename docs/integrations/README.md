# Pluribus — AI editors and agent systems

This directory is the **integration hub**: how to connect Pluribus as **shared institutional memory** (MCP-first) across editors and agent runtimes.

**Product truth:** Pluribus is the **memory substrate and cognitive extension** for agents—not a generic sidecar. **MCP** is the primary agent operating surface; **REST** is for tests, verification, service integration, and admin—see [memory-doctrine.md](../memory-doctrine.md).

**Terminology:** use **ingest channel** (e.g. advisory `source`) and **distill mode** (e.g. `pluribus_distill_origin` on candidates) consistently—see [memory-doctrine.md](../memory-doctrine.md).

---

## Start here

| Doc | Purpose |
|-----|---------|
| **[matrix.md](matrix.md)** | Platform comparison — MCP, rules, skills, maturity |
| **[generic-mcp.md](generic-mcp.md)** | Any MCP-capable client — URLs, auth, default tool loop |
| **[cursor.md](cursor.md)** | Cursor — MCP, rules, workflow |
| **[claude-code.md](claude-code.md)** | Claude Code (CLI) — MCP, rules |
| **[claude-desktop.md](claude-desktop.md)** | Claude Desktop — stdio MCP, limitations |
| **[openclaw.md](openclaw.md)** | OpenClaw — MCP CLI, gateway, aggressive defaults |
| **[opencode.md](opencode.md)** | OpenCode — `opencode.json` MCP (remote/local), AGENTS.md, skills |
| **[continue.md](continue.md)** | Continue — MCP config |
| **[zed.md](zed.md)** | Zed — MCP where supported |
| **[vscode.md](vscode.md)** | VS Code — MCP / Copilot-adjacent setups |

**Copy-paste artifacts** (rules, skills, example JSON) live under **`../../integrations/`** at repo root—see [integrations/README.md](../../integrations/README.md).

---

## Default agent loop (all platforms)

1. **Recall before substantive work:** **`recall_context`** (task text in; bundle + `mcp_context` out). Alias: **`memory_context_resolve`**.
2. **Capture experience:** **`record_experience`** (summary) or opportunistic **`memory_log_if_relevant`** / **`auto_log_episode_if_relevant`**. Alias: **`mcp_episode_ingest`**.
3. **Gate risky changes:** `enforcement_evaluate` when the product calls for it.
4. **Optional depth:** curation / contradictions / relationships tools—pull-based, not required for the core learning loop.

Details: [mcp-poc-contract.md](../mcp-poc-contract.md), [mcp-usage.md](../mcp-usage.md).

---

## Doctrine guardrails (all integrations)

- **No** project / task / workspace / **scope** as memory partitions or required recall selectors—see [anti-regression.md](../anti-regression.md).
- Memory is **global**; **tags** + **retrieval_query** shape the situation.
- **Non-destructive evolution**—no “delete memory” as the user story; candidates and promotion handle learning.

---

## Verification

Example configs are **syntactic templates**; always point **`url`** / **`CONTROL_PLANE_URL`** at your real Pluribus base (default dev: `http://127.0.0.1:8123`). With **`PLURIBUS_API_KEY`** on the server, set **`X-API-Key`** (HTTP MCP) or **`CONTROL_PLANE_API_KEY`** (stdio adapter). Automated per-client testing is **not** shipped in this repo—validate in your client after paste.
