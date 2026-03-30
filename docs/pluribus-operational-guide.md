# Pluribus — operations and deployment (public)

This complements [INSTALL.md](../INSTALL.md) and [deployment-poc.md](deployment-poc.md) with a **single operational narrative**: what you must configure, what is optional, and how CI validates the stack.

---

## Deployment shapes

| Shape | Command / notes |
|-------|-----------------|
| **Published image + Compose (public default)** | `docker compose -f docker-compose.install.yml` + `PLURIBUS_IMAGE=ghcr.io/<owner>/pluribus:<tag>` — [pluribus-container-install.md](pluribus-container-install.md), [pluribus-image-release-policy.md](pluribus-image-release-policy.md). |
| **Clone + build (dev)** | Repo root: `docker compose up -d` — builds API from `./control-plane`, **:8123**, migrations on boot. |
| **Postgres + Redis only** | `docker compose up -d postgres redis` — run `./controlplane` on host; **Boot** applies embedded SQL on startup. |
| **Local Postgres only** | [INSTALL.md](../INSTALL.md) Option B. |

---

## Required vs optional config

Copy **`control-plane/configs/config.example.yaml`** → **`configs/config.local.yaml`** (gitignored), then run with **`CONFIG=configs/config.local.yaml`**. The image ships committed **`configs/config.yaml`** for Compose service DNS defaults.

| Area | Required | Notes |
|------|----------|--------|
| **`postgres.dsn`** | Yes (for real deployments) | Docker compose wires this via env in compose files. |
| **`PLURIBUS_API_KEY`** | No | If set, REST and **`POST /v1/mcp`** require **`X-API-Key`** (or **`?token=`** on MCP only). |
| **`mcp.disabled`** | No | If `true`, MCP endpoint is not mounted. |
| **`enforcement`** | No | Feature flags for pre-change enforcement. |
| **`evidence.root_path`** | No | Filesystem root for evidence artifacts if used. |
| **Startup wait** | No | `startup.db_wait_timeout_seconds`, `startup.db_wait_interval_millis` — tune slow DB. |

Full field list: [control-plane/README.md](../control-plane/README.md) (Config section).

---

## Health and readiness

| Endpoint | Meaning |
|----------|---------|
| **`GET /healthz`** | Process alive. |
| **`GET /readyz`** | DB reachable **and** core baseline present (**`memories`** table exists after boot’s SQL replay) (else **503**). |

Load balancers should use **`readyz`** for traffic.

---

## Schema (fresh Postgres)

- SQL files: `control-plane/migrations/*.sql`.
- Applied on **every control-plane startup** (idempotent DDL; **no** version ledger, **no** supported in-place upgrade from prior schemas). **Pre-release:** there are no GA installs to upgrade — use a **new** database or volume; do not point the server at legacy populated schemas expecting migration.
- Historical archive (obsolete flows): [archive/migration-unversioned-baseline.md](archive/migration-unversioned-baseline.md).

---

## CI and quality gates

| Gate | What runs |
|------|-----------|
| **Memory substrate proof (REST)** | **`cd control-plane && TEST_PG_DSN='…' make proof-rest`** — adversarial **`proof-*.json`** scenarios over HTTP + two-pass determinism; requires **Postgres + pgvector** on a **clean** DB. This is the **canonical behavioral proof** for the memory layer (see [evaluation.md](evaluation.md), [evidence/memory-proof.md](../evidence/memory-proof.md)). |
| **Episodic pipeline proof (REST)** | **`make proof-episodic`** (repo root or `control-plane/`) — runs **`proof-rest`** plus **`TestEpisodicProofSprintREST_Postgres`** (conflict, time skew, advisory/recall boundary, soak). See [evidence/episodic-proof.md](../evidence/episodic-proof.md), [episodic-similarity.md](episodic-similarity.md). |
| **Control-plane unit tests** | `cd control-plane && go test ./...` |
| **Control-plane integration (CI)** | **`make regression`** — Docker Postgres (no host ports), **`go test -tags=integration -count=1 ./...`**, includes YAML **`proof-scenarios/`** suite. |
| **Dockerfile smoke** | `docker build -f control-plane/Dockerfile control-plane` on every PR/push. |
| **GHCR publish** | Multi-arch image ( **`linux/386`**, **`linux/amd64`**, **`linux/arm/v7`**, **`linux/arm64`**, **`linux/ppc64le`**, **`linux/s390x`** — intersection of **`postgres:18-alpine`** and **`redis:7`**) push to **`ghcr.io/<owner>/pluribus`** — **only** after all gates above pass **and** on push to `main`/`master` or `v*` tags (not on PRs). See [pluribus-image-release-policy.md](pluribus-image-release-policy.md). |

**CI** enforces **`make regression`**. **Publishers and integrators** proving the substrate should run **`make proof-rest`** against pgvector-backed Postgres as well — it is the clearest **REST-first** receipt. Run **`make proof-episodic`** when validating the **advisory episodic** and **distillation → materialize** paths under pressure — [evidence/episodic-proof.md](../evidence/episodic-proof.md).

---

## Cursor / MCP operator notes

- **Canonical:** MCP **over HTTP** at `{base}/v1/mcp` — [mcp-service-first.md](mcp-service-first.md).
- **Compat:** stdio `pluribus-mcp` — [cmd/pluribus-mcp/README.md](../control-plane/cmd/pluribus-mcp/README.md).
- Cursor + MCP: [mcp-usage.md](mcp-usage.md).

---

## Optional feature defaults (operator clarity)

- **Authentication**: off unless `PLURIBUS_API_KEY` is set (see `docs/authentication.md`).
- **Semantic retrieval**: **on by default** for situation matching (pgvector + embeddings). Lexical/tag search still refines candidates. Set `recall.semantic_retrieval.enabled: false` only when you intentionally avoid embedding calls; misconfiguration logs `[SEMANTIC ERROR]` and falls back to lexical.
- **Triggered recall**: optional behavior gate controlled by recall config.
- **Pattern elevation**: optional and config-driven.
- **Eval/stress harnesses**: operator-invoked; run via `make eval` / `make stress-eval`.

---

## Troubleshooting

- **503 on `/readyz`:** DB down or core tables missing after boot — check logs and DSN; ensure Postgres is empty/greenfield for first start.
- **401 / 403 on API:** **`PLURIBUS_API_KEY`** set and missing/wrong **`X-API-Key`** (see [authentication.md](authentication.md), [api-contract.md](api-contract.md)).
- **Two compose projects:** Dev stack vs **`make regression`** use different compose projects — regression tears down only its own volumes — see [README.md](../README.md).
