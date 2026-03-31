# MCP usage — ensure the agent actually calls Pluribus

**Canonical surface:** `POST /v1/mcp` on the control-plane base URL (same host as REST). Stdio `pluribus-mcp` is a thin proxy to the same tools.

## What you must do

1. **Wire MCP** in the client (HTTP to `/v1/mcp` preferred). Reachability: `curl` **`/healthz`** and **`/readyz`** on the API base.
2. **Add rules** so the model is nudged every session—see [agent-rules.md](agent-rules.md). Without rules, tools sit unused.
3. **Default tool loop:** **`recall_context`** before substantive work; **`record_experience`** (or opportunistic **`memory_log_if_relevant`**) after learning signals. Aliases **`memory_context_resolve`** / **`mcp_episode_ingest`** unchanged. Order and options: [mcp-usage.md](../../mcp-usage.md#recall-driven-workflow-recommended-order).

## What MCP gives you

- Tools in the **same loop** as codegen—no separate “memory app.”
- **JSON-RPC** tool calls with bodies aligned to REST (see [mcp-poc-contract.md](../../mcp-poc-contract.md)).

## Auth

If the server sets **`PLURIBUS_API_KEY`**, MCP HTTP clients must send **`X-API-Key`**. For stdio, set **`CONTROL_PLANE_API_KEY`**. See [authentication.md](../../authentication.md).

## Optional automation

- Server-side **auto-distill** from advisory episodes depends on **`distillation`** config (`distillation.enabled`, `distillation.auto_from_advisory_episodes`). If disabled, distill explicitly via **`POST /v1/episodes/distill`** or MCP **`episode_distill_explicit`**.

## Verify it is working

- **MCP:** `tools/list` lists **`recall_context`** and **`record_experience`** first; **`recall_context`** returns `mcp_context` + `recall_bundle` when the pool has matches.
- **Data:** new rows in advisory episodes and/or pending candidates over time—not just empty recall.
