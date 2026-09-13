# Pluribus — mandatory loop

**Pluribus** is the governed memory control plane. Chat is not durable memory until ingested; **probationary memory** can be formed **at ingest** when the summary carries deterministic learning signals.

> **Compliance (Phase 2):** These instructions guide agent behavior. **Compliance is measured by Pluribus telemetry** (`compliance_evaluate`, `/v1/compliance/*`), but clients may still skip tools unless the client runtime enforces them. Pluribus does not block local file edits outside MCP.

**If** your available tools include Pluribus (`recall_context`, `record_experience`, or HTTP/RPC equivalents your client documents for this server), **then**:

## Recall (before acting)

Run **`recall_context`** (tags + `retrieval_query` / situation text) **before** editing or recommending changes when the task involves any of:

- Multiple files, refactors, or new modules
- API, schema, migrations, auth, or enforcement behavior
- Debugging failing tests, CI, or production-like incidents
- Security, privacy, or compliance-sensitive changes
- Anything you would need to explain to a new teammate in more than two sentences

**Do not** skip **Pluribus** recall because the task “seems small” once you have read files—if it matches the list, **`recall_context`** first.

**Do not** defer recall on substantive work when **Pluribus** MCP is connected—no “I’ll recall later” for multi-file refactors, architecture or API shifts, incident investigation, or non-trivial feature work.

**Clock / last-time:** pass RFC3339 **`occurred_after` / `occurred_before`** on **`recall_context`**. The server does not parse “last Friday.” Write **`occurred_at`** on **`record_experience`** when you know when it happened. Write one dense claim, not an essay.

## Assemble working context (you curate)

Recall returns **candidates you can use**, not finished context. Prefer **`agent_grounding`**: each line is **`[memory_id] statement`**. **You** apply what belongs to this task:

1. Read Continuity / Constraints / Experience. Treat a line as usable only if it constrains **this** task.
2. Discard cross-project, unused-looking, or low-applicability junk.
3. Act using the kept statements (constraints first, then decisions, failures, patterns).
4. After you **used** a line, keep its **`memory_id`** for **`used_memory_ids`** (upvote is the receipt of use).
5. Do **not** treat the raw JSON dump as the working context.

## Housekeeping (when chores exist)

After **`recall_context`** or **`wakeup_context`**, check for **pool maintenance** (open curation chores). The server surfaces at most one line in **`mcp_context.housekeeping`** or **`housekeeping`**; you may also call **`list_chores`**.

**If** a chore is present:

1. Read the chore (type, statement snippets, allowed **`actions`**).
2. **If you can judge:** call **`resolve_chore`** with **`chore_id`**, **`action`**, and **`agent_id`** (required — use a stable client id, e.g. `cursor:<hostname>`, `claude-code:<hostname>`).
3. **If you cannot judge** (insufficient context, need a human): do **not** vote randomly; note why you deferred in the next **`record_experience`**.
4. **Corroboration:** **`min_resolvers`** distinct **`agent_id`** hashes must agree on the same action (default **1** on a solo hive; operators may raise it). A memory's **own author** never counts toward the threshold.
5. **Actions:** `quarantine_review` → **`release`** (→ `pending`, never `active`) or **`delete`**; `contradiction` → **`keep_subject`** / **`keep_related`** / **`coexist`**; `duplicate_pair` → **`consolidate`** or **`distinct`**.

**Do not** skip housekeeping because substantive work feels urgent — one tool call when you can judge helps every future agent on the shared pool.

**Do not** treat empty chores as failure — when the pool is clean, this step is a no-op.

## Proof and smoke rows (automation hygiene)

When **you** create throwaway test or demo memories (CI receipts, local smoke, scenario markers), tag them **`ephemeral`** plus **`proof-scenario`** or **`smoke-shared-memory`**. Do **not** leave durable-looking constraints in the shared pool without ephemeral tags — agents use **`memory_feedback`** and chores if any slip through.

## Curate (utility feedback)

After you act on work informed by **`recall_context`** or **`wakeup_context`**:

1. **Used:** pass those **`memory_id`**s as **`used_memory_ids`** on **`record_experience`**. The server applies **`helpful`** (upvote). Do **not** also call **`memory_feedback`** with **`helpful`** for the same IDs.
2. **Misled / wrong / stale:** **`memory_feedback`** with **`harmful`**, **`wrong`**, or **`outdated`** (reason required).
3. **Recalled but unused:** no upvote.

This is how agents on a trusted network police the shared pool without a separate curator AI. Ranking uses utility over time; chores handle structural mess (dupes, contradictions, quarantine).

## Record (after outcomes)

Run **`record_experience`** after you:

- Land a fix or merge-worthy change
- Resolve a non-obvious bug or test failure
- Make or confirm a design or product decision in chat

**Do not** skip **`record_experience`** because recall returned little or nothing—**empty recall ≠ skip record**.

## Memory formation at ingest

**`record_experience`** (same path as **`mcp_episode_ingest`**) **POSTs `/v1/advisory-episodes`**. Write one dense claim, not an essay. Optional **`occurred_at`** (RFC3339). The server **immediately** qualifies **plausible** experience: keyword signals, **`mcp:event:*`**, experiment/benchmark language, or rich situational text can create **probationary** `memories` at **authority 1–2** and link the ingest row. **Ranking** separates strong from weak over time. **Clear noise** is rejected — not memory. Clock/last-time questions on **`recall_context`** read these experiences.

## Doctrine (Pluribus)

- Memory is **tags + situation**; do not treat **project**, **workspace**, **task**, or **scope** as required memory partitions for recall or enforcement truth (Pluribus **anti-regression** and **memory-doctrine** in this repo).
- Legacy tool names: **`memory_context_resolve`**, **`mcp_episode_ingest`** — same **Pluribus** constraints.

## When Pluribus MCP is not available

State once: *Pluribus MCP not in tool list; proceeding without recall/record.* Then continue—**do not** pretend you ran tools you cannot call.
