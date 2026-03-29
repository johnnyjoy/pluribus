#!/usr/bin/env bash
# Reset a local Postgres database for proof runs (requires dropdb/createdb on PATH).
# Usage: ./scripts/proof-fresh-db.sh [dbname]
# Example: ./scripts/proof-fresh-db.sh pluribus_proof
set -euo pipefail
DBNAME="${1:-pluribus_proof}"
dropdb --if-exists "$DBNAME" 2>/dev/null || true
createdb "$DBNAME"
echo "Empty database ready: $DBNAME"
echo "Then: cd control-plane && TEST_PG_DSN='postgres://USER:PASS@127.0.0.1:5432/${DBNAME}?sslmode=disable' make proof-rest"
