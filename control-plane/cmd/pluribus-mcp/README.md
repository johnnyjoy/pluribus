# pluribus-mcp

**Compatibility-only** thin **stdio MCP** adapter: forwards **`tools/call`** to control-plane HTTP. **Prefer MCP on the service:** **`POST /v1/mcp`** on the API — see **[mcp-service-first.md](../../docs/mcp-service-first.md)** and **[mcp-migration-stdio-to-http.md](../../docs/mcp-migration-stdio-to-http.md)**.

Shared tool definitions and HTTP mapping: **`internal/mcp`** (same as HTTP MCP).

## Run

```bash
cd control-plane
export CONTROL_PLANE_URL=http://127.0.0.1:8123   # or your server
export CONTROL_PLANE_API_KEY=...                  # set to your PLURIBUS_API_KEY value when server auth is enabled
go run ./cmd/pluribus-mcp
```

## Tools

Same tool list as **[mcp-poc-contract.md](../../docs/mcp-poc-contract.md)** and **`internal/mcp/tools.go`**: `health`, `recall_compile`, `recall_get`, `recall_run_multi`, `memory_create`, `memory_promote`, `curation_digest`, `curation_materialize`, `enforcement_evaluate`. **Project** routes are **HTTP only** (not separate MCP tools in this surface).

**Note:** This binary exposes **tools only** (no `prompts/*` or `resources/*`); use HTTP MCP for those.

## Build

```bash
go build -o pluribus-mcp ./cmd/pluribus-mcp
```

## Release builds (GitHub Releases)

CI publishes `pluribus-mcp` binaries as `tar.gz` assets on Git tags `v*`:

- `pluribus-mcp-linux-<arch>.tar.gz` (extracts to `pluribus-mcp`)
- `SHA256SUMS.txt` (checksums for the tarballs)

After download, extract the tarball, then run the binary with:

- `CONTROL_PLANE_URL` (API base URL, e.g. `http://127.0.0.1:8123`)
- `CONTROL_PLANE_API_KEY` only if the server has **`PLURIBUS_API_KEY`** enabled
