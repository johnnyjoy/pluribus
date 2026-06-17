# Local Pluribus Server — Upgrade Runbook

Upgrade from **previous local deployment** to **current repository release candidate** (`git describe` / `--version` output).

**This runbook does not upgrade your server automatically.** Execute steps manually with confirmation at destructive points.

---

## Scope

- Control-plane HTTP API + Postgres persistence
- Optional stdio MCP proxy (`pluribus-mcp`)
- Phase 11I telemetry + Phase 11K guarded utility policy tables

## Assumptions

- You operate the server (systemd, bare binary, or Docker Compose)
- Postgres is reachable from the control-plane host
- You can take a **full logical backup** before upgrade
- Current deployed version is **unknown to this doc** — capture `PLURIBUS_OLD_VERSION` yourself

## Variables

```bash
PLURIBUS_HOME=/path/to/pluribus
PLURIBUS_BACKUP_DIR=/path/to/backups/pluribus-YYYYMMDD
PLURIBUS_DB_DSN=postgres://USER:PASSWORD@HOST:5432/controlplane?sslmode=disable
PLURIBUS_OLD_VERSION=<your-current-version>
PLURIBUS_NEW_VERSION=$(cd "$PLURIBUS_HOME" && ./control-plane/controlplane --version | head -1)
CONFIG=${CONFIG:-$PLURIBUS_HOME/control-plane/configs/config.local.yaml}
PLURIBUS_BASE_URL=http://127.0.0.1:8123
PLURIBUS_API_KEY=<your-key-if-auth-enabled>
```

---

## Phase 0 — Pre-upgrade checklist

- [ ] Read [local-server-backup-rollback.md](local-server-backup-rollback.md)
- [ ] Read [local-upgrade-config-checklist.md](local-upgrade-config-checklist.md)
- [ ] `make regression` passed on upgrade candidate (maintainer) OR you accept release-candidate artifact
- [ ] Record `PLURIBUS_OLD_VERSION`
- [ ] Maintenance window scheduled

---

## Phase 1 — Backup (required)

Follow [local-server-backup-rollback.md](local-server-backup-rollback.md).

**STOP if backup not verified.**

---

## Phase 2 — Build / package new version

```bash
cd "$PLURIBUS_HOME"
git fetch && git checkout <release-commit>   # or use your packaging process
make build
./control-plane/controlplane --version
./scripts/build-proof.sh
```

**Docker alternative:**

```bash
make image PLURIBUS_VERSION=<tag>
export PLURIBUS_IMAGE=pluribus:<tag>
```

---

## Phase 3 — Migration dry-run (optional but recommended)

On a **disposable** machine (not production DB):

```bash
./scripts/migration-dry-run.sh
cat artifacts/local-upgrade-migration-dry-run.json
```

---

## Phase 4 — Config review

1. Diff your `config.local.yaml` against `control-plane/configs/config.example.yaml`
2. Ensure `postgres.dsn` unchanged unless intentional
3. Preserve `PLURIBUS_API_KEY` in environment
4. See [local-upgrade-config-checklist.md](local-upgrade-config-checklist.md)

---

## Phase 5 — Upgrade (stop → replace → start)

### 5.1 Stop service

```bash
sudo systemctl stop pluribus-control-plane
# OR: docker compose stop control-plane
```

### 5.2 Deploy new binary

```bash
make build
# Replace running binary paths or update Docker image tag
```

### 5.3 Start service

```bash
sudo systemctl start pluribus-control-plane
# OR: docker compose up -d control-plane
```

Watch logs for:

```text
schema: applied 000N_....sql
controlplane <version> listening on ...
```

First boot applies any missing idempotent migrations.

---

## Phase 6 — Post-upgrade smoke tests

```bash
export PLURIBUS_BASE_URL PLURIBUS_API_KEY
./scripts/smoke/local-post-upgrade-verify.sh \
  --base-url "$PLURIBUS_BASE_URL" \
  ${PLURIBUS_API_KEY:+--api-key "$PLURIBUS_API_KEY"}
```

Individual scripts:

- `scripts/smoke/local-rest-smoke.sh`
- `scripts/smoke/local-mcp-smoke.sh`
- `scripts/smoke/local-telemetry-smoke.sh`
- `scripts/smoke/local-utility-policy-smoke.sh`

---

## Phase 7 — MCP verification

```bash
./scripts/smoke/local-mcp-smoke.sh --base-url "$PLURIBUS_BASE_URL"
```

Expect ≥50 tools including `agent_telemetry_*` and `agent_utility_*`.

---

## Phase 8 — Telemetry / utility verification

Telemetry smoke creates one session row (low impact). Utility smoke is **read-only** (summary + applications list).

---

## Rollback decision points

| After step | If failure | Action |
|------------|------------|--------|
| Build proof | compile fail | Fix checkout; do not deploy |
| Start / logs | migrate error | Rollback DB + binary |
| REST smoke | health/compile fail | Rollback |
| MCP smoke | tools missing | Wrong binary — rollback |
| Telemetry | 404/500 | Check migrations 0012; rollback if broken |

Rollback: [local-server-backup-rollback.md](local-server-backup-rollback.md)

---

## Final acceptance criteria

- [ ] `controlplane --version` shows new build
- [ ] `/healthz` and `/readyz` OK
- [ ] Recall compile returns bundle
- [ ] MCP `tools/list` ≥50 tools
- [ ] Telemetry session start works
- [ ] Utility policy summary works
- [ ] Existing memories still queryable (spot-check)

---

## Release notes

See [releases/local-upgrade-release-notes.md](releases/local-upgrade-release-notes.md).
