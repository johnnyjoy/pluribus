<div align="center">

<img src="Pluribus_logo.png" alt="Pluribus" width="280">

</div>

# Pluribus

**Pluribus** is the **open control plane** for governed AI memory: **Postgres-backed** durable rows, **situation-shaped recall**, **pre-change enforcement**, and **curation**—without treating chat or logs as source of truth.

**Recall** is the name of the **model and doctrine** this repository embodies (constraints, decisions, patterns, failures, and related kinds that outlast any single session). Pluribus is the **runnable service**—HTTP API, MCP, shared global memory pool—described in [docs/memory-doctrine.md](docs/memory-doctrine.md).

> **Usage:** Pluribus only accumulates memory if the agent **runs it every substantive pass**.  
> Default loop: **`recall_context` → plan → act → `record_experience`**.  
> Skip recall or skip record and you get no durable progress—same as amnesia.

**New here?** Three steps: **[`docs/get-started.md`](docs/get-started.md)**.

---

## Table of contents

- [Get started (new users)](docs/get-started.md)
- [What this is](#what-this-is)
- [Why use it](#why-use-it)
- [Quick start (Docker — recommended)](#quick-start-docker--recommended)
- [Using Pluribus with AI agents](#using-pluribus-with-ai-agents)
- [Ensuring your agent uses Pluribus](#ensuring-your-agent-uses-pluribus)
- [Using Pluribus with AI editors and agent systems](#using-pluribus-with-ai-editors-and-agent-systems)
- [SDKs for custom agents (Go + Python)](#sdks-for-custom-agents-go--python)
- [MCP configuration by client](#mcp-configuration-by-client)
- [Multi-agent shared memory](#multi-agent-shared-memory)
- [Installation paths](#installation-paths)
- [Configuration](#configuration)
- [Everyday API usage (smoke)](#everyday-api-usage-smoke)
- [Documentation map](#documentation-map)
- [Repository layout](#repository-layout)
- [Verification and development](#verification-and-development)
- [Contributing](#contributing)

---

## What this is

- **One HTTP API** (port **8123** in the default stack) for memory lifecycle, recall bundles, enforcement, and related workflows.
- **MCP over HTTP** at **`POST /v1/mcp`** on the **same** base URL—**service-first MCP** is canonical; stdio MCP is compatibility-only.
- **Global memory pool** in **Postgres** (with **pgvector** for optional semantic retrieval). Correlation is **tags + retrieval text**, not per-agent silos.
- **Repo governance files** (`constitution.md`, `active.md`, `workorders/`, `evidence/`) describe how *this* repository uses recall; Pluribus is the engine you can run for any codebase or team that adopts the same discipline.

**Canonical product model:** [docs/memory-doctrine.md](docs/memory-doctrine.md) · **System shape:** [docs/architecture.md](docs/architecture.md)

---

## Why use it

| Benefit | What it avoids |
|--------|----------------|
| **Continuity** | Re-deriving the same constraints and failures every session |
| **Shared truth** | Fragmented “memory” trapped in one editor or one chat |
| **Authority-aware recall** | Flattening everything into one similarity score |
| **Enforcement before risky edits** | Shipping changes that violate stated decisions or patterns |
| **Curated learning** | Promoting noise; the loop favors **validated**, **typed** statements |

---

## Quick start (Docker — recommended)

**Prerequisites:** Docker Compose v2, `curl` (optional: `jq`).

From the **repository root**:

```bash
docker compose up -d
```

This starts **Postgres** (pgvector image), **Redis**, and **controlplane** on **`http://127.0.0.1:8123`**. On boot the API waits for the database, applies embedded baseline SQL, and verifies core tables. Data persists in the **`recall_pgdata`** volume until you remove it.

**Check that it is up:**

```bash
curl -sS http://127.0.0.1:8123/healthz
curl -sS http://127.0.0.1:8123/readyz
```

Both should return `ok` when startup has finished.

**Next — use an AI agent with Pluribus:** add MCP (**`http://127.0.0.1:8123/v1/mcp`**) in your editor and paste the loop from **[`integrations/pluribus-instructions.md`](integrations/pluribus-instructions.md)** (or your platform’s pack under **[`integrations/`](integrations/)**). Full walkthrough: **[`docs/pluribus-quickstart.md`](docs/pluribus-quickstart.md)** §4 and **[`docs/mcp-usage.md`](docs/mcp-usage.md)**. **Cursor:** **[`integrations/cursor/README.md`](integrations/cursor/README.md)**.

**Prefer a published image (no local build)?** Use **`docker-compose.install.yml`** and a registry image—see [INSTALL.md](INSTALL.md) and [docs/pluribus-container-install.md](docs/pluribus-container-install.md).

**Reset the dev database completely** (destructive): `docker compose down -v` then `docker compose up -d`. Pre-release builds assume a **fresh or disposable** Postgres; there is no GA-grade versioned migration story yet—see [INSTALL.md](INSTALL.md).

---

## Using Pluribus with AI agents

Point every client at the **same** API base URL (default **`http://127.0.0.1:8123`**). The process is **concurrent**; Postgres is the shared store. Protocol details: [docs/mcp-service-first.md](docs/mcp-service-first.md).

**Make memory habitual (not just connected):** [docs/integrations/usage.md](docs/integrations/usage.md) — behavioral loop, verification, failure modes — and [docs/usage/ensuring-agent-usage.md](docs/usage/ensuring-agent-usage.md) for full operational detail.

### Using Pluribus with AI editors and agent systems

**Pluribus is MCP-first** for agents: institutional memory and cognitive extension—not a generic side tool. **REST** remains the service, test, and admin boundary.

| Platform | Guide | Pack (`pluribus-instructions.md`, native templates, `skill.md`) |
|----------|-------|-------------------------------------------|
| **Hub + matrix** | [docs/integrations/README.md](docs/integrations/README.md) | [docs/integrations/matrix.md](docs/integrations/matrix.md) |
| **Cursor** | [docs/integrations/cursor.md](docs/integrations/cursor.md) | [integrations/cursor/](integrations/cursor/) |
| **Claude Code** | [docs/integrations/claude-code.md](docs/integrations/claude-code.md) | [integrations/claude-code/](integrations/claude-code/) |
| **Claude Desktop** | [docs/integrations/claude-desktop.md](docs/integrations/claude-desktop.md) | [integrations/claude-desktop/](integrations/claude-desktop/) |
| **OpenClaw** | [docs/integrations/openclaw.md](docs/integrations/openclaw.md) | [integrations/openclaw/](integrations/openclaw/) |
| **OpenCode** | [docs/integrations/opencode.md](docs/integrations/opencode.md) | [integrations/opencode/](integrations/opencode/) |
| **Continue** | [docs/integrations/continue.md](docs/integrations/continue.md) | [integrations/continue/](integrations/continue/) |
| **Zed** | [docs/integrations/zed.md](docs/integrations/zed.md) | [integrations/zed/](integrations/zed/) |
| **VS Code** | [docs/integrations/vscode.md](docs/integrations/vscode.md) | [integrations/vscode/](integrations/vscode/) |
| **Any MCP client** | [docs/integrations/generic-mcp.md](docs/integrations/generic-mcp.md) | [integrations/generic-mcp/](integrations/generic-mcp/) |

**Why integrate early:** recall and episodic ingest work best when they are **default habits**—see [docs/integrations/README.md](docs/integrations/README.md) and [docs/memory-doctrine.md](docs/memory-doctrine.md).

Each **`integrations/<platform>/`** pack includes a pointer **`rules.md`**, editor-native templates where applicable, **[`integrations/pluribus-instructions.md`](integrations/pluribus-instructions.md)** (canonical loop text), **`skill.md`**, **`README.md`**, and usually **`mcp-config.example.json`**—directive templates, **do not commit secrets**.

**Cursor (full “plugin” pack):** **[`integrations/cursor/`](integrations/cursor/)** bundles MCP JSON (**`mcp-config.json`**, no-auth + LAN examples), **`pluribus.mdc`** (single canonical rule), Agent Skill, **[`prompts.md`](integrations/cursor/prompts.md)**, **[`commands.md`](integrations/cursor/commands.md)**, **[`helper/verify-mcp.sh`](integrations/cursor/helper/verify-mcp.sh)**, and **[`plugin-plan.md`](integrations/cursor/plugin-plan.md)** (what Cursor actually supports). Prefer **user-level** **`~/.cursor/mcp.json`** and **user rules** so Pluribus applies in every repository—see **[`integrations/cursor/README.md`](integrations/cursor/README.md)**.

**VS Code (real extension):** **[`integrations/vscode/extension/`](integrations/vscode/extension/)** — **Recall Context**, **Record Experience**, **View Learnings** against the control-plane REST API; Explorer sidebar + Output channel. See **[`integrations/vscode/README.md`](integrations/vscode/README.md)** and **[`docs/integrations/vscode.md`](docs/integrations/vscode.md)**.

**Optional auth:** if the server has **`PLURIBUS_API_KEY`** set, send **`X-API-Key`** on HTTP MCP and in **`headers`** below; if unset, omit them. [docs/authentication.md](docs/authentication.md).

### SDKs for custom agents (Go + Python)

If you are building your own agent runtime, use the minimal SDKs (**recall → work → record**):

- [docs/sdk/README.md](docs/sdk/README.md)
- [docs/sdk/go.md](docs/sdk/go.md)
- [docs/sdk/python.md](docs/sdk/python.md)

Examples:

- `examples/go/minimal_loop/main.go`
- `examples/python/minimal_loop.py`

### MCP configuration by client

Use these as templates; adjust host, port, and paths for your machine.

**Endpoint (HTTP MCP):** `http://127.0.0.1:8123/v1/mcp` (JSON-RPC `POST`).

**Fallback (stdio):** build `cd control-plane && go build -o pluribus-mcp ./cmd/pluribus-mcp`, then run the binary with **`CONTROL_PLANE_URL=http://127.0.0.1:8123`** (and **`CONTROL_PLANE_API_KEY`** when auth is on). Same tools as HTTP; prompts/resources are HTTP-only on the service.

#### Cursor

**Global** `~/.cursor/mcp.json` is ideal so Pluribus MCP is available in **every** repository you open; use **repository-local** `.cursor/mcp.json` only for overrides. [Cursor MCP reference](https://cursor.com/docs/context/mcp).

**Preferred — HTTP MCP** (no local `pluribus-mcp` binary):

```json
{
  "mcpServers": {
    "pluribus": {
      "url": "http://127.0.0.1:8123/v1/mcp"
    }
  }
}
```

**With API key** (use an env var; do not commit secrets):

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

**Alternative — stdio** (if HTTP transport is unavailable):

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

Restart Cursor after edits. Troubleshooting: [docs/mcp-usage.md](docs/mcp-usage.md#cursor-specific-behavior).

#### Claude Desktop

Config file (typical): macOS `~/Library/Application Support/Claude/claude_desktop_config.json`, Windows `%APPDATA%\Claude\claude_desktop_config.json`, Linux `~/.config/Claude/claude_desktop_config.json`. Claude usually expects a **stdio** server—point **`command`** at your **`pluribus-mcp`** binary and set **`CONTROL_PLANE_URL`**.

```json
{
  "mcpServers": {
    "pluribus": {
      "command": "/absolute/path/to/recall/control-plane/pluribus-mcp",
      "env": {
        "CONTROL_PLANE_URL": "http://127.0.0.1:8123"
      }
    }
  }
}
```

Add **`CONTROL_PLANE_API_KEY`** to **`env`** when the server uses **`PLURIBUS_API_KEY`**. Fully quit and reopen Claude Desktop after saving.

#### OpenClaw

OpenClaw reads MCP settings from its own config (commonly under **`~/.openclaw/`** — exact filename and schema depend on your OpenClaw version). Use the **stdio** pattern: **`command`** = path to **`pluribus-mcp`**, **`env.CONTROL_PLANE_URL`** = `http://127.0.0.1:8123` (or your host). If your build documents **remote / SSE / URL** MCP servers, configure the URL to **`http://<host>:8123/v1/mcp`** and **`X-API-Key`** when auth is enabled—match the [HTTP verification](docs/mcp-migration-stdio-to-http.md) `curl` there.

#### OpenCode

Add a **`mcp.pluribus`** entry in **`opencode.json`** at the **repository root** or in **`~/.config/opencode/opencode.json`**: **`type": "remote"`**, **`url": "http://127.0.0.1:8123/v1/mcp"`**, **`oauth": false`**. When **`PLURIBUS_API_KEY`** is set on the server, add **`headers`** with **`X-API-Key`** (e.g. **`{env:PLURIBUS_API_KEY}`**). Example: [integrations/opencode/mcp-config.example.json](integrations/opencode/mcp-config.example.json). Full guide: [docs/integrations/opencode.md](docs/integrations/opencode.md). [OpenCode MCP servers](https://dev.opencode.ai/docs/mcp-servers).

#### VS Code, Zed, Windsurf, Claude Code (CLI)

Follow that editor’s MCP panel: **HTTP** → base URL **`http://127.0.0.1:8123`**, MCP path **`/v1/mcp`**; **stdio** → **`pluribus-mcp`** + **`CONTROL_PLANE_URL`**. Claude Code often supports a repository-level MCP file similar to Cursor; use the same **HTTP** or **stdio** blocks as above.

#### Workflow and deeper docs

Recommended tool order (ground → act → enforce → learn): [docs/mcp-usage.md](docs/mcp-usage.md#recall-driven-workflow-recommended-order). Full client matrix and edge cases: [docs/mcp-usage.md](docs/mcp-usage.md).

---

## Ensuring your agent uses Pluribus

**MCP is only useful if tools are called.** Loop: **`recall_context` → plan → act → `record_experience`** (aliases **`memory_context_resolve`** / **`mcp_episode_ingest`**). Wire MCP **and** install the platform **native** rules per **`integrations/<platform>/README.md`** (from **[`pluribus-instructions.md`](integrations/pluribus-instructions.md)**), plus **`snippets/context-prime.txt`**, **`skill.md`**. See **[docs/usage/ensuring-agent-usage.md](docs/usage/ensuring-agent-usage.md)** and [docs/usage/snippets/](docs/usage/snippets/).

---

## Multi-agent shared memory

Memory rows live in a **global pool**. **Agent A** can write durable decisions and patterns; **Agent B** (later, different session, different tool) calls **recall** with overlapping **tags** and sees the same substrate—**enforcement** can then flag proposals that contradict established memory.

Step-by-step demo (curl): **[docs/walkthrough-multi-agent.md](docs/walkthrough-multi-agent.md)**. Single-agent continuity: [docs/walkthrough-single-agent.md](docs/walkthrough-single-agent.md).

---

## Installation paths

| Path | When to use |
|------|-------------|
| **`docker compose up -d`** (repo root) | Default **developer** stack; builds API from `./control-plane`. |
| **`docker-compose.install.yml` + GHCR image** | **Operator / no Go toolchain**; set `PLURIBUS_IMAGE` per [INSTALL.md](INSTALL.md). |
| **Local binary** | `cd control-plane && make build && ./controlplane` against your own Postgres—see [INSTALL.md](INSTALL.md) option B. |

Build details: [BUILD.md](BUILD.md). Image build from source: `make image` (see [INSTALL.md](INSTALL.md)).

---

## Configuration

- **Compose (default):** the `controlplane` container uses **`CONFIG=/config/config.yaml`**. The baked-in defaults match the Compose service names (`postgres`, `redis`). Source: [control-plane/configs/config.yaml](control-plane/configs/config.yaml).
- **Local overrides (not committed):** **`control-plane/configs/config.local.yaml`** is gitignored; set **`CONFIG`** to that path when running on the host. See [.gitignore](.gitignore).
- **Environment (Compose):** optional **`PLURIBUS_API_KEY`** for API key auth—see [docker-compose.yml](docker-compose.yml) comment and [docs/authentication.md](docs/authentication.md).
- **Deeper topics** (recall weights, promotion gates, Redis, evidence paths): [control-plane/README.md](control-plane/README.md) and [docs/pluribus-operational-guide.md](docs/pluribus-operational-guide.md).

---

## Everyday API usage (smoke)

Same **create + recall** flow as [docs/pluribus-quickstart.md](docs/pluribus-quickstart.md) §3 — use that section for copy-paste and explanation. Quick duplicate:

```bash
curl -sS -X POST http://127.0.0.1:8123/v1/memories \
  -H 'Content-Type: application/json' \
  -d '{"kind":"constraint","authority":9,"statement":"Run tests before deploy.","tags":["demo","release"]}'

curl -sS -X POST http://127.0.0.1:8123/v1/recall/compile \
  -H 'Content-Type: application/json' \
  -d '{"tags":["demo","release"],"retrieval_query":"deploy safety"}' | jq .
```

Route index: [docs/http-api-index.md](docs/http-api-index.md). RC1-oriented examples: [docs/api-contract.md](docs/api-contract.md).

---

## Documentation map

| Need | Doc |
|------|-----|
| **Minimal path (new users)** | [docs/get-started.md](docs/get-started.md) |
| **Integration adoption & verification** | [docs/integrations/usage.md](docs/integrations/usage.md) |
| **Doctrine (highest authority)** | [docs/memory-doctrine.md](docs/memory-doctrine.md) |
| **Reviewer / CI guardrails** | [docs/anti-regression.md](docs/anti-regression.md) |
| **Quickstart** | [docs/pluribus-quickstart.md](docs/pluribus-quickstart.md) |
| **HTTP + MCP route map** | [docs/http-api-index.md](docs/http-api-index.md) |
| **Pre-change enforcement** | [docs/pre-change-enforcement.md](docs/pre-change-enforcement.md) |
| **Curation loop** | [docs/curation-loop.md](docs/curation-loop.md) |
| **Operations** | [docs/pluribus-operational-guide.md](docs/pluribus-operational-guide.md) |
| **Full index** | [docs/README.md](docs/README.md) |
| **AI editor / agent integrations** | [docs/integrations/README.md](docs/integrations/README.md) |
| **Agent actually uses memory** | [docs/usage/ensuring-agent-usage.md](docs/usage/ensuring-agent-usage.md) |

---

## Repository layout

| Purpose | Location |
|---------|----------|
| Control-plane API (Go module) | `control-plane/` |
| Integration pack (`pluribus-instructions.md`, native templates, `skill.md`, MCP examples) | `integrations/` (+ [docs/integrations/](docs/integrations/)) |
| Default Compose stack | [docker-compose.yml](docker-compose.yml) |
| Governing law (this repo) | `constitution.md` |
| Current focus | `active.md` |
| Work orders | `workorders/` |
| Evidence artifacts | `evidence/` |

Work order format: [docs/work-order-format.md](docs/work-order-format.md). Curation style: [docs/memory-curation.md](docs/memory-curation.md). File-based retrieval order (for agents working *in* this repo): [docs/retrieval-order.md](docs/retrieval-order.md).

---

## Verification and development

**Unit tests and vet** (from `control-plane/`):

```bash
cd control-plane
make build
go test ./...
go vet ./...
```

**Repo Makefile targets** (from root): `make test`, `make eval`, `make regression`—see [BUILD.md](BUILD.md).

**Substrate proof** (integration test against Postgres; needs **`TEST_PG_DSN`**; uses a **clean** database for deterministic scenarios): `cd control-plane && make proof-rest` — details [docs/evaluation.md](docs/evaluation.md), receipt [evidence/memory-proof.md](evidence/memory-proof.md). **Episodic pipeline stress proof** (advisory episodes, distillation, curation, materialize, recall/enforcement): `make proof-episodic` — [evidence/episodic-proof.md](evidence/episodic-proof.md). **Normal `docker compose` restarts** reuse your existing database; the clean-DB rule applies to **that** harness only.

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Changes should respect **global memory**, **tags + situation**, and the **memory doctrine**.
