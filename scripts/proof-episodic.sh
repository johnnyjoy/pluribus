#!/usr/bin/env bash
# Episodic REST proof: Postgres DSN required (fresh DB recommended — see evidence/episodic-proof.md).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
if [[ -z "${TEST_PG_DSN:-}" ]]; then
  echo "TEST_PG_DSN is not set. Example:" >&2
  echo "  export TEST_PG_DSN='postgres://user:pass@localhost:5432/pluribus_proof?sslmode=disable'" >&2
  exit 1
fi
exec make -C "$ROOT" proof-episodic
