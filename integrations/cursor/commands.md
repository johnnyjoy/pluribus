# Pluribus — commands and repeatable patterns in Cursor

Cursor does **not** run Pluribus by itself—you drive **MCP tools** from **Agent** after the server is configured. This file documents **patterns** that behave like “commands.”

## Built-in Cursor entry points

| Entry | Use |
|--------|-----|
| **Agent chat** | Invoke **`recall_context`** / **`record_experience`** via MCP (natural language or explicit tool calls). |
| **Cursor Commands** (`.cursor/commands/*.md`) | This repo ships workflow commands under **`.cursor/commands/`** (e.g. benefit-eval). Add your own markdown commands that *tell the agent* to run Pluribus tools—there is no separate Pluribus CLI inside Cursor. |
| **Rules + Skills** | **`pluribus.mdc`** / **`skills/pluribus/`** keep the loop top-of-mind so Agent reaches for tools without you typing prompts every time. |

## Repeatable “before / after” pattern

1. **Before:** In Agent, say: “Run **`recall_context`** for: &lt;paste task&gt;” with tags / retrieval text as needed.
2. **Work:** Plan, edit, test.
3. **After:** “Run **`record_experience`** summarizing outcome and lessons.”

## Optional shell check (outside Cursor)

From the repo: `integrations/cursor/helper/verify-mcp.sh` — confirms HTTP MCP responds (does not prove Agent will call tools).

## Slash commands

Cursor **slash commands** are editor features (e.g. `/van`, `/plan` in this repo). They are **not** Pluribus RPC. Use them to structure *your* workflow; pair with **rules** so the same session still runs **`recall_context`** / **`record_experience`** when appropriate.
