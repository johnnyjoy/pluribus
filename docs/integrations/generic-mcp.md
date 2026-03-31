# Generic MCP client

Any client that speaks **MCP over HTTP** (JSON-RPC `POST` to a single URL) can use Pluribus as the **default memory system**.

## Endpoint

- **URL:** `http://<host>:<port>/v1/mcp` (dev default: `http://127.0.0.1:8123/v1/mcp`)
- **Method:** `POST`
- **Headers:** `Content-Type: application/json`; if the server uses **`PLURIBUS_API_KEY`**, add **`X-API-Key: <key>`** (some clients allow only `Authorization`—use what your client supports; Pluribus expects **`X-API-Key`** on the API)

## Auth

| Server | Client |
|--------|--------|
| No `PLURIBUS_API_KEY` | Omit key headers |
| `PLURIBUS_API_KEY` set | Send same value as **`X-API-Key`** |

On **`POST /v1/mcp` only**, query **`?token=`** may work as an alternative—see [authentication.md](../authentication.md).

## Default tools (automation-first)

- **`recall_context`** — recall before work (minimal args: task text); alias **`memory_context_resolve`**.
- **`record_experience`** — episodic capture (ingest channel `source=mcp`; not canonical until promotion); alias **`mcp_episode_ingest`**.
- Optional: **`memory_log_if_relevant`** / **`auto_log_episode_if_relevant`** for deterministic auto-ingest signals.

Layer-2 tools (curation, contradictions, etc.) are **optional**—see [mcp-poc-contract.md](../mcp-poc-contract.md).

## JSON-RPC surface

Same as other MCP servers: `initialize`, `tools/list`, `tools/call`, optional `prompts/list` / `resources/list` on the **HTTP** adapter (stdio binary may omit some surfaces).

## Behavioral pack

**[integrations/pluribus-instructions.md](../../integrations/pluribus-instructions.md)** (behavior), **[integrations/generic-mcp/skill.md](../../integrations/generic-mcp/skill.md)**, **[integrations/generic-mcp/skills/pluribus/SKILL.md](../../integrations/generic-mcp/skills/pluribus/SKILL.md)** (Agent Skills format), **[integrations/generic-mcp/README.md](../../integrations/generic-mcp/README.md)**.

## Examples

See **[integrations/generic-mcp/examples.json](../../integrations/generic-mcp/examples.json)** for minimal JSON-RPC shapes (illustrative only).
