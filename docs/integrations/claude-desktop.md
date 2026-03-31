# Claude Desktop

Claude Desktop historically favors **stdio MCP servers**. Pluribus ships **`pluribus-mcp`** for that path; **HTTP MCP** may be available depending on Desktop version—prefer **stdio** for maximum compatibility.

## Config file (typical paths)

- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`
- Linux: `~/.config/Claude/claude_desktop_config.json`

## Stdio example

Build the binary: `cd control-plane && go build -o pluribus-mcp ./cmd/pluribus-mcp`

Use **absolute path** to the binary in **`command`**. Set **`env.CONTROL_PLANE_URL`** to your API base.

Copy-ready: **[integrations/claude-desktop/mcp-config.example.json](../../integrations/claude-desktop/mcp-config.example.json)** (same shape as [mcp-usage.md](../mcp-usage.md)).

## Rules + skill

**Canonical:** **[integrations/pluribus-instructions.md](../../integrations/pluribus-instructions.md)**. Paste **[integrations/claude-desktop/custom-instructions.template.md](../../integrations/claude-desktop/custom-instructions.template.md)** into Desktop **custom instructions**; add **`snippets/context-prime.txt`**. Optional **[integrations/claude-desktop/skill.md](../../integrations/claude-desktop/skill.md)**. Pack: **[integrations/claude-desktop/README.md](../../integrations/claude-desktop/README.md)**.

## Limitations

- **Prompts/resources** on the **HTTP** service may not appear when using **stdio** `pluribus-mcp` (tools are aligned; see [mcp-service-first.md](../mcp-service-first.md)).
- Fully quit and reopen Desktop after config changes.
