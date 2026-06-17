# Local Pluribus Upgrade — Config Checklist

Use placeholders only. Do not commit secrets.

## Required before upgrade

| Variable / config | Required | Example placeholder | Secret? |
|-------------------|----------|---------------------|---------|
| `CONFIG` | Yes (non-Docker) | `configs/config.local.yaml` | No |
| `postgres.dsn` in config | Yes | `postgres://user:****@127.0.0.1:5432/controlplane?sslmode=disable` | Yes (password) |
| `server.bind` | Yes | `:8123` | No |
| `PLURIBUS_API_KEY` | Recommended prod | `<generate-strong-key>` | **Yes** |

## MCP stdio proxy (optional)

| Variable | Required | Example | Secret? |
|----------|----------|---------|---------|
| `CONTROL_PLANE_URL` | If using stdio MCP | `http://127.0.0.1:8123` | No |
| `CONTROL_PLANE_API_KEY` | If auth enabled | same as `PLURIBUS_API_KEY` | **Yes** |

## Optional features

| Variable / config | When needed | Upgrade impact |
|-------------------|-------------|----------------|
| `redis.enabled` | Cache layer | Restart required after config change |
| `synthesis.enabled` | Backend LLM synthesis | Off by default; no Ollama required |
| `mcp.disabled` | Disable HTTP MCP | Set `true` only if intentional |
| `enforcement.enabled` | Pre-change gate | Default on (RC1) |
| `similarity.enabled` / `distillation.enabled` | Advisory pipeline | Review config diff |

## Pre-upgrade capture

```bash
# Record current version (after upgrade candidate built)
PLURIBUS_HOME=/path/to/pluribus
$PLURIBUS_HOME/control-plane/controlplane --version
curl -sS http://127.0.0.1:8123/healthz

# Backup config (no secrets in git)
cp "$CONFIG" "$PLURIBUS_BACKUP_DIR/config.yaml.$(date -u +%Y%m%dT%H%M%SZ)"
```

## Post-upgrade verify

```bash
PLURIBUS_BASE_URL=http://127.0.0.1:8123
PLURIBUS_API_KEY=<your-key>   # if auth enabled
./scripts/smoke/local-post-upgrade-verify.sh --base-url "$PLURIBUS_BASE_URL"
```

See [local-server-upgrade-runbook.md](local-server-upgrade-runbook.md) and [local-server-backup-rollback.md](local-server-backup-rollback.md).
