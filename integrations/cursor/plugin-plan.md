# Pluribus “plugin” model for Cursor — what we actually ship

## What Cursor supports (reality)

| Mechanism | Role | Official docs |
|-----------|------|----------------|
| **`mcp.json`** (user `~/.cursor/mcp.json` or repo `.cursor/mcp.json`) | Registers MCP servers (HTTP or stdio). **This is the foundation** for Pluribus tools in Agent. | [MCP](https://cursor.com/docs/context/mcp) |
| **`.cursor/rules` files** (`*.mdc` / `*.md`) | Versioned instructions + frontmatter (`description`, `globs`, `alwaysApply`). See [Rules](https://cursor.com/docs/rules). |
| **User rules** | Global Agent instructions in **Cursor Settings → Rules**. | [Rules](https://cursor.com/docs/rules) |
| **Agent Skills** (`SKILL.md` under `.cursor/skills/` or `~/.cursor/skills/`) | Packaged agent behaviors. | [Skills](https://cursor.com/docs/context/skills) |
| **`AGENTS.md`** | Plain markdown instructions; optional alternative to heavy rule sets. | [Rules](https://cursor.com/docs/rules) |
| **Cursor extension API** (`vscode.cursor.mcp.registerServer`) | For **extension authors** registering MCP from code—not what this repo ships. | [MCP extension API](https://cursor.com/docs/context/mcp-extension-api) |

## Conclusion (A / B / C)

- **A — A first-party “Pluribus” VSIX in this repo:** **No.** We do not ship a Cursor extension binary here.
- **B — MCP + rules + skills + prompts + docs as the practical “plugin”:** **Yes.** This is the **integration pack** under `integrations/cursor/`. Users merge JSON, copy rules/skills, and paste prompts—it is honest and matches how Cursor is meant to be extended today.
- **C — Sidecar helpers:** **Optional.** Small scripts (e.g. `helper/verify-mcp.sh`) only where they reduce friction (verify HTTP MCP), not a parallel architecture.

## How this complements the Pluribus Cursor rule

- The **rule** (e.g. **`pluribus.mdc`**) states the **non‑negotiable loop** in Agent context.
- The **pack** makes the loop **easier to run**: MCP wired, prompts for recall/record, verification, and one canonical rule (stricter anti-deferral included).
- Neither replaces the other: **rule = obligation**, **pack = ergonomics + visibility**.

## Limitations (unchanged)

- Pluribus must be **running** (e.g. Docker); Cursor does not bundle the server.
- **User rules** apply to **Agent (Chat)**; not all Cursor AI surfaces use the same rule stack—see [Rules](https://cursor.com/docs/rules).
- **Team rules** from the org dashboard can layer on top; this pack does not configure them.
