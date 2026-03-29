ARCHIVED — NOT ACTIVE SYSTEM TRUTH

---

# Pluribus Focus Layer — Future Sprint Design Brief

## Status

Proposed future sprint.  
Not part of the current proof-of-concept acceptance bar.

---

# Purpose

Capture the idea of a **focus layer** for Pluribus so it can be implemented after the core memory/recall system reaches a more polished and dependable state.

This layer is meant to improve:

- return to focus after interruption
- thread continuity
- short-term working-state recovery
- ability to resume multi-step work without drift
- practical usefulness for real agent workflows

---

# Problem

Even with strong memory and recall, an agent can still lose track of:

- the current objective
- what subtask is in progress
- what was decided in the current line of work
- what is blocked
- what the very next step should be

Durable memory helps preserve knowledge, but it does not automatically provide **working focus**.

That means an agent may still:

- wander after interruption
- restart analysis from too far back
- reload too much context
- fail to distinguish long-term truths from current working state

---

# Core Idea

Introduce a lightweight **focus and thread state layer** on top of the existing Pluribus memory system.

This layer should preserve and restore the practical elements needed to continue work effectively, including:

- what we are doing now
- why we are doing it
- what matters most
- what is blocked
- what should happen next

The intent is not to replace durable memory, but to complement it with a **working-state model**.

---

# Goals

## Primary goals

1. Help an agent resume work after interruption with minimal drift
2. Preserve the current objective and immediate next step
3. Distinguish active working state from long-term memory
4. Improve thread continuity across sessions
5. Reduce the amount of irrelevant context needed to get back on target

## Secondary goals

1. Support better multi-step work over time
2. Improve usefulness for coding, planning, and ongoing project execution
3. Borrow effective ideas from focus/productivity tools without turning Pluribus into a generic productivity app

---

# Non-Goals

This future sprint is **not** intended to build:

- a full productivity app
- a personal dashboard
- a calendar system
- a broad task manager replacement
- a UI-heavy system as the primary focus
- a second memory system separate from Pluribus authority

This layer should remain tightly connected to the existing memory/recall architecture.

---

# Why This Matters

The current system is strongest at:

- durable memory
- recall of stored knowledge
- constraint and decision reuse
- reduction of amnesia and drift

But practical work often depends on something more immediate:

- current focus
- open thread
- last meaningful state
- what was being attempted
- what to do next

This is especially important for:

- long-running coding efforts
- interrupted sessions
- multi-agent coordination
- resuming older work
- keeping work aligned with the present objective

---

# Working Model

The focus layer should likely sit above durable memory and below client UX.

Conceptually:

```text
Client / Agent
    ↓
Focus Layer
    ↓
Recall / Memory / Promotion / Control-Plane
    ↓
Durable Memory Store
````

The focus layer is not a replacement for recall.
It is a way to assemble and preserve the **active working frame**.

---

# Core Concepts

## 1. Active Focus

A concise statement of what matters most right now.

Examples:

* deploy the docker compose stack and verify MCP connectivity
* prove benefit with baseline vs recall eval
* implement recall ranking upgrade
* fix contradiction visibility in debug output

This should answer:

> What is the current main objective?

---

## 2. Working Thread

A bounded representation of the current line of work.

This may include:

* current task
* current subtask
* decisions made within this thread
* assumptions in play
* recent progress
* unresolved questions

This should answer:

> What thread of work are we currently in?

---

## 3. Resumable Brief

A short resume packet compiled for re-entry.

A good resumable brief may include:

* objective
* current status
* constraints
* blockers
* immediate next step
* open questions
* freshness or verification warning if needed

This should answer:

> If I came back cold, what do I need to know to continue effectively?

---

## 4. Blockers

Explicit record of what is preventing progress.

Examples:

* missing real-host deployment test
* MCP client not yet connected
* ranking test is flaky
* unclear contradiction display policy

This should answer:

> What is stopping forward movement right now?

---

## 5. Next Step

The immediate next useful move.

Not a large goal.
Not a roadmap.
A direct next action.

Examples:

* run docker compose on server
* run recall_get before task prompts
* compare baseline A vs system C on seeded eval
* archive current RIU milestone

This should answer:

> What should happen next?

---

## 6. Focus Freshness / Staleness

Not forgetting, but awareness that some working-state context may be outdated.

Examples:

* this focus thread is from 3 weeks ago
* verify assumptions before acting
* deployment target may have changed
* blocked item may no longer be blocked

This should answer:

> Is the current working-state packet still fresh enough to trust directly?

---

# Design Principles

## 1. Durable memory and focus are not the same thing

Durable memory contains:

* decisions
* constraints
* lessons
* failures
* proven patterns

Focus state contains:

* current objective
* active subtask
* blockers
* next step
* current thread status

Do not confuse these two layers.

---

## 2. Focus should be lightweight

This should not become an uncontrolled dump of session history.

It should be:

* short
* structured
* resumable
* behavior-shaping

---

## 3. Focus should be easy to refresh

The system should support:

* updating focus
* closing a thread
* replacing stale state
* carrying forward only what matters

---

## 4. Focus should compile context, not replace it

The focus layer should improve:

* return to work
* task continuity
* short-term direction

It should not replace recall, memory ranking, or long-term knowledge.

---

## 5. Focus should aid agents, not just humans

This layer should be useful to:

* Cursor
* MCP clients
* local agents
* orchestration flows
* future multi-agent systems

---

# Example Data Model (Conceptual)

## Active Focus Record

```json
{
  "focus_id": "uuid",
  "title": "POC deployment via MCP",
  "objective": "Deploy control-plane with docker compose and verify external MCP client end-to-end flow",
  "status": "active",
  "priority": "high",
  "correlation_tags": ["optional namespace for recall / search"],
  "updated_at": "timestamp"
}
```

## Working Thread Record

```json
{
  "thread_id": "uuid",
  "focus_id": "uuid",
  "current_subtask": "Verify recall_get and memory_create from pluribus MCP client",
  "summary": "POC run is near completion; need real-host verification",
  "constraints": [
    "control-plane is authoritative",
    "MCP first for external agents"
  ],
  "blockers": [
    "real host smoke test not yet recorded"
  ],
  "next_step": "Run deployment-poc on server and execute poc-e2e walkthrough",
  "updated_at": "timestamp"
}
```

## Resumable Brief

```json
{
  "focus_id": "uuid",
  "objective": "Prove benefit of Pluribus recall via MCP",
  "status_summary": "Baseline and seeded recall tests completed; latest run shows strong constraint obedience",
  "blockers": [
    "Need broader eval across more task families"
  ],
  "next_step": "Run additional eval on contradiction handling and cross-project transfer",
  "freshness_note": "Verify deployment environment before resuming",
  "updated_at": "timestamp"
}
```

These are conceptual examples only, not a final schema.

---

# Likely System Behaviors

A useful focus layer might support behaviors like:

## Resume Focus

Given a project or thread, return:

* current objective
* current status
* blockers
* next step
* relevant constraints

## Set Focus

Create or update the current objective for a project or thread.

## Advance Thread

Mark that a step was completed and replace it with the next step.

## Pause Thread

Record current state before switching away.

## Close Thread

Mark thread complete and preserve summary as durable memory if warranted.

## Return-to-Focus Compile

Compile a resumable brief for an agent re-entering the work.

---

# Potential API / MCP Ideas (Future)

These are possible later endpoints or tools, not current commitments.

## Possible service endpoints

* `POST /v1/focus/set`
* `GET /v1/focus/current`
* `POST /v1/thread/update`
* `GET /v1/thread/resume`
* `POST /v1/thread/pause`
* `POST /v1/thread/close`

## Possible MCP tools

* `focus_get`
* `focus_set`
* `thread_resume`
* `thread_update`
* `thread_pause`
* `thread_close`

These should only be implemented if they fit the core architecture cleanly.

---

# Success Criteria

This future sprint would be successful if it can demonstrate:

## 1. Better return to work

An agent can resume a paused task with materially less drift.

## 2. Clear current direction

The system can tell the agent:

* what it is doing
* what matters now
* what the next step is

## 3. Reduced context overload

The agent no longer needs a giant context reload just to resume cleanly.

## 4. Improved thread continuity

Separate threads of work remain distinct and resumable.

## 5. Better practical usefulness

The system becomes more helpful for real multi-step project work.

---

# Evaluation Ideas for a Future Sprint

When this feature is eventually built, useful evals might include:

## Test 1 — Resume after interruption

Pause work, return later, and measure:

* drift
* recovery speed
* correctness of next step

## Test 2 — Thread switching

Switch between two active threads and see if the system restores the correct one.

## Test 3 — Long-running task continuity

Track whether the system helps maintain direction across multiple sessions.

## Test 4 — Freshness awareness

Resume an old thread and verify that the system warns where assumptions may be stale.

## Test 5 — Focus vs durable memory separation

Ensure the system does not confuse:

* “what is currently active”
  with
* “what remains true long-term”

---

# Risks

## 1. Scope creep

This can easily turn into a productivity app if not constrained.

## 2. Duplicate state

Focus data must not become a competing memory authority.

## 3. Noise

If focus packets get too verbose, they will become another source of clutter.

## 4. Staleness confusion

Old focus state must be identifiable without causing hard forgetting.

## 5. Over-engineering

The simplest useful version is probably much smaller than the tempting version.

---

# Recommended Timing

This should be a **post-POC / post-core-hardening sprint**.

Recommended order:

1. prove benefit of the core memory/recall system
2. deploy and test the system in real workflows
3. harden recall, promotion, and contradiction handling
4. then add the focus layer as the next productivity multiplier

---

# Summary

The focus layer is a future enhancement intended to improve:

* thread continuity
* return to focus
* active working-state recovery
* practical usefulness for long-running AI-assisted work

It should not replace durable memory.
It should sit above it and help agents resume work with less drift and less wasted context.

The key idea is simple:

> durable memory preserves what remains true
> focus state preserves what matters right now

That distinction should guide the entire design.

---

# Proposed Next-Step Placeholder

When ready, create a future sprint plan such as:

* `plan-focus-layer.md`
* `plan-thread-resume-layer.md`
* `plan-active-focus-system.md`

with:

* final scope
* concrete schema
* API choices
* evaluation criteria
* implementation order