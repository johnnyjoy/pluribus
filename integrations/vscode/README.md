# VS Code

Two integration paths:

1. **REST extension (in-repo)** — **`extension/`** — commands and an Explorer sidebar against the control-plane **HTTP API** (no MCP runtime inside the extension).
2. **MCP + Copilot-style instructions** — add Pluribus MCP per extension docs; **`mcp-config.example.json`** for shape; copy **`github-copilot-instructions.template.md`** → **`.github/copilot-instructions.md`**, then merge **`snippets/context-prime.txt`** ([Custom instructions](https://code.visualstudio.com/docs/copilot/customization/custom-instructions)); add **`skill.md`** if you want the step table.

**Canonical behavior:** [`pluribus-instructions.md`](../pluribus-instructions.md).

**`recall_context` → plan → act → `record_experience`** for substantive agent work (MCP path). The extension uses REST equivalents: **`POST /v1/recall/compile`**, **`POST /v1/advisory-episodes`**, **`GET /v1/curation/pending`**.

## Extension (`extension/`)

**Commands:** `Pluribus: Recall Context`, `Pluribus: Record Experience`, `Pluribus: View Learnings` (pending curation queue).

**UI:** Explorer → **Pluribus** view — shows last recall/record/pending snippets; full JSON in **Output → Pluribus**.

**Settings:**

| Key | Default | Purpose |
|-----|---------|---------|
| `pluribus.baseUrl` | `http://127.0.0.1:8123` | API base (no trailing slash) |
| `pluribus.apiKey` | *(empty)* | `X-API-Key` when the server uses `PLURIBUS_API_KEY` |

**Build / run from source:**

```bash
cd integrations/vscode/extension
npm install
npm run compile
```

In VS Code: **Run and Debug** → “Extension Development Host”, or **`F5`** with a launch config that loads this folder.

**Package (optional):** `npm install -g @vscode/vsce` then `vsce package` from `extension/` (produces a `.vsix`).

**Verify:** With `docker compose up`, run **Recall Context** — Output should show HTTP 200 and a recall bundle JSON. **Record Experience** should return 201 when similarity/advisory episodes are enabled. **View Learnings** calls **`GET /v1/curation/pending`**.
