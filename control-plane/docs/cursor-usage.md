# Cursor usage pattern

Use Pluribus (control-plane) with Cursor so AI coding stays aligned with project memory and avoids drift.

**Canonical boundary:** Pluribus integrates with Cursor through **MCP and HTTP**, not as an editor language server. See [Pluribus — LSP vs MCP boundary](../../docs/pluribus-lsp-mcp-boundary.md).

---

## Mental model

**gopls** — code understanding in the editor (hover, definitions, references, diagnostics). Local, editor-owned.

**Pluribus** — continuity and governance (recall, drift, enforcement, curation). Reach it via **MCP** (`POST /v1/mcp` on your API URL) or **direct HTTP** (`/v1/...`).

They work **together**; neither replaces the other.

---

## Connect Cursor to Pluribus (MCP — the “one URL” that applies)

| Approach | What you configure |
|----------|-------------------|
| **Preferred** | MCP over HTTP: your Pluribus **base URL** and **`POST /v1/mcp`** — [mcp-service-first.md](../../docs/mcp-service-first.md). |
| **Compat** | Stdio **`pluribus-mcp`** with **`CONTROL_PLANE_URL`** — [cmd/pluribus-mcp/README.md](../cmd/pluribus-mcp/README.md). |

That wires the **agent** to tools (projects, recall, memory, curation, enforcement). It does **not** change how the Go extension talks to **gopls**.

**Troubleshooting Cursor MCP UI:** [mcp-usage.md](../../docs/mcp-usage.md).

---

## Optional: server-side LSP signals (not editor LSP)

If **`lsp.enabled: true`** on the server, Pluribus can call **gopls inside the server process** to enrich recall/drift when requests include fields like **`repo_root`** and **`lsp_focus_path`**. That is **not** something you point Cursor’s LSP settings at. Details: [lsp-features.md](lsp-features.md).

---

## Flow (recall → work → drift)

1. **Start with recall**, then gather only the external metadata your task explicitly requires.

2. **Compile recall bundle** for the task (tool or curl):
   ```bash
   curl -s -X POST http://localhost:8123/v1/recall/compile -H 'Content-Type: application/json' -d @recall-request.json | jq .
   ```
   Save the response (target, task, governing constraints, decisions, known failures, applicable patterns).

3. **Give Cursor** the task description, the recall bundle, and the relevant files. Ask for a proposal or implementation.

4. **Get the proposal** (e.g. diff, plan, or code).

5. **Run drift check** on the proposal text (tool or curl). Body is **`DriftCheckRequest`**: required **`proposal`**, optional **`tags`**, **`slow_path_required`**, **`is_followup_check`**, **`repo_root`**, **`touched_symbols`** — see `internal/drift/types.go` / `internal/runmulti/types.go`.
   ```bash
   jq -n --rawfile p proposal.txt '{proposal: ($p | rtrimstr("\n"))}' > drift-request.json
   curl -s -X POST http://localhost:8123/v1/drift/check -H 'Content-Type: application/json' -d @drift-request.json | jq .
   ```
   If the result is `"passed": false`, the proposal violates stored constraints or failure patterns—revise before applying.

6. **After meaningful work**, run curation digest/materialize to store durable learning (state/decision/failure/pattern/constraint). Promote selectively to avoid transcript soup.

---

## Principles

- **Recall first:** Compile and attach the recall bundle so the agent has constraints and decisions in context.
- **Drift after:** Check the proposal against memory before committing.
- **Curate sparingly:** Promote only memories that will help future tasks.
- **Do not conflate planes:** Editor intelligence (gopls) ≠ Pluribus (MCP/HTTP).
