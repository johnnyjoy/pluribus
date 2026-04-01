# Pluribus — AI editors and agent systems

**Integrations are behavioral:** each pack under **`integrations/<platform>/`** ships **`rules.md`** (pointer to **[`pluribus-instructions.md`](../../integrations/pluribus-instructions.md)** plus the **native** file for that editor), **`skill.md`** (WHEN→DO), and **`README.md`** (exact install steps). MCP wiring is separate from “the agent actually calls tools.”

**Non-negotiable loop:** **`recall_context` → work → `record_experience`** on every substantive task. Legacy names **`memory_context_resolve`** / **`mcp_episode_ingest`** are compatibility aliases only.

**After Docker is up:** configure MCP in your client ([mcp-usage.md](../mcp-usage.md)), then paste **[`pluribus-instructions.md`](../../integrations/pluribus-instructions.md)** (or copy the **`.mdc` / skill** from your platform’s folder). Without rules, tools may sit unused.

**Adoption (habit, not exposure):** [usage.md](usage.md) · **tiers + verify:** [matrix.md](matrix.md) · **skills → tools:** [skills-model.md](skills-model.md).

Doctrine: [memory-doctrine.md](../memory-doctrine.md). End-to-end first run: [pluribus-quickstart.md](../pluribus-quickstart.md) §4.

---

## Start here

| Doc | Purpose |
|-----|---------|
| **[matrix.md](matrix.md)** | Platform comparison |
| **[generic-mcp.md](generic-mcp.md)** | Any MCP client |
| **[cursor.md](cursor.md)** | Cursor |
| **[claude-code.md](claude-code.md)** | Claude Code |
| **[claude-desktop.md](claude-desktop.md)** | Claude Desktop |
| **[openclaw.md](openclaw.md)** | OpenClaw |
| **[opencode.md](opencode.md)** | OpenCode |
| **[continue.md](continue.md)** | Continue |
| **[zed.md](zed.md)** | Zed |
| **[vscode.md](vscode.md)** | VS Code |

**Artifacts:** [integrations/README.md](../../integrations/README.md)

---

## Default loop (directive)

1. **`recall_context`** — pre-action, full task text.  
2. **Plan / reason** on bundle.  
3. **Act**.  
4. **`record_experience`** — post-action, short summary.  

**Optional:** enforcement / curation / contradictions—pull-only; **never** replace 1–4.

---

## Doctrine guardrails

- **No** project / task / workspace / **scope** as required recall partitions—[anti-regression.md](../anti-regression.md).
- Memory is **global**; **tags** + situation text shape recall.

---

## Verification

Templates are not secrets; set real URLs and API keys locally. Re-validate after client upgrades.
