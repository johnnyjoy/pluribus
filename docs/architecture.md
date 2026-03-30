# Architecture — memory, recall, enforcement

This document states how the **Recall / Pluribus** stack is shaped **when viewed through** [memory-doctrine.md](memory-doctrine.md). It is not a duplicate of every endpoint; it is the **structural story**.

---

## Global memory pool

- **Durable memory** lives in the database (`memories` and related tables) behind the **control-plane** API.
- The pool is **global** — not a set of isolated per-container truths.
- **Tags**, **kinds**, and **authority** are first-class; they replace “which silo am I in?” as the mental model.

---

## Situational recall

- **Recall** assembles a **bundle** for the current question: continuity, constraints, experience (and related groupings per API version).
- The bundle is **selected**, not a full database dump — token-bounded and relevance-ranked.
- Inputs are **situation-shaped**: natural-language **retrieval query**, optional **tags**, optional symbols/LSP hints where configured — **not** a required container selector.

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

## Control-plane placement

- **HTTP** — canonical integration surface (`/v1/memory/*`, `/v1/recall/*`, `/v1/enforcement/*`, curation, evidence, …).
- **MCP over HTTP** — `POST /v1/mcp` with tools and prompts aligned to the doctrine.
- **Postgres** — authoritative durable store; **Redis** optional for cache where configured.

---

## Verification (REST-first)

- **Proof of core behavior** is established at the **REST API** first: **`make proof-rest`** in `control-plane/` with **`TEST_PG_DSN`** (Postgres with **pgvector**, clean DB). See [evaluation.md](evaluation.md) and [evidence/memory-proof.md](../evidence/memory-proof.md).
- **Episodic advisory and distillation** are covered by embedded **`proof-episodic-*.json`** (in **`proof-rest`**) and by **`make proof-episodic`** (adds sprint integration tests). See [evidence/episodic-proof.md](../evidence/episodic-proof.md) and [episodic-similarity.md](episodic-similarity.md).
- **MCP** is tested as a **thin adapter** once REST behavior is locked; it is not a substitute for service-boundary proof.
- **LSP**-assisted recall is **optional**; it does not define the memory contract.
- **CI** runs a broader **`make regression`** gate (integration tests, including YAML proof scenarios); that complements but does not replace the **canonical REST proof harness** for substrate truth.

---

## Related docs

- [memory-doctrine.md](memory-doctrine.md)
- [anti-regression.md](anti-regression.md)
- [pluribus-public-architecture.md](pluribus-public-architecture.md)
- [http-api-index.md](http-api-index.md) — canonical route + MCP map  
- [api-contract.md](api-contract.md) — RC1 subset narrative  
- [evaluation.md](evaluation.md) — **`make proof-rest`** and supporting targets
