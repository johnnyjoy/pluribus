# Product Doctrine

Implementation doctrine for Pluribus product decisions. Not marketing copy.

## Core position

Pluribus should **make existing agents better** through durable, auditable, lifecycle-aware memory exposed via **MCP and REST/API**.

Pluribus should **not** require users to buy another AI subscription to make their AI subscription useful.

The product trap to avoid:

```text
“Please buy another AI subscription to help with your AI subscription.”
```

The correct position:

```text
Use your existing agentic client.
Connect it to Pluribus through MCP or REST/API.
Pluribus provides durable, auditable, lifecycle-aware memory.
```

## Interface doctrine

- **MCP and REST/API are both primary interfaces** for agentic clients.
- Neither interface is secondary for recall, memory formation, or lifecycle semantics.
- Agentic clients call Pluribus; they do not call internal retrieval plumbing.

## Memory doctrine alignment

- Agentic clients may **curate** memory (suggest, ingest, promote).
- Pluribus **enforces** memory integrity (formation gates, lifecycle, preservation, utility, contradiction policy).
- AI may suggest; Pluribus must enforce.

## Internal retrieval plumbing

Embeddings, if used, are **retrieval plumbing**, not product identity.

- Optional, replaceable, server-side only
- Invisible to agentic clients except through honest recall behavior and bundle metadata
- Disabled by default until hostile gates and agentic evaluation justify otherwise

## Optional backend capabilities

Optional in-process capabilities (e.g. run-multi synthesis) exist for **operator-controlled** scenarios. They are not required for core memory value and must not become a second product the user must operate.

## Provider drift guardrail

Do not prescribe Ollama, OpenAI, Anthropic, or any specific embedding provider in core phases unless explicitly scoped to provider integration.

Agentic clients must not depend on provider-specific embedding configuration.

See: [agent-interface-boundary.md](./agent-interface-boundary.md), [memory-doctrine.md](./memory-doctrine.md).
