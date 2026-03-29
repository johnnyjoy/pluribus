# Pluribus — compose-first install (published image)

**Canonical path:** pull the **published** Pluribus image from **GHCR**, run **PostgreSQL** + **Redis** via Compose, start **without** building Go from source.

Build-from-source remains valid for **developers**; it is **not** the default public story — see [pluribus-public-architecture.md](pluribus-public-architecture.md).

---

## What you need

| Dependency | Required? | Notes |
|------------|-----------|--------|
| **PostgreSQL** | **Yes** | Durable memory and migrations run against Postgres. |
| **Redis** | **Recommended** | Default `config.yaml` enables Redis for caching; **run the Redis service** in Compose for parity with dev stacks. |
| **Docker / Compose** | **Yes** | Compose v2. |

---

## 1. Choose an image tag

Images are published by CI to:

`ghcr.io/<your-github-username-lowercase>/pluribus:<tag>`

See [pluribus-image-release-policy.md](pluribus-image-release-policy.md) for tags (`main`, `sha-*`, `v*`, `latest`).

---

## 2. Configure env

From the repo root:

```bash
cp pluribus.install.env.example .env
# Edit .env: set PLURIBUS_IMAGE=ghcr.io/<owner>/pluribus:main (or your tag)
```

Or pass the file without copying:

```bash
docker compose -f docker-compose.install.yml --env-file pluribus.install.env.example up -d
```
(After editing `pluribus.install.env.example` with your real image.)

---

## 3. Start the stack

```bash
docker compose -f docker-compose.install.yml up -d
```

This starts **postgres**, **redis**, and **controlplane** on **:8123** (first pull may take a minute).

---

## 4. Health and readiness

| Endpoint | Meaning |
|----------|---------|
| **`GET /healthz`** | Process is up. |
| **`GET /readyz`** | Postgres reachable **and** migrations **current** — use this before treating the service as ready. |

```bash
curl -sS http://127.0.0.1:8123/healthz
curl -sS http://127.0.0.1:8123/readyz
```

---

## 5. Connect

- **HTTP:** `http://127.0.0.1:8123/v1/...` — see [pluribus-quickstart.md](pluribus-quickstart.md).  
- **MCP over HTTP:** `POST http://127.0.0.1:8123/v1/mcp` — [mcp-service-first.md](mcp-service-first.md).  

**Stdio `pluribus-mcp`** is **compat-only** — not part of this install path.

---

## 6. Local dev stack (build from source)

The default repo root **`docker-compose.yml`** still **builds** the control-plane image from `./control-plane` for **local development**. Use **`docker-compose.install.yml`** when you want a **published** image instead.

---

## References

- [pluribus-operational-guide.md](pluribus-operational-guide.md) — config, auth, CI  
- [INSTALL.md](../INSTALL.md) — alternate paths (local Postgres)  
- [deployment-poc.md](deployment-poc.md) — deeper runbook  
