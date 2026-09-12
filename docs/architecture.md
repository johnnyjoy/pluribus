# Architecture — memory, recall, enforcement

How the **Pluribus** stack is shaped when viewed through [memory-doctrine.md](memory-doctrine.md). Structural story — not every endpoint. **Wire map:** [http-api-index.md](http-api-index.md). **RC1 narrative:** [api-contract.md](api-contract.md).

---

## Public summary

**Pluribus** (the **`pluribus`** service in Compose) is **governed, durable memory**: typed rows, **recall compilation** from a **shared pool**, **pre-change enforcement**, **curation** (digest → materialize), drift checks, and evidence links — **Postgres** + **HTTP**.

**Memory-first:** durable memory is **not** owned by silos. **Tags** and **`retrieval_query`** describe the **situation**. Ontology: [ontology.md](ontology.md).

| Access | Role |
|--------|------|
| **REST** `GET/POST /v1/*` | Any HTTP client |
| **MCP over HTTP** `POST /v1/mcp` | Agents — tools, prompts, resources |
| **Stdio** `pluribus-mcp` | **Compatibility only** — forwards to HTTP |

Editor **LSP ≠ Pluribus** — see [lsp-mcp-boundary.md](lsp-mcp-boundary.md). Service-first details: [mcp-service-first.md](mcp-service-first.md).

**Mechanics:** Write → Recall → Enforce → Curate. Proof receipts: [proof/README.md](proof/README.md).

---

## Global memory pool

- **Durable memory** lives in Postgres (`memories` and related tables) behind the **pluribus** API.
- The pool is **global** — not isolated per-container truths.
- **Tags**, **kinds**, and **authority** are first-class; they replace “which silo am I in?” as the mental model.

---

## Situational recall

- **Recall** assembles a **bundle** for the current question: continuity, constraints, experience (and related groupings per API version).
- The bundle is **selected**, not a full database dump — token-bounded and relevance-ranked.
- Inputs are **situation-shaped**: natural-language **retrieval query**, optional **tags**, optional **`repo_root`** (basename adds a soft affinity token), optional symbols/LSP hints where configured — **not** a required container selector.

**Situational affinity ranking** (additive, global pool preserved): [architecture/situational-affinity-ranking.md](architecture/situational-affinity-ranking.md).

---

## Hybrid retrieval

The recall pipeline combines, as configured:

- **Semantic** similarity (embeddings / pgvector where enabled),
- **Lexical** and **tag** overlap,
- **Authority** and **constraint** salience,
- **Policy** (triggered recall, limits, RIE caps, etc.).

There is **no** “search only inside container X” as the product contract.

---

## Authority dominance and constraint priority

- **Higher authority** memory ranks higher and participates in **enforcement** when binding.
- **Constraints** can **block** or **require review** against proposals — they are not advisory by default when classified as binding by the server.

---

## Experience → memory → behavior → learning

1. **Experience** — raw outcomes and narrative may exist in chat or evidence; they are **not** canon until distilled.
2. **Memory** — promoted, typed rows with authority (constraints, decisions, patterns, failures, …).
3. **Behavior** — agents **recall** before acting; **enforcement** evaluates risky proposals against binding memory.
4. **Learning** — **curation_digest** / **materialize** (and other promote paths) move validated learning into durable memory.

---

## Runtime placement

- **HTTP** — canonical integration surface (`/v1/memory/*`, `/v1/recall/*`, `/v1/enforcement/*`, curation, evidence, …).
- **MCP over HTTP** — `POST /v1/mcp` with tools and prompts aligned to the doctrine.
- **Postgres** — authoritative durable store; **Redis** optional for cache where configured.
- **Go module path** — still `control-plane/` (directory name); **binary / Compose service** — **`pluribus`**.

---

## Verification (REST-first)

- **Proof of core behavior** at **REST** first: **`make proof-rest`** in `control-plane/` with **`TEST_PG_DSN`**. See [evaluation.md](evaluation.md) and [evidence/memory-proof.md](../evidence/memory-proof.md).
- **Episodic lane:** **`make proof-episodic`**. See [evidence/episodic-proof.md](../evidence/episodic-proof.md) and [advisory/episodic-similarity.md](advisory/episodic-similarity.md).
- **MCP** is a **thin adapter** once REST behavior is locked.
- **CI:** **`make regression`** complements REST proof (YAML scenarios under `control-plane/proof-scenarios/`).

---

## Related docs

| Doc | Purpose |
|-----|---------|
| [memory-doctrine.md](memory-doctrine.md) | Canonical product model |
| [anti-regression.md](anti-regression.md) | Reviewer guardrails |
| [http-api-index.md](http-api-index.md) | Full route + MCP map |
| [evaluation.md](evaluation.md) | Proof commands |
| [operate/guide.md](operate/guide.md) | Config, health, migrations |
