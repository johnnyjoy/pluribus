# Control-plane — design entry (current)

**Purpose:** Point readers to **operational** and **interface** truth. Long-form historical design (pre–memory-first cutover, projects/targets narrative) is **[archive/control-plane-design-and-starter-legacy.md](archive/control-plane-design-and-starter-legacy.md)** — **archived**, not authoritative.

---

## Where truth lives

| Need | Doc / code |
|------|------------|
| **Product model** | [memory-doctrine.md](memory-doctrine.md) |
| **Every HTTP route + MCP tool** | [http-api-index.md](http-api-index.md) |
| **RC1 subset narrative (examples)** | [api-contract.md](api-contract.md) |
| **MCP tool → HTTP** | [mcp-poc-contract.md](mcp-poc-contract.md) |
| **Run, compose, schema boot** | [deployment-poc.md](deployment-poc.md), [control-plane/README.md](../control-plane/README.md) |
| **Router source** | `control-plane/internal/apiserver/router.go` |

---

## Mental model (short)

The **control-plane** is a **memory-first** Go service: **Postgres** holds durable **`memories`**; **recall** compiles situation-shaped bundles; **enforcement** gates proposals; **curation** turns post-work text into candidates → **materialize**. **MCP** and **REST** hit the **same** handlers.

Do not use this page as an API spec — use **http-api-index** and Go types.
