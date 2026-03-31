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
2. Load **[integrations/pluribus-instructions.md](../../integrations/pluribus-instructions.md)** via **[integrations/openclaw/policy.template.md](../../integrations/openclaw/policy.template.md)** + **`snippets/context-prime.txt`** into the agent’s pinned policy; merge **[integrations/openclaw/skill.md](../../integrations/openclaw/skill.md)** if you want the step table (**recall_context** → work → **record_experience**).
3. Enforce **`recall_context` → work → `record_experience`** every substantive run. (Aliases **`memory_context_resolve`** / **`mcp_episode_ingest`**.) Pull curation/contradictions only when debugging.

## Example stub

**[integrations/openclaw/mcp-config.example.json](../../integrations/openclaw/mcp-config.example.json)** holds **illustrative** env blocks—translate into **`openclaw mcp set`** arguments per OpenClaw’s current CLI.

## Limitations

- OpenClaw releases evolve quickly—re-verify MCP registration against **your** OpenClaw version’s docs.
