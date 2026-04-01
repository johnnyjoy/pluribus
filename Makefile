# Recall repo — release/test-drive automation entrypoints.
#
# GitHub CI runs: (1) cd control-plane && go test ./...  (2) make regression.
# Plain `go test ./...` does NOT compile //go:build integration tests — use `make regression`,
# `make integration-go` (ephemeral Postgres + host Go, no Compose image build), or `make ci-local`.
.PHONY: test regression integration-go ci-local eval stress-eval api-test integration-test test-drive image pg-textsearch-image lexical-backfill lexical-reindex lexical-verify pg-textsearch-eval

COMPOSE_REGRESSION := docker compose -p recall-regression -f docker-compose.regression.yml
ARTIFACTS_DIR ?= artifacts

# Local Pluribus image (control-plane) with embedded version. Tags: $(IMAGE_NAME):$(PLURIBUS_VERSION) and $(IMAGE_NAME):local
# Override: make image PLURIBUS_VERSION=1.2.3  or  VERSION=1.2.3 make image
PLURIBUS_VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
IMAGE_NAME ?= pluribus

image:
	docker build -t $(IMAGE_NAME):$(PLURIBUS_VERSION) -t $(IMAGE_NAME):local \
		--build-arg VERSION=$(PLURIBUS_VERSION) \
		-f control-plane/Dockerfile control-plane
	@echo "Images: $(IMAGE_NAME):$(PLURIBUS_VERSION) $(IMAGE_NAME):local — set PLURIBUS_IMAGE=$(IMAGE_NAME):$(PLURIBUS_VERSION) for docker-compose.install.yml"

# Docker regression: ephemeral Postgres (no host ports) + integration tests inside regression-runner.
# -p recall-regression matches compose file `name:` so teardown never hits the dev `recall` project.
# Requires Docker Compose v2. Always tears down regression volumes after the run (success or failure).
regression:
	$(COMPOSE_REGRESSION) run --rm --build regression-runner \
		&& $(COMPOSE_REGRESSION) down -v --remove-orphans \
		|| { $(COMPOSE_REGRESSION) down -v --remove-orphans; exit 1; }

# Host Go + ephemeral Postgres (Docker on localhost). Sets TEST_PG_DSN / TEST_PG_RESET_SCHEMA — you are
# not testing the DSN. Avoids Compose image/buildx when you only need integration-tagged tests locally.
integration-go:
	@./scripts/run-integration-tests.sh

# Experimental: build Postgres 18 + pgvector + pg_textsearch image (see docs/experiments/pg-textsearch-container.md).
pg-textsearch-image:
	docker build -t pluribus-postgres-pg-textsearch:local -f docker/pg-textsearch/Dockerfile docker/pg-textsearch

# Lexical projection ETL (requires PG_TEXTSEARCH_EVAL_DSN or DATABASE_URL to Postgres with pg_textsearch loaded).
lexical-backfill:
	cd control-plane && go run ./cmd/pg-textsearch-eval -dsn="$${PG_TEXTSEARCH_EVAL_DSN:-$${DATABASE_URL:-postgres://controlplane:controlplane@127.0.0.1:5432/controlplane?sslmode=disable}}" backfill

lexical-reindex:
	cd control-plane && go run ./cmd/pg-textsearch-eval -dsn="$${PG_TEXTSEARCH_EVAL_DSN:-$${DATABASE_URL:-postgres://controlplane:controlplane@127.0.0.1:5432/controlplane?sslmode=disable}}" reindex

lexical-verify:
	cd control-plane && go run ./cmd/pg-textsearch-eval -dsn="$${PG_TEXTSEARCH_EVAL_DSN:-$${DATABASE_URL:-postgres://controlplane:controlplane@127.0.0.1:5432/controlplane?sslmode=disable}}" verify

# Full automated eval: ephemeral Docker Postgres + seed + reindex + query suite + artifacts (see docs/experiments/pg-textsearch-etl.md).
pg-textsearch-eval:
	@./scripts/pg-textsearch-eval.sh

# Core unit and package tests for the authoritative module.
test:
	$(MAKE) -C control-plane test

# Local gate matching .github/workflows/ci.yml (unit tests + Docker integration suite).
ci-local: test regression

# Run deterministic evaluation harness and emit lightweight artifacts.
eval:
	@mkdir -p $(ARTIFACTS_DIR)
	cd control-plane && go test ./internal/eval -run TestEvaluationHarness -v | tee ../$(ARTIFACTS_DIR)/eval-report.txt
	@printf '{\n  "suite": "eval",\n  "artifact": "artifacts/eval-report.txt",\n  "generated_at_utc": "%s"\n}\n' "$$(date -u +%Y-%m-%dT%H:%M:%SZ)" > $(ARTIFACTS_DIR)/eval-report.json

# Run stress-focused eval scenarios (subset) and emit lightweight artifacts.
stress-eval:
	@mkdir -p $(ARTIFACTS_DIR)
	cd control-plane && go test ./internal/eval -run 'TestEvaluationHarness|TestDetectTriggersFromScenarios' -v | tee ../$(ARTIFACTS_DIR)/stress-report.txt
	@printf '{\n  "suite": "stress-eval",\n  "artifact": "artifacts/stress-report.txt",\n  "generated_at_utc": "%s"\n}\n' "$$(date -u +%Y-%m-%dT%H:%M:%SZ)" > $(ARTIFACTS_DIR)/stress-report.json

# REST/API-focused integration tests (host-managed Postgres DSN required).
api-test:
	cd control-plane && TEST_PG_DSN="$${TEST_PG_DSN}" go test -tags=integration -v ./cmd/controlplane -run TestIntegration_rest

# Full integration-tagged control-plane tests (host-managed Postgres DSN required).
integration-test:
	cd control-plane && TEST_PG_DSN="$${TEST_PG_DSN}" go test -tags=integration -v ./cmd/controlplane

# Reasonable one-command technical-preview proof path.
test-drive: test eval

# Episodic advisory + distillation + curation + recall/enforcement boundary (host Postgres; matches CI regression stack).
.PHONY: proof-episodic
proof-episodic:
	cd control-plane && TEST_PG_DSN="$${TEST_PG_DSN}" $(MAKE) proof-episodic
