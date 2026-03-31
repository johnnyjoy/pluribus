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

## Rules

Desktop does not use Cursor-style rule files. Apply **system prompt** or **project instructions** manually, or keep a short doc open—see **[integrations/claude-desktop/rules.md](../../integrations/claude-desktop/rules.md)** for pasteable text.

## Limitations

- **Prompts/resources** on the **HTTP** service may not appear when using **stdio** `pluribus-mcp` (tools are aligned; see [mcp-service-first.md](../mcp-service-first.md)).
- Fully quit and reopen Desktop after config changes.
