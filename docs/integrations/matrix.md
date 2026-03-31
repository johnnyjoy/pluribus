# Platform comparison matrix

| Platform | MCP | Rules / instructions | Skills / command packs | Config / packaging | Recommended Pluribus mode | Notes |
|----------|-----|----------------------|---------------------------|--------------------|---------------------------|-------|
| **Cursor** | Yes (HTTP or stdio) | Yes (`.cursor/rules`) | Yes (Cursor skills optional) | `.cursor/mcp.json` | HTTP **`POST /v1/mcp`** + repo rules | First-class; see [cursor.md](cursor.md) |
| **Claude Code** | Yes (project + user MCP) | Yes (CLAUDE.md, instructions) | Skills via product | MCP JSON in project | HTTP or stdio to same API | See [claude-code.md](claude-code.md) |
| **Claude Desktop** | Yes (stdio typical) | App system prompt (manual) | N/A | `claude_desktop_config.json` | **stdio** `pluribus-mcp` | HTTP MCP depends on Desktop version—prefer stdio per [mcp-usage.md](../mcp-usage.md) |
| **OpenClaw** | Yes (`openclaw mcp` CLI + gateway) | Config + gateway docs | Skills / plugins ecosystem | `openclaw` CLI + config | Register Pluribus like any MCP server | See [openclaw.md](openclaw.md); verify against [OpenClaw MCP docs](https://docs.openclaw.ai/cli/mcp) |
| **OpenCode** | Yes (`mcp` remote or local in `opencode.json`) | Yes (`instructions`, `AGENTS.md`, `.opencode/`) | Yes (`.opencode/skills/`) | `opencode.json` (project or `~/.config/opencode/`) | Remote HTTP **`/v1/mcp`**, set **`oauth`: false**; optional stdio | See [opencode.md](opencode.md); [OpenCode MCP servers](https://dev.opencode.ai/docs/mcp-servers) |
| **Continue** | Yes | `.continue` rules / prompts | Continue prompts | `config.yaml` / JSON | HTTP MCP recommended | See [continue.md](continue.md) |
| **Zed** | Yes (recent versions) | Project/docs | As supported | Settings | HTTP MCP where available | See [zed.md](zed.md) |
| **VS Code** | Yes (MCP extension / built-in path) | `.github/copilot-instructions.md` etc. | Varies | `mcp.json` / workspace | HTTP MCP | See [vscode.md](vscode.md) |
| **Generic MCP** | N/A | Paste rules into system prompt | Portable markdown | Any JSON-RPC MCP client | **`/v1/mcp`** | See [generic-mcp.md](generic-mcp.md) |

**Maturity:** Cursor + generic HTTP MCP are the most **documented in-repo** (see [mcp-usage.md](../mcp-usage.md), [evaluation.md](../evaluation.md)). Other rows follow vendor docs; re-verify after major client upgrades.
