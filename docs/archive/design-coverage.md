ARCHIVED — NOT ACTIVE SYSTEM TRUTH

---

# Design coverage

This document maps the control-plane design (see [control-plane-design-and-starter.md](control-plane-design-and-starter.md) and the HTML extract) to implementation status so optional or deferred items (e.g. Redis, optional backend synthesis) are visible and not missed.

**Legend:** ✅ done | 🟡 partial | ⏳ not started | ➖ N/A

---

## 1. What this system is

| Item | Status | Notes |
|------|--------|--------|
| Memory-governed execution control plane | ✅ | Implemented |
| Durable external cognition layer | ✅ | Postgres + evidence + recall/drift/curation |
| Define target, store memory, compile recall, drift check, evidence | ✅ | All present |

---

## 2. The problem it solves

| Item | Status | Notes |
|------|--------|--------|
| Replace chat-as-state with structured memory and recall | ✅ | Core loop implemented |
| Stay on target; explicit memory | ✅ | Target, task, recall bundle, drift |

---

## 3. Core operating principles (3.1–3.8)

| Principle | Status | Notes |
|-----------|--------|--------|
| 3.1 No important state in chat | ✅ | State in Postgres/evidence |
| 3.2 Immediate memory is compiled | ✅ | Recall compiler |
| 3.3 Durable memory is explicit | ✅ | memory_objects, tags, links |
| 3.4 Promotion is selective | ✅ | Curation service, promote/reject |
| 3.5 Current project governs | ✅ | Recall/scoping by project |
| 3.6 Evidence outranks vibes | ✅ | Evidence service; authority/scoring |
| 3.7 Fast by default, slow when risky | 🟡 | Preflight + cache; full “slow path” not wired |
| 3.8 Model proposes; system decides | ✅ | Optional backend synthesis for run-multi; curation promotes |

---

## 4. Architecture

| Item | Status | Notes |
|------|--------|--------|
| Operator → controlplane API | ✅ | chi router, /v1/* |
| projects, targets, memory, tasks, recall, drift, evidence | ✅ | All services |
| optional backend synthesis (run-multi) | ✅ | `internal/synthesis` — Ollama / OpenAI / Anthropic; single process |
| Modular monolith | ✅ | Primary surface: single `controlplane` binary |

---

## 5. Service model

| Service | Status | Notes |
|---------|--------|--------|
| Projects, Targets, Tasks, Memory, Curation, Recall, Drift, Evidence | ✅ | Implemented |
| Optional backend synthesis | ✅ | Config-gated; Ollama / OpenAI / Anthropic for run-multi only |

---

## 6–9. Memory taxonomy, scope, authority, applicability

| Item | Status | Notes |
|------|--------|--------|
| Memory kinds (decision, constraint, failure, pattern, candidate, evidence, contradiction) | ✅ | Types and scoring |
| Scope model | ✅ | Scope fields and recall filtering |
| Authority 1–10 | ✅ | Used in recall/drift/curation |
| Applicability (governing, advisory, analogical, experimental) | ✅ | In memory model |

---

## 10. Storage model

| Store | Status | Notes |
|-------|--------|--------|
| PostgreSQL (durable truth) | ✅ | All core tables + migrations |
| Filesystem (evidence blobs) | ✅ | Evidence storage by digest |
| **Redis (optional cache only)** | ✅ | **Done.** internal/cache (Store, RedisStore), keys for recall bundle + preflight. Config: redis.enabled, recall.cache_ttl_seconds. Not authoritative. |

---

## 11. Full repo layout

| Area | Status | Notes |
|------|--------|--------|
| cmd/ controlplane | ✅ | Canonical single server process |
| internal/ projects, targets, tasks, memory, curation, recall, drift, evidence, tooling, **synthesis**, **cache** | ✅ | cache added |
| configs, migrations, scripts | ✅ | Present |

---

## 12–15. Config and build

| Item | Status | Notes |
|------|--------|--------|
| YAML config (server, postgres, auth, evidence, recall, curation, optional synthesis) | ✅ | app.Config |
| **redis, recall.cache_ttl_seconds** | ✅ | In config and example |
| Makefile, go.mod | ✅ | Build and deps |

---

## 16. Migrations

| Item | Status | Notes |
|------|--------|--------|
| 0001–0011 (projects through recall_bundles) | ✅ | Applied in order |

---

## 17–20. Shared and app code

| Item | Status | Notes |
|------|--------|--------|
| pkg/api (enums, Scope) | ✅ | Used |
| app LoadConfig, Boot (DB, EvidenceRoot, APIKeys, **Cache**) | ✅ | Redis optional in Boot |
| httpx (router, middleware, JSON) | ✅ | Used |
| obs (logger) | 🟡 | If present; metrics not fully wired |

---

## 21–28. Domain packages

| Package | Status | Notes |
|---------|--------|--------|
| projects, targets, tasks | ✅ | CRUD + handlers |
| memory (Create, Search) | ✅ | Repo + service |
| curation (Evaluate, ListPending, MarkPromoted/Rejected) | ✅ | Salience config |
| recall (Compile, Preflight, **cache integration**) | ✅ | Cache get/set in Compile and Preflight |
| drift (Check) | ✅ | Checker + repo |
| evidence (Create, storage) | ✅ | By kind+digest |
| tooling (gopls client for recall/drift) | ✅ | In-process in `controlplane`; `internal/tooling` |
| synthesis (**optional run-multi LLM**) | ✅ | Ollama / OpenAI / Anthropic adapters; see [../control-plane/docs/backend-synthesis.md](../control-plane/docs/backend-synthesis.md) |

---

## 29–34. Binaries

| Binary | Status | Notes |
|--------|--------|--------|
| controlplane | ✅ | All routes, optional Redis, in-process LSP client, optional `synthesis` for run-multi |

---

## 35–38. Scripts, first-run, workflow, Cursor usage

| Item | Status | Notes |
|------|--------|--------|
| dev-up, migrate (obsolete stub) | ✅ | DSN, postgres (and optional redis); schema in Boot |
| First-run (DB, migrate, run servers) | ✅ | Documented |
| Example workflow (create project/target/task, recall, drift, candidates) | ✅ | README/scripts |
| Cursor usage pattern | ✅ | [docs/ai-recall-cursor-focus.md](ai-recall-cursor-focus.md), [docs/cursor-verify-recall.md](cursor-verify-recall.md) |

---

## 39–42. Verdict and “what’s next”

| Item | Status | Notes |
|------|--------|--------|
| Real repo, schema, CRUD, recall compiler, drift checker, HTTP API (+ MCP proxy) | ✅ | No bundled shell CLI; use **curl** / **pluribus-mcp** |
| **Redis cache** | ✅ | Implemented (optional) |
| **Optional backend synthesis** | ✅ | Implemented (`synthesis` config; providers Ollama, OpenAI, Anthropic) |
| Promotion pipeline, evidence linking, target/task updates | 🟡 | Partial (promote exists; evidence link endpoints may be minimal) |
| Contradiction detection, multi-agent | ⏳ | Partial / evolving |
| **LSP (OE-4)** | ✅ | gopls via `tooling.GoplsClient` in `controlplane` (recall symbol boost, drift ref risk) |

---

## Optional / deferred (from HTML extract)

| Design item | Status | Notes |
|-------------|--------|--------|
| Redis for hot recall bundles, preflight, tag index | ✅ | Recall + preflight + memory tag search (OE-1) cached |
| Single-store cache backend (no fallback chains) | ✅ | One Redis client; no chain |
| “Do not make Redis authoritative” | ✅ | Cache is read-through only; Postgres is truth |
| Optional backend synthesis for run-multi | ✅ | `synthesis.enabled`; no separate reasoner service |

---

## How to use this doc

- **Before a release or “design complete” check:** Walk sections 1–42 and confirm status matches reality.
- **When adding a feature:** Add a row or update status so the next reviewer sees it.
- **When design doc changes:** Update the corresponding rows and add any new deferred items to “Optional / deferred.”
