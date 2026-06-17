# Local Pluribus Server — Backup and Rollback

**Doctrine:** Rollback means **restore database backup + previous binary/config**, not reversing migrations in place.

Pluribus replays idempotent embedded SQL on boot; there is **no down-migration** and **no schema version ledger**.

---

## Pre-upgrade backup checklist

- [ ] Record current version: `./control-plane/controlplane --version` (or Docker image tag)
- [ ] Record git commit / release tag used to build current binary
- [ ] Export config (redact secrets in notes): `cp $CONFIG $PLURIBUS_BACKUP_DIR/`
- [ ] **Postgres logical backup** (required)
- [ ] Optional: copy evidence directory if used (`evidence.root_path` in config)
- [ ] Optional: Docker image save if using containers

### Variables

```bash
PLURIBUS_HOME=/path/to/pluribus
PLURIBUS_BACKUP_DIR=/path/to/backups/pluribus-$(date -u +%Y%m%dT%H%M%SZ)
PLURIBUS_DB_DSN=postgres://USER:PASSWORD@HOST:5432/controlplane?sslmode=disable
PLURIBUS_OLD_VERSION=<record-before-upgrade>
```

---

## Postgres backup

```bash
mkdir -p "$PLURIBUS_BACKUP_DIR"
pg_dump -Fc "$PLURIBUS_DB_DSN" -f "$PLURIBUS_BACKUP_DIR/controlplane-pre-upgrade.dump"
# Verify dump readable
pg_restore -l "$PLURIBUS_BACKUP_DIR/controlplane-pre-upgrade.dump" | head
```

**Confirm explicitly before proceeding:** backup file exists and `pg_restore -l` succeeds.

---

## Config backup

```bash
cp "${CONFIG:-configs/config.local.yaml}" "$PLURIBUS_BACKUP_DIR/config.yaml"
# Do NOT commit backup files containing secrets
```

---

## Binary / container backup

**Bare metal:**

```bash
cp "$PLURIBUS_HOME/control-plane/controlplane" "$PLURIBUS_BACKUP_DIR/controlplane.$PLURIBUS_OLD_VERSION"
cp "$PLURIBUS_HOME/control-plane/pluribus-mcp" "$PLURIBUS_BACKUP_DIR/pluribus-mcp.$PLURIBUS_OLD_VERSION"
```

**Docker:**

```bash
docker tag "$PLURIBUS_OLD_IMAGE" "$PLURIBUS_BACKUP_DIR/pluribus:rollback-$PLURIBUS_OLD_VERSION"
docker save -o "$PLURIBUS_BACKUP_DIR/pluribus-image.tar" "$PLURIBUS_OLD_IMAGE"
```

---

## Capture schema state (informational)

```bash
psql "$PLURIBUS_DB_DSN" -c "\dt" > "$PLURIBUS_BACKUP_DIR/tables-pre-upgrade.txt"
psql "$PLURIBUS_DB_DSN" -c "SELECT COUNT(*) FROM memories;" >> "$PLURIBUS_BACKUP_DIR/counts-pre-upgrade.txt"
```

There is no `schema_migrations` table — table list is the best available snapshot.

---

## Service stop (before restore or binary swap)

```bash
# systemd example — adjust unit name
sudo systemctl stop pluribus-control-plane

# Docker Compose example
docker compose -f docker-compose.yml stop control-plane
```

---

## Rollback procedure

### When to rollback

- Boot fails after upgrade
- Smoke tests fail (`scripts/smoke/local-post-upgrade-verify.sh`)
- Unexpected data loss or auth breakage
- Migration apply errors on startup

### Steps

1. **Stop** new server (see above).
2. **Restore database** (destructive to post-upgrade writes):

```bash
# ⚠️ DESTRUCTIVE: drops/recreates objects per pg_restore flags — confirm backup path
dropdb --if-exists -h HOST -U USER controlplane   # OR restore to new DB and swap DSN
createdb -h HOST -U USER controlplane
pg_restore -d "$PLURIBUS_DB_DSN" "$PLURIBUS_BACKUP_DIR/controlplane-pre-upgrade.dump"
```

3. **Restore previous binary/image** from `$PLURIBUS_BACKUP_DIR`.
4. **Restore config** if changed during upgrade.
5. **Start** previous version.
6. **Verify:** healthz + smoke scripts against old binary.

---

## Post-rollback verification

```bash
curl -sS http://127.0.0.1:8123/healthz
./scripts/smoke/local-rest-smoke.sh --base-url http://127.0.0.1:8123
```

---

## Failure modes

| Failure | Likely cause | Action |
|---------|--------------|--------|
| Boot loop on migrate | Corrupt DB / partial apply | Restore backup |
| Auth failures | API key mismatch | Restore config + env |
| Empty recall | Wrong DSN / empty DB | Verify DSN; restore if wrong DB |
| MCP tools missing | Old binary | Confirm binary/image tag |

---

## Go / no-go (backup)

**NO-GO** if `pg_dump` not completed or untested before upgrade.
