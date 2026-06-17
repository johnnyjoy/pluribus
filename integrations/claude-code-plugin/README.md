# Pluribus — Claude Code plugin (first-party)

This directory is a **Claude Code plugin**: skills, hooks, optional agent, and **MCP** wiring so Pluribus stays present in the agent loop. **Ranking, authority, promotion, and doctrine remain on the Pluribus control plane**—this package only orchestrates and teaches the loop.

## What it does

| Component | Behavior |
|-----------|----------|
| **MCP** | Plugin ships `.mcp.json` pointing at **`http://127.0.0.1:8123/v1/mcp`** (same service-first endpoint as Cursor). |
| **Hooks** | `SessionStart`: orientation + `GET /healthz`, then **`POST /v1/mcp`** `tools/call` **`wakeup_context`** (same tool surface as plugin MCP)—compact **identity** (`kind=state` from the server) + **governing_memory** injected when the API is up (set `PLURIBUS_HOOK_WAKEUP=off` to skip). `UserPromptSubmit`: optional **`POST /v1/recall/compile`** preview for substantive prompts (same semantics as MCP `recall_context`; bounded size). `PostToolUseFailure`: **optional** reminder to consider `record_experience`—**off by default** (see env). |
| **Skills** | Model-invoked `/pluribus:*` skills for recall, record, health, curation, and doctrine. |
| **Agent** | `memory-first-investigator`: recall-first investigation; no local “truth” policy. |

Nothing here implements a second memory system.

## Automatic vs manual

- **Automatic (hooks):** session primer + **wake-up** (identity + governing memory from Pluribus when reachable); prompt-time recall **preview** when the server is up and the prompt is long enough (see `hooks/user-prompt-recall.sh`); optional post-failure hint if enabled.
- **Manual / tool-driven:** use MCP **`recall_context`** and **`record_experience`** (or names your client exposes); plugin skills document when and how.

## Environment variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `PLURIBUS_BASE_URL` | `http://127.0.0.1:8123` | API base for hook scripts (`healthz`, `recall/compile`). |
| `PLURIBUS_API_KEY` | *(unset)* | Sent as `X-API-Key` to HTTP hooks when set. |
| `PLURIBUS_HOOK_RECALL` | `on` | Set to `off` to disable UserPromptSubmit HTTP recall preview. |
| `PLURIBUS_HOOK_WAKEUP` | `on` | Set to `off` to disable SessionStart HTTP MCP `wakeup_context` injection (health + orientation still run). |
| `PLURIBUS_HOOK_WAKEUP_MAX_IDENTITY` | `4` | Max identity (`state`) lines shown from wake-up JSON. |
| `PLURIBUS_HOOK_WAKEUP_MAX_GOVERNING` | `8` | Max governing-memory lines (kind + truncated statement). |
| `PLURIBUS_HOOK_FAILURE_HINT` | `off` | Set to `on` to enable PostToolUseFailure `record_experience` reminder. |

With API key required on the server, merge **`mcp.with-auth.example.json`** into your MCP config (or add headers in user settings). The default `.mcp.json` has **no** `X-API-Key` line so empty headers do not break handshakes.

## Install

Normal installation uses Claude Code’s **plugin manager**, not a raw `claude` flag.

### Claude Desktop (no terminal)

In the **desktop app**, use the **+** next to the prompt → **Plugins** to see installed plugins and commands. **Add plugin** opens the plugin browser (marketplaces you have configured, including Anthropic’s official catalog). **Manage plugins** enables, disables, or uninstalls. Scopes: **user**, **project**, or **local-only**—same semantics as CLI installs.

After you add the **`pluribus-repo`** marketplace (once) via CLI or any supported flow, **pluribus** should appear in that browser like any other catalog plugin.

### Where plugins run (product constraints)

Per Anthropic’s current docs: **plugins are not available in remote (cloud) sessions**; **local** and **SSH** sessions are where plugin hooks, bundled MCP from `.mcp.json`, and skills apply. Design implication: **hooks + plugin MCP won’t run in remote-only workflows**—users there need **connectors / manual MCP** in that environment if the product allows it.

This does **not** change Pluribus server semantics; it only bounds **where** the plugin package is active.

### From this repository (local marketplace)

1. Clone or open the repo and `cd` to the **repository root**.
2. **Register the catalog** (directory that contains **`integrations/.claude-plugin/marketplace.json`**):

   ```bash
   claude plugin marketplace add ./integrations
   ```

   Or in an interactive session: **`/plugin marketplace add ./integrations`** (same path).

3. **Install the plugin** (user scope is default; use **`/plugin`** UI to pick project/local scope if needed):

   ```bash
   claude plugin install pluribus@pluribus-repo
   ```

   Or run **`/plugin`** → **Discover** → choose **pluribus** → install.

4. Apply changes: **`/reload-plugins`**.

Official reference: [Discover and install plugins](https://code.claude.com/docs/en/discover-plugins), [Plugin marketplaces](https://code.claude.com/docs/en/plugin-marketplaces).

### Development only (`--plugin-dir`)

To test **unpacked** plugin changes without going through a marketplace, you can run:

```bash
claude --plugin-dir ./integrations/claude-code-plugin
```

This is for **development and CI-style checks**, not the usual install path. Prefer **`plugin marketplace add` + `plugin install`** for anything you rely on day to day.

After editing plugin files, **`/reload-plugins`** still applies.

### L0 identity (non-empty `identity` in wake-up)

Wake-up **identity** is whatever the server returns for active **`kind=state`** memories in the pool—the plugin does **not** hardcode it. For a minimal bootstrap after a fresh deploy, run from the repo root (server must accept **`POST /v1/memories`** with your API key):

```bash
export PLURIBUS_BASE_URL=http://127.0.0.1:8123   # default if unset
export PLURIBUS_API_KEY=...                      # if required
./scripts/seed-l0-identity-memories.sh
```

Then confirm with MCP **`wakeup_context`** or `POST /v1/recall/wakeup` that **`identity`** is populated.

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
| `SessionStart` | `hooks/session-start.sh` | Health + orientation; loads **`wakeup_context`** via HTTP MCP and injects bounded identity + governing memory (Pluribus-owned selection). |
| `UserPromptSubmit` | `hooks/user-prompt-recall.sh` | Injects bounded compile preview; skips short prompts. |
| `PostToolUseFailure` | `hooks/post-tool-failure-hint.sh` | Optional failure hint; **disabled by default**. |

## Agent

- **`memory-first-investigator`** — subagent for recall-first debugging and investigation (`agents/memory-first-investigator.md`).

## MCP configuration path

- **Plugin:** `integrations/claude-code-plugin/.mcp.json`
- **Auth example:** `integrations/claude-code-plugin/mcp.with-auth.example.json`

Canonical HTTP MCP URL for Pluribus: **`{base}/v1/mcp`**. Default base **`http://127.0.0.1:8123`**.

## Marketplace readiness (not publishing)

- Version in **`integrations/claude-code-plugin/.claude-plugin/plugin.json`** (semver), mirrored in **`integrations/.claude-plugin/marketplace.json`** for the bundled catalog.
- **Local catalog:** **`integrations/.claude-plugin/marketplace.json`** lists plugin **`pluribus`** with **`source`: `./claude-code-plugin`** (paths relative to the **`integrations/`** marketplace root).
- Wider distribution: add the same catalog via **Git URL** or publish per [Create a plugin marketplace](https://code.claude.com/docs/en/plugin-marketplaces); official submission (when ready): [plugin submit](https://code.claude.com/docs/en/plugins) (in-app forms linked from docs).

## Risks and limits

- Hooks depend on **`curl`**, **`jq`**, and network access to `PLURIBUS_BASE_URL`; they **fail open** (no blocking) on errors.
- If **`identity`** in wake-up is empty, the pool has no suitable **`state`** rows yet—seed (see above) or add memories through normal ingest; the hook only displays what the server returns.
- UserPromptSubmit recall is a **preview**, not a substitute for MCP tools in all clients.
- **PostToolUseFailure** hints are easy to overuse; they stay **off** unless `PLURIBUS_HOOK_FAILURE_HINT=on`.

## License

See **`LICENSE.md`** (same Purinus License v1.0 as the repository unless noted otherwise).
