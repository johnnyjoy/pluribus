# BUILD.md — build Pluribus & `pluribus-mcp` (release truth)

This file is the **build-from-source** and **release-artifacts** reference.

---

## What you are building

- `control-plane` (server): Go binary `controlplane` (HTTP API; also the canonical MCP-over-HTTP implementation at `POST /v1/mcp`)
- `pluribus-mcp` (compat adapter): thin stdio JSON-RPC server that forwards `tools/call` to the control-plane HTTP API

---

## Prerequisites

- Go `1.22+`
- Docker + Docker Compose v2 (only if you want to run the Docker regression suite)

---

## Build from source (local)

### 1. Build control-plane

```bash
cd control-plane
go build -o controlplane ./cmd/controlplane
```

### 2. Build `pluribus-mcp`

```bash
cd control-plane
go build -o pluribus-mcp ./cmd/pluribus-mcp
```

---

## Test (local)

### 1. Control-plane unit tests

```bash
cd control-plane
go test ./...
```

### 1b. Repo-level test-drive commands

From repo root:

```bash
make test
make eval
make stress-eval
make test-drive
```

Optional host-managed integration targets:

```bash
# automated: starts ephemeral Postgres and runs full integration-tagged suite (generated DSN)
make integration-go

# manual: you supply TEST_PG_DSN
make api-test
make integration-test
```

**Experimental — pg_textsearch (BM25 lexical layer):** `make pg-textsearch-image`, **`make pg-textsearch-eval`** (full automated eval + artifacts), `make lexical-backfill` / `lexical-reindex` / `lexical-verify`, overlay `docker-compose.pg-textsearch.yml`, docs under [docs/experiments/](docs/experiments/).

Artifacts written by eval targets:

- `artifacts/eval-report.txt`
- `artifacts/eval-report.json`
- `artifacts/stress-report.txt`
- `artifacts/stress-report.json`

### 2. Authoritative Docker regression suite

From the repo root:

```bash
make regression
```

---

## Build the Docker image (control-plane)

From the repo root:

```bash
make image
```

This builds the `control-plane/Dockerfile` image and embeds the same `VERSION` the build uses.

---

## Build outputs vs published releases

### Control-plane container image (GHCR)

Published by CI to `ghcr.io/<owner>/pluribus` (multi-arch) — see:

- [docs/pluribus-image-release-policy.md](docs/pluribus-image-release-policy.md)

### `pluribus-mcp` binaries (GitHub Releases)

CI publishes release artifacts as `tar.gz` per Linux architecture on Git tags `v*`:

- supported targets (architectures):
  - `linux/386`
  - `linux/amd64`
  - `linux/arm/v7`
  - `linux/arm64`
  - `linux/ppc64le`
  - `linux/s390x`
- assets: `pluribus-mcp-linux-<suffix>.tar.gz` where `<suffix>` matches the architecture above
- assets: `pluribus-memory-<tag>.vsix` — VS Code extension built from `integrations/vscode/extension/`
- assets: `pluribus-integration-packs-<tag>.zip` — snapshot of `integrations/cursor`, `claude-code`, `generic-mcp`, `opencode`
- assets: `SHA256SUMS.txt` (checksums for the tarballs, `.vsix`, and integration zip)

After extracting a tarball, the archive root contains a single binary: `pluribus-mcp`.

---

## Run notes

### Control-plane server

`controlplane` serves HTTP on `:8123` by default (see `control-plane/configs/config.example.yaml`).

### Secure it with `PLURIBUS_API_KEY`

If `PLURIBUS_API_KEY` is set on the server environment (non-empty after trim):

- REST endpoints and `POST /v1/mcp` require `X-API-Key: <PLURIBUS_API_KEY>`

If `PLURIBUS_API_KEY` is not set:

- the API is publicly accessible, and the server logs an explicit warning at startup

This is intentional for technical preview local/LAN evaluation.

### Run `pluribus-mcp` (stdio compat)

Example:

```bash
export CONTROL_PLANE_URL=http://127.0.0.1:8123
export CONTROL_PLANE_API_KEY=<your PLURIBUS_API_KEY value> # only needed when server auth is enabled
./pluribus-mcp
```

