ARCHIVED — NOT ACTIVE SYSTEM TRUTH

---

# POC end-to-end walkthrough (~30 min)

**Prefer a single checklist with PASS/FAIL criteria?** Use **[poc-manual-proof-run.md](poc-manual-proof-run.md)**.

**Prerequisite:** Stack up + **database initialized** (schema applied) per [deployment-poc.md](deployment-poc.md) §3.

**Base URL:** `export BASE=http://127.0.0.1:8123` (or your host IP).

---

## 1. Health

```bash
curl -sS "$BASE/healthz"
```

Expect: `ok`

---

## 2. Tag namespace

```bash
export POC_TAG=poc-demo
```

---

## 3. Recall (compile) — structured bundle

```bash
curl -sS -X POST "$BASE/v1/recall/compile" \
  -H 'Content-Type: application/json' \
  -d "{\"tags\":[\"$POC_TAG\"],\"max_per_kind\":5,\"retrieval_query\":\"POC walkthrough\"}" | jq .
```

Empty buckets are normal until you add memory.

---

## 4. Recall (GET) — same semantics, query string

```bash
curl -sS "$BASE/v1/recall/?tags=$POC_TAG&max_per_kind=5&query=POC+walkthrough" | jq .
```

---

## 5. Run-multi (bounded path)

```bash
curl -sS -X POST "$BASE/v1/recall/run-multi" \
  -H 'Content-Type: application/json' \
  -d "{\"query\":\"POC smoke query\",\"tags\":[\"$POC_TAG\"],\"merge\":false,\"promote\":false}" | jq .
```

**Caveat:** Full server-side run-multi LLM behavior requires **`synthesis.enabled: true`** and a valid **`synthesis`** provider in control-plane config (see [../control-plane/docs/backend-synthesis.md](../control-plane/docs/backend-synthesis.md)). With defaults (`synthesis.enabled: false`), **`POST /v1/recall/run-multi`** reports that server-side run-multi synthesis is not configured — use the client LLM or enable synthesis explicitly.

---

## 6. Optional — promote

Only if your promotion gates allow it (see `promotion` in config):

```bash
curl -sS -X POST "$BASE/v1/memory/promote" \
  -H 'Content-Type: application/json' \
  -d "{\"type\":\"decision\",\"content\":\"POC promoted decision\",\"tags\":[\"$POC_TAG\"]}" | jq .
```

---

## 7. Memory effect on later recall

After creating memory (via **`POST /v1/memory`** or successful promote), repeat step **3** — bundles should include new items when tags/kinds match.

---

## 8. MCP adapter smoke

From `control-plane/`:

```bash
export CONTROL_PLANE_URL="$BASE"
go run ./cmd/pluribus-mcp &
# In another terminal, use your MCP client, or send a single JSON-RPC line (advanced).
```

See [mcp-poc-contract.md](mcp-poc-contract.md).

---

## Scripted variant

```bash
./scripts/poc-e2e.sh
```

(requires `curl`, `jq`; uses same steps when `BASE` unset defaults to `http://127.0.0.1:8123`)
