# VS Code (extension + optional MCP)

## REST extension (orchestrator)

**Running Pluribus:** use **Docker Compose** from the repo (see root **`README.md`**) for a local API, or point **`pluribus.baseUrl`** at another host via **user/local** settings. **Do not commit** site-specific IPs or LAN addresses into the repository.

The in-repo **Pluribus AI** extension under **[integrations/vscode/extension/](../../integrations/vscode/extension/)** (v0.2+) is an **event-driven orchestrator**:

- **Health:** periodic `GET /healthz`, status bar + sidebar.
- **Auto recall:** `POST /v1/recall/compile` on **task start** and **debug start** (optional debounced **save**).
- **Auto record:** `POST /v1/advisory-episodes` on **task process non-zero exit** (`source: vscode-orchestrator`); optional **debug end** (default **off**).
- **Metrics:** local counters + Output logging — **no** network telemetry.
- **Policy:** **no** client-side memory ranking or promotion; all semantics remain on the control plane.

Details, settings, and `.vsix` packaging: **[integrations/vscode/README.md](../../integrations/vscode/README.md)**.

## MCP (optional)

VS Code’s MCP story is vendor-specific. Use **[integrations/vscode/mcp-config.example.json](../../integrations/vscode/mcp-config.example.json)** and your Copilot/MCP extension’s docs. Endpoint: `http://127.0.0.1:8123/v1/mcp` (or deployed host).

## Rules + skill

Copy **[integrations/vscode/github-copilot-instructions.template.md](../../integrations/vscode/github-copilot-instructions.template.md)** → **`.github/copilot-instructions.md`**; merge **`snippets/context-prime.txt`**. Skill table: **[integrations/vscode/skill.md](../../integrations/vscode/skill.md)**.

## Cursor

The same `.vsix` may install in **Cursor**; behavior should be verified manually (tasks/debug events). **VS Code** is the tested reference for this repository.

## Limitations

- Auto record on task failure requires **`onDidEndTaskProcess`** (tasks with a real process).
- **Save**-triggered recall is **off** by default (noise).
- **Hard enforcement** is intentionally out of scope — soft nudges and defaults only.
