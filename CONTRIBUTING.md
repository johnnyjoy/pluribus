# Contributing to Recall / Pluribus

Thank you for helping improve governed agent memory. This repo optimizes for **memory integrity**, not feature churn.

## Before you open a PR

1. Read **[docs/memory-doctrine.md](docs/memory-doctrine.md)** — canonical product model (highest authority).
2. Read **[docs/anti-regression.md](docs/anti-regression.md)** — what reviewers will reject.
3. Run tests from **`control-plane/`**:
   ```bash
   cd control-plane && go test ./... && go vet ./...
   ```

Guard tests under **`control-plane/internal/guardrails/`** enforce memory-first copy and JSON shapes for core paths; do not disable them without a doctrine update.

## Design features without “containers”

- Model user intent as **situation** + **tags** + **retrieval query**, not as a silo selector.
- Persist truth as **typed memory** with **authority**; use **recall** to assemble context and **enforcement** to gate risky proposals.
- Prefer **curation_digest → materialize** for new durable learning, not raw chat dumps.

If you think you need a new “partition” or “owner ID” for memory, **stop** and open a design discussion; the default answer is **no** unless `memory-doctrine.md` is explicitly revised.

## Evaluate your change

Ask:

- Does it respect **global memory** and **situational recall**?
- Does it add **required** correlation IDs to recall, enforcement, or core memory APIs?
- Does MCP or docs teach **silos** instead of **tags + query**?
- Would it fail **[docs/anti-regression.md](docs/anti-regression.md)** checklist?

## Docs and prompts

- MCP prompt bodies live in **`control-plane/internal/mcp/*.md`** — each must end with the **Pluribus doctrine (MCP)** footer (see any prompt file).
- Public narrative docs should link **memory-doctrine.md** where the model matters.

## Code layout

- **Authoritative API:** `control-plane/` (separate Go module).
- Repo root holds constitution, work orders, and top-level docs.

## License / compliance

Follow repository license and any org policies for secrets and PII. Do not commit credentials.
