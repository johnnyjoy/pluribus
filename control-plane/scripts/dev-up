#!/usr/bin/env bash
# Dev reminder: start Postgres, then run the control-plane servers.
# Run from control-plane/ directory.
#
# 1. Create DB (once):  createdb controlplane
# 2. (skip) Schema applies on first ./controlplane start
# 3. Build:              make build
# 4. Start controlplane: ./controlplane   (or CONFIG=configs/config.local.yaml ./controlplane)
# 5. Health:             curl http://localhost:8123/healthz
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"
echo "Control-plane dev-up: run from $ROOT_DIR"
echo "  1. createdb controlplane   (if not exists)"
echo "  2. (embedded SQL runs when you start ./controlplane — no separate migrate step)"
echo "  3. make build"
echo "  4. ./controlplane"
echo "  5. curl http://localhost:8123/healthz"
