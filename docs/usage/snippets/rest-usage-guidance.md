# REST usage — when MCP is unavailable or for automation

Use the **same base URL** as MCP (default **`http://127.0.0.1:8123`**). Send **`Content-Type: application/json`**. If **`PLURIBUS_API_KEY`** is set on the server, add **`X-API-Key: <secret>`** to every request except **`GET /healthz`**.

**Full route map:** [http-api-index.md](../../http-api-index.md). **Detail examples:** [api-contract.md](../../api-contract.md).

---

## Compile recall (grounding)

```bash
curl -sS -X POST http://127.0.0.1:8123/v1/recall/compile \
  -H 'Content-Type: application/json' \
  -d '{"tags":["myteam","service"],"retrieval_query":"deploy and migration constraints"}'
```

Adjust **`tags`** and **`retrieval_query`** to your situation. Response shape: `internal/recall/types.go` (`RecallBundle`).

---

## Create an advisory episode (experience log)

```bash
curl -sS -X POST http://127.0.0.1:8123/v1/advisory-episodes \
  -H 'Content-Type: application/json' \
  -d '{"summary":"Incident: retry storm on checkout; fixed by rate limit; decision: cap burst per IP.","source":"manual","tags":["checkout","incident"]}'
```

**`summary`** is required. **`source`** is the **ingest channel** (e.g. `manual`, `mcp`). Episodes are **advisory** until distilled and promoted.

**Note:** Some deployments return **403** if episodic/similarity features are disabled—operational policy, not a client bug.

---

## Distill to candidates (explicit)

Requires **`distillation.enabled`** in server config. Typically **`episode_id`** from the create response:

```bash
curl -sS -X POST http://127.0.0.1:8123/v1/episodes/distill \
  -H 'Content-Type: application/json' \
  -d '{"episode_id":"<uuid-from-advisory-episodes-response>"}'
```

Alternately, some builds accept a **`summary`** payload for distillation—see [http-api-index.md](../../http-api-index.md) and `episode_distill_explicit` in [mcp-poc-contract.md](../../mcp-poc-contract.md).

---

## Durable memory (direct write)

For explicit canonical rows (decisions, constraints, patterns, failures), use **`POST /v1/memory`** or **`POST /v1/memories`**—see [api-contract.md](../../api-contract.md). Prefer the **curation** path for learning from narrative.

---

## Scripts and CI

REST is the right surface for **cron**, **webhooks**, **load tests**, and **proof targets** (`make proof-rest`, `make proof-episodic`)—see [evaluation.md](../../evaluation.md).
