<div align="center">

<img src="Pluribus_logo.png" alt="Pluribus" width="280">

*Out of many agents, one memory.*

</div>

# Pluribus

**Pluribus** is the **open control plane** for governed AI memory: **Postgres-backed** durable rows, **situation-shaped recall**, **pre-change enforcement**, and **curation** — without treating chat or logs as source of truth.

> **Usage:** **`recall_context` → plan → act → `record_experience`**. Skip the loop and you get amnesia.

**Start here:** **[`docs/get-started.md`](docs/get-started.md)** (run → connect → optional doctrine).

---

## What this is

| Piece | Location |
|-------|----------|
| **Runnable API** | **`pluribus`** service on **`:8123`** (`docker compose up -d`) |
| **Go implementation** | `control-plane/` module |
| **Agent packs** | [`integrations/`](integrations/) + [`pluribus-instructions.md`](integrations/pluribus-instructions.md) |
| **Canonical model** | [`docs/memory-doctrine.md`](docs/memory-doctrine.md) |
| **Full doc index** | [`docs/README.md`](docs/README.md) |

---

## Quick start

```bash
docker compose up -d
make smoke-shared-memory
```

Add MCP at **`http://127.0.0.1:8123/v1/mcp`** and paste **[`integrations/pluribus-instructions.md`](integrations/pluribus-instructions.md)** into your editor rules. Details: **[get-started](docs/get-started.md)** · **[MCP usage](docs/mcp-usage.md)** · **[Cursor pack](integrations/cursor/README.md)**.

**The memory database is sacred** — in-place upgrades only; see **[operate/guide](docs/operate/guide.md)**.

---

## Why use it

| Benefit | What it avoids |
|--------|----------------|
| **Continuity** | Re-deriving constraints and failures every session |
| **Shared truth** | Memory trapped in one chat or editor |
| **Authority-aware recall** | Flattening everything into one similarity score |
| **Enforcement** | Shipping changes that violate stated decisions |
| **Curated learning** | Promoting noise without review |

---

## Repository layout

| Path | Purpose |
|------|---------|
| `control-plane/` | Go API (`make build` → `./pluribus`) |
| `integrations/` | Per-editor MCP + rules packs |
| `docs/` | Public documentation ([index](docs/README.md)) |
| `scripts/` | Upgrade, backup, proof, verification |
| `evidence/` | Proof artifacts |

Internal maintainer notes: [`docs/meta/`](docs/meta/) (not product truth).

---

## Verification

```bash
make regression          # CI gate
make smoke-shared-memory # quick shared-memory check
```

Substrate proof: **`make proof-rest`** · Integration scenarios: **[proof/](docs/proof/README.md)** · [CONTRIBUTING.md](CONTRIBUTING.md)
