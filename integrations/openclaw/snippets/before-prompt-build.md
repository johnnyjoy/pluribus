# Optional before_prompt_build hook

Call Pluribus before the model sees the user turn. This is a client hook, not a watcher AI.

- Session start / empty prompt: `wakeup_context`.
- User text present: `recall_context` with that text as `task`. If the user named a clock window, compute RFC3339 `occurred_after` / `occurred_before` yourself.
- Do not auto-ingest the prompt or tool results. After a judged outcome, `record_experience` with one dense claim and optional `occurred_at`.
