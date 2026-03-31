# Cursor — Pluribus integration pack (“plugin”)

This folder is the **realistic** Cursor integration: **MCP config + rules + optional skill + prompts + helpers + docs**. There is **no separate VSIX** in this repository; see [`plugin-plan.md`](plugin-plan.md) for what Cursor actually supports.

## What “plugin” means here

A **config and behavior pack** that makes Pluribus **native-feeling** in Cursor: MCP wired, rules/skills aligned, copy-paste prompts, and a small HTTP check script. It **complements** the Pluribus Cursor rule—it does not replace it.

## Ideal install (user scope — every repository)

1. **MCP (global):** merge [`mcp-config.json`](mcp-config.json) (or [`mcp-config.no-auth.json`](mcp-config.no-auth.json) if the API has no key) into **`~/.cursor/mcp.json`**. For a machine on your LAN, start from [`mcp-config.lan.example.json`](mcp-config.lan.example.json) and set host/port. See [Cursor MCP](https://cursor.com/docs/context/mcp).
2. **Agent Skill (global):** copy **`skills/pluribus/`** → **`~/.cursor/skills/pluribus/`** ([Agent Skills](https://cursor.com/docs/context/skills)).
3. **User rules:** **Cursor Settings → Rules → User rules** — paste [`pluribus-instructions.md`](../pluribus-instructions.md) + [`snippets/context-prime.txt`](snippets/context-prime.txt) ([Rules](https://cursor.com/docs/rules)).
4. **Repository rule (optional, team):** copy [`pluribus.mdc`](pluribus.mdc) → **`.cursor/rules/pluribus.mdc`** so clones share the same **`.cursor/rules`** entry (frontmatter + `alwaysApply` / `globs`).

**Stricter rule:** [`pluribus-stricter.mdc`](pluribus-stricter.mdc) — see [`rules.md`](rules.md). Do not enable two `alwaysApply` defaults without intent.

**Loop:** **`recall_context` → plan → act → `record_experience`**.

## Pack contents

| File | Purpose |
|------|---------|
| [`mcp-config.json`](mcp-config.json) | Default HTTP MCP + `X-API-Key` from env |
| [`mcp-config.example.json`](mcp-config.example.json) | Same default (kept for older docs) |
| [`mcp-config.no-auth.json`](mcp-config.no-auth.json) | No auth header |
| [`mcp-config.lan.example.json`](mcp-config.lan.example.json) | Example LAN base URL |
| [`pluribus.mdc`](pluribus.mdc) | Default Cursor rule (`.mdc`) |
| [`pluribus-stricter.mdc`](pluribus-stricter.mdc) | Optional stricter rule |
| [`rules.md`](rules.md) | How to choose / install rules |
| [`prompts.md`](prompts.md) | Copy-paste Agent nudges |
| [`commands.md`](commands.md) | Patterns & Cursor entry points |
| [`plugin-plan.md`](plugin-plan.md) | What Cursor supports vs what we ship |
| [`skill.md`](skill.md) | Compact WHEN→DO |
| [`skills/pluribus/`](skills/pluribus/) | Agent Skill folder |
| [`snippets/context-prime.txt`](snippets/context-prime.txt) | Short prime for rules |
| [`AGENTS.template.md`](AGENTS.template.md) | Optional root **`AGENTS.md`** |
| [`helper/verify-mcp.sh`](helper/verify-mcp.sh) | Curl JSON-RPC `initialize` against `/v1/mcp` |

## How the rule and this pack work together

- **Rule** (`pluribus.mdc` / user rules): tells Agent **what to do** (loop + constraints).  
- **Pack** (MCP JSON + skill + prompts): makes it **easier to do** and **harder to forget** (tools available, nudges ready, verify script).

## Verify Pluribus is actually used

1. **MCP alive:** run `./helper/verify-mcp.sh` from this directory (or set `PLURIBUS_URL` / `PLURIBUS_API_KEY`).  
2. **Cursor sees tools:** Agent → confirm **`pluribus`** / **`recall_context`** appears in tool list after restart.  
3. **Behavior:** on a non-trivial task, Agent should call **`recall_context`** before large edits and **`record_experience`** after a meaningful outcome.  
4. **Server-side:** optional—check episodic / candidate activity via your usual API or DB tools (not Cursor-specific).

## Troubleshooting

| Issue | What to check |
|-------|----------------|
| No Pluribus tools | **`~/.cursor/mcp.json`** merged correctly; Cursor restarted; Pluribus **`docker compose up`** / API reachable (`curl /healthz`). |
| 401 / auth | Server has **`PLURIBUS_API_KEY`** → set **`PLURIBUS_API_KEY`** in env and keep **`X-API-Key`** in MCP config; or use [`mcp-config.no-auth.json`](mcp-config.no-auth.json) when API has no key. |
| Agent ignores recall | Strengthen rules ([`pluribus-stricter.mdc`](pluribus-stricter.mdc)), add [`prompts.md`](prompts.md) to user rules, ensure skill is installed. |
| Wrong host on LAN | Edit URL in MCP config; firewall allows port **8123** (or your mapped port). |

Full doc: [`docs/integrations/cursor.md`](../../docs/integrations/cursor.md) · Memory model: [`docs/memory-doctrine.md`](../../docs/memory-doctrine.md).
