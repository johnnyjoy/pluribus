# Cursor — enforcement tier (Phase 12D)

Pluribus does **not** ship a Cursor Marketplace plugin or VSIX from this repository. This pack is **MCP config + rules + skill + helpers**.

## Classification

| Field | Value |
|-------|-------|
| **Platform max enforcement tier** | **3** — Cursor supports MCP, rules, skills, commands, and (per [Cursor Plugins docs](https://cursor.com/docs/plugins)) plugin **hooks** for context injection. It does **not** block native Write/Edit when the agent skips MCP. Rules are **advisory** for the agent loop. |
| **Current Pluribus integration tier** | **2** — Global/repo MCP wiring + mandatory-loop **rules** + skill + `helper/verify-mcp`. No first-party `.cursor-plugin` with hooks in-repo. |
| **Gap** | **below_platform_capability** — Could add an official Cursor plugin manifest with SessionStart-style hooks calling Pluribus wake-up; still cannot reach Tier 5 without a custom enforcement MCP that gates native tools (out of scope). |
| **Status label** | **semi_mandatory** — Tools available + strong rules; agent can still ignore loop or use native tools without MCP. |
| **Second memory system** | **No** — pack points at control plane only. |

## What this pack does

- Registers HTTP MCP → `POST /v1/mcp`
- Teaches `recall_context` → work → `record_experience` via rules/skill
- When recall/wakeup surfaces **housekeeping**, agents should **`resolve_chore`** or defer with reason in **`record_experience`**
- Verifies endpoint with `./helper/verify-mcp`

## What this pack does not do

- Block agent edits without recall
- Auto-call MCP tools on every prompt (no in-repo Cursor hooks)
- Ship as VSIX or published Cursor Marketplace plugin
- Run Phase 11I telemetry or 11K utility tools automatically

## Verification

```bash
scripts/integrations/verify-cursor-pack.sh --static
PLURIBUS_BASE_URL=http://127.0.0.1:8123 scripts/integrations/verify-cursor-pack.sh
```

## Honest product claim

> Pluribus Cursor integration makes the memory loop **easy and visible**; it is **not fully mandatory** because Cursor cannot force MCP tool usage over native editor tools.
