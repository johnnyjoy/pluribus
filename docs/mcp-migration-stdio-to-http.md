# Migrating from stdio `pluribus-mcp` to HTTP MCP

## Summary

| Aspect | stdio (`cmd/pluribus-mcp`) | HTTP (control-plane) |
|--------|---------------------------|----------------------|
| Entry | Host runs binary, speaks MCP on stdin/stdout | **`POST /v1/mcp`** on the API base URL |
| Auth | Env **`CONTROL_PLANE_API_KEY`** → forwarded as **`X-API-Key`** | Send **`X-API-Key`** on the MCP request |
| Tools | `tools/list`, `tools/call` | Same methods |
| Extra | No prompts/resources on the binary | **`prompts/list`**, **`prompts/get`**, **`resources/list`**, **`resources/read`** on the service |

## Client configuration sketch

1. Base URL = control-plane API (e.g. `http://127.0.0.1:8123`).
2. MCP transport = HTTP to **`/v1/mcp`** (per your client’s MCP-over-HTTP support).
3. Pass the same API key you would use for `curl` to any **`/v1/*`** route.

## When to keep stdio

- Client only supports stdio MCP (e.g. some IDE integrations with no HTTP MCP).
- Air-gapped debugging where running the binary beside the IDE is simpler than pointing at a URL.

The stdio binary is **not** deprecated for removal; it is **non-canonical** for new setups that can use the service URL directly.

## Verification

```bash
curl -sS -X POST "$BASE/v1/mcp" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $KEY" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

Expect a JSON-RPC **`result`** with a **`tools`** array.
