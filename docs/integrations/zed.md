# Zed

Zed has been adding **MCP** support; exact UI and config paths depend on version. Use Pluribus like any other MCP server: **HTTP** endpoint **`http://<host>:8123/v1/mcp`** with **`X-API-Key`** when required.

## Setup

1. Open Zed **Settings** / **MCP** (wording varies by version).
2. Add a server with base URL matching your Pluribus deployment and path **`/v1/mcp`**.
3. Add auth header if needed.

**Example stub:** **[integrations/zed/mcp-config.example.json](../../integrations/zed/mcp-config.example.json)** — illustrative only; confirm against [Zed documentation](https://zed.dev/docs) for MCP.

## Rules + skill

**Canonical:** **[integrations/pluribus-instructions.md](../../integrations/pluribus-instructions.md)**. Paste **[integrations/zed/agent-context.template.md](../../integrations/zed/agent-context.template.md)** into Zed **Agent** instructions (no shared repo path across versions); add **`snippets/context-prime.txt`**. Use **[integrations/zed/skill.md](../../integrations/zed/skill.md)** for the step table. Pack: **[integrations/zed/README.md](../../integrations/zed/README.md)**.

## Limitations

- MCP availability and schema differ by Zed release—**verify in-product** after upgrades.
