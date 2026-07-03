# Generic MCP

1. **MCP:** `POST /v1/mcp` per root **README** / **`examples.json`**. Server advertises **55 tools** (regenerate doc: `UPDATE_MCP_TOOLS_DOC=1 go test ./internal/mcp/ -run TestMCPToolsDocMatchesRegistry` from `control-plane/`).
2. Paste **[`pluribus-instructions.md`](../pluribus-instructions.md)** + **`snippets/context-prime.txt`** into system / developer instructions.
3. **Agent Skills (optional):** copy **`skills/pluribus/`** if the client supports Agent Skills.

**Core loop:** **`recall_context`** or **`wakeup_context`** → plan → act → **`record_experience`**.

**Phase 11I/11K (optional, server-side):** `agent_telemetry_*` and `agent_utility_*` tools are available on the same MCP surface; examples in **`examples.json`**. Plugins do not auto-run telemetry — agents or hooks must call these tools.

**Enforcement tier:** **1–2** (rules + MCP availability). **Not mandatory** unless your client enforces tool use.

**Verify:**

```bash
scripts/integrations/verify-generic-mcp.sh --static
PLURIBUS_BASE_URL=http://127.0.0.1:8123 scripts/integrations/verify-generic-mcp.sh
```
