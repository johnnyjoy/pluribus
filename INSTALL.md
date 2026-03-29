# Install Guide

**Compose-first (published image):** [docs/pluribus-container-install.md](docs/pluribus-container-install.md) — pull **`ghcr.io/<owner>/pluribus:<tag>`**, set `PLURIBUS_IMAGE`, use [`docker-compose.install.yml`](docker-compose.install.yml) + [`pluribus.install.env.example`](pluribus.install.env.example). No Go build required.

**Local image (build from this repo):** from repo root, `make image` runs `docker build` with `--build-arg VERSION` (default: `git describe --tags --always --dirty`, or override `make image PLURIBUS_VERSION=1.2.3`). Produces tags `pluribus:local` and `pluribus:<version>` with the version string embedded in the `controlplane` binary. Point `PLURIBUS_IMAGE` at one of those tags to use [`docker-compose.install.yml`](docker-compose.install.yml) without pulling from GHCR.

**Quickstart (HTTP smoke):** [docs/pluribus-quickstart.md](docs/pluribus-quickstart.md).
**Authentication behavior:** [docs/authentication.md](docs/authentication.md).
**Evaluation/stress guide:** [docs/evaluation.md](docs/evaluation.md).

**Database posture (pre-release):** the server **does not** implement versioned schema upgrades. There are **no** GA releases or install bases to migrate yet — use a **fresh** Postgres (or `docker compose down -v` when resetting). Pointing the API at an arbitrary old schema is unsupported.

This repository supports three distinct setup paths:

- **Compose-first with published image (operator-style):** no local Go build required.
- **Compose-first with local image:** build image from this repo, then run the same install compose path.
- **Local binaries + local Postgres (developer-style):** run `controlplane` directly on host.

## Prerequisites

- Docker + Docker Compose v2
- Go 1.22+ (for local builds/tests)
- `make`

## Option A: Compose-first (recommended)

From repo root:

```bash
docker compose up -d
```

This starts the default local stack:

- `postgres`
- `redis`
- `controlplane` (HTTP API on `:8123`)

`controlplane` startup waits for DB, applies embedded baseline SQL, and verifies core tables exist (fresh Postgres intended).

### Verify

```bash
curl -sS http://127.0.0.1:8123/healthz
curl -sS http://127.0.0.1:8123/readyz
```

Expected:
- `/healthz` returns `ok` when process is alive.
- `/readyz` returns `ok` when startup + schema checks are complete.

## Option B: Local binaries + local Postgres

1) Create DB:

```bash
createdb controlplane
```

2) Build (from `control-plane/`):

```bash
cd control-plane
make build
```

3) Run API (first start applies embedded SQL to a fresh database):

```bash
./controlplane
```

## Automated Regression (required CI tenet)

From repo root:

```bash
make regression
```

What it does:

- Spins up an **ephemeral Postgres** only for the regression run
- Uses Docker Compose project `recall-regression`
- Does **not** publish DB ports to host
- Runs control-plane tests (including integration-tagged tests) inside Docker
- Tears down only the regression project volumes/network

## CI

GitHub Actions on push/PR to `main`/`master` runs **two** jobs (see [`.github/workflows/ci.yml`](.github/workflows/ci.yml)):

1. **`go test ./...`** in **`control-plane/`** — unit tests for the authoritative API module.
2. **`make regression`** — authoritative gate (ephemeral Docker Postgres + integration tests).

## Standing benefit baseline

Lean procedure (correctness vs benefit): [docs/foundational-beta-benefit-baseline.md](docs/foundational-beta-benefit-baseline.md)

## Related Docs

- Root overview: [`README.md`](README.md)
- Control-plane details: [`control-plane/README.md`](control-plane/README.md)
- Deployment runbook: [`docs/deployment-poc.md`](docs/deployment-poc.md)
