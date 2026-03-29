# Pluribus container image — registry, tags, architectures

This document matches **GitHub Actions** in [`.github/workflows/ci.yml`](../.github/workflows/ci.yml).

---

## Registry and image name

| Item | Value |
|------|--------|
| **Registry** | **GitHub Container Registry (GHCR)** — `ghcr.io` |
| **Image** | **`ghcr.io/<github-owner-lowercase>/pluribus`** |

The image contains **only the control-plane API** binary (`controlplane` + `entrypoint.sh` + committed **`config.yaml`** installed as **`/config/config.yaml`**). **PostgreSQL** and **Redis** are **not** baked in; use Compose (see [pluribus-container-install.md](pluribus-container-install.md)). Override settings with **`CONFIG`** and a mounted file; for host development copy **`config.example.yaml`** → **`configs/config.local.yaml`** (see [control-plane/README.md](../control-plane/README.md)).

---

## Architectures

CI publishes every **`linux/*`** platform that appears on **both** official **`postgres:18-alpine`** and **`redis:7`** (the Compose dependencies in [`docker-compose.yml`](../docker-compose.yml)). Re-check anytime those image tags change:

```bash
docker manifest inspect postgres:18-alpine | jq ...
docker manifest inspect redis:7 | jq ...
```

| Platform | Support |
|----------|---------|
| `linux/386` | Yes |
| `linux/amd64` | Yes |
| `linux/arm/v7` | Yes |
| `linux/arm64` | Yes |
| `linux/ppc64le` | Yes |
| `linux/s390x` | Yes |

**Not** published (only one of the two deps ships it for these tags): e.g. `linux/riscv64` (Postgres only), `linux/mips64le` (Redis only), `linux/arm/v5` / `linux/arm/v6` (only one image each).

Built with `docker buildx` in CI (QEMU emulates non-`amd64` platforms on the GitHub runner, so publish jobs take longer than a two-arch build).

---

## Quality gates before publish

Publication runs **only** on `push` to `main` / `master` or `v*` tags (not on pull requests). **All** of the following must succeed first:

1. **`go test ./...`** in **`control-plane/`**  
2. **`make regression`** — control-plane integration suite (Docker Postgres, proof scenarios)  
3. **Dockerfile smoke build** — `docker build -f control-plane/Dockerfile control-plane`  

If any step fails, **no** image is pushed.

---

## Tag semantics

| Trigger | Tags pushed |
|---------|-------------|
| Push to **`main`** or **`master`** | `ghcr.io/<owner>/pluribus:<branch>` and `ghcr.io/<owner>/pluribus:sha-<short>` |
| Push **version tag** `v*` (e.g. `v0.4.0`) | `ghcr.io/<owner>/pluribus:v0.4.0` (full tag string) and `ghcr.io/<owner>/pluribus:latest` |

**`latest`** is updated only when a **`v*`** tag is pushed — not on every branch push.

**Rolling development:** use the **`main`** (or **`master`**) tag.

**Immutable releases:** use the **`v*`** tag (and optionally rely on **`latest`** for the last semver release).

---

## Pull (examples)

```bash
docker pull ghcr.io/<owner>/pluribus:main
docker pull ghcr.io/<owner>/pluribus:v0.4.0
```

---

## Package visibility

New GHCR packages may default to **private**. For public distribution, set the package **public** in the GitHub org/user **Packages** settings.

---

## Release notes

When cutting a release, mention:

- Image: `ghcr.io/<owner>/pluribus:<semver-tag>`
- Architectures: `linux/386`, `linux/amd64`, `linux/arm/v7`, `linux/arm64`, `linux/ppc64le`, `linux/s390x` (see table above)
- Install: [pluribus-container-install.md](pluribus-container-install.md), [pluribus-quickstart.md](pluribus-quickstart.md)  
- Proof index: [pluribus-proof-index.md](pluribus-proof-index.md)  
- Scope: [pluribus-release-scope.md](pluribus-release-scope.md)
