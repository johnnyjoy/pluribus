# VS Code extension — Pluribus orchestrator

Two integration paths:

1. **This extension (REST)** — event-driven orchestration + manual commands against the control plane **HTTP API**. **Does not embed MCP**; memory semantics stay on the server.
2. **MCP + Copilot-style instructions** — optional; see **`mcp-config.example.json`** and **`github-copilot-instructions.template.md`**.

## What the extension does (v0.2+)

| Area | Behavior |
|------|-----------|
| **Health** | `GET /healthz` on activate, on config change, and on a configurable interval |
| **Auto recall** | `POST /v1/recall/compile` on **task start** and **debug start** (optional: debounced **save**) |
| **Auto record** | `POST /v1/advisory-episodes` with `source: vscode-orchestrator` on **task process non-zero exit**; optional **debug end** (off by default — noisy) |
| **Manual** | Recall / record commands remain; tags default from settings |
| **Visibility** | Explorer sidebar (connection, metrics line, last auto/manual snippets), **Output → Pluribus**, **status bar** |
| **Metrics** | Local counters (auto/manual recall & record, failures, nudges); **`Pluribus: Show metrics`** logs a line |

**Not** implemented client-side: ranking, authority, promotion, or doctrine — only transport and triggers.

## Settings (`pluribus.*`)

| Key | Default | Notes |
|-----|---------|--------|
| `baseUrl` | `http://127.0.0.1:8123` | API root |
| `apiKey` | *(empty)* | `X-API-Key` if server requires it |
| `defaultTags` | `vscode` | Comma-separated tags for requests |
| `orchestrator.enabled` | `true` | Master switch for hooks |
| `orchestrator.recallOnTaskStart` | `true` | |
| `orchestrator.recallOnDebugStart` | `true` | |
| `orchestrator.recallOnSave` | `false` | Can be chatty |
| `orchestrator.saveDebounceMs` | `2500` | |
| `orchestrator.maxRecallTotal` | `24` | `max_total` for compile |
| `orchestrator.recordOnTaskProcessFailure` | `true` | Needs shell/process tasks |
| `orchestrator.recordOnDebugEnd` | `false` | Every debug stop — usually too loud |
| `healthCheckIntervalMinutes` | `5` | `0` = no periodic check |
| `nudges.whenDisconnected` | `true` | One soft warning per session on network-style recall failure |
| `nudges.afterConsecutiveTaskFailures` | `0` | `>0` → info message at N consecutive task failures |
| `metrics.logIntervalMinutes` | `0` | `>0` → periodic metrics line in Output |

## Testing against Pluribus

- **Local:** start the stack with the repo’s **Docker Compose** (see root **`README.md`**) so the API is reachable at the default **`http://127.0.0.1:8123`**.
- **Another host:** set **`pluribus.baseUrl`** (and your editor’s MCP URL, if used) to that server’s base URL — e.g. **User** settings in VS Code / Cursor, or an untracked local settings file.
- **Do not commit** machine-specific or LAN IPs/hostnames into git; keep environment-specific URLs out of the repository so defaults stay portable.

## Commands

- **Pluribus: Check health**
- **Pluribus: Show metrics**
- **Pluribus: Show output**
- **Pluribus: Recall Context (manual)** / **Record Experience (manual)** / **View Learnings** / **Refresh sidebar**

## Build / local install

```bash
cd integrations/vscode/extension
npm install
npm run compile
```

**Run (Extension Development Host):** open `integrations/vscode/extension` in VS Code → **Run and Debug** → launch extension (or **F5** if `.vscode/launch.json` is present).

**Pack `.vsix` (no marketplace):**

```bash
cd integrations/vscode/extension
npx --yes @vscode/vsce package --no-dependencies
```

Install: **Extensions** → **⋯** → **Install from VSIX…**.

## Cursor compatibility

The same VS Code extension **often** loads in **Cursor** (Cursor is VS Code–compatible). **Not verified in CI** in this repo: treat **VS Code** as the reference; after installing the `.vsix` in Cursor, confirm **tasks/debug** events fire and **Output → Pluribus** shows auto recall/record. Differences in task/debug internals are possible — use metrics + logs to validate.

## Event hooks (implementation)

| Source | Event | Auto recall | Auto record |
|--------|--------|-------------|-------------|
| Task | `tasks.onDidStartTask` | Yes (query = task name + workspace) | — |
| Task process | `tasks.onDidEndTaskProcess` | — | Yes if `exitCode` defined and ≠ 0 |
| Debug | `debug.onDidStartDebugSession` | Yes | — |
| Debug | `debug.onDidTerminateDebugSession` | — | Optional (`recordOnDebugEnd`) |
| Save | `onDidSaveTextDocument` | Optional (`recallOnSave`) | — |
| Config | `onDidChangeConfiguration` | — | Re-runs health |

Tasks **without** a process (e.g. some `CustomExecution` tasks) do not get `onDidEndTaskProcess` — no auto record on those failures.

## REST mapping

- Recall: `POST /v1/recall/compile` (same semantics as MCP `recall_context` on the server).
- Record: `POST /v1/advisory-episodes` (`source` = `vscode-orchestrator` or `vscode-manual`).
- Pending: `GET /v1/curation/pending` (manual command).

**Canonical agent loop** (for Copilot/MCP elsewhere): [`../pluribus-instructions.md`](../pluribus-instructions.md).
