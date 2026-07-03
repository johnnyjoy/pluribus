# Platform comparison matrix (Phase 12D)

Enforcement tiers: **0** doc only · **1** rules+MCP · **2** auto recall trigger · **3** mandatory preflight · **4** mandatory postflight · **5** closed loop.

| Platform | Integration path | Platform max tier | Current Pluribus tier | Gap | Status | Install verify | Smoke verify | Distribution | Mandatory-loop confidence | Next action |
|----------|------------------|-------------------|----------------------|-----|--------|----------------|--------------|--------------|---------------------------|-------------|
| **Cursor** | `integrations/cursor/` | 3 | 2 | below_platform_capability | semi_mandatory | `verify-cursor-pack.sh --static` | `helper/verify-mcp` + live MCP | Pack only (not VSIX/marketplace) | Medium — rules+MCP, agent can skip | Optional future Cursor plugin with hooks |
| **VS Code** | `integrations/vscode/extension/` | 4 | 3–4 | meets_platform_maximum (REST scope) | semi_mandatory | `verify-vscode-extension.sh --static` | Task/debug orchestrator + checklist command | VSIX local; **not** Marketplace-published | Medium-high for task/debug; low for Copilot chat | Document MCP optional path |
| **OpenClaw** | `integrations/openclaw/` | 2 | 1–2 | below_platform_capability | advisory_only | Manual + policy template | `openclaw mcp` CLI (vendor) | Template pack | Low — policy + MCP registration | Refresh vendor CLI docs |
| **Claude Code (light)** | `integrations/claude-code/` | 2 | 1 | below_platform_capability | advisory_only | MCP example JSON | MCP tools visible in Code | Template | Low | Point users to first-party plugin |
| **Claude Code (plugin)** | `integrations/claude-code-plugin/` | 5 | 3 | below_platform_capability | semi_mandatory | `verify-claude-code-plugin.sh --static` | SessionStart + UserPromptSubmit hooks | Local marketplace catalog; **not** Anthropic submit | Medium-high locally; **none** in remote cloud | Optional PreToolUse (deferred — risky) |
| **Claude Desktop** | `integrations/claude-desktop/` | 1 | 1 | meets_platform_maximum | template_only | stdio MCP config template | Restart + tools/list | Template | Low — instructions only | Honest tier label in README |
| **OpenCode** | `integrations/opencode/` | 2 | 1 | below_platform_capability | template_only | AGENTS template | Vendor MCP list | Template | Low | Strengthen AGENTS template |
| **Continue** | `integrations/continue/` | 2 | 1 | below_platform_capability | template_only | `.continue/rules` | Continue MCP UI | Template | Low | Rules + MCP example |
| **Zed** | `integrations/zed/` | 2 | 1 | below_platform_capability | template_only | agent-context template | Zed MCP settings | Template | Low | MCP verify steps |
| **Generic MCP** | `integrations/generic-mcp/` | 2 | 1–2 | template_only_but_honest | measured_only | `verify-generic-mcp.sh --static` | curl initialize | Portable examples | Low unless client enforces tools | Keep examples current |

**Tiers (legacy doc):** **1** = deepest in-repo packs. **2** = strong MCP + templates. **3** = MCP + minimal rules.

**Adoption:** [usage.md](usage.md) · **Skills → tools:** [skills-model.md](skills-model.md).

**MCP surface:** `POST /v1/mcp`, **59 tools** (includes `list_chores` / `resolve_chore` agent-driven curation) — verify with `scripts/integrations/verify-mcp-surface.sh`.

**Do not claim:** marketplace publication, mandatory enforcement on advisory-only platforms, or automatic telemetry without tool calls.
