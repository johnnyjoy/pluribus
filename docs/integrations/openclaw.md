# OpenClaw

OpenClaw is an **MCP-capable** agent/gateway ecosystem. Treat Pluribus as a **standard MCP server**: register the control-plane **HTTP MCP** endpoint or the **`pluribus-mcp`** stdio adapter the same way you register other MCP backends.

## Official mechanics

- **CLI:** OpenClaw documents MCP under **`openclaw mcp`** — list, show, set, unset. See **[OpenClaw MCP CLI](https://docs.openclaw.ai/cli/mcp)** (verify subcommands for your version).
- **Gateway / config:** Self-hosted setups use project or user config; **hot-reload** behavior depends on version—follow current OpenClaw docs.

**Do not** invent a fake `openclaw.json` schema here. Use **`openclaw mcp set`** (or the documented equivalent) to point at:

- **HTTP:** `http://<host>:8123/v1/mcp` + **`X-API-Key`** when auth is on, or  
- **Stdio:** path to **`pluribus-mcp`** + `CONTROL_PLANE_URL`.

## First-class usage pattern

1. Register Pluribus MCP (HTTP preferred if your gateway supports URL + headers).
2. Encode **aggressive defaults**: recall first, ingest after meaningful work—see **[integrations/openclaw/rules.md](../../integrations/openclaw/rules.md)** and **[integrations/openclaw/skills.md](../../integrations/openclaw/skills.md)**.
3. Use **`recall_context`** + **`record_experience`** as the primary pair (aliases **`memory_context_resolve`** / **`mcp_episode_ingest`**); layer-2 tools optional.

## Example stub

**[integrations/openclaw/mcp-config.example.json](../../integrations/openclaw/mcp-config.example.json)** holds **illustrative** env blocks—translate into **`openclaw mcp set`** arguments per OpenClaw’s current CLI.

## Limitations

- OpenClaw releases evolve quickly—re-verify MCP registration against **your** OpenClaw version’s docs.
