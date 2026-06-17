# Test Isolation

Phase 11E proves that Pluribus tests and proof targets exercise the **current repository checkout**, not Cursor’s configured production MCP server, not a production Pluribus URL, and not a globally installed old `pluribus-mcp` binary.

## What is the Pluribus under test?

The **control-plane Go module in this git checkout** (`/projects/pluribus/control-plane`). Validation uses:

- Direct Go package calls (`formation.Gate`, `formationquality.Evaluator`, handlers)
- `httptest` servers wired to checkout routers/handlers
- In-process MCP handlers (`internal/mcp.NewHTTPHandler`)
- Optional `go run ./cmd/pluribus-mcp` or `go build -o $BIN ./cmd/pluribus-mcp` with **`cmd.Dir` set to the checkout** and `CONTROL_PLANE_URL` pointing at a **local httptest or docker-compose** server

## How unit tests call current checkout code

Go tests import `control-plane/internal/...` packages compiled from the working tree. No external Pluribus service is required for formation, recall benchmark, or isolation gates.

## How REST tests avoid production URLs

REST integration tests use `httptest.NewServer` with routers built from the checkout. URLs are `http://127.0.0.1:<ephemeral>` — never production hosts.

## How MCP tests avoid Cursor’s configured MCP server

MCP tests call `internal/mcp` HTTP handlers directly or spawn `go run ./cmd/pluribus-mcp` with `CONTROL_PLANE_URL` set to the **local httptest server URL** from the same test (`mcp_memory_formation_integration_test.go`). Tests do **not** read `.cursor/mcp.json` or connect to editor-configured MCP servers.

## Binary invocation rules

| Pattern | Allowed? |
|---------|----------|
| `go run ./cmd/pluribus-mcp` with `cmd.Dir` = checkout | Yes |
| `go build -o /tmp/... ./cmd/pluribus-mcp` in proof scripts | Yes |
| `exec.Command("pluribus-mcp")` (PATH lookup) | **No** |
| `exec.LookPath("pluribus-mcp")` | **No** |

## Automated guard

```bash
make test-codebase-isolation
make proof-codebase-isolation
```

Package: `control-plane/internal/testisolation/`

Artifact: `artifacts/codebase-test-isolation.json`

Hard thresholds (all must be **0**):

- `production_mcp_dependency_count`
- `cursor_mcp_config_dependency_count`
- `global_binary_dependency_count`
- `unqualified_path_binary_count`
- `external_endpoint_dependency_count`

## Allowlisted patterns

Documented in `testisolation.DefaultAllowlist()`:

- `go run ./cmd/pluribus-mcp` — builds from checkout
- `localhost` / `127.0.0.1` — local test endpoints
- `CONTROL_PLANE_URL` bound to `srv.URL` from httptest

## What this does not claim

- Future tests added without review are automatically safe
- Docker proof scripts never touch external networks (they use local compose)
- Production agent behavior is proven

## See also

- [formation-escape-hatches.md](formation-escape-hatches.md)
- [memory-formation-quality.md](memory-formation-quality.md)
