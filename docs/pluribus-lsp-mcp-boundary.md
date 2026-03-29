# Pluribus — LSP vs MCP boundary (canonical doctrine)

**Status:** Stable product boundary — documentation and mental model, not an implementation spec.

This document **locks** how Pluribus relates to **editor LSP** and to **MCP**, so the system does not drift into protocol confusion or scope creep.

---

## Doctrine statement

> **Pluribus is a continuity and governance service exposed via HTTP and MCP. It is not an LSP server and does not replace editor language servers.**

---

## Mental model (one paragraph)

**Use gopls (or your editor’s language server) for code understanding** — symbols, definitions, references, diagnostics in the IDE.

**Use Pluribus via MCP or HTTP for memory, continuity, and workflow discipline** — recall bundles, drift, enforcement, curation ([http-api-index.md](http-api-index.md)).

These planes are **complementary**, not interchangeable.

---

## Three planes

| Plane | Role | Examples |
|-------|------|----------|
| **Code intelligence** | Local, editor-driven semantics | gopls, hover, go-to-definition, references, diagnostics |
| **Continuity / governance** | Durable scoped memory and checks | Pluribus: recall, drift, enforcement, curation, evidence |
| **Integration (agents)** | Tooling protocol for AI clients | MCP over HTTP (`POST /v1/mcp`), optional stdio `pluribus-mcp` → HTTP |

---

## Non-negotiable rules

1. **Pluribus is not an LSP server.** Do not expose Pluribus as an LSP endpoint for the editor, do not document “point Cursor’s Go language server at Pluribus,” and do not imply Pluribus replaces gopls.

2. **MCP (and HTTP) is the integration surface for agents.** The legitimate “one URL” story is **one base URL for the Pluribus API** (REST + `POST /v1/mcp`), not a URL that substitutes for editor LSP.

3. **gopls remains local and editor-owned.** Cursor/VS Code use gopls over stdio; that is expected and supported. Pluribus does not interfere with it.

4. **In-process LSP client only.** When `lsp.enabled` is true, the control-plane runs an **LSP client** to **gopls** **inside the server process** to enrich recall and drift (e.g. symbols, reference counts). That is **server-side** enrichment, not a protocol offered to the editor.

5. **No LSP proxies or multiplexers in scope.** LSP multiplexers, editor-facing LSP-over-HTTP bridges, and hybrid MCP/LSP adapters are **out of scope** unless explicitly approved as a **separate** product initiative.

---

## “One URL” — correct meaning

| Question | Answer |
|----------|--------|
| **Yes** | One **Pluribus** base URL for **HTTP** and **MCP** (`POST /v1/mcp`). |
| **No** | One URL that makes the **editor** treat Pluribus as its **language server**. That is not supported and not a goal. |

---

## FAQ

**Q1 — What is Pluribus responsible for?**  
Continuity, governed memory, recall compilation, drift, enforcement, curation — not IDE code intelligence.

**Q2 — What is gopls responsible for?**  
Code structure and editor semantics in the workspace.

**Q3 — What is MCP responsible for?**  
Agent/tool integration with Pluribus (tools, prompts, resources on the MCP surface).

**Q4 — Why not unify LSP + MCP + HTTP into one protocol?**  
They solve different problems. Forcing one abstraction increases complexity, security surface, and user confusion without a proportional benefit.

**Q5 — Can clients send symbol names without server-side gopls?**  
Yes. Compile requests may include **`symbols`** from any source; server-side gopls is optional for auto-fill and reference-count features. See [control-plane/docs/lsp-features.md](../control-plane/docs/lsp-features.md).

---

## Future boundary note

If **deeper editor integration** is ever desired (e.g. a pass-through or multiplexer that combines gopls with Pluribus signals), that would be:

- a **deliberate** product/design effort,
- likely a **separate** component or initiative,
- **not** implied by current Pluribus scope.

Until then, this boundary remains fixed.

---

## References

| Doc | Purpose |
|-----|---------|
| [http-api-index.md](http-api-index.md) | HTTP routes + MCP column; LSP affects only optional recall/drift fields |
| [pluribus-public-architecture.md](pluribus-public-architecture.md) | Canonical access model (HTTP + MCP) |
| [mcp-service-first.md](mcp-service-first.md) | MCP on the API |
| [mcp-usage.md](mcp-usage.md) | Cursor + MCP troubleshooting |
| [control-plane/docs/cursor-usage.md](../control-plane/docs/cursor-usage.md) | Cursor workflow (recall / drift) |
| [control-plane/docs/lsp-features.md](../control-plane/docs/lsp-features.md) | Server-side LSP **client** features (config, HTTP fields) |
