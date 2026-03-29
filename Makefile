# Recall repo — release/test-drive automation entrypoints.
#
# GitHub CI runs: (1) cd control-plane && go test ./...  (2) make regression.
# Plain `go test ./...` does NOT compile //go:build integration tests — use `make regression`
# or `make ci-local` before pushing to match CI.
.PHONY: test regression ci-local eval stress-eval api-test integration-test test-drive image

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
