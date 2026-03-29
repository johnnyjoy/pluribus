# Walkthrough: Single-agent continuity

Goal: show that **stored memory** restores context on resume and prevents repeating a known failure — without treating “project” as the product center.

## Setup

1. Start stack:

```bash
docker compose up -d
```

2. Use a **tag namespace** for this demo (shared memory pool; tags correlate the rows):

```bash
TAG_NS="single-agent-demo"
```

## Step 1: seed durable memory

```bash
curl -sS -X POST http://127.0.0.1:8123/v1/memory \
  -H 'Content-Type: application/json' \
  -d "{\"kind\":\"decision\",\"authority\":8,\"statement\":\"Use one canonical query path for listing endpoints.\",\"tags\":[\"${TAG_NS}\",\"api\"]}"

curl -sS -X POST http://127.0.0.1:8123/v1/memory \
  -H 'Content-Type: application/json' \
  -d "{\"kind\":\"failure\",\"authority\":9,\"statement\":\"Duplicated query builders caused inconsistent pagination and outages.\",\"tags\":[\"${TAG_NS}\",\"api\"]}"

curl -sS -X POST http://127.0.0.1:8123/v1/memory \
  -H 'Content-Type: application/json' \
  -d "{\"kind\":\"pattern\",\"authority\":7,\"statement\":\"Centralize list filtering and pagination in one shared function.\",\"tags\":[\"${TAG_NS}\",\"api\"]}"
```

## Step 2: simulate resume

```bash
curl -sS -X POST http://127.0.0.1:8123/v1/recall/compile \
  -H 'Content-Type: application/json' \
  -d "{\"tags\":[\"${TAG_NS}\",\"api\"],\"retrieval_query\":\"resume listing endpoint work\"}" | jq .
```

Expected:

- recall bundle includes previously saved decision/failure/pattern context,
- continuity is reconstructed **from memory**, not from chat logs.

## Step 3: confirm failure-avoidance signal

```bash
curl -sS -X POST http://127.0.0.1:8123/v1/enforcement/evaluate \
  -H 'Content-Type: application/json' \
  -d "{\"proposal_text\":\"Add a second custom SQL query path for list endpoints to move faster.\"}" | jq .
```

Expected:

- response recommends reject/revise path,
- known failure is surfaced as a reason to avoid duplicated query paths.

## What this proves

- memory persists across interruption,
- recall restores relevant continuity,
- known failures can shape next actions before execution.

Optional: for curation or evidence flows beyond this walkthrough, use the exact routes and bodies in [http-api-index.md](http-api-index.md) — **persistence scaffolding**, not the memory partition model ([memory-doctrine.md](memory-doctrine.md)).
