# Pluribus — Cursor rules (packaging)

**Pluribus** integration: canonical behavior is **[`pluribus-instructions.md`](../pluribus-instructions.md)** — mandatory **Pluribus** `recall_context` → plan → act → `record_experience` when tools exist; doctrine in that file.

## Rule file (single canonical)

- **File:** [`pluribus.mdc`](pluribus.mdc) — includes stricter anti-deferral language (no separate “stricter” pack).
- **Install:** copy to **`.cursor/rules/pluribus.mdc`** (repository) **or** paste **[`pluribus-instructions.md`](../pluribus-instructions.md)** into **User rules** — see [`README.md`](README.md).  
- **Intent:** **Pluribus** memory-first **tags + situation** only; no **scope**-shaped partitions.

## Usage notes

| Goal | Action |
|------|--------|
| **Same Pluribus behavior in every repository** | **User rules** + global **`~/.cursor/mcp.json`** + **`~/.cursor/skills/pluribus/`** — see [`README.md`](README.md). |
| **Shared Pluribus rules for everyone who clones** | Commit **`.cursor/rules/pluribus.mdc`** in the repo. |
| **Prime the model** | Also paste [`snippets/context-prime.txt`](snippets/context-prime.txt) into user or repository rules. |
| **Agent Skill** | [`skills/pluribus/SKILL.md`](skills/pluribus/SKILL.md) — **Pluribus** Agent Skills format. |

## What “good” looks like

- **Pluribus** **recall** runs before substantive edits when MCP is available.  
- **Pluribus** **record** runs after meaningful outcomes (not after every trivial keystroke).  
- Noise is not promoted; durable **Pluribus** lessons are what matter (`docs/memory-doctrine.md`).

## Plain copy (no frontmatter)

Use **`pluribus-instructions.md`** or the body of **`pluribus.mdc`** below the YAML `---` block.
