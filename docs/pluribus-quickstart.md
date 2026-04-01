# Pluribus — quickstart (first five minutes)

**Goal:** Run the service, verify health, **create durable memory** in the **shared global pool**, run **recall** with **tags + situation text**, and know where **MCP over HTTP** and the **memory doctrine** live. See [memory-doctrine.md](memory-doctrine.md) (canonical model).

**Prerequisites:** Docker Compose v2, `curl`, `jq` (optional).

**Ultra-short path:** [get-started.md](get-started.md). **Agents:** when **§2** is healthy, go to **§4** (§3 optional). Rules: **[`integrations/pluribus-instructions.md`](../integrations/pluribus-instructions.md)** · **[`integrations/`](../integrations/)** packs.

---

## 1. Start the stack (clone — usual first path)

From the **repository root** (builds the API from `./control-plane`):

```bash
docker compose up -d
```

Starts Postgres, Redis, and **controlplane** on **`http://127.0.0.1:8123`**. The API waits for the DB, applies embedded baseline SQL, then serves.

---

## Alternative: published image only (no local build)

For a **registry** install (no `git clone` build), use **`docker-compose.install.yml`** — [pluribus-container-install.md](pluribus-container-install.md), [pluribus-image-release-policy.md](pluribus-image-release-policy.md). Set **`PLURIBUS_IMAGE`**, then:

```bash
docker compose -f docker-compose.install.yml --env-file pluribus.install.env.example up -d
```

(Edit the env file with **your** image path first.)

---

## 2. Verify liveness and readiness

```bash
curl -sS http://127.0.0.1:8123/healthz
curl -sS http://127.0.0.1:8123/readyz
```

Both should return `ok` once startup completes.

## Auth default (technical preview)

- If `PLURIBUS_API_KEY` is unset, API auth is disabled.
- If `PLURIBUS_API_KEY` is set, clients must send `X-API-Key`.

This behavior is intentional for low-friction localhost/LAN test-drive usage.

---

## 3. Seed memory and compile recall (smoke)

Memory is **global**; **tags** and **retrieval_query** shape the situation — no silo selector.

```bash
curl -sS -X POST http://127.0.0.1:8123/v1/memories \
  -H 'Content-Type: application/json' \
  -d '{"kind":"constraint","authority":9,"statement":"Use Postgres for durable data.","tags":["demo","quickstart"]}'

curl -sS -X POST http://127.0.0.1:8123/v1/recall/compile \
  -H 'Content-Type: application/json' \
  -d '{"tags":["demo","quickstart"],"retrieval_query":"durable data"}' | jq .
```

You should see structured recall output from the **shared** memory pool. **[memory-doctrine.md](memory-doctrine.md)** is the authority on what memory is and is not.

---

## 4. Connect via MCP (Cursor, Claude Desktop, others)

**Minimal mental model:** Pluribus speaks **HTTP** on port **8123**. Your agent app needs **one** MCP entry pointing at **`http://127.0.0.1:8123/v1/mcp`** (same machine) plus **rules or pasted instructions** so the model actually **calls** `recall_context` / `record_experience` — tools alone are not enough. Step-by-step clients: **[mcp-usage.md](mcp-usage.md#configure-pluribus-in-your-client)**. **Cursor** users: copy **[`integrations/cursor/mcp-config.json`](../integrations/cursor/mcp-config.json)** into **`~/.cursor/mcp.json`** (merge), restart Cursor, then add **[`integrations/pluribus-instructions.md`](../integrations/pluribus-instructions.md)** to **User rules** — **[`integrations/cursor/README.md`](../integrations/cursor/README.md)**.

Canonical MCP surface: **`POST /v1/mcp`** — [mcp-service-first.md](mcp-service-first.md). **Do not** treat the stdio `pluribus-mcp` binary as the default — it is **compat-only**; see [mcp-migration-stdio-to-http.md](mcp-migration-stdio-to-http.md).

---

## 5. Run evaluation quickly

```bash
make eval
make stress-eval
```

Artifacts appear in `artifacts/`.

**Canonical REST proof (memory substrate):** Postgres **+ pgvector**, clean DB — `cd control-plane && TEST_PG_DSN='postgres://…' make proof-rest`. This is the primary “does the service behave as claimed?” path; details in [evaluation.md](evaluation.md) and [evidence/memory-proof.md](../evidence/memory-proof.md).

**Episodic lane proof** (advisory ingest → similar → distill → review → materialize → recall/enforcement): same DSN — `make proof-episodic` from repo root or `cd control-plane && make proof-episodic`. Inventory: [evidence/episodic-proof.md](../evidence/episodic-proof.md).

---

## 6. Where to go next

| Need | Doc |
|------|-----|
| **Canonical memory model** | [memory-doctrine.md](memory-doctrine.md), [anti-regression.md](anti-regression.md) |
| Architecture (one story) | [architecture.md](architecture.md), [pluribus-public-architecture.md](pluribus-public-architecture.md) |
| MCP usage | [mcp-usage.md](mcp-usage.md) |
| Operations (config, auth, CI) | [pluribus-operational-guide.md](pluribus-operational-guide.md), [INSTALL.md](../INSTALL.md), [deployment-poc.md](deployment-poc.md) |
| Proof receipts | [pluribus-proof-index.md](pluribus-proof-index.md) |
| Release scope / what’s deferred | [pluribus-release-scope.md](pluribus-release-scope.md) |
| More API examples | [control-plane/README.md](../control-plane/README.md) |

---

## 7. Automated regression (maintainers)

```bash
make regression
```

Runs integration tests (including YAML **proof scenarios**) against ephemeral Postgres — same batch gate as CI. For the **adversarial REST invariant harness** (`proof-*.json`, determinism), use **`make proof-rest`** as in [evaluation.md](evaluation.md) — that is the clearest pre-public **substrate** receipt. For **episodic** coverage on top (sprint scenarios + full JSON suite), use **`make proof-episodic`** — [evidence/episodic-proof.md](../evidence/episodic-proof.md).
