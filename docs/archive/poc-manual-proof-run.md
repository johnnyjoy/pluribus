ARCHIVED — NOT ACTIVE SYSTEM TRUTH

---

# Manual POC proof run — “prove it to yourself”

**Audience:** You, on a machine where you can run Docker and `curl`.  
**Time:** ~15–25 minutes first time (includes compose + **database initialization** — empty Postgres gets the API schema).  
**Stage:** Yes — the repo is at **POC stage**: stack + HTTP authority + thin MCP adapter are implemented; this guide is the **repeatable manual acceptance** path.

**Deeper detail:** [deployment-poc.md](deployment-poc.md) (bring-up), [poc-e2e-walkthrough.md](poc-e2e-walkthrough.md) (same steps, alternate layout), [mcp-poc-contract.md](mcp-poc-contract.md) (MCP ↔ HTTP).

---

## What “passing” means

You have **personally verified** that:

| # | Claim | How you know |
|---|--------|----------------|
| A | Stack runs | `healthz` returns `ok` |
| B | DB is initialized | **`/readyz`** is `ok` (core tables + migrations current) |
| C | Recall is server-side | JSON bundles return from `/v1/recall/*` |
| D | Run-multi path exists | HTTP 200 + JSON body from `/v1/recall/run-multi` (see caveats below) |
| E | Memory can be created and show up in recall | `POST /v1/memory` then compile shows your statement |
| F | (Optional) MCP is thin | `pluribus-mcp` proxies to same URLs — no second brain |

---

## 0. Prerequisites

- Docker + Compose v2, `curl`, `jq` (migrations run **inside control-plane on startup** — no host `psql` required).
- Ports **8123**, **5432**, **6379** free (or adjust compose / URLs).

---

## 1. Bring up the stack + initialize the database

Postgres starts **empty**; **control-plane** runs embedded baseline SQL on boot (no `schema_migrations` table).

Follow **[deployment-poc.md](deployment-poc.md)** through:

1. `docker compose up -d`
2. `curl -sS http://127.0.0.1:8123/healthz` → **`ok`**
3. `curl -sS http://127.0.0.1:8123/readyz` → **`ok`** (confirms DB + core schema)

**PASS:** `healthz` and `readyz` are `ok` (check `docker compose logs controlplane` if migrations fail).

**FAIL:** Fix compose / Postgres first — nothing else will prove the product.

---

## 2. Set base URL

```bash
export BASE=http://127.0.0.1:8123
```

(On another host, use `http://SERVER_IP:8123` and ensure the firewall allows it.)

---

## 3. Health (sanity)

```bash
curl -sS "$BASE/healthz"
```

**PASS:** prints `ok`.

---

## 4. Tag namespace (no workspace HTTP in current router)

```bash
export POC_TAG=poc-proof
```

**PASS:** Variable is non-empty.

---

## 5. Recall compile (empty DB is OK)

```bash
curl -sS -X POST "$BASE/v1/recall/compile" \
  -H 'Content-Type: application/json' \
  -d "{\"tags\":[\"$POC_TAG\"],\"max_per_kind\":5,\"retrieval_query\":\"POC proof run\"}" | jq .
```

**PASS:** HTTP 200 and JSON with bundle structure (buckets may be empty).

---

## 6. GET recall (same semantics, query string)

```bash
curl -sS "$BASE/v1/recall/?tags=$POC_TAG&max_per_kind=5&query=POC+proof+run" | jq .
```

**PASS:** HTTP 200, JSON analogous to step 5.

---

## 7. Run-multi (bounded pipeline)

```bash
curl -sS -X POST "$BASE/v1/recall/run-multi" \
  -H 'Content-Type: application/json' \
  -d "{\"query\":\"POC manual proof\",\"tags\":[\"$POC_TAG\"],\"merge\":false,\"promote\":false}" | jq .
```

**PASS:** HTTP 200 and a JSON body you can inspect.

**Important caveat:** A **full** server-side LLM-backed run-multi needs **`synthesis.enabled: true`** and a valid provider in control-plane config (see [../control-plane/docs/backend-synthesis.md](../control-plane/docs/backend-synthesis.md)). Default config leaves synthesis **disabled** — **`POST /v1/recall/run-multi`** will indicate run-multi is not configured for backend synthesis; that is **expected** until you opt in. Prefer client-side synthesis unless you explicitly enable backend synthesis.

---

## 8. Create memory, then prove recall sees it

Insert a durable memory tagged **`poc`**:

```bash
curl -sS -X POST "$BASE/v1/memory" \
  -H 'Content-Type: application/json' \
  -d "{
    \"kind\": \"decision\",
    \"authority\": 5,
    \"statement\": \"POC manual proof: server-authoritative memory is visible to recall.\",
    \"tags\": [\"$POC_TAG\"]
  }" | tee /tmp/poc-memory.json | jq .
```

**PASS:** HTTP 200 (or 201) and JSON including an `id` for the memory.

Re-run compile and confirm the statement appears in the bundle (field names vary by bucket; search the output for your sentence or use `jq`):

```bash
curl -sS -X POST "$BASE/v1/recall/compile" \
  -H 'Content-Type: application/json' \
  -d "{\"tags\":[\"$POC_TAG\"],\"max_per_kind\":5,\"retrieval_query\":\"POC manual proof\"}" | jq . | tee /tmp/poc-recall-after.json
grep -q 'POC manual proof' /tmp/poc-recall-after.json && echo 'PASS: statement visible in recall output' || echo 'CHECK: inspect /tmp/poc-recall-after.json — tagging/kind may affect bucket'
```

**PASS:** Your statement text appears in the JSON, or you can see the memory id/content in the appropriate bucket.

---

## 9. Optional — promote

Only if promotion policy allows (see `promotion` in config):

```bash
curl -sS -X POST "$BASE/v1/memory/promote" \
  -H 'Content-Type: application/json' \
  -d "{\"type\":\"decision\",\"content\":\"POC promoted line\",\"tags\":[\"$POC_TAG\"]}" | jq .
```

**PASS:** JSON shows `"promoted": true` (or a clear `reason` if gated).

---

## 10. Optional — MCP adapter smoke

From repo `control-plane/`:

```bash
export CONTROL_PLANE_URL="$BASE"
go run ./cmd/pluribus-mcp
```

Use your MCP client against stdio, or follow **[mcp-poc-contract.md](mcp-poc-contract.md)** for tool names and shapes.

**PASS:** Tools call the **same** HTTP API; adapter does not invent rankings.

---

## 11. Scripted shortcut

With the stack already up and the DB initialized:

```bash
./scripts/poc-e2e
```

**PASS:** Script exits 0 and prints project id at the end.

Default `BASE` is `http://127.0.0.1:8123`. Override:

```bash
BASE=http://your-host:8123 ./scripts/poc-e2e
```

---

## If something fails

| Symptom | See |
|--------|-----|
| `connection refused` on 8123 | [deployment-poc.md §7](deployment-poc.md) — compose / logs |
| 500 on API after health works | Migrations not applied — [deployment-poc.md §3](deployment-poc.md) |
| run-multi not configured / error | Backend synthesis off by default — enable `synthesis` in config or use client-side synthesis |
| Memory created but not in recall | Tags / `max_per_kind` / kind buckets — inspect JSON; adjust tags to match |

---

## Done?

If steps **1–8** pass, you have **manual proof** of: **deploy → API → recall → run-multi path → create memory → recall reflects memory**. That is the POC bar this repo was built for.

Optional next hardening (not required to call the POC “real”): run the same checklist on a **non-laptop** server, add CI for `compose up` + `poc-e2e` (DB schema applies on control-plane boot), publish a Docker image for `pluribus-mcp`.
