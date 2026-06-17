---
name: memory-health
description: Check that Pluribus is reachable before relying on memory tools. Use when MCP errors, timeouts, or disconnects appear, or when validating a new install.
---

# Memory health

**HTTP (control plane)**

- `GET ${PLURIBUS_BASE_URL:-http://127.0.0.1:8123}/healthz` with optional header `X-API-Key` when `PLURIBUS_API_KEY` is set.

**MCP**

- Service-first endpoint: **`POST /v1/mcp`** on the same base URL (see plugin `.mcp.json`).
- If tools fail, verify URL, API key, and that the server process is running.

If **`wakeup_context`** returns empty **`identity`**, the active pool likely has no **`state`** memories yet—bootstrap with normal ingest or run **`scripts/seed-l0-identity-memories.sh`** from the repo (see Claude Code plugin README).

This skill does **not** change server policy; it only helps confirm connectivity.
