# Platform comparison matrix

| Tier | Platform | MCP | Rules / instructions | `skill.md` / packs | Config / packaging | Recommended Pluribus mode | Verify |
|------|----------|-----|----------------------|---------------------------|--------------------|---------------------------|--------|
| **1** | **Cursor** | Yes (HTTP or stdio) | **User rules** + optional **`.cursor/rules/pluribus.mdc`** | **`skill.md`**, **`skills/pluribus/`** → **`~/.cursor/skills/`** | **`~/.cursor/mcp.json`** (ideal) | HTTP **`POST /v1/mcp`** + rules | [helper/verify-mcp.sh](../../integrations/cursor/helper/verify-mcp.sh); tools in Agent |
| **1** | **OpenClaw** | Yes (CLI + gateway) | **`policy.template.md`** | **`openclaw/skill.md`** | **`openclaw mcp set`** per [openclaw.md](openclaw.md) | HTTP or stdio | MCP listed; policy loads loop |
| **2** | **Claude Code** | Yes | **`CLAUDE.template.md`** → **`CLAUDE.md`** | **`skill.md`**, **`skills/pluribus/`** | Project MCP JSON | HTTP or stdio | Tools visible in Code |
| **2** | **Claude Desktop** | Yes (stdio typical) | **`custom-instructions.template.md`** | **`skill.md`**, **`skills/pluribus/`** | `claude_desktop_config.json` | **stdio** `pluribus-mcp` | Restart; check MCP |
| **2** | **OpenCode** | Yes | **`AGENTS.template.md`** | **`skill.md`**, **`skills/pluribus/`** | `opencode.json` | Remote **`/v1/mcp`**, **`oauth`: false** | `opencode` MCP list |
| **3** | **Continue** | Yes | **`.continue/rules/pluribus.md`** | **`skill.md`** | `config.yaml` / JSON | HTTP MCP | Rules + MCP in Continue |
| **3** | **Zed** | Yes | **`agent-context.template.md`** | **`skill.md`** | Settings | HTTP where available | Agent instructions + MCP |
| **1** | **VS Code** | Yes | **`.github/copilot-instructions.md`** template + optional **`extension/`** | **`skill.md`** | `mcp.json` + **`pluribus.baseUrl`** | HTTP MCP or **REST extension** | Recall/record commands; Output + Explorer sidebar |
| **3** | **Generic MCP** | — | **[`pluribus-instructions.md`](../../integrations/pluribus-instructions.md)** | **`skill.md`**, **`skills/pluribus/`** | Any client | **`/v1/mcp`** | JSON-RPC `tools/list` |

**Tiers:** **1** = deepest in-repo packs (MCP + rules + prompts/helpers where applicable). **2** = strong MCP + templates. **3** = MCP + minimal rules; verify against vendor docs after upgrades.

**Adoption and failure modes:** [usage.md](usage.md). **Skills → tools:** [skills-model.md](skills-model.md).

**Maturity:** Cursor + generic HTTP MCP are the most **documented in-repo** (see [mcp-usage.md](../mcp-usage.md), [evaluation.md](../evaluation.md)). Other rows follow vendor docs; re-verify after major client upgrades.
