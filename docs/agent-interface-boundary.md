# Agent Interface Boundary

Pluribus exposes **MCP and REST/API as both primary interfaces** for agentic clients. There is no secondary interface tier for recall or memory.

## Correct architecture

```text
Agentic client
  -> MCP or REST/API
      -> Pluribus
          -> memory storage
          -> lifecycle enforcement
          -> utility / reputation
          -> recall ranking
          -> optional internal retrieval plumbing
```

The agent calls **Pluribus**. The agent does **not** call an embedder, embedding provider, or model endpoint.

## Forbidden architecture

```text
Agentic client
  -> embedding provider
```

```text
Agentic client
  -> second AI subscription
      -> Pluribus
```

If Pluribus uses embeddings internally, that detail is **optional internal plumbing**. It must not become an agent contract, MCP tool requirement, or REST/API client obligation.

## Interface rules

| Rule | Detail |
|------|--------|
| Primary interfaces | MCP and REST/API are peers for agentic recall and memory |
| Embedder visibility | Not agent-facing; optional server-side only |
| Semantic default | Disabled; lexical recall is the safe floor |
| Metadata honesty | `semantic_retrieval` on compile bundles describes behavior; it does not instruct clients to configure providers |
| Parity | Lifecycle, date bounds, `recall_mode`, and `include_status` must behave the same through MCP and REST/API |
| Usefulness | Phase 11B harness compares memory vs no-memory and MCP vs REST through deterministic task fixtures — see [agent-memory-usefulness.md](./agent-memory-usefulness.md) |

## Product boundary

Pluribus must remain useful **without** requiring users to buy, run, or understand a second AI system. Internal AI-like helpers (optional synthesis, optional embeddings) are subordinate, auditable, and cannot override memory doctrine.

Pluribus is the agent’s **memory system**, not the agent itself.

## Provider drift guardrail

Do **not** prescribe Ollama, OpenAI, Anthropic, or any specific embedding provider in core Pluribus phases unless the phase is explicitly about provider integration.

Use generic terms:

- configured internal embedding provider
- project-supported embedder
- controlled retrieval plumbing

Agentic clients must not depend on provider-specific embedding details.

See also: [product-doctrine.md](./product-doctrine.md), [memory-doctrine.md](./memory-doctrine.md), [recall-quality.md](./recall-quality.md).
