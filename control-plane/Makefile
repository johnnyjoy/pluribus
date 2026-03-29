# Build all control-plane binaries.
.PHONY: build build-controlplane build-pluribus-mcp clean test regression

# Full regression (Docker): ephemeral Postgres + integration tests — run from repo root via parent Makefile.
regression:
	$(MAKE) -C .. regression

build: build-controlplane build-pluribus-mcp

build-controlplane:
	go build -o controlplane ./cmd/controlplane

build-pluribus-mcp:
	go build -o pluribus-mcp ./cmd/pluribus-mcp

clean:
	rm -f controlplane pluribus-mcp

test:
	go test ./...

# Adversarial REST proof harness (requires Postgres + migrations). From control-plane/: TEST_PG_DSN=... make proof-rest
.PHONY: proof-rest
proof-rest:
	go test -tags=integration -count=1 -v ./internal/eval/... -run TestProofHarnessREST_Postgres
