# Local Upgrade Release Notes

**Upgrade from:** previous local deployment (version unknown — capture `controlplane --version` before upgrade)

**Upgrade to:** release candidate at `v1.2.2-11-g4478297-dirty` (commit `4478297`, branch `main`)

---

## Summary

This release candidate completes the Phase 11 agent memory loop (usefulness benchmarks through guarded utility policy) and Phase 12 regression hardening. Suitable for local server upgrade **after backup** and **post-upgrade smoke verification**.

---

## Phase 11B–11K capabilities

| Phase | Capability |
|-------|------------|
| 11B | Agent memory usefulness benchmark |
| 11C | Cognitive memory benefit hardening |
| 11D | Memory formation quality gate |
| 11E | Formation escape hatches + codebase test isolation |
| 11F | Agent-facing memory contract |
| 11G | MCP/REST contract parity + endpoint coverage |
| 11H | Agent contract obedience telemetry |
| 11I | Persisted telemetry + utility candidates |
| 11J | Automatic recall telemetry hooks + Postgres proof |
| 11K | Guarded utility candidate application policy |

## Phase 12

| Phase | Capability |
|-------|------------|
| 12A | Savage repo/docs audit |
| 12B | Regression greenline (integration test repair) |
| 12C | Upgrade readiness (this package) |

---

## Database migrations added (if upgrading from pre-11I)

Applies on boot (idempotent):

- `0012_agent_memory_use_telemetry.sql` — telemetry sessions, recall/decision/output events, violations, utility candidates
- `0013_agent_utility_applications.sql` — guarded utility application ledger

Earlier migrations `0001`–`0011` required for full feature set.

---

## New MCP tools

9× `agent_telemetry_*` + 8× `agent_utility_*` (55 tools total). See [http-api-index.md](../http-api-index.md) and [mcp-tools.md](../mcp-tools.md).

---

## New REST endpoints

- `/v1/agent/telemetry/*`
- `/v1/agent/utility/policy/*`

See [http-api-index.md](../http-api-index.md).

---

## Config / environment

- `PLURIBUS_API_KEY` — recommended for production auth
- `CONFIG` — path to YAML with `postgres.dsn`
- Review [local-upgrade-config-checklist.md](../local-upgrade-config-checklist.md)

No new **required** env vars beyond existing Postgres DSN in config.

---

## Breaking changes

None identified for MCP tool names or REST paths. Phase 11 HTTP hard reject (400 on junk advisory ingest) is behavioral — clients expecting 201+rejected should update.

---

## Known risks

- **Dirty git tree** if built from working copy — prefer clean tag checkout
- **No schema version table** — rollback requires DB restore
- **Unknown prior local version** — verify migrations 0012/0013 apply cleanly on first boot
- Recall keyword-bridge changes (Phase 12B) affect candidate generation — benchmark gates pass

---

## Upgrade checklist

1. [local-server-backup-rollback.md](../local-server-backup-rollback.md)
2. [local-server-upgrade-runbook.md](../local-server-upgrade-runbook.md)
3. `./scripts/smoke/local-post-upgrade-verify.sh`

---

## Verification

```bash
make build && ./control-plane/controlplane --version
./scripts/migration-dry-run.sh
make regression   # maintainer gate
```
