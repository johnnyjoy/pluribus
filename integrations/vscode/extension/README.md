# Pluribus AI

**Editor-side helper for [Pluribus](https://github.com/johnnyjoy/pluribus)** — the governed memory control plane for AI agents.

**Extension ID:** `pluribus.pluribus-ai`. If you previously installed **`pluribus.pluribus-memory`**, uninstall the old extension before installing this one — the identifier changed.

This extension connects **VS Code** and **Cursor** to your running Pluribus API over **REST**. It does **not** replace Pluribus or implement memory policy: ranking, authority, and ingest rules stay on the **server**.

## What you get

- **Health** — periodic checks against `GET /healthz`, status bar + sidebar.
- **Orchestration** — optional automatic **recall** before tasks / debug sessions and **advisory record** after task failures (configurable).
- **Manual actions** — recall, record, browse pending curation queue.
- **Metrics** — local counters in Output (no telemetry).

Full settings, events, and install-from-source steps: **[`../README.md`](../README.md)** (parent folder in this repo).

## Requirements

- A reachable Pluribus control plane (e.g. **Docker Compose** from the repo, or your own deployment).
- Set **`pluribus.baseUrl`** (and **`apiKey`** if the server uses `PLURIBUS_API_KEY`).

For **Chat / MCP tools** (`recall_context`, `record_experience`, …), configure your editor’s MCP separately — see **`integrations/cursor/`** in the Pluribus repo.

## License

Same terms as the [Pluribus repository](https://github.com/johnnyjoy/pluribus) (see root `LICENSE.md`).
