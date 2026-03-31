# OpenCode

[OpenCode](https://dev.opencode.ai/) is an open-source terminal (and desktop / extension) AI coding agent. It supports **MCP** natively via **`mcp`** in **`opencode.json`** (project root) or **`~/.config/opencode/opencode.json`**. Pluribus fits as **remote** HTTP MCP (preferred, service-first) or **local** stdio via **`pluribus-mcp`**.

Official references: [MCP servers](https://dev.opencode.ai/docs/mcp-servers), [Config](https://dev.opencode.ai/docs/config), [Rules / AGENTS.md](https://dev.opencode.ai/docs/rules).

## MCP (preferred — remote HTTP)

Merge the **`mcp.pluribus`** block from **[integrations/opencode/mcp-config.example.json](../../integrations/opencode/mcp-config.example.json)** into your OpenCode config. Adjust **`url`** for non-local Pluribus.

- Set **`oauth": false`** so OpenCode does not treat Pluribus as an OAuth MCP server (Pluribus uses **`X-API-Key`** when auth is enabled).
- When the control plane has **`PLURIBUS_API_KEY`** set, add **`headers`** with **`X-API-Key`** — use **`{env:PLURIBUS_API_KEY}`** per [OpenCode variable substitution](https://dev.opencode.ai/docs/config#env-vars). If the server has **no** API key, **omit** **`headers`** entirely (do not send an empty key).

OpenCode names tools with an MCP prefix (e.g. **`pluribus_*`**). You can nudge the model with **`use pluribus`** or document defaults in **AGENTS.md** (see below).

**Context caveat:** OpenCode warns that many MCP tools increase token use—keep Pluribus enabled and disable heavier MCPs you do not need. You can also scope MCP tools per [agent](https://dev.opencode.ai/docs/mcp-servers#per-agent).

## MCP (fallback — local stdio)

Use **`type": "local"`** with a **`command`** array pointing at your **`pluribus-mcp`** binary and **`environment`** for **`CONTROL_PLANE_URL`** (and **`CONTROL_PLANE_API_KEY`** when the server uses **`PLURIBUS_API_KEY`**). Build: `cd control-plane && go build -o pluribus-mcp ./cmd/pluribus-mcp`. Same behavior as other clients; see [mcp-usage.md](../mcp-usage.md).

## Rules + core skill

- **`instructions`** / **`AGENTS.md`** / **`.opencode/skills/`** — see [OpenCode rules](https://dev.opencode.ai/docs/rules).
- **Canonical:** **[integrations/pluribus-instructions.md](../../integrations/pluribus-instructions.md)**. Follow **[integrations/opencode/AGENTS.template.md](../../integrations/opencode/AGENTS.template.md)** for **`AGENTS.md`** (or merge the same text into **`opencode.json`** **`instructions`**). Optionally copy **[integrations/opencode/skills/pluribus/](../../integrations/opencode/skills/pluribus/)** to **`.opencode/skills/pluribus/`** or paste **[integrations/opencode/skill.md](../../integrations/opencode/skill.md)** into **`AGENTS.md`**. Pack: **[integrations/opencode/README.md](../../integrations/opencode/README.md)**.

## Verification

After editing config, restart OpenCode. Use **`opencode mcp list`** to confirm the server is registered (see [OpenCode MCP docs](https://dev.opencode.ai/docs/mcp-servers#manage)). This repo does not run automated tests against OpenCode releases—re-check after major OpenCode upgrades.

## Limitations

- **Remote URL** must reach a live Pluribus control plane (**`POST /v1/mcp`**).
- **Tool count** affects context; trim other MCP servers if you hit limits.
