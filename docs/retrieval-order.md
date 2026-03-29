# Canonical retrieval order

The orchestrator and recall pipeline assemble context in this order. Do not reverse or skip steps.

1. **Constitution** — `constitution.md`: purpose, domain boundaries, naming rules, forbidden patterns, non-negotiable architecture, coding style, performance priorities, entrypoints and invariants. Loaded first so project law is always present.

2. **Active / current work order** — `active.md` and the current work order (e.g. `workorders/WO-0001.md`): what we are working on right now.

3. **Recent decisions** — Decisions relevant to the domain from recall (filtered by tags/project, ranked by authority and relevance). Injected after the work order so the model sees prior decisions in context.

4. **Constraints, patterns, failures** — From recall: constraints (must-follow rules), patterns (preferred approaches), failures (what not to repeat). Assembled in that order in the prompt.

5. **Evidence for affected files** — (When implemented) Evidence or context for files listed in the work order’s “Files involved” section. Not yet wired in the current pipeline.

The orchestrator’s `assemblePrompt` implements (1)–(4). Evidence (5) is reserved for a future phase.
