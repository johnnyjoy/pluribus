# Cursor

Cursor is a **first-class** target: **HTTP MCP** to Pluribus, **user- or project-scoped rules**, and optional **Agent Skills**. Cursor’s model is documented in **[Rules](https://cursor.com/docs/rules)** (project rules, user rules, team rules, **`AGENTS.md`**) and **[Agent Skills](https://cursor.com/docs/context/skills)**.

## Install scope: user vs project

**Pluribus ideal:** install at **user level** so recall + episodic capture are default in **every** repo.

| Piece | **User (recommended)** | **Project (optional)** |
|--------|-------------------------|-------------------------|
| **MCP** | **`~/.cursor/mcp.json`** | **`.cursor/mcp.json`** |
| **Agent Skill** | **`~/.cursor/skills/pluribus/`** | rarely needed if global skill exists |
| **Rules** | **Cursor Settings → Rules → User rules** — paste [`pluribus-instructions.md`](../../integrations/pluribus-instructions.md) + [`integrations/cursor/snippets/context-prime.txt`](../../integrations/cursor/snippets/context-prime.txt) | **`.cursor/rules/pluribus.mdc`** from [`integrations/cursor/pluribus.mdc`](../../integrations/cursor/pluribus.mdc) (`.mdc` + frontmatter for apply mode / globs) |

**Team / org:** dashboard **Team rules** can layer on top (precedence **Team → Project → User** per [Cursor](https://cursor.com/docs/rules)). Pluribus does not replace that—use Team rules only if your org requires them.

**Note:** User rules apply to **Agent (Chat)**; they are not the same as Inline Edit (⌘K) behavior—see [Cursor Rules FAQ](https://cursor.com/docs/rules).

## MCP (recommended)

**Preferred:** HTTP to the control-plane (no local binary required).

- Config: **global** **`~/.cursor/mcp.json`** (recommended) or **repository** `.cursor/mcp.json`
- URL: `http://127.0.0.1:8123/v1/mcp` (adjust for your host)
- With API key: `headers.X-API-Key` → `${env:PLURIBUS_API_KEY}`

Full templates: [mcp-usage.md](../mcp-usage.md), copy-ready file: **[integrations/cursor/mcp-config.example.json](../../integrations/cursor/mcp-config.example.json)**.

**Fallback:** stdio **`pluribus-mcp`** with `CONTROL_PLANE_URL`—see root [README.md](../../README.md).

## Rules + skill (quick reference)

- **Canonical loop text:** **[integrations/pluribus-instructions.md](../../integrations/pluribus-instructions.md)**  
- **Project rule file:** **[integrations/cursor/pluribus.mdc](../../integrations/cursor/pluribus.mdc)** → **`.cursor/rules/pluribus.mdc`**  
- **Skill pack:** **[integrations/cursor/skills/pluribus/](../../integrations/cursor/skills/pluribus/)** → **`~/.cursor/skills/pluribus/`** (recommended)  
- **AGENTS.md (optional):** **[integrations/cursor/AGENTS.template.md](../../integrations/cursor/AGENTS.template.md)**  
- **Compact table:** **[integrations/cursor/skill.md](../../integrations/cursor/skill.md)**  
- **Pack index:** **[integrations/cursor/README.md](../../integrations/cursor/README.md)**

## Workflow

1. Start Pluribus (`docker compose up -d` or your deployment).
2. Enable MCP in Cursor; confirm **`tools/list`** includes Pluribus tools.
3. **Before** substantive work: **`recall_context`**. **After** meaningful outcomes: **`record_experience`**. (Aliases **`memory_context_resolve`** / **`mcp_episode_ingest`**.)

Troubleshooting: [mcp-usage.md](../mcp-usage.md#cursor-specific-behavior).

## Limitations

- Cursor does not ship Pluribus; you must run the API.
- Tool lists are **runtime**—rebuild `pluribus-mcp` after server upgrades if using stdio.
