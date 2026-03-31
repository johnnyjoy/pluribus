# Claude Code (CLI)

Claude Code supports **MCP servers** from the project or user configuration. Pluribus should be registered like any other MCP server pointing at the **same** control-plane URL you use for HTTP tests.

## MCP

- **HTTP:** If your Claude Code build supports **URL-based MCP**, use `http://127.0.0.1:8123/v1/mcp` and **`X-API-Key`** when auth is enabled.
- **Stdio:** Point **`command`** at **`pluribus-mcp`** and set **`CONTROL_PLANE_URL`** (and **`CONTROL_PLANE_API_KEY`** when needed).

**Exact file locations and schema** change with Anthropic releases—use **`/mcp` or MCP settings** inside Claude Code to register servers, then align with our example: **[integrations/claude-code/mcp-config.example.json](../../integrations/claude-code/mcp-config.example.json)** (comments describe merge strategy; JSON may need to be split per vendor format).

Official reference: [Claude Code MCP documentation](https://docs.anthropic.com/en/docs/claude-code/mcp) (verify current URL).

## Rules / skill

- **Canonical:** **[integrations/pluribus-instructions.md](../../integrations/pluribus-instructions.md)**. Copy **[integrations/claude-code/CLAUDE.template.md](../../integrations/claude-code/CLAUDE.template.md)** to project-root **`CLAUDE.md`**, then append **`snippets/context-prime.txt`** ([Claude Code](https://docs.anthropic.com/en/docs/claude-code)).
- **Agent Skill:** copy **[integrations/claude-code/skills/pluribus/](../../integrations/claude-code/skills/pluribus/)** into your project skills path, or paste **[integrations/claude-code/skill.md](../../integrations/claude-code/skill.md)** into **`CLAUDE.md`**. Pack: **[integrations/claude-code/README.md](../../integrations/claude-code/README.md)**.

## Limitations

- Schema for “where MCP JSON lives” is **vendor-defined**—do not treat the example file as a drop-in without checking your Claude Code version.
