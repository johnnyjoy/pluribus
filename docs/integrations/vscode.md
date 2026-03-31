# VS Code (MCP / Copilot-adjacent)

Visual Studio Code can use MCP through **extensions** or **workspace configuration**, depending on Microsoft’s current shipping model (GitHub Copilot agent mode, MCP extension, etc.). Pluribus does **not** ship a VSIX in this repo—integrate via **standard MCP client** configuration pointing at **`/v1/mcp`**.

## MCP URL

- **Endpoint:** `http://127.0.0.1:8123/v1/mcp` (or your deployed host)
- **Header:** `X-API-Key` when `PLURIBUS_API_KEY` is set on the server

**Example stub:** **[integrations/vscode/mcp-config.example.json](../../integrations/vscode/mcp-config.example.json)** — merge into whatever MCP config your VS Code setup expects (often a JSON file under `.vscode/` or user settings—**vendor-specific**).

## Rules + skill

**Canonical:** **[integrations/pluribus-instructions.md](../../integrations/pluribus-instructions.md)**. Copy **[integrations/vscode/github-copilot-instructions.template.md](../../integrations/vscode/github-copilot-instructions.template.md)** to **`.github/copilot-instructions.md`** ([VS Code custom instructions](https://code.visualstudio.com/docs/copilot/customization/custom-instructions)); merge **`snippets/context-prime.txt`**. Use **[integrations/vscode/skill.md](../../integrations/vscode/skill.md)** for the step table. Pack: **[integrations/vscode/README.md](../../integrations/vscode/README.md)**.

## Limitations

- VS Code MCP story is **moving quickly**—confirm the config file name and schema for your Copilot / MCP extension version.
