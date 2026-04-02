# Claude Code

**First-party plugin (recommended):** [`../claude-code-plugin/README.md`](../claude-code-plugin/README.md) — install with `claude --plugin-dir ./integrations/claude-code-plugin` for skills, hooks, MCP template, and the `memory-first-investigator` agent.

---

1. Register Pluribus MCP; see **`mcp-config.example.json`** / root **README** (plugin ships **`.mcp.json`** at `integrations/claude-code-plugin/`).
2. Copy **`CLAUDE.template.md`** → project-root **`CLAUDE.md`**, then append **`snippets/context-prime.txt`**. ([Claude Code](https://docs.anthropic.com/en/docs/claude-code) loads **`CLAUDE.md`** automatically.)
3. **Optional (without plugin):** copy **`skills/pluribus/`** to your skills path, or paste **`skill.md`**.

**Canonical behavior:** [`pluribus-instructions.md`](../pluribus-instructions.md).

**`recall_context` → plan → act → `record_experience`** every substantive task.
