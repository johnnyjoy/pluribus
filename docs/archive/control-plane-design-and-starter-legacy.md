ARCHIVED — NOT ACTIVE SYSTEM TRUTH — superseded by [http-api-index.md](../http-api-index.md), [api-contract.md](../api-contract.md) (subset), and [memory-doctrine.md](../memory-doctrine.md).

---

# Control-plane design and starter implementation (legacy)

Full consolidated design + starter implementation for a Go-based long-horizon AI memory/recall system. This document is the version you can lift into Cursor and start turning into a real repo.

> **Operational truth:** For **current** build targets, Docker/compose, and schema apply (**`app.Boot`** runs embedded `migrations/*.sql` on each start; fresh Postgres), use **[control-plane/README.md](../control-plane/README.md)**, **[deployment-poc.md](deployment-poc.md)**, and root **[README.md](../README.md)** as authoritative. This file keeps **design intent** and an early tree sketch; lower sections may lag.

---

# 1. What this system is

This system is a **memory-governed execution control plane** for AI-assisted engineering.

It is not a chatbot. It is not a giant transcript. It is not "AI memory" in the mystical sense.

It is a **durable external cognition layer** whose job is to:

* define target
* store durable project memory
* capture candidate learnings
* compile immediate memory for a task
* check for drift before acceptance
* store evidence
* support one worker today and many later

The model is **stateless**. The system is **stateful**. That distinction is the center of gravity.

---

# 2. The problem it solves

LLMs fail on large work for predictable reasons: no durable recall, no authority hierarchy, no target lock, no memory curation, repeated rediscovery, repeated mistakes, drift from original goal, collapse into generic textbook patterns, chat history becomes the accidental database.

This system fixes that by replacing "chat as state" with: structured memory, task-oriented recall, explicit target, drift checks, tool-grounded evidence.

> **Stay on target.** And do it with explicit memory, not transcript sludge.

---

# 3. Core operating principles

## 3.1 No important state lives in chat
Chat is an interface, never the source of truth.

## 3.2 Immediate memory is compiled
It is not stored directly. It is assembled from: target, task, durable memory, relevant failures, relevant patterns, current local context.

## 3.3 Durable memory is explicit
Everything important is a typed object.

## 3.4 Promotion is selective
Not every event becomes memory. Most things should die.

## 3.5 Current project governs
Past projects can inform but do not override current governing memory.

## 3.6 Evidence outranks vibes
Anything backed by tests, logs, benchmarks, diffs, or actual outputs should rank higher than unsupported prose.

## 3.7 Fast by default, slow when risky
Routine work should be cheap. Risky work should force deeper recall and validation.

## 3.8 The model proposes; the system decides
The LLM can propose code and candidate memories. It does not get to silently define truth.

---

# 4. Architecture

```text
Operator / Cursor / Future Orchestrator  |  v  controlplane API  |  +-----------+-----------+-----------+  |  |  |  |  v  v  v  v
 projects  targets  memory  tasks  |  |  |  +-----+-----+-----+-----+  |  |  v  v  recall  drift  |  |  +-----+-----+  |  v  evidence  |  v  optional backend synthesis for run-multi  |  v  local or remote LLM (when enabled)
```

Begin as a **modular monolith** for controlplane, not a microservice circus. Split later if needed.

---

# 5. Service model

Expose as logical services even if initially in one binary: **Projects** (identity, scope), **Targets** (what we're achieving now), **Tasks** (bounded work), **Memory** (durable objects), **Curation** (candidate evaluation and promotion), **Recall** (immediate memory compilation), **Drift** (proposal validation), **Evidence** (proof artifacts), **Reasoner** (optional thin wrapper around model backend). Optional **LSP** (gopls) for symbol/reference signals is wired **inside** `controlplane` via `internal/tooling`, not a separate HTTP tool service.

---

# 6. Memory taxonomy

**Decision** — A chosen path (e.g. "Query cookbook is canonical.").  **Constraint** — A rule (e.g. "Do not maintain duplicate query-building paths.").  **Failure** — Reusable bad outcome (e.g. "Prior refactor broke fluent terminators.").  **Pattern** — Reusable successful approach (e.g. "Single-path builder authority.").  **Candidate** — Possible future memory object.  **Evidence** — Proof artifact.  **Contradiction** — Explicit record of conflict between memories or proposal and law.

---

# 7. Scope model

Every memory object must be scoped. Allowed: `global`, `project`, `domain`, `subsystem`, `task`, `symbol`, `file`, `session`. Examples: naming law = `project`; query builder rule = `domain=query`; one-off migration = `task`; failure tied to one class = `symbol`. Without scope, recall turns into soup.

---

# 8. Authority model

Scale 1–10: 1 weak suggestion, 3 candidate-level weak, 5 promoted local lesson, 7 strong reusable project memory, 8 near-governing, 9 canonical/governing, 10 constitutional. Authority affects recall ranking, contradiction resolution, drift checks, promotion thresholds.

---

# 9. Applicability model

**governing** | **advisory** | **analogical** | **experimental**. Current project constraint = governing; prior project pattern = analogical. Stops past project memories from hijacking current project.

---

# 10. Storage model

**PostgreSQL** — Primary durable truth: projects, targets, tasks, memory_objects, tags, links, candidate_events, evidence metadata, drift_checks, recall_bundles.  **Filesystem** — Evidence blobs (logs, diffs, test output, build output).  **Redis later** — Optional hot cache for recall bundles, active target; not authoritative.

---

# 11. Full repo layout

```text
control-plane/  README.md  Makefile  go.mod  /cmd  controlplane  /configs  config.example.yaml  /migrations  0001_projects.sql .. 0011_recall_bundles.sql  /internal  /app  config.go  boot.go  /httpx  response.go  middleware.go  router.go  /obs  logger.go  /projects  types.go  repo.go  service.go  handlers.go  /targets  types.go  repo.go  service.go  handlers.go  /tasks  types.go  repo.go  service.go  handlers.go  /memory  types.go  repo.go  search.go  service.go  handlers.go  /curation  types.go  salience.go  repo.go  service.go  handlers.go  /recall  types.go  preflight.go  repo.go  compiler.go  service.go  handlers.go  /drift  types.go  checker.go  repo.go  service.go  handlers.go  /evidence  types.go  storage.go  repo.go  service.go  handlers.go  /tooling  (gopls client, helpers)  /reasoning  types.go  client.go  /pkg/api  enums.go  common.go  /scripts  dev-up.sh  migrate.sh  /var/evidence
```

---

# 12–15. Config and build

**go.mod:** `module control-plane`, go **1.22**, chi v5, uuid, lib/pq, yaml.v3. *(Verify current `go.mod` in repo.)*

**README:** Control Plane = Go-based external cognition layer. Services: projects, targets, tasks, durable memory, candidate memory, recall compilation, drift checking, evidence. Core loop: Target → Task → Recall → Reason → Drift → Curate → Repeat. **Binaries (current Makefile):** controlplane, **pluribus-mcp**. Storage: Postgres (durable), filesystem (evidence), Redis optional. Principles: current project governs, evidence beats vibes, immediate memory is compiled, model is stateless.

**Makefile:** `make build` — controlplane, pluribus-mcp; test, clean.

**config.example.yaml:** server.bind, synthesis (optional run-multi backend: enabled, provider, model, …), postgres.dsn, evidence.root_path, recall.default_max_items_per_kind, curation (candidate_threshold, review_threshold, promote_threshold). HTTP auth: **`PLURIBUS_API_KEY`** in the environment (see `internal/httpx/auth.go`).

---

# 16. Migrations (summary)

0001_projects, 0002_targets, 0003_tasks, 0004_memory_objects, 0005_memory_tags, 0006_memory_links, 0007_candidate_events, 0008_evidence_records, 0009_memory_evidence_links, 0010_drift_checks, 0011_recall_bundles. Full SQL for each is in the original design paste or must be generated from the schema described in sections 21–28 (projects, targets, tasks, memory, curation, recall, drift, evidence).

---

# 17–20. Shared and app code

**pkg/api:** enums (MemoryKind, ScopeType, Applicability, Status), Scope struct.  **internal/app:** Config (LoadConfig from YAML), Boot (DB, EvidenceRoot, APIKeys).  **internal/httpx:** WriteJSON, WriteError, APIKeyMiddleware, NewRouter + healthz.  **internal/obs:** NewLogger.

---

# 21–28. Domain packages (signatures and behavior)

**projects:** Project, CreateRequest; Repo Create/GetBySlug/GetByID/List; Service Create/GetByID/GetBySlug/List; Handlers CRUD + by-slug.  **targets:** Target (Goal, NonGoals, SuccessCriteria, ConstraintsSummary, Phase, Priority); Repo Create/GetByID/ListByProject/GetActiveByProject; Service same; Handlers Create/GetByID/ListByProject/GetActiveByProject.  **tasks:** Task (ProjectID, TargetID, Title, Description, Status, Priority, RiskLevel, Payload); Repo Create/GetByID/ListByProject; Service Create/GetByID/ListByProject; Handlers Create/GetByID/ListByProject.  **memory:** MemoryObject (Kind, Scope, Authority, Applicability, Statement, Tags, …); Repo Create + Search (by project, tags, status=active); Service Create/Search; Handlers Create/Search.  **curation:** CandidateEvent; SalienceConfig, ScoreText (directives, canonical, failure signals, speculation); Repo Create/ListPendingByProject/UpdatePromotionStatus; Service Evaluate/ListPending/MarkPromoted/MarkRejected; Handlers Evaluate/ListPending/MarkPromoted/MarkRejected.  **recall:** CompileRequest, RecallBundle (Target, Task, GoverningConstraints/Decisions, KnownFailures, ApplicablePatterns); PreflightRequest/Result (risk level, required actions); Compiler (Memory, Tasks, Targets) Compile; Repo CreateBundle; Service Compile/Preflight; Handlers Compile/Preflight.  **drift:** CheckRequest (Proposal), CheckResult (Passed, Violations, Warnings); checker (duplicate responsibility, fluent regression); Repo CreateCheck; Service Check; Handlers Check.  **evidence:** Record, CreateRequest; Storage Save (digest, path); Repo Create; Service Create; Handlers Create.  **tooling:** gopls-backed LSP client used by recall/drift when enabled; legacy git/rg/test helpers remain in package for tests.  **reasoning:** GenerateRequest/Response; Client Generate; Handlers Generate (`cmd/reasoner`).

---

# 29–34. Binaries

**cmd/controlplane:** Load config, Boot container, wire projects/targets/tasks/memory/curation/recall/drift/evidence handlers under /v1, APIKey middleware, ListenAndServe; optional run-multi backend synthesis via `internal/synthesis` when `synthesis.enabled` (single process).

---

# 35. Scripts

**scripts/dev-up.sh:** Reminder to start postgres then run controlplane. **scripts/migrate.sh:** Obsolete stub; schema runs in **server `Boot`** only.

---

# 36. First-run sequence

Create DB `controlplane`, make build, run controlplane (schema on first boot), curl healthz on 8123.

---

# 37. Example workflow

Seed memory (`POST /v1/memory`). Compile recall (`POST /v1/recall/compile` with tags + situation text). Drift check (`POST /v1/drift/check`). List pending candidates (`GET /v1/curation/pending` — no query params). See archived note at top for current docs.

---

# 38. Cursor usage pattern

Compile a recall bundle for the situation. Give the agent: recall bundle, relevant files, and a bounded change description. Run drift check. If a lesson emerged, run curation digest → materialize. Promote only useful memories. That stops long work from becoming transcript soup.

---

# 39. What this gives you now

Real repo layout, real schema, real CRUD beginnings, persistent projects/targets/tasks/memory/candidates/recall bundles/drift checks/evidence metadata, functional recall compiler, functional drift checker. Enough to start operating.

---

# 40. What is still intentionally incomplete

Candidate-to-memory promotion workflow, stronger memory search filtering, target/task update flows, contradiction detection, evidence linking to memory objects, real LLM backend wiring, tool allowlists and safety controls, LSP integration, task claiming and multi-agent. Phase-two and beyond.

---

# 41. What to build next

Highest-value next: promotion pipeline (candidate → durable memory), memory search by kind/scope/authority, evidence link endpoints, list/read for evidence/drift/recall bundles, target update, task status updates, optional backend synthesis tuning, tool-api safety guardrails.

---

# 42. Final verdict

This is a **serious, buildable, large-scale starting point**: not browser-bound, not transcript-bound, not fake-person AI, not "just use more context", not dependency soup, not hype. It is a concrete external memory-and-recall control plane for long-horizon AI work.
