# Pluribus — public architecture (canonical)

Single **public-facing** story: what exists, how clients connect, and what is compatibility-only. **Authority:** [memory-doctrine.md](memory-doctrine.md), [architecture.md](architecture.md), **[http-api-index.md](http-api-index.md)** (full wire map), [api-contract.md](api-contract.md) (RC1 narrative subset).

---

## What Pluribus is

**Pluribus** (the control-plane service) is **governed, durable memory**: typed memory rows (constraints, decisions, patterns, failures, object lessons, state), **recall compilation** from a **shared pool**, **pre-change enforcement**, **curation** (digest → materialize), drift checks, and evidence links — **Postgres** + **HTTP**.

**Memory-first:** durable memory is **not** owned by silos. **Tags** and **retrieval_query** describe the **situation**. Exact JSON keys per route: **Go `json` tags** listed from [http-api-index.md](http-api-index.md). Narrative examples for a subset: [api-contract.md](api-contract.md). Ontology: [pluribus-memory-first-ontology.md](pluribus-memory-first-ontology.md).

Repo-root files (`constitution.md`, work orders, `evidence/`) are **adjacent patterns** for agent workflows. **Authoritative** execution and durable memory are **`control-plane`** HTTP APIs and Postgres.

---

## Why memory-first (short)

- **Long context ≠ durable memory** — transport capacity does not give authority ordering, persistence, or enforcement.
- **Selective recall** — candidates are ranked (authority, fit, recency, policy signals, optional similarity); the model gets a **bounded bundle**, not a transcript dump.
- **Enforcement** — proposals can be checked against binding memory before risky changes.
- **Shared learning** — validated lessons can influence later agent steps when promoted with evidence discipline.

---

## How it works (mechanics)

1. **Write** — Typed memory rows with tags and authority (direct API or materialize from curation).
2. **Recall** — `recall_get` / `recall_compile` assemble a situation-shaped bundle (prefer **tags** + **retrieval_query**).
3. **Enforce** — `enforcement_evaluate` gates proposals against binding memory.
4. **Curate** — `curation_digest` → review → `curation_materialize` for durable promotion paths.

Authority and salience can move with observed outcomes; learning stays reason-coded and inspectable, not opaque training.

---

## Canonical access model (service-first)

| Path | Role | When to use |
|------|------|-------------|
| **Direct HTTP** | `GET/POST /v1/*` | Any client that speaks REST. |
| **MCP over HTTP** | `POST /v1/mcp` — JSON-RPC 2.0 | MCP-capable clients; tools, **prompts**, **resources** ship here. |
| **Stdio MCP** (`pluribus-mcp`) | Forwards to HTTP | **Compatibility only** — not the default deployment shape. |

**Rule:** The **service** is the product. Canonical attachment is the **API URL** (and optional API key).

Details: [mcp-service-first.md](mcp-service-first.md), [mcp-poc-contract.md](mcp-poc-contract.md).

---

## Editor LSP vs Pluribus

Pluribus is **not** an LSP server and does **not** replace **gopls**. **MCP over HTTP** (and REST) is the **agent** integration surface. See [pluribus-lsp-mcp-boundary.md](pluribus-lsp-mcp-boundary.md).

---

## Prompts and resources

On **`POST /v1/mcp`**, Pluribus exposes **tools**, **prompts**, and **resources** (markdown URIs). They encode correct usage — not decoration. Versioning: **`SurfaceVersion`** in listings — [mcp-prompt-resource-versioning.md](mcp-prompt-resource-versioning.md).

---

## Feature stack (current release)

| Area | Behavior |
|------|----------|
| **Memory** | Durable **shared** rows (kind, authority, tags); promotion and evidence linking. |
| **Recall** | Compiles a **bundle** from the **shared pool**; inputs per **`CompileRequest`** / GET query ([http-api-index.md](http-api-index.md)). |
| **Enforcement** | Evaluates proposals against **binding** memory before change. |
| **Curation** | Digest summaries → candidates → materialize. |
| **Drift** | Structured checks against memory and policy. |
| **Proof** | YAML + integration tests — [proof-scenarios.md](proof-scenarios.md), [pluribus-proof-index.md](pluribus-proof-index.md). |

---

## Compatibility (non-authoritative)

| Item | Status |
|------|--------|
| **`cmd/pluribus-mcp` stdio** | Optional compat path to HTTP. |
| **Embeddings / vector retrieval** | **Not** canonical recall authority — see [pluribus-release-scope.md](pluribus-release-scope.md). |
| **Advisory episodic similarity** | Optional **advisory** layer — [episodic-similarity.md](episodic-similarity.md). |

---

## References

| Doc | Purpose |
|-----|---------|
| [pluribus-container-install.md](pluribus-container-install.md) | Compose-first install — GHCR image |
| [pluribus-image-release-policy.md](pluribus-image-release-policy.md) | Registry, tags, CI gates |
| [pluribus-quickstart.md](pluribus-quickstart.md) | First-run path |
| [pluribus-operational-guide.md](pluribus-operational-guide.md) | Config, health, migrations |
| [mcp-usage.md](mcp-usage.md) | MCP + Cursor usage (one doc) |
| [control-plane-design-and-starter.md](control-plane-design-and-starter.md) | Design entry (pointers); legacy deep doc archived |
| [http-api-index.md](http-api-index.md) | **Canonical** HTTP + MCP route map |
| [mcp-discipline-doctrine.md](mcp-discipline-doctrine.md) | When to use which tool |
