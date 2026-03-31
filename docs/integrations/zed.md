# Zed

Zed has been adding **MCP** support; exact UI and config paths depend on version. Use Pluribus like any other MCP server: **HTTP** endpoint **`http://<host>:8123/v1/mcp`** with **`X-API-Key`** when required.

## Setup

1. Open Zed **Settings** / **MCP** (wording varies by version).
2. Add a server with base URL matching your Pluribus deployment and path **`/v1/mcp`**.
3. Add auth header if needed.

**Example stub:** **[integrations/zed/mcp-config.example.json](../../integrations/zed/mcp-config.example.json)** — illustrative only; confirm against [Zed documentation](https://zed.dev/docs) for MCP.

## Rules

Use **[integrations/zed/rules.md](../../integrations/zed/rules.md)** as assistant / project instructions if Zed exposes a field for them.

## Limitations

- MCP availability and schema differ by Zed release—**verify in-product** after upgrades.
