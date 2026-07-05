# Recall repo — release/test-drive automation entrypoints.
#
# GitHub CI runs: (1) cd control-plane && go test ./...  (2) make regression.
# Plain `go test ./...` does NOT compile //go:build integration tests — use `make regression`,
# `make integration-go` (ephemeral Postgres + host Go, no Compose image build), or `make ci-local`.
.PHONY: test regression integration-go ci-local eval stress-eval api-test integration-test test-drive image pg-textsearch-image lexical-backfill lexical-reindex lexical-verify pg-textsearch-eval build build-control-plane build-mcp migration-dry-run migrate-status

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
		sh -c 'PROOF_AGENT_TELEMETRY_POSTGRES=1 AGENT_TELEMETRY_POSTGRES_BENCHMARK=1 \
		go test -tags=integration -count=1 ./internal/agenttelemetry/... \
		-run TestProofAgentTelemetryPostgresHardThresholds -v && \
		PROOF_GUARDED_UTILITY_POSTGRES=1 GUARDED_UTILITY_POSTGRES_BENCHMARK=1 \
		go test -tags=integration -count=1 ./internal/utilitypolicy/... \
		-run TestProofGuardedUtilityPostgresHardThresholds -v && \
		go test -tags=integration -count=1 -p 1 ./...' \
		&& $(COMPOSE_REGRESSION) down -v --remove-orphans \
		|| { $(COMPOSE_REGRESSION) down -v --remove-orphans; exit 1; }

# Host Go + ephemeral Postgres (Docker on localhost). Sets TEST_PG_DSN / TEST_PG_RESET_SCHEMA — you are
# not testing the DSN. Avoids Compose image/buildx when you only need integration-tagged tests locally.
integration-go:
	@./scripts/run-integration-tests

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

# Full automated eval: ephemeral Docker Postgres + seed + reindex + query suite + artifacts (see docs/experiments/pg-textsearch-eval.md).
pg-textsearch-eval:
	@./scripts/pg-textsearch-eval

# Core unit and package tests for the authoritative module.
test:
	$(MAKE) -C control-plane test

# Reproducible local binaries with version/commit/build-time metadata (see docs/reports/phase12c-version-build-metadata.md).
build:
	$(MAKE) -C control-plane build

build-control-plane:
	$(MAKE) -C control-plane build-controlplane

build-mcp:
	$(MAKE) -C control-plane build-pluribus-mcp

migration-dry-run migrate-status:
	$(MAKE) -C control-plane migration-dry-run

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

# Deployed benefit receipts: same proof-scenario suite against a live control-plane.
# Requires CONTROL_PLANE_URL or PLURIBUS_PROOF_BASE_URL (e.g. http://host:8123).
.PHONY: proof-deployed-benefit-receipts
proof-deployed-benefit-receipts:
	@chmod +x scripts/proof-deployed-benefit-receipts.sh
	@./scripts/proof-deployed-benefit-receipts.sh

# Doctrine drift regression (unit): pending-in-pool guardrails, lifecycle recall fixtures, binding dampener, MCP doc sync.
.PHONY: doctrine-regression
doctrine-regression:
	cd control-plane && go test ./internal/guardrails/... ./internal/recall/lifecyclebenchmark/... ./pkg/api/... ./internal/recall/... \
		-run 'TestMemoryDoctrine|TestApplyHistoricalScoreCap|TestEffectiveBinding|TestLifecycleRecallBenchmark' -count=1
	cd control-plane && go test ./internal/mcp/... -run TestMCPToolsDocMatchesRegistry -count=1

# Doctrine integration proofs (ephemeral Postgres): YAML benefit receipts + MCP record→recall continuity.
.PHONY: doctrine-regression-integration
doctrine-regression-integration:
	@./scripts/run-integration-tests -run 'TestIntegration_proofScenarioSuite|TestIntegration_HTTP_MCP_recordExperienceRecallContinuity' -count=1 -v

# Phase 2 install smoke: one command after docker compose — shared memory write/recall/enforcement.
.PHONY: smoke-shared-memory
smoke-shared-memory:
	@chmod +x scripts/smoke-shared-memory.sh
	@./scripts/smoke-shared-memory.sh

# Reasonable one-command technical-preview proof path.
test-drive: test eval

# Episodic advisory + distillation + curation + recall/enforcement boundary (host Postgres; matches CI regression stack).
.PHONY: proof-episodic proof-mcp test-mcp
proof-episodic:
	cd control-plane && TEST_PG_DSN="$${TEST_PG_DSN}" $(MAKE) proof-episodic

# Phase 1 MCP: unit/behavior tests (no Docker).
test-mcp:
	cd control-plane && go test ./internal/mcp/... -count=1

# Phase 1 MCP: Docker Compose proof (healthz, readyz, JSON-RPC initialize/list/call + hostile cases).
proof-mcp:
	@./scripts/proof-mcp-docker.sh

# Phase 1 MCP: Docker authenticated HTTP MCP proof.
proof-mcp-auth:
	@chmod +x scripts/proof-mcp-docker-auth.sh
	@./scripts/proof-mcp-docker-auth.sh

# Phase 1 MCP: stdio adapter proof against running control-plane.
proof-mcp-stdio:
	@chmod +x scripts/proof-mcp-stdio.sh
	@./scripts/proof-mcp-stdio.sh

# Phase 1 MCP close-out: unit tests + all Docker/stdio proofs.
proof-mcp-all: test-mcp proof-mcp proof-mcp-auth proof-mcp-stdio

# Phase 2 agent-loop compliance: unit/hostile tests + in-process MCP scenarios (+ optional Docker).
.PHONY: proof-agent-loop proof-agent-loop-docker proof-agent-loop-all test-agent-loop
test-agent-loop:
	cd control-plane && go test ./internal/compliance/... ./internal/mcp/ -run 'TestEvaluateSession|TestMCPTelemetry|TestProofAgentLoopScenarios' -count=1

proof-agent-loop:
	@chmod +x scripts/proof-agent-loop-compliance.sh
	@PROOF_AGENT_LOOP_SKIP_DOCKER=1 ./scripts/proof-agent-loop-compliance.sh

proof-agent-loop-docker:
	@chmod +x scripts/proof-agent-loop-compliance.sh
	@./scripts/proof-agent-loop-compliance.sh

proof-agent-loop-all: test-agent-loop proof-agent-loop-docker

# Phase 3 recall benchmark: hostile labeled corpus + metrics gate.
.PHONY: test-recall-benchmark recall-benchmark-gate recall-benchmark-baseline recall-benchmark-report recall-benchmark-all
test-recall-benchmark:
	cd control-plane && go test ./internal/recall/benchmark/... -run 'TestBenchmark|TestDomainConfusion|TestRecallBenchmarkRegression' -count=1

recall-benchmark-gate:
	cd control-plane && RECALL_BENCHMARK_GATE=1 go test ./internal/recall/benchmark/... -run TestRecallBenchmarkGate -count=1 -v

recall-benchmark-baseline:
	cd control-plane && RECALL_BENCHMARK_BASELINE=1 go test ./internal/recall/benchmark/... -run TestRecallBenchmarkBaseline -count=1 -v

recall-benchmark-report: recall-benchmark-baseline

recall-benchmark-all: recall-benchmark-baseline test-recall-benchmark recall-benchmark-gate

.PHONY: recall-benchmark-hybrid recall-benchmark-hybrid-gate recall-benchmark-compare recall-benchmark-all-modes
recall-benchmark-hybrid:
	cd control-plane && RECALL_BENCHMARK_HYBRID=1 go test ./internal/recall/benchmark/... -run TestRecallBenchmarkHybridGate -count=1 -v

recall-benchmark-hybrid-gate: recall-benchmark-hybrid

recall-benchmark-compare:
	cd control-plane && RECALL_BENCHMARK_COMPARE=1 go test ./internal/recall/benchmark/... -run TestRecallBenchmarkCompare -count=1 -v

recall-benchmark-all-modes: recall-benchmark-gate recall-benchmark-hybrid-gate recall-benchmark-compare

.PHONY: recall-benchmark-real-embedder test-embedding-staleness test-real-embedder-fallback
recall-benchmark-real-embedder:
	cd control-plane && RECALL_BENCHMARK_REAL_EMBEDDER=1 go test ./internal/recall/benchmark/... -run TestRecallBenchmarkRealEmbedder -count=1 -v

test-embedding-staleness:
	cd control-plane && go test ./internal/memory/... -run 'TestEmbedding|TestStale|TestMissing|TestDimension|TestSemanticRetrievalDefault|TestLiveEmbedder' -count=1

test-real-embedder-fallback:
	cd control-plane && go test ./internal/memory/... ./internal/recall/... -run 'TestEmbedder|TestSemanticRetrievalReportsFallbackReason' -count=1

recall-benchmark-live-compare:
	cd control-plane && RECALL_BENCHMARK_LIVE_COMPARE=1 RECALL_BENCHMARK_REAL_EMBEDDER=1 go test ./internal/recall/benchmark/... -run 'TestRecallBenchmarkLiveCompare|TestRecallBenchmarkRealEmbedder' -count=1 -v

# Phase 5 memory formation: hostile formation quality gate.
.PHONY: test-memory-formation memory-formation-report
test-memory-formation:
	cd control-plane && go test ./internal/formation/... -count=1

# Phase 8 lifecycle recall: hostile current vs historical gate.
.PHONY: test-lifecycle-recall
test-lifecycle-recall:
	cd control-plane && go test ./internal/recall/lifecyclebenchmark/... -count=1

# Phase 7 memory utility and reputation foundations.
.PHONY: test-memory-utility memory-utility-report
test-memory-utility:
	cd control-plane && go test ./internal/utility/... -count=1

# Phase 9B memory preservation: TTL/archive doctrine, historical recovery proofs.
.PHONY: test-memory-preservation
test-memory-preservation:
	cd control-plane && go test ./internal/memory/... ./internal/recall/... ./internal/guardrails/... ./internal/mcp/... -count=1 -run 'Test(TTL|Archive|Preserv|HistoricalRecall|ParseCompileDate|FilterByDate|NoProduction|Advisory|Durable|ListExpired|BuildMemoryContextResolveCompileBody_date|ValidateToolArguments_recallContext_invalid|UtilityScoreBounded|ExpireMemories)'

memory-utility-report:
	cd control-plane && go test ./internal/utility/benchmark/... -run TestMemoryUtilityBenchmarkGate -count=1 -v | tee ../artifacts/memory-utility-report.txt

memory-formation-report:
	cd control-plane && go test ./internal/formation/benchmark/... -run TestMemoryFormationBenchmarkCases -count=1 -v | tee ../artifacts/memory-formation-report.txt

# Phase 11B agentic memory usefulness harness (deterministic, no LLM, no external SaaS).
.PHONY: test-agent-memory-usefulness agent-memory-usefulness-benchmark proof-agent-memory-effectiveness
test-agent-memory-usefulness:
	cd control-plane && go test ./internal/agentusefulness/... -count=1

agent-memory-usefulness-benchmark:
	@mkdir -p artifacts
	cd control-plane && AGENT_MEMORY_USEFULNESS_BENCHMARK=1 go test ./internal/agentusefulness/... -run TestAgentMemoryUsefulnessBenchmarkWritesArtifact -count=1 -v

proof-agent-memory-effectiveness:
	@mkdir -p artifacts
	cd control-plane && AGENT_MEMORY_USEFULNESS_GATE=1 PROOF_AGENT_MEMORY_EFFECTIVENESS=1 go test ./internal/agentusefulness/... -run 'TestAgentMemoryUsefulnessGate|TestAgentMemoryUsefulnessBenchmarkWritesArtifact' -count=1 -v

# Phase 11C research-backed cognitive memory usefulness hardening.
.PHONY: test-cognitive-memory-usefulness cognitive-memory-usefulness-benchmark proof-cognitive-memory-benefit
test-cognitive-memory-usefulness:
	cd control-plane && go test ./internal/agentusefulness/... -count=1

cognitive-memory-usefulness-benchmark:
	@mkdir -p artifacts
	cd control-plane && COGNITIVE_MEMORY_USEFULNESS_BENCHMARK=1 go test ./internal/agentusefulness/... -run TestCognitiveMemoryBenchmarkArtifact -count=1 -v

proof-cognitive-memory-benefit:
	@mkdir -p artifacts
	cd control-plane && COGNITIVE_MEMORY_USEFULNESS_GATE=1 COGNITIVE_MEMORY_USEFULNESS_BENCHMARK=1 PROOF_COGNITIVE_MEMORY_BENEFIT=1 go test ./internal/agentusefulness/... -run 'TestCognitiveMemoryUsefulnessGate|TestCognitiveMemoryBenchmarkArtifact' -count=1 -v

# Phase 11D cognitive memory formation quality gates.
.PHONY: test-memory-formation-quality memory-formation-quality-benchmark proof-memory-formation-quality
test-memory-formation-quality:
	cd control-plane && go test ./internal/formationquality/... -count=1

memory-formation-quality-benchmark:
	@mkdir -p artifacts
	cd control-plane && MEMORY_FORMATION_QUALITY_BENCHMARK=1 go test ./internal/formationquality/... -run TestFormationQualityBenchmarkArtifact -count=1 -v

proof-memory-formation-quality:
	@mkdir -p artifacts
	cd control-plane && MEMORY_FORMATION_QUALITY_GATE=1 MEMORY_FORMATION_QUALITY_BENCHMARK=1 PROOF_MEMORY_FORMATION_QUALITY=1 go test ./internal/formationquality/... -run 'TestFormationQualityGate|TestFormationQualityBenchmarkArtifact' -count=1 -v

# Phase 11E formation escape hatches and codebase test isolation.
.PHONY: test-formation-escape-hatches formation-escape-hatch-benchmark proof-formation-escape-hatches test-codebase-isolation proof-codebase-isolation
test-formation-escape-hatches:
	cd control-plane && go test ./internal/formation/... -run 'TestPromote|TestProbationary|TestFormationEscape|TestDirectCreateRegression|TestFormationQualityFixtureLoads' -count=1

formation-escape-hatch-benchmark:
	@mkdir -p artifacts
	cd control-plane && FORMATION_ESCAPE_HATCH_BENCHMARK=1 go test ./internal/formation/... -run TestFormationEscapeHatchBenchmarkArtifact -count=1 -v

proof-formation-escape-hatches:
	@mkdir -p artifacts
	cd control-plane && FORMATION_ESCAPE_HATCH_GATE=1 FORMATION_ESCAPE_HATCH_BENCHMARK=1 PROOF_FORMATION_ESCAPE_HATCHES=1 go test ./internal/formation/... -run 'TestFormationEscapeHatchGate|TestFormationEscapeHatchBenchmarkArtifact' -count=1 -v

test-codebase-isolation:
	cd control-plane && go test ./internal/testisolation/... -count=1

proof-codebase-isolation:
	@mkdir -p artifacts
	cd control-plane && CODEBASE_ISOLATION_GATE=1 PROOF_CODEBASE_ISOLATION=1 go test ./internal/testisolation/... -run TestCodebaseIsolationGate -count=1 -v

# Phase 11F agent-facing memory contract and usage discipline.
.PHONY: test-agent-memory-contract agent-memory-contract-benchmark proof-agent-memory-contract
.PHONY: test-agent-memory-contract-parity agent-memory-contract-parity-benchmark proof-agent-memory-contract-parity
.PHONY: test-agent-memory-endpoint-coverage agent-memory-endpoint-coverage-benchmark proof-agent-memory-endpoint-coverage
test-agent-memory-contract:
	cd control-plane && go test ./internal/agentcontract/... -count=1

agent-memory-contract-benchmark:
	@mkdir -p artifacts
	cd control-plane && AGENT_MEMORY_CONTRACT_BENCHMARK=1 go test ./internal/agentcontract/... -run TestAgentMemoryContractBenchmarkArtifact -count=1 -v

proof-agent-memory-contract:
	@mkdir -p artifacts
	cd control-plane && PROOF_AGENT_MEMORY_CONTRACT=1 go test ./internal/agentcontract/... -run TestProofAgentMemoryContractHardThresholds -count=1 -v

test-agent-memory-contract-parity:
	cd control-plane && go test ./internal/agentcontract/... -run 'TestWakeup|TestMCP|TestREST|TestRunMulti|TestAllAgent|TestMissingField' -count=1

agent-memory-contract-parity-benchmark:
	@mkdir -p artifacts
	cd control-plane && AGENT_MEMORY_CONTRACT_PARITY_BENCHMARK=1 go test ./internal/agentcontract/... -run TestAgentMemoryContractParityBenchmarkArtifact -count=1 -v

proof-agent-memory-contract-parity:
	@mkdir -p artifacts
	cd control-plane && PROOF_AGENT_MEMORY_CONTRACT_PARITY=1 go test ./internal/agentcontract/... -run TestProofAgentMemoryContractParityHardThresholds -count=1 -v

test-agent-memory-endpoint-coverage:
	cd control-plane && go test ./internal/agentcontract/... -run 'TestAgentFacing|TestWakeup|TestRunMulti|TestMCP|TestREST|TestAllAgent' -count=1

agent-memory-endpoint-coverage-benchmark:
	@mkdir -p artifacts
	cd control-plane && AGENT_MEMORY_ENDPOINT_COVERAGE_BENCHMARK=1 go test ./internal/agentcontract/... -run TestAgentMemoryEndpointCoverageBenchmarkArtifact -count=1 -v

proof-agent-memory-endpoint-coverage:
	@mkdir -p artifacts
	cd control-plane && PROOF_AGENT_MEMORY_ENDPOINT_COVERAGE=1 go test ./internal/agentcontract/... -run TestProofAgentMemoryEndpointCoverageHardThresholds -count=1 -v

# Phase 11H agent contract obedience and memory-use telemetry.
.PHONY: test-agent-contract-obedience agent-contract-obedience-benchmark proof-agent-contract-obedience
test-agent-contract-obedience:
	cd control-plane && go test ./internal/agentobedience/... -count=1

agent-contract-obedience-benchmark:
	@mkdir -p artifacts
	cd control-plane && AGENT_CONTRACT_OBEDIENCE_BENCHMARK=1 go test ./internal/agentobedience/... -run TestAgentContractObedienceBenchmarkArtifact -count=1 -v

proof-agent-contract-obedience:
	@mkdir -p artifacts
	cd control-plane && PROOF_AGENT_CONTRACT_OBEDIENCE=1 go test ./internal/agentobedience/... -run TestProofAgentContractObedienceHardThresholds -count=1 -v

# Phase 11I agent telemetry persistence and live loop integration.
.PHONY: test-agent-telemetry-persistence agent-telemetry-persistence-benchmark proof-agent-telemetry-persistence
test-agent-telemetry-persistence:
	cd control-plane && go test ./internal/agenttelemetry/... -count=1

agent-telemetry-persistence-benchmark:
	@mkdir -p artifacts
	cd control-plane && AGENT_TELEMETRY_PERSISTENCE_BENCHMARK=1 go test ./internal/agenttelemetry/... -run TestAgentTelemetryPersistenceBenchmarkArtifact -count=1 -v

proof-agent-telemetry-persistence:
	@mkdir -p artifacts
	cd control-plane && PROOF_AGENT_TELEMETRY_PERSISTENCE=1 go test ./internal/agenttelemetry/... -run TestProofAgentTelemetryPersistenceHardThresholds -count=1 -v

# Phase 11J automatic recall telemetry hooks and Postgres persistence proof.
.PHONY: test-automatic-recall-telemetry automatic-recall-telemetry-benchmark proof-automatic-recall-telemetry
.PHONY: test-agent-telemetry-postgres agent-telemetry-postgres-benchmark proof-agent-telemetry-postgres

test-automatic-recall-telemetry:
	cd control-plane && go test ./internal/agenttelemetry/... -run 'TestAuto|TestRecordAuto|TestEvaluateRejects|TestTelemetry|TestHostileAuto' -count=1

automatic-recall-telemetry-benchmark:
	@mkdir -p artifacts
	cd control-plane && AUTOMATIC_RECALL_TELEMETRY_BENCHMARK=1 go test ./internal/agenttelemetry/... -run TestAutomaticRecallTelemetryBenchmarkArtifact -count=1 -v

proof-automatic-recall-telemetry:
	@mkdir -p artifacts
	cd control-plane && PROOF_AUTOMATIC_RECALL_TELEMETRY=1 go test ./internal/agenttelemetry/... -run TestProofAutomaticRecallTelemetryHardThresholds -count=1 -v

test-agent-telemetry-postgres:
	cd control-plane && go test -tags=integration ./internal/agenttelemetry/... -run 'TestPostgres' -count=1

agent-telemetry-postgres-benchmark:
	@mkdir -p artifacts
	cd control-plane && AGENT_TELEMETRY_POSTGRES_BENCHMARK=1 TEST_PG_DSN="$${TEST_PG_DSN}" go test -tags=integration ./internal/agenttelemetry/... -run TestAgentTelemetryPostgresBenchmarkArtifact -count=1 -v

proof-agent-telemetry-postgres:
	@mkdir -p artifacts
	cd control-plane && PROOF_AGENT_TELEMETRY_POSTGRES=1 TEST_PG_DSN="$${TEST_PG_DSN}" go test -tags=integration ./internal/agenttelemetry/... -run TestProofAgentTelemetryPostgresHardThresholds -count=1 -v

# Phase 11K guarded utility candidate application policy.
.PHONY: test-guarded-utility-policy guarded-utility-policy-benchmark proof-guarded-utility-policy
.PHONY: test-guarded-utility-postgres guarded-utility-postgres-benchmark proof-guarded-utility-postgres

test-guarded-utility-policy:
	cd control-plane && go test ./internal/utilitypolicy/... -count=1

guarded-utility-policy-benchmark:
	@mkdir -p artifacts
	cd control-plane && GUARDED_UTILITY_POLICY_BENCHMARK=1 go test ./internal/utilitypolicy/... -run TestGuardedUtilityPolicyBenchmarkArtifact -count=1 -v

proof-guarded-utility-policy:
	@mkdir -p artifacts
	cd control-plane && PROOF_GUARDED_UTILITY_POLICY=1 go test ./internal/utilitypolicy/... -run TestProofGuardedUtilityPolicyHardThresholds -count=1 -v

test-guarded-utility-postgres:
	cd control-plane && go test -tags=integration ./internal/utilitypolicy/... -run 'TestPostgres' -count=1

guarded-utility-postgres-benchmark:
	@mkdir -p artifacts
	cd control-plane && GUARDED_UTILITY_POSTGRES_BENCHMARK=1 TEST_PG_DSN="$${TEST_PG_DSN}" go test -tags=integration ./internal/utilitypolicy/... -run TestPostgresPolicyProofArtifact -count=1 -v

proof-guarded-utility-postgres:
	@mkdir -p artifacts
	cd control-plane && PROOF_GUARDED_UTILITY_POSTGRES=1 TEST_PG_DSN="$${TEST_PG_DSN}" go test -tags=integration ./internal/utilitypolicy/... -run TestProofGuardedUtilityPostgresHardThresholds -count=1 -v

.PHONY: verify-integrations-static verify-integrations-mcp

verify-integrations-static:
	scripts/integrations/verify-cursor-pack.sh --static
	scripts/integrations/verify-claude-code-plugin.sh --static
	scripts/integrations/verify-vscode-extension.sh --static
	scripts/integrations/verify-generic-mcp.sh --static
	scripts/integrations/verify-mcp-surface.sh --static
	scripts/integrations/verify-housekeeping-enforcement.sh --static

verify-integrations-mcp:
	PLURIBUS_BASE_URL="$${PLURIBUS_BASE_URL:-http://127.0.0.1:8123}" scripts/integrations/verify-mcp-surface.sh
