# POC deployment runbook — Pluribus / Recall stack

**Goal:** Bring up **Postgres**, **Redis**, and **control-plane** so the HTTP API is reachable for recall, run-multi, and promotion. **Not** hardened for public internet (default: **`PLURIBUS_API_KEY`** unset — open API).

**Compose-first with a published image (no local Go build):** [pluribus-container-install.md](pluribus-container-install.md) — `docker-compose.install.yml` + `PLURIBUS_IMAGE` from GHCR.

---

## Prerequisites

| Requirement | Notes |
|-------------|--------|
| **Docker** + **Docker Compose** v2 | `docker compose version` |
| **Ports free** | **5432** (Postgres), **6379** (Redis), **8123** (control-plane) on the host, unless you change `docker-compose.yml` |
| **Disk** | Postgres volume persists data in Docker volume `recall_pgdata` |

**Optional (host-only / CI):** **Go 1.22+** for `go test` / local `./controlplane`. Schema SQL runs inside **server boot** only (no separate reconcile CLI). **Do not** run `go run ./control-plane/...` from repo root: `control-plane` is a **separate Go module** (`module control-plane`).

**Remote / server:** Binding **`8123:8123`** exposes the API on **all interfaces** by default. Use firewall rules; do **not** rely on default auth for any shared network.

---

## CI / regression (merge gate)

GitHub Actions ([`.github/workflows/ci.yml`](../.github/workflows/ci.yml)) runs:

1. **`go test ./...`** in **`control-plane/`** — authoritative Go module for the API.
2. **`make regression`** — **authoritative control-plane** suite (Docker ephemeral Postgres, includes **`-tags=integration`**).
3. **Dockerfile smoke build** — `docker build -f control-plane/Dockerfile control-plane`.
4. **GHCR publish** (push to `main`/`master` or `v*` tags only, after 1–3 pass) — multi-arch **`ghcr.io/<owner>/pluribus`** — [pluribus-image-release-policy.md](pluribus-image-release-policy.md).

Local parity: run **`make regression`** from repo root before merge. Standing benefit baseline (not CI): [foundational-beta-benefit-baseline.md](foundational-beta-benefit-baseline.md).

---

## 1. Clone and enter repo root

```bash
cd /path/to/recall   # repository root (contains docker-compose.yml)
```

---

## 2. Start the stack

```bash
docker compose up -d
```

**Services:** `postgres` (**`pgvector/pgvector:pg18`** in [`docker-compose.yml`](../docker-compose.yml)), `redis`, **`controlplane`**. On start, **control-plane** waits for DB, applies embedded baseline SQL (`migrations/*.sql`), verifies core tables exist, then serves HTTP on **8123**. Use a **fresh** Postgres database. **Pre-release:** there is **no** supported in-place database upgrade path and no GA install base to migrate — do not treat boot SQL replay as upgrading older schemas.

**Check:**

```bash
docker compose ps
curl -sS http://127.0.0.1:8123/healthz   # liveness
curl -sS http://127.0.0.1:8123/readyz    # readiness (DB + core schema)
```

---

## 3. How schema apply works (fresh install)

| Piece | Location / behavior |
|-------|---------------------|
| **SQL files** | [`control-plane/migrations/*.sql`](../control-plane/migrations/) — lexicographic order |
| **Runner** | [`internal/migrate`](../control-plane/internal/migrate) + embedded **`migrations/*.sql`** — runs every file each boot (idempotent DDL); **no** version table and **no** upgrade engine |
| **When it runs** | **`app.Boot`** in [`internal/app/boot.go`](../control-plane/internal/app/boot.go) — waits for DB (startup timeout + retry), applies SQL, checks core tables (e.g. `memories`); each **control-plane** start |

**Compose stack:** **`postgres`**, **`redis`**, **`controlplane`** — schema work happens **inside the control-plane process** during `Boot`, not in a separate migrate job.

Historical archive (obsolete `schema_migrations` / host `migrate.sh` narratives): [archive/migration-unversioned-baseline.md](archive/migration-unversioned-baseline.md).

---

## 4. Configuration (Docker)

The **controlplane** image bakes [`control-plane/configs/config.yaml`](../control-plane/configs/config.yaml) and starts through [`control-plane/scripts/entrypoint.sh`](../control-plane/scripts/entrypoint.sh):

- **Postgres DSN:** `postgres://controlplane:controlplane@postgres:5432/controlplane?sslmode=disable` (service name **`postgres`** from inside the compose network).
- **Redis:** `redis:6379`.
- **Auth:** omit **`PLURIBUS_API_KEY`** for local/controlled dev; set it for any shared or public network (see [authentication.md](authentication.md), [api-contract.md](api-contract.md)).
- **Startup wait knobs:** `startup.db_wait_timeout_seconds`, `startup.db_wait_interval_millis`.

To customize, rebuild the image after editing the file, or add a volume mount in `docker-compose.yml` (advanced).

---

## 5. Health and readiness

| Check | Command |
|-------|---------|
| **Liveness** | `curl -sS http://HOST:8123/healthz` → `ok` |
| **Readiness** | `curl -sS http://HOST:8123/readyz` → `ok` when DB ping succeeds and the **`memories`** table exists (baseline applied on boot) |
| Postgres | `docker compose exec postgres pg_isready -U controlplane -d controlplane` |
| Redis | `docker compose exec redis redis-cli ping` → `PONG` |

---

## 6. Optional: verify from another machine

Use the **host’s LAN IP** (not `127.0.0.1`):

```bash
curl -sS http://SERVER_IP:8123/healthz
curl -sS http://SERVER_IP:8123/readyz
```

Ensure firewall allows **TCP 8123** (and **5432** only if you intentionally expose Postgres).

---

## 7. Common failures

| Symptom | Likely cause | What to do |
|---------|----------------|------------|
| `connection refused` on :8123 | Controlplane not up or still building | `docker compose logs -f controlplane` |
| **`readyz` 503** | DB down or migrations failed during boot | `docker compose logs controlplane` |
| API 500 on use | Rare if `readyz` ok | App logs |
| `port is already allocated` | Another process uses 5432/6379/8123 | Stop conflicting service or change **ports:** in `docker-compose.yml` |
| Postgres unhealthy | First start / disk | `docker compose logs postgres` |
| Run-multi “not configured” for server-side synthesis | **`synthesis.enabled: false`** (default) | Enable **`synthesis`** in config or use client LLM — [backend-synthesis.md](../control-plane/docs/backend-synthesis.md), [mcp-poc-contract.md](mcp-poc-contract.md) |

---

## 8. Stop / reset

```bash
docker compose down
```

**Remove Postgres data** (destructive):

```bash
docker compose down -v
```

---

## Next

- **MCP thin server:** [`control-plane/cmd/pluribus-mcp`](../control-plane/cmd/pluribus-mcp) — [mcp-poc-contract.md](mcp-poc-contract.md).
- **End-to-end walkthrough:** [poc-e2e-walkthrough.md](poc-e2e-walkthrough.md).
- **Unversioned baseline (archive):** [archive/migration-unversioned-baseline.md](archive/migration-unversioned-baseline.md).
