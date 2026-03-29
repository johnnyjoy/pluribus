ARCHIVED — NOT ACTIVE SYSTEM TRUTH

---

# Gap analysis: *AI Coding and Recall* vs Recall

**Date:** 2026-03-22  
**Source document:** `AI Coding and Recall.html` (repo **root** — there is no copy under `docs/`).  
**Plain-text extraction:** `docs/ai-coding-and-recall-extracted.txt` (HTML stripped; note the export still begins with ChatGPT UI chrome before the substantive thread).

**Goal framing (from the source thread):** A **learning memory system** for AI-assisted coding — not “more context,” but **durable, structured, queryable continuity**: constitution → active/work → decisions/constraints → **evidence-backed** recall, with **curation**, **drift control**, and **network-accessible** services so the model stays a **stateless client** of truth.

---

## 1. What the source document prescribes (condensed)

| Theme | Intent |
|--------|--------|
| **Four (+1) layers** | Constitution, current state, work orders, **queryable evidence**; memory as **curated operational** units, not a diary. |
| **Retrieval hierarchy** | Constitution > current work order > recent decisions > evidence > codebase search; **discipline** to load slices before coding. |
| **Decision objects** | Typed, scoped, evidence-linked, promotable, supersede-able — **authority** and ranking, not flat text. |
| **Cognitive control plane** | Memory authority, curation, recall, constraints, **drift** — **external** to the LLM. |
| **Tooling** | LSP = code reality; MCP = nervous system; CLI/git = evidence acquisition; vector DB = **episodic only**, not truth; filesystem = reviewable substrate. |
| **Curation loop** | After meaningful work: what changed, what was learned, link evidence, promote when validated — **quality** determines success. |
| **Service architecture** | Memory, recall, evidence, drift (and sketched curation/target/skill services) **queryable over the network**. |
| **Anti-patterns** | Storing every turn, unchecked model writes to canonical memory, vector-first retrieval, speculation as truth. |

**Product boundary (this repo):** The source table uses “LSP = code reality” in a **conceptual** sense. In **Pluribus**, **editor** LSP remains **gopls** (local). The service uses an **in-process LSP client** only for server-side recall/drift enrichment; it does **not** expose an LSP server to the IDE. **Cursor/agents** integrate via **MCP/HTTP** — [pluribus-lsp-mcp-boundary.md](pluribus-lsp-mcp-boundary.md).

---

## 2. What we have implemented (aligned)

### 2.1 Network-accessible control plane

- **Go `control-plane`** with HTTP **v1** APIs: projects, **recall** (`compile`, `compile-multi`, `GET` bundle, `preflight`, **`run-multi`**), **memory** (`create`, `search`, **`promote`**), **drift** (`check`), **ingest** (`cognition`, commit), **evidence** linking/scoring paths used in promotion.
- **Postgres** as durable store; **Redis** where used; migrations on startup; **`/readyz`** for operational readiness.
- **Thin MCP server** (`pluribus-mcp`) exposing core operations to clients (e.g. Cursor).

### 2.2 Memory model close to “decision objects”

- Structured **kinds** (constraints, decisions, patterns, failures, etc.), **tags**, **authority**, **scope**, promotion workflow including **`evidence_ids`**, **policy composite / gates** (Hive Phase 9), **`require_review` → pending** paths.
- **Contradiction** handling and **RIU-style** applicability/recall intelligence (structured ranking signals, not raw retrieval alone).

### 2.3 Recall + “compile immediate memory”

- **Recall compilation** and **run-multi** orchestration with merge, debug surfaces, and integration with promotion policy.
- **LSP integration** (Hive Phase 8): auto symbols, reference counts feeding recall — matches the doc’s “anchor memory to real symbols.”

### 2.4 Drift and slow-path discipline

- **Drift check** service and wiring through **compile-multi / preflight / run-multi** flows (Hive Phase 7+), including follow-up checks where designed.

### 2.5 File-system substrate (documented)

- Repo documents **`constitution.md`**, **`active.md`**, **`workorders/`**, optional **`evidence/`** — adjacent file layout; **authoritative** memory is **control-plane + DB** per README.

### 2.6 Operational memory culture (docs)

- **`docs/memory-curation.md`**, **`docs/retrieval-order.md`**, work-order templates — align with “curated operational memory” and hierarchy **in principle** (see gaps below for wiring).

---

## 3. What we have *not* done (or only partially)

| Gap | Source idea | Current state |
|-----|-------------|---------------|
| **Evidence in the retrieval bundle (step 5)** | Load **evidence for affected files** before coding. | **Optional:** `recall.evidence_in_bundle` adds compact **`supporting_evidence`** to each **`MemoryItem`** ([evidence-in-recall.md](evidence-in-recall.md)); full file-level “affected files” preload remains **not** automatic. **`docs/retrieval-order.md`** orchestrator still implements (1)–(4) only unless extended. |
| **End-to-end curation *loop*** | After every meaningful task: capture learning, link evidence, promote — **habit or automation**. | **APIs and policies** exist; **no first-class “session closer” daemon** or enforced agent skill that always runs post-task (workflow gap). |
| **Automatic “reject proposed change”** | Pre-flight **constraint binding** → validate → allow/reject. | **Drift check** informs; **full automated gate** on arbitrary patches is **not** the same as doc’s ideal **enforce** step. |
| **`decisions.md` append-only log** | Simple **B** in minimal layout. | Project emphasizes **structured memory + DB** via control-plane; **root `decisions.md`** may exist but is **not** the single system of record — intentional divergence, but worth explicit product choice. |
| **Advisory episodic similarity** | “Similar past episodes,” **not** authority. | **Shipped:** `advisory_episodes` + lexical/tag `POST /v1/advisory-episodes/similar` ([episodic-similarity.md](episodic-similarity.md)); must stay **subordinate** to canon in retrieval order. |
| **Evidence *blob* store** | Object storage / large artifacts, hashes, screenshots. | **Metadata + linking + scoring** in-service; **large blob** store and rich artifact pipeline **not** fully spelled out as in the sketch. |
| **Separate “Target service”** | Explicit target/work-order service in SOA diagram. | **Work orders** are **files + docs**; not a dedicated networked target microservice. |
| **Full curation REST surface** | Sketched `POST /curation/evaluate|promote|merge|review`. | **Promotion + ingest** cover parts of this; **not** a standalone curation API family as drawn. |
| **Skills as executable procedures** | Bridge memory ↔ tools ↔ action. | **Lives in Cursor/Codex ecosystem**; **not** versioned inside this repo as runnable procedures bound to control-plane. |
| **Clean extraction artifact** | N/A | Extracted text includes **ChatGPT sidebar/UI** lines at the top; for a **reader-only** artifact, a second pass could drop lines before “You said:”. |

---

## 4. Recommendations (prioritized for a *learning* memory system)

1. **Wire “evidence for affected files” into recall compile**  
   Close the loop the document treats as **non-optional**: given WO “Files involved,” attach **evidence records** (or summaries + links) into the **same bundle** as constraints/decisions, respecting the **Constitution > WO > decisions > evidence** order already documented.

2. **Productize the curation loop**  
   Choose one: **(a)** a **Cursor command + checklist** (human-in-the-loop), **(b)** an **ingest/cognition** convention after each task, or **(c)** a small **worker** that prompts for promotion when `run-multi` or drift returns learning signals. Without this, learning **stagnates** regardless of API quality.

3. **Make “learning” measurable**  
   Define 2–3 **metrics**: promotion rate with `evidence_ids`, contradiction resolution latency, drift warnings per WO — aligned with the doc’s claim that **curation quality** is the bottleneck.

4. **Keep advisory similarity subordinate**  
   Treat `advisory_episodes` / similar-case retrieval as **“have we seen something like this?”** only — **never** ahead of authority-ranked recall in client orchestration.

5. **Clarify files vs control-plane**  
   Document **when** edits go to **workorders / constitution / evidence** vs **`POST /v1/memory`** and other `/v1/*` APIs. Reduces agent confusion: **service is authoritative** for durable memory; repo files are optional human workflow.

6. **Optional: trim the extracted HTML text**  
   Regenerate `docs/ai-coding-and-recall-extracted.txt` starting at the first **“You said:”** / substantive message to produce a **clean archival transcript** for humans and future RAG.

---

## 5. One-line verdict

**You have built the hard part of the vision — a networked cognitive control plane with structured memory, recall, drift, LSP-aware ranking, evidence-aware promotion, and MCP access.** The largest remaining gaps for the **learning** story are **(1) deeper retrieval-orchestration** (e.g. full “affected files” preload; bounded **`evidence_in_bundle`** exists — see §3), **(2) habitual use of the curation digest/materialize path** (the APIs exist; enforcement is organizational), and **(3) operational wiring** — **`POST /v1/enforcement/evaluate`** provides a structured **pre-change gate** against **binding** memory (config **`enforcement.enabled`**, RC1 default **on** when omitted; set **`enabled: false`** to disable; heuristic policy), while **`/v1/drift/check`** remains primarily a **client-interpreted** signal unless CI/agents treat **`passed`** / **`block_execution`** as hard stops; neither blocks git merges by itself.
