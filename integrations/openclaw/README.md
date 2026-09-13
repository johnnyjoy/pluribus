# OpenClaw

1. Register Pluribus via **`openclaw mcp`**; values from **`mcp-config.example.json`**.
2. Set **`toolFilter.include`** to the loop names in that file (`wakeup_context`, `recall_context`, `record_experience`, `memory_feedback`, `list_chores`, `resolve_chore`, `health`).
3. Pin **`policy.template.md`** + **`snippets/context-prime.txt`** in gateway system/policy.
4. Merge **`skill.md`** if you need the step table inline.
5. **Never** auto-capture every tool result into Pluribus (`autoCapture: *` is forbidden). Record after a judged outcome only.

Do not replace OpenClaw `MEMORY.md`, dreaming, or `memory_search`. Pluribus is the shared store those files cannot be.

**Deep adoption (policy + MCP; no fake lifecycle APIs):** [`adoption.md`](adoption.md).

**Canonical behavior:** [`pluribus-instructions.md`](../pluribus-instructions.md).

**`recall_context` → plan → act → `record_experience`** every substantive run.
