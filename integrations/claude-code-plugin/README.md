# Pluribus — Claude Code plugin (first-party)

This directory is a **Claude Code plugin**: skills, hooks, optional agent, and **MCP** wiring so Pluribus stays present in the agent loop. **Ranking, authority, promotion, and doctrine remain on the Pluribus control plane**—this package only orchestrates and teaches the loop.

## What it does

| Component | Behavior |
|-----------|----------|
| **MCP** | Plugin ships `.mcp.json` pointing at **`http://127.0.0.1:8123/v1/mcp`** (same service-first endpoint as Cursor). |
| **Hooks** | `SessionStart`: orientation + `GET /healthz` reachability. `UserPromptSubmit`: optional **`POST /v1/recall/compile`** preview for substantive prompts (same semantics as MCP `recall_context`; bounded size). `PostToolUseFailure`: **optional** reminder to consider `record_experience`—**off by default** (see env). |
| **Skills** | Model-invoked `/pluribus:*` skills for recall, record, health, curation, and doctrine. |
| **Agent** | `memory-first-investigator`: recall-first investigation; no local “truth” policy. |

Nothing here implements a second memory system.

## Automatic vs manual

- **Automatic (hooks):** session primer; prompt-time recall **preview** when the server is up and the prompt is long enough (see `hooks/user-prompt-recall.sh`); optional post-failure hint if enabled.
- **Manual / tool-driven:** use MCP **`recall_context`** and **`record_experience`** (or names your client exposes); plugin skills document when and how.

## Environment variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `PLURIBUS_BASE_URL` | `http://127.0.0.1:8123` | API base for hook scripts (`healthz`, `recall/compile`). |
| `PLURIBUS_API_KEY` | *(unset)* | Sent as `X-API-Key` to HTTP hooks when set. |
| `PLURIBUS_HOOK_RECALL` | `on` | Set to `off` to disable UserPromptSubmit HTTP recall preview. |
| `PLURIBUS_HOOK_FAILURE_HINT` | `off` | Set to `on` to enable PostToolUseFailure `record_experience` reminder. |

With API key required on the server, merge **`mcp.with-auth.example.json`** into your MCP config (or add headers in user settings). The default `.mcp.json` has **no** `X-API-Key` line so empty headers do not break handshakes.

## Install (local)

From a machine with **Claude Code** installed:

```bash
cd /path/to/pluribus
claude --plugin-dir ./integrations/claude-code-plugin
```

Or install from a **marketplace** once you publish a catalog; see **`marketplace.example.json`** (paths are examples—adjust `source` to your checkout).

After changes, use **`/reload-plugins`** in Claude Code.

## Verify (static)

No Claude binary required:

```bash
bash integrations/claude-code-plugin/scripts/verify-plugin.sh
```

## Skills (namespace `pluribus`)

| Skill folder | Invoked as | Role |
|--------------|------------|------|
| `recall-context` | `/pluribus:recall-context` | When to call `recall_context` before substantive work. |
| `record-experience` | `/pluribus:record-experience` | When and how to `record_experience`. |
| `memory-health` | `/pluribus:memory-health` | Reachability / MCP sanity. |
| `curation-pending` | `/pluribus:curation-pending` | Pending curation tools—advisory only. |
| `pluribus-doctrine` | `/pluribus:pluribus-doctrine` | Tags + situation; server owns truth. |

## Hooks

| Event | Script | Why |
|-------|--------|-----|
| `SessionStart` | `hooks/session-start.sh` | Primes context and reports API health. |
| `UserPromptSubmit` | `hooks/user-prompt-recall.sh` | Injects bounded compile preview; skips short prompts. |
| `PostToolUseFailure` | `hooks/post-tool-failure-hint.sh` | Optional failure hint; **disabled by default**. |

## Agent

- **`memory-first-investigator`** — subagent for recall-first debugging and investigation (`agents/memory-first-investigator.md`).

## MCP configuration path

- **Plugin:** `integrations/claude-code-plugin/.mcp.json`
- **Auth example:** `integrations/claude-code-plugin/mcp.with-auth.example.json`

Canonical HTTP MCP URL for Pluribus: **`{base}/v1/mcp`**. Default base **`http://127.0.0.1:8123`**.

## Marketplace readiness (not publishing)

- Version in `.claude-plugin/plugin.json` (semver).
- **`marketplace.example.json`** shows a catalog entry shape; adjust `source` and run `claude plugin marketplace add` per Claude Code docs when you publish a real marketplace repo.
- Official submission (when ready): see [Claude Code plugins — submit](https://code.claude.com/docs/en/plugins) (in-app forms linked from docs).

## Risks and limits

- Hooks depend on **`curl`**, **`jq`**, and network access to `PLURIBUS_BASE_URL`; they **fail open** (no blocking) on errors.
- UserPromptSubmit recall is a **preview**, not a substitute for MCP tools in all clients.
- **PostToolUseFailure** hints are easy to overuse; they stay **off** unless `PLURIBUS_HOOK_FAILURE_HINT=on`.

## License

See **`LICENSE.md`** (same Purinus License v1.0 as the repository unless noted otherwise).
