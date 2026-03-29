# Walkthrough: Multi-agent shared memory

Goal: show that memory captured by **Agent A** improves **Agent B** behavior later — **shared durable memory** in the global pool, not a per-session silo.

## Setup

```bash
docker compose up -d
TAG_NS="multi-agent-demo"
```

## Agent A: captures validated memory

```bash
curl -sS -X POST http://127.0.0.1:8123/v1/memory \
  -H 'Content-Type: application/json' \
  -d "{\"kind\":\"decision\",\"authority\":8,\"statement\":\"Use feature flags for staged rollout of API changes.\",\"tags\":[\"${TAG_NS}\",\"rollout\"]}"

curl -sS -X POST http://127.0.0.1:8123/v1/memory \
  -H 'Content-Type: application/json' \
  -d "{\"kind\":\"pattern\",\"authority\":8,\"statement\":\"Rollout order: canary -> 10% -> 50% -> 100% with rollback checkpoints.\",\"tags\":[\"${TAG_NS}\",\"rollout\"]}"
```

## Agent B: starts later and compiles recall

```bash
curl -sS -X POST http://127.0.0.1:8123/v1/recall/compile \
  -H 'Content-Type: application/json' \
  -d "{\"tags\":[\"${TAG_NS}\",\"rollout\"],\"retrieval_query\":\"prepare API rollout plan\"}" | jq .
```

Expected:

- Agent B receives Agent A’s decision/pattern context in bundle output.

## Agent B: validates risky alternative

```bash
curl -sS -X POST http://127.0.0.1:8123/v1/enforcement/evaluate \
  -H 'Content-Type: application/json' \
  -d "{\"proposal_text\":\"Ship rollout to 100% immediately without staging.\"}" | jq .
```

Expected:

- enforcement flags unsafe proposal against established pattern/decision.

## What this proves

- memory is shared through the control-plane, not tied to one agent session,
- later agents benefit from earlier validated experience,
- behavior improves through **durable governed memory** (shared pool + tags).
