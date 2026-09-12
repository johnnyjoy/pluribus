# Integration packs (behavioral) — **Pluribus**

**Canonical behavior:** [`pluribus-instructions.md`](pluribus-instructions.md) — mandatory loop when MCP tools are available:

```text
wakeup_context / recall_context
  → pool maintenance (resolve_chore + agent_id when chores exist)
  → plan → act → memory_feedback (when recall items mattered)
  → record_experience
```

**One copy in `pluribus-instructions.md`.** Platform folders ship **native install artifacts only** — README, templates, `skills/pluribus/SKILL.md`, MCP JSON. **`skill.md` / `rules.md`** are pointer stubs; do not duplicate loop text there.

| Platform | Folder |
|----------|--------|
| **Cursor** | [`cursor/`](cursor/) — `pluribus.mdc`, skills, MCP JSON |
| **Claude Code** | [`claude-code/`](claude-code/) · plugin: [`claude-code-plugin/`](claude-code-plugin/) |
| **Claude Desktop** | [`claude-desktop/`](claude-desktop/) |
| **OpenClaw** | [`openclaw/`](openclaw/) |
| **OpenCode** | [`opencode/`](opencode/) |
| **Continue** | [`continue/`](continue/) |
| **Zed** | [`zed/`](zed/) |
| **VS Code** | [`vscode/`](vscode/) (extension + Copilot template) |
| **Any MCP** | [`generic-mcp/`](generic-mcp/) |

**Doc hub (index only):** [docs/integrations/README.md](../docs/integrations/README.md)

**Verify static packs:** `make verify-integrations-static`

Skip a loop step on substantive work → misconfigured client, not a Pluribus bug.
