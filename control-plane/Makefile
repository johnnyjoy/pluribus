# Build all control-plane binaries with embedded version metadata.
.PHONY: build build-controlplane build-pluribus-mcp build-mcp clean test regression migration-dry-run migrate-status

GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
PLURIBUS_VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w \
	-X control-plane/internal/buildinfo.Version=$(PLURIBUS_VERSION) \
	-X control-plane/internal/buildinfo.GitCommit=$(GIT_COMMIT) \
	-X control-plane/internal/buildinfo.BuildTime=$(BUILD_TIME)

# Full regression (Docker): ephemeral Postgres + integration tests — run from repo root via parent Makefile.
regression:
	$(MAKE) -C .. regression

build: build-controlplane build-pluribus-mcp

build-controlplane:
	go build -ldflags="$(LDFLAGS)" -o controlplane ./cmd/controlplane

build-pluribus-mcp build-mcp:
	go build -ldflags="$(LDFLAGS)" -o pluribus-mcp ./cmd/pluribus-mcp

# Disposable Postgres migration proof (writes ../artifacts/local-upgrade-migration-dry-run.json).
migration-dry-run migrate-status:
	@../scripts/migration-dry-run.sh

clean:
	rm -f controlplane pluribus-mcp

test:
	go test ./...

# Adversarial REST proof harness (requires Postgres + migrations). From control-plane/: TEST_PG_DSN=... make proof-rest
.PHONY: proof-rest test-mcp
proof-rest:
	go test -tags=integration -count=1 -v ./internal/eval/... -run TestProofHarnessREST_Postgres

# MCP schema quality + JSON-RPC behavior tests (no Docker).
test-mcp:
	go test ./internal/mcp/... -count=1

# Episodic REST proof: full proof-rest JSON suite + sprint integration scenarios (same DSN + integration tag).
.PHONY: proof-episodic
proof-episodic:
	go test -tags=integration -count=1 -v ./internal/eval/... -run 'TestProofHarnessREST_Postgres|TestEpisodicProofSprintREST_Postgres'
