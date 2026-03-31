# Pluribus — Cursor rules (packaging)

This file describes **how to install** the Pluribus loop as Cursor **rules**. Canonical behavior text is always **[`pluribus-instructions.md`](../pluribus-instructions.md)**.

## Default rule (recommended starting point)

- **File:** [`pluribus.mdc`](pluribus.mdc)  
- **Install:** copy to **`.cursor/rules/pluribus.mdc`** (repository) **or** follow [`README.md`](README.md) for **user rules** (paste canonical markdown from `pluribus-instructions.md` in Settings).  
- **Intent:** **`recall_context` → plan → act → `record_experience`**; memory-first **tags + situation** only; no **scope** partitions (`docs/anti-regression.md`).

## Stricter variant (optional)

- **File:** [`pluribus-stricter.mdc`](pluribus-stricter.mdc)  
- **Use when:** you want stronger language (“do not defer recall,” “empty recall ≠ skip record”).  
- **Install:** copy to **`.cursor/rules/pluribus-stricter.mdc`** — **do not** install both **`pluribus.mdc`** and **`pluribus-stricter.mdc`** with **`alwaysApply: true`** unless you intend overlapping instructions; usually **replace** the default file or set **`alwaysApply: false`** on one and scope with **`globs`**.

## Usage notes

| Goal | Action |
|------|--------|
| **Same behavior in every repository** | **User rules** + global **`~/.cursor/mcp.json`** + **`~/.cursor/skills/pluribus/`** — see [`README.md`](README.md). |
| **Shared behavior for everyone who clones** | Commit **`.cursor/rules/pluribus.mdc`** (or stricter) in the repo. |
| **Prime the model** | Also paste [`snippets/context-prime.txt`](snippets/context-prime.txt) into user or repository rules. |
| **Skill packaging** | [`skills/pluribus/SKILL.md`](skills/pluribus/SKILL.md) — Agent Skills format. |

## What “good” looks like

- **Recall** runs before substantive edits when MCP is available.  
- **Record** runs after meaningful outcomes (not after every trivial keystroke).  
- **Promotion** of noise is avoided; durable lessons are what matter (`docs/memory-doctrine.md`).

## Plain copy (no frontmatter)

If you only need markdown: use **`pluribus-instructions.md`** or the body of **`pluribus.mdc`** below the YAML `---` block.
