# Get started

**Single entry path.** Everything else is in [Documentation index](README.md).

---

## 1. Run Pluribus

**Default (clone + build):**

```bash
docker compose up -d
curl -sS http://127.0.0.1:8123/healthz   # ok
curl -sS http://127.0.0.1:8123/readyz    # ok
make smoke-shared-memory                 # PASS: Shared memory works.
```

Starts **Postgres**, **Redis**, and the **`pluribus`** API on **`http://127.0.0.1:8123`**. Data persists in the **`recall_pgdata`** volume.

**Published image (no local build):** [guides/container-install.md](guides/container-install.md).

**Local binary:** `cd control-plane && make build && ./pluribus` (Postgres required — see [operate/guide.md](operate/guide.md)).

**Hands-on HTTP lab:** [guides/quickstart-lab.md](guides/quickstart-lab.md).

**Sacred database:** upgrades are **in-place** migrations — never `docker compose down -v` on a pool with real memories. See [operate/guide.md](operate/guide.md) and `scripts/upgrade-in-place.sh`.

---

## 2. Connect your AI app

1. **MCP** → `http://127.0.0.1:8123/v1/mcp` ([mcp-usage.md](mcp-usage.md))
2. **Rules** → paste [`integrations/pluribus-instructions.md`](../integrations/pluribus-instructions.md) (or your platform pack under [`integrations/`](../integrations/))
3. **Verify** → [usage/ensuring-agent-usage.md](usage/ensuring-agent-usage.md)

| Platform | Pack |
|----------|------|
| **Hub** | [integrations/README.md](../integrations/README.md) |
| **Cursor** | [integrations/cursor/README.md](../integrations/cursor/README.md) |

Tools alone are not enough — the agent must **call** `recall_context`, **`resolve_chore`** when chores exist, and **`record_experience`**.

---

## 3. Why it works this way (optional)

[memory-doctrine.md](memory-doctrine.md) — canonical model. [architecture.md](architecture.md) — system shape.

---

## Maintainers

| Need | Doc |
|------|-----|
| CI / regression | [evaluation.md](evaluation.md), `make regression` |
| Build | [../BUILD.md](../BUILD.md) |
| Release | [release/README.md](release/README.md) |
| Proof receipts | [proof/README.md](proof/README.md) |
