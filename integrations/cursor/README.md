# Cursor

**Ideal — user level (every project):** wire Pluribus once in your Cursor **user** scope so Agent behaves the same everywhere. Cursor separates **[user vs project rules](https://cursor.com/docs/rules)** and supports **global** MCP config; **Agent Skills** in **`~/.cursor/skills/`** apply across workspaces.

1. **MCP (global):** merge **`mcp-config.example.json`** → **`~/.cursor/mcp.json`** so **`pluribus`** is always available (repo-local **`.cursor/mcp.json`** only if you need per-project overrides).
2. **Agent Skill (global):** copy **`skills/pluribus/`** → **`~/.cursor/skills/pluribus/`** ([Agent Skills](https://cursor.com/docs/context/skills)).
3. **Rules — user (global Agent instructions):** open **Cursor Settings → Rules → User rules** and paste **[`pluribus-instructions.md`](../pluribus-instructions.md)** plus **`snippets/context-prime.txt`** (plain markdown; no frontmatter). That applies to **Agent (Chat)** across all projects ([rules overview](https://cursor.com/docs/rules)).
4. **Rules — project (team / repo):** optionally copy **`pluribus.mdc`** → **`.cursor/rules/pluribus.mdc`** so everyone who clones the repo gets the same **project rule** (version-controlled **`alwaysApply` / `globs` / `description`** in one file).

**Optional:** root **`AGENTS.md`** from [`AGENTS.template.md`](AGENTS.template.md) (CLI / simple instruction file); compact table in **`skill.md`**.

**Loop:** **`recall_context` → plan → act → `record_experience`**.
