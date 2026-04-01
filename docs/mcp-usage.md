# MCP usage (Pluribus)

Single guide for **how** to use Pluribus through MCP from **popular AI clients** (Cursor, Claude Desktop, and others). **Transport:** [mcp-service-first.md](mcp-service-first.md) (HTTP MCP canonical; stdio adapter compat-only). **Tool → route mapping:** [mcp-poc-contract.md](mcp-poc-contract.md) (**Dual-layer MCP** — Layer 1 default loop: **`recall_context`** before work, **`record_experience`** after; stable aliases **`memory_context_resolve`** / **`mcp_episode_ingest`** unchanged). **Timing / canon vs candidates:** [mcp-discipline-doctrine.md](mcp-discipline-doctrine.md).

Pluribus is most effective when used in a **before/after loop**: **recall** before substantive work, **record** after meaningful outcomes. The MCP **`initialize`** response includes **`instructions`** that reinforce this loop for connected clients.

**Ingestion policy:** `record_experience` creates probationary memory **at ingest** when the summary is **plausible** (signals, event tags, or rich context); otherwise the row is rejected. **Ranking and authority**, not a strict “only high-confidence” filter at intake, determine what dominates recall—see [memory-doctrine.md](memory-doctrine.md) §F.

---

## Prerequisites

1. **Control-plane is running** and reachable (e.g. `docker compose up -d` from the repo root → `http://127.0.0.1:8123`). Check [pluribus-quickstart.md](pluribus-quickstart.md).
2. Know whether **API auth** is on: if the server has **`PLURIBUS_API_KEY`** set, every MCP client must send that value (see [authentication.md](authentication.md)). If unset, you can omit headers.

---

## Configure Pluribus in your client

Pluribus exposes MCP in **two** ways:

| Transport | When to use |
|-----------|-------------|
| **HTTP** — `POST /v1/mcp` on the API base URL | **Preferred** when your client supports **remote / Streamable HTTP** MCP (fewer moving parts; includes **prompts** and **resources** on the service). |
| **stdio** — binary [`cmd/pluribus-mcp`](../control-plane/cmd/pluribus-mcp) | **Widest compatibility** (Claude Desktop, older Cursor flows, any client that only runs a local command). Forwards to the same HTTP API; **tools only** (no prompts/resources on the binary). |

After editing any MCP config file, **fully restart** the host application (Cursor, Claude Desktop, OpenCode, …) so it reloads servers.

### Cursor

**Config files:** per-repository [`.cursor/mcp.json`](../.cursor/mcp.json) (shareable with the team) or global `~/.cursor/mcp.json`. Cursor merges them; see [Cursor MCP docs](https://cursor.com/docs/context/mcp).

**Option A — HTTP MCP (recommended).** Point Cursor at the **same URL** the API uses:

```json
{
  "mcpServers": {
    "pluribus": {
      "url": "http://127.0.0.1:8123/v1/mcp"
    }
  }
}
```

If the server has **`PLURIBUS_API_KEY`** set, add headers (use an env var—do not commit secrets):

```json
{
  "mcpServers": {
    "pluribus": {
      "url": "http://127.0.0.1:8123/v1/mcp",
      "headers": {
        "X-API-Key": "${env:PLURIBUS_API_KEY}"
      }
    }
  }
}
```

**Option B — stdio `pluribus-mcp`.** Build the binary once: `cd control-plane && go build -o pluribus-mcp ./cmd/pluribus-mcp`. Then wire an **absolute** path (or `${workspaceFolder}`) so Cursor can spawn it:

```json
{
  "mcpServers": {
    "pluribus": {
      "command": "${workspaceFolder}/control-plane/pluribus-mcp",
      "env": {
        "CONTROL_PLANE_URL": "http://127.0.0.1:8123",
        "CONTROL_PLANE_API_KEY": "${env:PLURIBUS_API_KEY}"
      }
    }
  }
}
```

Omit **`CONTROL_PLANE_API_KEY`** when the server does not use **`PLURIBUS_API_KEY`**. See [Cursor-specific behavior](#cursor-specific-behavior) below for troubleshooting.

### Claude Desktop

Claude Desktop loads **`mcpServers`** from a JSON file (typical paths: macOS `~/Library/Application Support/Claude/claude_desktop_config.json`, Windows `%APPDATA%\Claude\claude_desktop_config.json`, Linux `~/.config/Claude/claude_desktop_config.json`). It is oriented toward **stdio** servers: use the **`pluribus-mcp`** binary with **`CONTROL_PLANE_URL`** (and **`CONTROL_PLANE_API_KEY`** when auth is enabled). Build as in [cmd/pluribus-mcp/README.md](../control-plane/cmd/pluribus-mcp/README.md).

Example shape (adjust the **command** path to your machine):

```json
{
  "mcpServers": {
    "pluribus": {
      "command": "/home/you/recall/control-plane/pluribus-mcp",
      "env": {
        "CONTROL_PLANE_URL": "http://127.0.0.1:8123"
      }
    }
  }
}
```

If your Claude Desktop build documents **URL-based** MCP servers, you can try the same **`http://…/v1/mcp`** endpoint as Cursor; capability varies by version—**stdio remains the portable default.**

### OpenCode

OpenCode reads MCP servers from **`mcp`** in **`opencode.json`** (per-repo file) or **`~/.config/opencode/opencode.json`**. Prefer **remote** HTTP to the control plane; set **`oauth": false`** so OpenCode does not treat Pluribus as OAuth MCP. Copy **[integrations/opencode/mcp-config.example.json](../integrations/opencode/mcp-config.example.json)** and merge; add **`headers.X-API-Key`** with **`{env:PLURIBUS_API_KEY}`** when the server uses **`PLURIBUS_API_KEY`**. **Local stdio** is supported via **`type": "local"`** and the **`pluribus-mcp`** binary—see [integrations/opencode.md](integrations/opencode.md) and [OpenCode MCP servers](https://dev.opencode.ai/docs/mcp-servers).

### VS Code, Zed, Windsurf, and other editors

Use that product’s MCP documentation. If it supports **HTTP / Streamable** MCP, configure **`http://<host>:8123/v1/mcp`** and **`X-API-Key`** when needed. If it only supports **stdio**, use **`pluribus-mcp`** as for Claude Desktop.

### Verify the connection

**HTTP:**

```bash
curl -sS -X POST "http://127.0.0.1:8123/v1/mcp" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

Expect a JSON-RPC **`result`** containing **`tools`**. With auth: add `-H "X-API-Key: $PLURIBUS_API_KEY"`.

More detail: [mcp-migration-stdio-to-http.md](mcp-migration-stdio-to-http.md).

---

## What MCP is for

MCP is the **agent** control interface for:

- recall retrieval (`recall_get`, `recall_compile`, optional `recall_run_multi`)
- pre-change validation (`enforcement_evaluate`)
- durable writes (`memory_create`, `memory_promote`) and curation (`curation_digest`, `curation_materialize`)
- **memory formation (advisory only):** **`record_experience`** / **`mcp_episode_ingest`** → `POST /v1/advisory-episodes` with ingest channel **`source: mcp`**; visibility into candidates (`curation_pending`, `curation_promotion_suggestions`, `curation_strengthened`)

It is **not** the editor Go language server — **gopls** stays local. See [pluribus-lsp-mcp-boundary.md](pluribus-lsp-mcp-boundary.md).

### Memory formation (experience → episode → candidate)

MCP is a **producer** of advisory **experience**, not a shortcut to canonical memory:

- **`record_experience`** and **`mcp_episode_ingest`** map to the same **`POST /v1/advisory-episodes`** handler with ingest channel **`source: mcp`**. Configure **`mcp.memory_formation`** in server YAML to turn episodic ingest off or tune minimum length / keyword requirements (deterministic; no LLM in the gate).
- When **`distillation.auto_from_advisory_episodes`** is enabled, **auto-distill** uses the same pipeline as REST ingest; proposals record **`pluribus_distill_origin`** **`auto:mcp`** (distill mode **auto_from_advisory_mcp**; see [memory-doctrine.md](memory-doctrine.md) Terminology).
- **`curation_pending`**, **`curation_promotion_suggestions`**, and **`curation_strengthened`** expose **candidate** state for review; promotion to durable memory remains **`curation_materialize`** (or governed **`auto-promote`** when explicitly enabled in config). MCP does **not** auto-promote by default.

See [episodic-similarity.md](episodic-similarity.md), [curation-loop.md](curation-loop.md), and [memory-doctrine.md](memory-doctrine.md) (Terminology: ingest channel vs distill mode).

### Automated proof (HTTP MCP primary)

Regression coverage for **memory formation through MCP** uses the **same** **`POST /v1/mcp`** JSON-RPC surface as production (initialize → tools/list → tools/call), not a separate mock. Integration tests live under **`control-plane/cmd/controlplane/`** with build tag **`integration`** (e.g. `mcp_memory_formation_integration_test.go`). They require **`TEST_PG_DSN`** and prove MCP ingest → advisory episode (`source: mcp`) → optional auto-distill → curation visibility tools, plus **deduplication** of repeated identical MCP ingests within a configurable time window (`mcp.memory_formation.dedup_*`). A **stdio** subprocess smoke (`go run ./cmd/pluribus-mcp`) is included in the same integration package as a **secondary** check; the **HTTP MCP** path remains the primary automated proof. See [evaluation.md](evaluation.md) and [evidence/episodic-proof.md](../evidence/episodic-proof.md).

---

## Recall-driven workflow (recommended order)

1. **Ground** — **`recall_context`** (or **`memory_context_resolve`**) **before** substantive work; raw **`recall_get`** / **`recall_compile`** when you need full wire control. Use **tags** + task text per [memory-doctrine.md](memory-doctrine.md).
2. **Act** — propose or execute a bounded change.
3. **Gate** — `enforcement_evaluate` with **proposal_text** before large refactors or policy moves (when enforcement is enabled).
4. **Learn** — after meaningful work, **`record_experience`** (or **`mcp_episode_ingest`** / opportunistic **`memory_log_if_relevant`**); optional governed path `curation_digest` → review → `curation_materialize`.

Tool **`arguments`** must match the same JSON as the underlying REST body or GET query mapping ([http-api-index.md](http-api-index.md), Go `json` tags). They do **not** define where memory lives. See [pluribus-memory-first-ontology.md](pluribus-memory-first-ontology.md).

**Prompts (embedded):** `pluribus_memory_grounding`, `pluribus_pre_change_enforcement`, `pluribus_memory_curation`, `pluribus_canon_vs_advisory` — names in `control-plane/internal/mcp/prompts.go`.

On **`recall_context`** / **`memory_context_resolve`** success, **`mcp_context`** may include short hints: **`decision_hint`** (always), **`relevance_hint`** (only when the bundle has matching memory rows), **`after_work_hint`** (wording adapts to empty vs non-empty pool). Details: [mcp-poc-contract.md](mcp-poc-contract.md).

---

## Manual CLI run (stdio adapter)

Without an IDE, from `control-plane/` after `go build -o pluribus-mcp ./cmd/pluribus-mcp`:

```bash
export CONTROL_PLANE_URL=http://127.0.0.1:8123
# only if auth is enabled on server:
export CONTROL_PLANE_API_KEY="${PLURIBUS_API_KEY}"
./pluribus-mcp
```

**IDE / agent configuration** is covered in [Configure Pluribus in your client](#configure-pluribus-in-your-client) above.

---

## Auth

- No `PLURIBUS_API_KEY` on server → MCP calls need no key.
- Server auth on → send `X-API-Key` (or client token path your deployment documents).

[authentication.md](authentication.md).

---

## Cursor-specific behavior

### Why there are no `tools/*.json` files for pluribus

Some Cursor MCP caches include **`tools/`** with static JSON per tool. **`pluribus-mcp`** is a **stdio** server: tools come from **`tools/list`** at runtime. The repo does not ship per-tool JSON under Cursor’s cache; missing files are **not** proof tools are absent — often the session disconnected before discovery.

### `STATUS.md` (“The MCP server errored”)

Cursor writes **`STATUS.md`** when the **last** stdio connection failed. Typical causes: binary crashed, Cursor closed the client (`Client closed`), pipe broke. **Fix:** Settings → MCP → restart pluribus; rebuild `go build -o pluribus-mcp ./cmd/pluribus-mcp` after code changes; restart Cursor if stuck.

### `health` worked, then “Not connected”

The **session** dropped. Reconnect MCP before relying on **`tools/list`** names. Tool names in **`tools/call`** must match **`tools/list`** output (Cursor may **prefix** display names in the UI).

### Checklist for a verification run

| Step | Action |
|------|--------|
| 1 | `cd control-plane && go build -o pluribus-mcp ./cmd/pluribus-mcp` |
| 2 | MCP config: **absolute path** to the **binary**, not the `cmd/pluribus-mcp` directory |
| 3 | `CONTROL_PLANE_URL` points at a **running** API |
| 4 | Reload MCP until Settings shows connected |
| 5 | If still errored, read MCP output log for stderr |

### Extra HTTP-only operations

Anything not exposed as an MCP tool in the default **tools/list** is still available via **direct HTTP** — see [http-api-index.md](http-api-index.md). Prefer memory-first flows (**tags** + **retrieval_query**).

---

## Manual proof template

Operator receipt sequence: [control-plane/proof-scenarios/functional-quality-workflow.yaml](../control-plane/proof-scenarios/functional-quality-workflow.yaml) (manual mode).

---

## References

- [mcp-service-first.md](mcp-service-first.md)
- [mcp-poc-contract.md](mcp-poc-contract.md)
- [curation-loop.md](curation-loop.md)
- [pre-change-enforcement.md](pre-change-enforcement.md)
- [proof-scenarios.md](proof-scenarios.md)
