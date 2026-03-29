# Authentication

Pluribus technical preview uses an intentional low-friction auth model:

- **If no token is configured, auth is disabled.**
- **If a token is configured, auth is enforced.**

This is deliberate for localhost/LAN evaluation and operator test-drives.

## Behavior Summary

| Server configuration | Result |
|---|---|
| `PLURIBUS_API_KEY` unset or empty | No API auth required |
| `PLURIBUS_API_KEY` set (non-empty) | API auth required |

When auth is enabled:

- REST endpoints require `X-API-Key: <PLURIBUS_API_KEY>`.
- MCP over HTTP (`POST /v1/mcp`) accepts `X-API-Key` and supports `?token=` for MCP compatibility.

Health/readiness endpoints remain available for operational checks.

## Local and LAN Use

For quick local/LAN test-drive:

- leave `PLURIBUS_API_KEY` unset,
- run compose,
- call APIs directly.

For shared/LAN environments where you need protection:

- set `PLURIBUS_API_KEY`,
- pass `X-API-Key` from clients and MCP adapters.

## REST Example (auth enabled)

```bash
curl -sS -X POST http://127.0.0.1:8123/v1/memories \
  -H 'Content-Type: application/json' \
  -H "X-API-Key: ${PLURIBUS_API_KEY}" \
  -d '{"kind":"constraint","authority":9,"statement":"Auth-enabled smoke.","tags":["auth-demo"]}'
```

## MCP Example (auth enabled)

`pluribus-mcp` compat adapter:

```bash
export CONTROL_PLANE_URL=http://127.0.0.1:8123
export CONTROL_PLANE_API_KEY="${PLURIBUS_API_KEY}"
./pluribus-mcp
```

MCP over HTTP direct calls should include the same API key header (or MCP token query compatibility path).

## Where auth is defined

- Contract-level behavior: [api-contract.md](api-contract.md) (subset) + full route map [http-api-index.md](http-api-index.md)
- Operational notes: `docs/pluribus-operational-guide.md`
- Build/run notes: `BUILD.md`
