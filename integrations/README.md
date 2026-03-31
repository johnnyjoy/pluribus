# Integration packs (behavioral)

**Canonical behavior text:** [`pluribus-instructions.md`](pluribus-instructions.md) (one copy; platform packs wrap or point to it).

Each folder is a **control surface**: **`rules.md`** (pointer to the canonical file + native path), **`skill.md`**, **`snippets/context-prime.txt`**, **`README.md`**, **`mcp-config.example.json`**, and optional **`skills/pluribus/`**. **Native** artifacts vary: e.g. **Cursor** — full pack in **`cursor/`** (`plugin-plan.md`, `prompts.md`, `commands.md`, `mcp-config.json`, `helper/`); prefer **user** MCP + **user rules** + **`~/.cursor/skills/`**, with **`pluribus.mdc`** optional per-repo; **Continue** **`.continue/rules/pluribus.md`**, **VS Code** **`.github/copilot-instructions.md`**, **Claude Code** **`CLAUDE.md`**.

**Mandatory loop:** **`recall_context` → plan → act → `record_experience`**. Skip a step on substantive work → misconfigured.

Hub index: **[docs/integrations/README.md](../docs/integrations/README.md)**.

```
integrations/
  pluribus-instructions.md     ← canonical behavior block
  cursor/          plugin-plan.md, prompts.md, commands.md, mcp-config.json,
                     pluribus.mdc, pluribus-stricter.mdc, rules.md, helper/,
                     skills/pluribus/SKILL.md, README.md
  claude-code/       CLAUDE.template.md (+ skills/pluribus/SKILL.md)
  claude-desktop/    custom-instructions.template.md
  openclaw/          policy.template.md
  opencode/          AGENTS.template.md (+ skills/pluribus/SKILL.md)
  continue/          rules/pluribus.md  → .continue/rules/pluribus.md
  zed/               agent-context.template.md
  vscode/            github-copilot-instructions.template.md
  generic-mcp/       (+ examples.json, skills/pluribus/SKILL.md)
  */snippets/context-prime.txt
```

**Never commit secrets.** Rename templates to your client’s expected paths.
