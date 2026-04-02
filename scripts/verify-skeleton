#!/usr/bin/env bash
# Verify layout and control-plane Go build, vet, test.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
echo "=== layout check ==="
test -f constitution.md && test -f active.md && test -f decisions.md
test -d evidence && test -d workorders && test -f workorders/WO-0001.md
echo "=== control-plane: go list ./... ==="
cd "$ROOT/control-plane"
go list ./...
echo "=== control-plane: go build controlplane ==="
go build -o /dev/null ./cmd/controlplane
echo "=== control-plane: go vet ./... ==="
go vet ./...
echo "=== control-plane: go test ./... ==="
go test ./...
echo "OK: skeleton verified."
