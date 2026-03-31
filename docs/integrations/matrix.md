# Platform comparison matrix

| Platform | MCP | Rules / instructions | `skill.md` / packs | Config / packaging | Recommended Pluribus mode | Notes |
|----------|-----|----------------------|---------------------------|--------------------|---------------------------|-------|
| **Cursor** | Yes (HTTP or stdio) | **User rules** (Settings) + optional **`.cursor/rules/pluribus.mdc`** | **`skill.md`**, **`skills/pluribus/`** → **`~/.cursor/skills/`** | **`~/.cursor/mcp.json`** (ideal) or repo | HTTP **`POST /v1/mcp`** + global rules | First-class; see [cursor.md](cursor.md) |
| **Claude Code** | Yes (project + user MCP) | Root **`CLAUDE.md`** from **`CLAUDE.template.md`** | **`skill.md`**, **`skills/pluribus/`** | MCP JSON in project | HTTP or stdio to same API | See [claude-code.md](claude-code.md) |
| **Claude Desktop** | Yes (stdio typical) | **`custom-instructions.template.md`** → app instructions | **`skill.md`**, **`skills/pluribus/`** | `claude_desktop_config.json` | **stdio** `pluribus-mcp` | HTTP MCP depends on Desktop version—prefer stdio per [mcp-usage.md](../mcp-usage.md) |
| **OpenClaw** | Yes (`openclaw mcp` CLI + gateway) | **`policy.template.md`** in gateway policy | **`integrations/openclaw/skill.md`** | `openclaw` CLI + config | Register Pluribus like any MCP server | See [openclaw.md](openclaw.md); verify against [OpenClaw MCP docs](https://docs.openclaw.ai/cli/mcp) |
| **OpenCode** | Yes (`mcp` remote or local in `opencode.json`) | **`AGENTS.md`** / **`instructions`**, **`AGENTS.template.md`** | **`skill.md`**, **`skills/pluribus/`** | `opencode.json` (project or `~/.config/opencode/`) | Remote HTTP **`/v1/mcp`**, set **`oauth`: false**; optional stdio | See [opencode.md](opencode.md); [OpenCode MCP servers](https://dev.opencode.ai/docs/mcp-servers) |
| **Continue** | Yes | **`.continue/rules/pluribus.md`** (from **`integrations/continue/rules/pluribus.md`**) | **`skill.md`** | `config.yaml` / JSON | HTTP MCP recommended | See [continue.md](continue.md) |
| **Zed** | Yes (recent versions) | **`agent-context.template.md`** → Agent instructions | **`skill.md`** | Settings | HTTP MCP where available | See [zed.md](zed.md) |
| **VS Code** | Yes (MCP extension / built-in path) | **`.github/copilot-instructions.md`** from template | **`skill.md`** | `mcp.json` / workspace | HTTP MCP | See [vscode.md](vscode.md) |
| **Generic MCP** | N/A | **[`pluribus-instructions.md`](../../integrations/pluribus-instructions.md)** into system prompt | **`skill.md`**, **`skills/pluribus/`** | Any JSON-RPC MCP client | **`/v1/mcp`** | See [generic-mcp.md](generic-mcp.md) |

**Maturity:** Cursor + generic HTTP MCP are the most **documented in-repo** (see [mcp-usage.md](../mcp-usage.md), [evaluation.md](../evaluation.md)). Other rows follow vendor docs; re-verify after major client upgrades.
