# pg_textsearch in Pluribus containers

## PostgreSQL version

**pg_textsearch** supports **PostgreSQL 17 and 18** (upstream README). This repo’s dev stack already uses **pg18** (`pgvector/pgvector:pg18`).

## Strategy chosen: pre-built `.deb` inside pgvector image

The [v1.0.0 release](https://github.com/timescale/pg_textsearch/releases) ships **zip files** per PG major and **amd64/arm64** containing `pg-textsearch-postgresql-18_*.deb`.

The Dockerfile at `docker/pg-textsearch/Dockerfile`:

1. `FROM pgvector/pgvector:pg18`
2. `curl` + `unzip` the matching zip (`TARGETARCH` from BuildKit)
3. `dpkg -i` the `.deb` (installs `pg_textsearch.so` and extension SQL into PG 18 paths)
4. Purge curl/unzip to keep the image smaller

**Why not build from source in the image?** Faster, reproducible, matches upstream release artifacts; avoids carrying `build-essential` in the final image.

**Alternative:** build from `pg_textsearch-*.tar.gz` with `make && make install` — use if a release `.deb` is unavailable for your arch.

## shared_preload_libraries

Upstream requires:

```text
shared_preload_libraries = 'pg_textsearch'
```

The **base** `pgvector` image does not set this. The overlay **`docker-compose.pg-textsearch.yml`** sets:

```yaml
command:
  - postgres
  - -c
  - shared_preload_libraries=pg_textsearch
```

If you later need **multiple** libraries, use a comma-separated list (order matters; follow Postgres docs).

## Rebuild

```bash
make pg-textsearch-image
# or
docker build -t pluribus-postgres-pg-textsearch:local -f docker/pg-textsearch/Dockerfile docker/pg-textsearch
```

Run stack:

```bash
docker compose -f docker-compose.yml -f docker-compose.pg-textsearch.yml up -d --build postgres
```

## Limitations

- **Regression runner** (`docker-compose.regression.yml`) still uses stock `pgvector:pg18` until this experiment is promoted — integration tests do not require pg_textsearch.
- **First-time init**: existing **data volumes** created without pg_textsearch may need a fresh volume or manual `ALTER SYSTEM` if you switch images mid-stream; prefer a **new volume** for clean experiments.

## Smoke SQL

```bash
docker compose -f docker-compose.yml -f docker-compose.pg-textsearch.yml exec -T postgres \
  psql -U controlplane -d controlplane -v ON_ERROR_STOP=1 \
  < scripts/experiments/pg_textsearch_smoke.sql
```
