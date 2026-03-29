# Pluribus — release readiness checklist

Hard gate before calling a revision **public-release ready**. Items are **binary** (done / not done).

---

## Documentation coherence

- [ ] **HTTP surface** matches runtime — [http-api-index.md](http-api-index.md) + [api-contract.md](api-contract.md) (subset narrative).
- [ ] **One public architecture story** — [pluribus-public-architecture.md](pluribus-public-architecture.md) (linked from root [README.md](../README.md)).
- [ ] **Quickstart** runnable — [pluribus-quickstart.md](pluribus-quickstart.md).
- [ ] **Operations** (config, health, migrations, CI) — [pluribus-operational-guide.md](pluribus-operational-guide.md), [INSTALL.md](../INSTALL.md).
- [ ] **Service-first** unmistakable: HTTP + **`POST /v1/mcp`** canonical; stdio labeled compat — [mcp-service-first.md](mcp-service-first.md).
- [ ] **Memory doctrine** is the behavioral authority — [memory-doctrine.md](memory-doctrine.md), [anti-regression.md](anti-regression.md).
- [ ] **MCP usage** single entry — [mcp-usage.md](mcp-usage.md).
- [ ] **Release scope** vs deferred explicit — [pluribus-release-scope.md](pluribus-release-scope.md).
- [ ] **Proof bundle** linked — [pluribus-proof-index.md](pluribus-proof-index.md), [proof-scenarios.md](proof-scenarios.md).
- [ ] **Doc map** — [docs/README.md](README.md).

---

## Operator bring-up (memory-first smoke)

On a running API (Compose or local binary), with optional `X-API-Key` if configured:

1. `GET /healthz` and `GET /readyz` return **200** / **`ok`** when DB is ready.
2. `POST /v1/memories` — create a **constraint** (or other kind) with **tags** in the shared pool.
3. `POST /v1/recall/compile` — body uses **`tags`** + **`retrieval_query`** only (memory-first path per doctrine).
4. `POST /v1/enforcement/evaluate` — **`proposal_text`** required; optional fields per **`EvaluateRequest`** ([http-api-index.md](http-api-index.md)).

Config synthesis on/off, run-multi **503** when synthesis disabled, and full curl matrices for legacy operator steps live in **archived** material: [archive/rc1-checklist.md](archive/rc1-checklist.md) — **not** active system truth.

---

## Technical gates

- [ ] **`go test ./...`** passes in **`control-plane/`**.
- [ ] **`make regression`** passes (integration proof suite).
- [ ] **README** “canonical” paths do not present stdio MCP as default.
- [ ] **Container publish** — CI + [pluribus-image-release-policy.md](pluribus-image-release-policy.md) aligned.
- [ ] **Compose-first install** — [pluribus-container-install.md](pluribus-container-install.md), [`docker-compose.install.yml`](../docker-compose.install.yml), [`pluribus.install.env.example`](../pluribus.install.env.example) accurate.

---

## Honesty

- [ ] **Limitations** (not multi-host, not embedding-authority) in proof or scope docs.
- [ ] **Advisory similarity** not presented as canonical recall — [episodic-similarity.md](episodic-similarity.md).

---

## Optional (release tag)

- [ ] Post-release roadmap — [pluribus-post-release-roadmap.md](pluribus-post-release-roadmap.md).

---

## Sign-off

| Role | Name | Date |
|------|------|------|
| Maintainer | | |

When required boxes are checked, the release is **documentation-ready**. **Operational** readiness (hosting, secrets, backups) is environment-specific.
