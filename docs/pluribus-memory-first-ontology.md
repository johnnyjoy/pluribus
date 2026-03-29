# Pluribus — memory-first ontology (companion)

> **Authority:** [memory-doctrine.md](memory-doctrine.md) is **higher precedence**. This file explains the same model in narrative form; if anything here conflicts, **update this file** to match the doctrine.

## What is memory truth

- **Durable memory** is stored in **`memories`** / **`memories_tags`**. Rows are **global and shared**; context is primarily **tag- and kind-shaped**, not owned by a silo.
- **Recall** and **enforcement** operate on that shared pool using **situation** inputs (natural-language query, tags, policy), not mandatory partition selectors.
- **Authority, constraints, decisions, patterns, failures** are properties of **memory**, not of an external container.

## What we do not claim

- We do **not** teach that governing memory “belongs to” a software initiative, ticket, or team silo.
- We do **not** teach recall as “only within partition X.”
- Optional HTTP fields that exist for **legacy database foreign keys** are **not** the product mental model; they must **not** be required for core recall, enforcement, or memory create/search. See [anti-regression.md](anti-regression.md).

## Protocol vs ontology

The disciplined sequence **Recall → Act → Validate → Update → Repeat** is **protocol** (good sequencing).  
**Memory-first ontology** means the **substrate** is shared memory; onboarding and MCP copy must keep that order in the reader’s head.

## See also

- [memory-doctrine.md](memory-doctrine.md)
- [architecture.md](architecture.md)
- [http-api-index.md](http-api-index.md) — wire map  
- [api-contract.md](api-contract.md) — RC1 subset narrative
- [pluribus-public-architecture.md](pluribus-public-architecture.md)
