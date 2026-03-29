<div align="center">

<img src="docs/assets/Pluribus_logo.png" alt="Pluribus" width="280">

</div>

# Pluribus

**Pluribus** is the **open control plane** for governed AI memory: **Postgres-backed** durable rows, **situation-shaped recall**, **pre-change enforcement**, and **curation**—without treating chat or logs as source of truth.

**Recall** is the name of the **model and doctrine** this repository embodies (constraints, decisions, patterns, failures, and related kinds that outlast any single session). Pluribus is the **runnable service**—HTTP API, MCP, shared global memory pool—described in [docs/memory-doctrine.md](docs/memory-doctrine.md).

---

## Table of contents

- [What this is](#what-this-is)
- [Why use it](#why-use-it)
- [Quick start (Docker — recommended)](#quick-start-docker--recommended)
- [Using Pluribus with AI agents](#using-pluribus-with-ai-agents)
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

**Prefer a published image (no local build)?** Use **`docker-compose.install.yml`** and a registry image—see [INSTALL.md](INSTALL.md) and [docs/pluribus-container-install.md](docs/pluribus-container-install.md).

**Reset the dev database completely** (destructive): `docker compose down -v` then `docker compose up -d`. Pre-release builds assume a **fresh or disposable** Postgres; there is no GA-grade versioned migration story yet—see [INSTALL.md](INSTALL.md).

---

## Using Pluribus with AI agents

Point every client at the **same** API base URL (default **`http://127.0.0.1:8123`**). The process is **concurrent**; Postgres is the shared store. Protocol details: [docs/mcp-service-first.md](docs/mcp-service-first.md).

**Optional auth:** if the server has **`PLURIBUS_API_KEY`** set, send **`X-API-Key`** on HTTP MCP and in **`headers`** below; if unset, omit them. [docs/authentication.md](docs/authentication.md).

### MCP configuration by client

Use these as templates; adjust host, port, and paths for your machine.

**Endpoint (HTTP MCP):** `http://127.0.0.1:8123/v1/mcp` (JSON-RPC `POST`).

**Fallback (stdio):** build `cd control-plane && go build -o pluribus-mcp ./cmd/pluribus-mcp`, then run the binary with **`CONTROL_PLANE_URL=http://127.0.0.1:8123`** (and **`CONTROL_PLANE_API_KEY`** when auth is on). Same tools as HTTP; prompts/resources are HTTP-only on the service.

#### Cursor

**Repository-local** `.cursor/mcp.json` or **global** `~/.cursor/mcp.json`. [Cursor MCP reference](https://cursor.com/docs/context/mcp).

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

#### VS Code, Zed, Windsurf, Claude Code (CLI)

Follow that editor’s MCP panel: **HTTP** → base URL **`http://127.0.0.1:8123`**, MCP path **`/v1/mcp`**; **stdio** → **`pluribus-mcp`** + **`CONTROL_PLANE_URL`**. Claude Code often supports a repository-level MCP file similar to Cursor; use the same **HTTP** or **stdio** blocks as above.

#### Workflow and deeper docs

Recommended tool order (ground → act → enforce → learn): [docs/mcp-usage.md](docs/mcp-usage.md#recall-driven-workflow-recommended-order). Full client matrix and edge cases: [docs/mcp-usage.md](docs/mcp-usage.md).

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

Memory is **global**; **tags** and **`retrieval_query`** shape the situation:

```bash
curl -sS -X POST http://127.0.0.1:8123/v1/memories \
  -H 'Content-Type: application/json' \
  -d '{"kind":"constraint","authority":9,"statement":"Run tests before deploy.","tags":["demo","release"]}'

curl -sS -X POST http://127.0.0.1:8123/v1/recall/compile \
  -H 'Content-Type: application/json' \
  -d '{"tags":["demo","release"],"retrieval_query":"deploy safety"}' | jq .
```

Full first-run narrative: [docs/pluribus-quickstart.md](docs/pluribus-quickstart.md). Route index: [docs/http-api-index.md](docs/http-api-index.md). RC1-oriented examples: [docs/api-contract.md](docs/api-contract.md).

---

## Documentation map

| Need | Doc |
|------|-----|
| **Doctrine (highest authority)** | [docs/memory-doctrine.md](docs/memory-doctrine.md) |
| **Reviewer / CI guardrails** | [docs/anti-regression.md](docs/anti-regression.md) |
| **Quickstart** | [docs/pluribus-quickstart.md](docs/pluribus-quickstart.md) |
| **HTTP + MCP route map** | [docs/http-api-index.md](docs/http-api-index.md) |
| **Pre-change enforcement** | [docs/pre-change-enforcement.md](docs/pre-change-enforcement.md) |
| **Curation loop** | [docs/curation-loop.md](docs/curation-loop.md) |
| **Operations** | [docs/pluribus-operational-guide.md](docs/pluribus-operational-guide.md) |
| **Full index** | [docs/README.md](docs/README.md) |

---

## Repository layout

| Purpose | Location |
|---------|----------|
| Control-plane API (Go module) | `control-plane/` |
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

**Substrate proof** (integration test against Postgres; needs **`TEST_PG_DSN`**; uses a **clean** database for deterministic scenarios): `cd control-plane && make proof-rest` — details [docs/evaluation.md](docs/evaluation.md), receipt [evidence/memory-proof.md](evidence/memory-proof.md). **Normal `docker compose` restarts** reuse your existing database; the clean-DB rule applies to **that** harness only.

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Changes should respect **global memory**, **tags + situation**, and the **memory doctrine**.
