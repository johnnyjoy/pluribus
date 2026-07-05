---
name: Pluribus
description: Pluribus — mandatory recall/record when MCP tools exist; memory-first tags + situation only
alwaysApply: true
---

# Pluribus — mandatory loop (when MCP tools are available)

**Pluribus** is the governed memory control plane. Chat and logs are not durable memory until distilled and curated.

**If** your available tools include Pluribus (`recall_context`, `record_experience`, or HTTP/RPC equivalents your client documents for this server), **then**:

## Recall (before acting)

Run **`recall_context`** (tags + `retrieval_query` / situation text) **before** editing or recommending changes when the task involves any of:

- Multiple files, refactors, or new modules
- API, schema, migrations, auth, or enforcement behavior
- Debugging failing tests, CI, or production-like incidents
- Security, privacy, or compliance-sensitive changes
- Anything you would need to explain to a new teammate in more than two sentences

**Do not** skip **Pluribus** recall because the task “seems small” once you have read files—if it matches the list, **`recall_context`** first.

## Housekeeping (when chores exist)

After recall/wakeup, if **`housekeeping`** is present or **`list_chores`** is non-empty: call **`resolve_chore`** with **`chore_id`**, **`action`**, and **`agent_id`** when you can judge; otherwise defer with reason in **`record_experience`**. See **[`pluribus-instructions.md`](../pluribus-instructions.md#housekeeping-when-chores-exist)**.

## Curate (utility feedback)

When recall items helped or misled you: **`memory_feedback`** (`helpful`, `harmful`, `wrong`, `outdated`). No vote is neutral.

## Record (after outcomes)

Run **`record_experience`** after you:

- Land a fix or merge-worthy change
- Resolve a non-obvious bug or test failure
- Make or confirm a design or product decision in chat

**Do not** skip **`record_experience`** because recall returned little or nothing—**empty recall ≠ skip record**.

## Doctrine (Pluribus)

- Memory is **tags + situation**; do not treat **project**, **workspace**, **task**, or **scope** as required memory partitions for recall or enforcement truth (Pluribus **anti-regression** / **memory-doctrine** in this repo).
- Legacy tool names: **`memory_context_resolve`**, **`mcp_episode_ingest`** — same **Pluribus** constraints.

## When Pluribus MCP is not available

State once: *Pluribus MCP not in tool list; proceeding without recall/record.* Then continue—**do not** pretend you ran tools you cannot call.
