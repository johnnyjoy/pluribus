# Continue

[Continue](https://continue.dev) supports **MCP servers** in its configuration. Point Continue at Pluribus’s **HTTP MCP** URL so the agent uses the same tools as other editors.

## Config

Continue typically uses a YAML or JSON config (path varies by version—**`~/.continue/config.json`** or project config). Add an MCP server entry whose **URL** is:

`http://127.0.0.1:8123/v1/mcp`

…with **headers** for **`X-API-Key`** when your Pluribus server uses **`PLURIBUS_API_KEY`**.

**Example stub:** **[integrations/continue/mcp-config.example.json](../../integrations/continue/mcp-config.example.json)** — **merge** keys into Continue’s actual schema per [Continue MCP docs](https://docs.continue.dev/customize/deep-dives/mcp) (verify URL).

## Rules + skill

**Canonical:** **[integrations/pluribus-instructions.md](../../integrations/pluribus-instructions.md)**. Copy **[integrations/continue/rules/pluribus.md](../../integrations/continue/rules/pluribus.md)** to **`.continue/rules/pluribus.md`** ([Continue rules](https://docs.continue.dev/customize/deep-dives/rules)); add **`snippets/context-prime.txt`** as another rule or YAML entry. Use **[integrations/continue/skill.md](../../integrations/continue/skill.md)** for the compact step table. Pack: **[integrations/continue/README.md](../../integrations/continue/README.md)**.

## Limitations

- Continue’s MCP schema has changed across versions—**validate** the merged config against Continue’s documentation for your release.
