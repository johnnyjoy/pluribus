# Ask Cursor to verify Recall

**MCP path:** Cursor’s Agent uses **`pluribus-mcp`** (`tools/call`). That binary only forwards to the control-plane HTTP API—see [mcp-poc-contract.md](mcp-poc-contract.md).

**Cursor quirks:** `STATUS.md` / missing `tools/*.json` under `mcps/user-pluribus/` / `Not connected` — see [mcp-usage.md](mcp-usage.md) (stdio servers vs static descriptors; reconnect after disconnect).

**Same API without MCP:** If a tool (e.g. **`memory_create`**) does not appear in **`tools/list`**, rebuild the **`pluribus-mcp`** binary, point Cursor at that binary, and restart Cursor ([mcp-poc-contract.md § Stale tools](mcp-poc-contract.md)). Until then, **`curl` to the same URLs** is equivalent to what the adapter does—not a different protocol.

**Memory first:** `POST /v1/memory` accepts **`CreateRequest`** (`kind`, `authority`, `statement`, optional `tags`, `applicability`, …) without silo selectors on the body. Use tags (e.g. `cursor-verify`) to correlate this check. Other routes: valid JSON keys only — see [http-api-index.md](http-api-index.md) and `internal/*/types.go`.

## How to run

1. Control-plane up ([deployment-poc.md](deployment-poc.md)).
2. **`go build -o pluribus-mcp ./cmd/pluribus-mcp`** in `control-plane/`; wire Cursor MCP to that **file**.
3. New Agent chat → **`/ask-cursor-verify-recall`**

Command: [`.cursor/commands/ask-cursor-verify-recall.md`](../.cursor/commands/ask-cursor-verify-recall.md).

## What you should see

- Baseline **`recall_get`** (or **`recall_compile`**) with your tag set → **`memory_create`** → recall again → constraint visible in bundle
- Short answer that follows the stored rule

If recall stays empty after writing memory, check logs, Redis, and bundle cache (memory service).

**Tag filters:** Prefer passing the same **`tags`** on recall as you used on **`memory_create`** so the bundle clearly includes your test rows. Empty tag query means “no tag filter” on retrieval (see server ranking/RIU docs if you tune policies).

---

**Older name:** [cursor-recall-benefit-test.md](cursor-recall-benefit-test.md) redirects here.
