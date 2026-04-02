# Pluribus experimental image: PostgreSQL 18 + pgvector + Timescale pg_textsearch (BM25).
# Upstream: https://github.com/timescale/pg_textsearch — requires shared_preload_libraries (see compose overlay).
#
# Build from repo root:
#   docker build -t pluribus-postgres-pg-textsearch:local -f docker/pg-textsearch/Dockerfile docker/pg-textsearch
#
# Pre-built .deb from GitHub Releases (pg18 amd64/arm64) is installed into the PGDG Postgres 18 image.

FROM pgvector/pgvector:pg18

USER root

ARG PG_TEXTSEARCH_VERSION=1.0.0
ARG TARGETARCH

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates curl unzip \
	&& case "$TARGETARCH" in \
		amd64) ARCH=amd64 ;; \
		arm64) ARCH=arm64 ;; \
		*) echo "unsupported TARGETARCH=$TARGETARCH (need amd64 or arm64)" >&2; exit 1 ;; \
	esac \
	&& curl -fsSL -o /tmp/pt.zip "https://github.com/timescale/pg_textsearch/releases/download/v${PG_TEXTSEARCH_VERSION}/pg-textsearch-v${PG_TEXTSEARCH_VERSION}-pg18-${ARCH}.zip" \
	&& unzip -o /tmp/pt.zip -d /tmp \
	&& dpkg -i /tmp/pg-textsearch-postgresql-18_*.deb \
	&& rm -rf /tmp/pt.zip /tmp/*.deb \
	&& apt-get purge -y --auto-remove unzip curl \
	&& rm -rf /var/lib/apt/lists/*

USER postgres
