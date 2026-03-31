# Claude Code (CLI)

Claude Code supports **MCP servers** from the project or user configuration. Pluribus should be registered like any other MCP server pointing at the **same** control-plane URL you use for HTTP tests.

## MCP

- **HTTP:** If your Claude Code build supports **URL-based MCP**, use `http://127.0.0.1:8123/v1/mcp` and **`X-API-Key`** when auth is enabled.
- **Stdio:** Point **`command`** at **`pluribus-mcp`** and set **`CONTROL_PLANE_URL`** (and **`CONTROL_PLANE_API_KEY`** when needed).

**Exact file locations and schema** change with Anthropic releases—use **`/mcp` or MCP settings** inside Claude Code to register servers, then align with our example: **[integrations/claude-code/mcp-config.example.json](../../integrations/claude-code/mcp-config.example.json)** (comments describe merge strategy; JSON may need to be split per vendor format).

Official reference: [Claude Code MCP documentation](https://docs.anthropic.com/en/docs/claude-code/mcp) (verify current URL).

## Rules / instructions

- Use project **`CLAUDE.md`** or Claude Code’s instructions field for **memory-first** behavior.
- Copy distilled rules from **[integrations/claude-code/rules.md](../../integrations/claude-code/rules.md)**.

## Skills

Portable templates: **[integrations/claude-code/skills.md](../../integrations/claude-code/skills.md)**.

## Limitations

- Schema for “where MCP JSON lives” is **vendor-defined**—do not treat the example file as a drop-in without checking your Claude Code version.
