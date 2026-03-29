# Walkthrough: Constraint enforcement

Goal: show that a known bad path is blocked or revised **before** execution, using **binding memory** and enforcement.

## Setup

```bash
docker compose up -d
TAG_NS="constraint-demo"
```

## Seed a hard constraint

```bash
curl -sS -X POST http://127.0.0.1:8123/v1/memory \
  -H 'Content-Type: application/json' \
  -d "{\"kind\":\"constraint\",\"authority\":10,\"statement\":\"Never deploy schema migrations without first running validation and rollback checks.\",\"tags\":[\"${TAG_NS}\",\"ops\"]}"
```

## Submit conflicting proposal

```bash
curl -sS -X POST http://127.0.0.1:8123/v1/enforcement/evaluate \
  -H 'Content-Type: application/json' \
  -d "{\"proposal_text\":\"Deploy migration directly to production now without validation to save time.\"}" | jq .
```

Expected:

- decision is not plain `allow`,
- `validation.next_action` indicates `reject` or `revise`,
- triggered memory includes the governing constraint.

## What this proves

- constraints are active control signals, not passive notes,
- unsafe proposals can be gated before execution,
- recall + enforcement combine to reduce repeated operational mistakes.
