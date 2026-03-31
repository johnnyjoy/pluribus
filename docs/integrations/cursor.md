# Cursor

Cursor is a **first-class** target: **HTTP MCP** to Pluribus, optional **Cursor Rules** under `.cursor/rules/`, and optional **skills**.

## MCP (recommended)

**Preferred:** HTTP to the control-plane (no local binary required).

- Config: **repository** `.cursor/mcp.json` or **global** `~/.cursor/mcp.json`
- URL: `http://127.0.0.1:8123/v1/mcp` (adjust for your host)
- With API key: `headers.X-API-Key` → `${env:PLURIBUS_API_KEY}`

Full templates: [mcp-usage.md](../mcp-usage.md), copy-ready file: **[integrations/cursor/mcp-config.example.json](../../integrations/cursor/mcp-config.example.json)**.

**Fallback:** stdio **`pluribus-mcp`** with `CONTROL_PLANE_URL`—see root [README.md](../../README.md).

## Rules

Paste or symlink content from **[integrations/cursor/rules.md](../../integrations/cursor/rules.md)** into `.cursor/rules/` (see [Cursor rules](https://cursor.com/docs/context/rules)).

## Skills / behavior templates

Portable behaviors: **[integrations/cursor/skills.md](../../integrations/cursor/skills.md)**. Optional: wrap as Cursor **skills** per your workflow.

## Workflow

1. Start Pluribus (`docker compose up -d` or your deployment).
2. Enable MCP in Cursor; confirm **`tools/list`** includes Pluribus tools.
3. Use **`recall_context`** early in substantive tasks; **`record_experience`** after meaningful outcomes (aliases **`memory_context_resolve`** / **`mcp_episode_ingest`** still valid).

Troubleshooting: [mcp-usage.md](../mcp-usage.md#cursor-specific-behavior).

## Limitations

- Cursor does not ship Pluribus; you must run the API.
- Tool lists are **runtime**—rebuild `pluribus-mcp` after server upgrades if using stdio.
