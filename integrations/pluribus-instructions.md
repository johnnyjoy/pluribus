# Pluribus — mandatory loop

**Pluribus** is the governed memory control plane. Chat is not durable memory until ingested; **probationary memory** can be formed **at ingest** when the summary carries deterministic learning signals.

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

## Record (after outcomes)

Run **`record_experience`** after you:

- Land a fix or merge-worthy change
- Resolve a non-obvious bug or test failure
- Make or confirm a design or product decision in chat

**Do not** skip **`record_experience`** because recall returned little or nothing—**empty recall ≠ skip record**.

## Memory formation at ingest

**`record_experience`** (same path as **`mcp_episode_ingest`**) **POSTs `/v1/advisory-episodes`**. The server **immediately** qualifies **plausible** experience: keyword signals, **`mcp:event:*`**, experiment/benchmark language, or rich situational text can create **probationary** `memories` at **authority 1–2** (advisory applicability) and link the ingest row. **Ranking** separates strong from weak over time. **Clear noise** is stored only as **`advisory_experiences`** with **`memory_formation_status: rejected`** — not memory.

## Doctrine (Pluribus)

- Memory is **tags + situation**; do not treat **project**, **workspace**, **task**, or **scope** as required memory partitions for recall or enforcement truth (Pluribus **anti-regression** and **memory-doctrine** in this repo).
- Legacy tool names: **`memory_context_resolve`**, **`mcp_episode_ingest`** — same **Pluribus** constraints.

## When Pluribus MCP is not available

State once: *Pluribus MCP not in tool list; proceeding without recall/record.* Then continue—**do not** pretend you ran tools you cannot call.
