# OpenCode

1. Merge **`mcp-config.example.json`** into **`opencode.json`** (repo or `~/.config/opencode/opencode.json`).
2. Add behavior + prime to **`AGENTS.md`** (see **`AGENTS.template.md`**) and/or **`instructions`** in **`opencode.json`** ([OpenCode rules](https://dev.opencode.ai/docs/rules)).
3. **Skills folder:** copy **`skills/pluribus/`** → **`.opencode/skills/pluribus/`**. **Or** paste **`skill.md`** into **`AGENTS.md`** if you skip the folder.
4. Ensure **`pluribus`** appears in tool-capable runs; nudge **`use pluribus`** if tools stall.

**Canonical behavior:** [`pluribus-instructions.md`](../pluribus-instructions.md).

**`recall_context` → plan → act → `record_experience`** every substantive task.
