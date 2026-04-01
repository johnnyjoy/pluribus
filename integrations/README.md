# Integration packs (behavioral) — **Pluribus**

**Pluribus** is the product surface agents connect to via MCP (or HTTP). **Canonical behavior:** [`pluribus-instructions.md`](pluribus-instructions.md) — mandatory **Pluribus** **`recall_context` → plan → act → `record_experience`** when tools are available; one copy, platform packs wrap or point to it.

Each folder is a **Pluribus control surface**: **`rules.md`** (pointer + native path), **`skill.md`**, **`snippets/context-prime.txt`**, **`README.md`**, **`mcp-config.example.json`**, and optional **`skills/pluribus/`**. **Native** artifacts vary: e.g. **Cursor** — full pack in **`cursor/`** (`plugin-plan.md`, `prompts.md`, `commands.md`, `mcp-config.json`, `helper/`); prefer **user** Pluribus MCP + **user rules** + **`~/.cursor/skills/`**, with **`pluribus.mdc`** optional per-repo; **Continue** **`.continue/rules/pluribus.md`**, **VS Code** **`.github/copilot-instructions.md`**, **Claude Code** **`CLAUDE.md`**.

Skip a **Pluribus** step on substantive work → misconfigured client.

Hub index: **[docs/integrations/README.md](../docs/integrations/README.md)**. **Adoption & verification:** **[docs/integrations/usage.md](../docs/integrations/usage.md)** · **[docs/integrations/matrix.md](../docs/integrations/matrix.md)** (tiers).

```
integrations/
  pluribus-instructions.md     ← canonical behavior block
  cursor/          plugin-plan.md, prompts.md, commands.md, mcp-config.json,
                     pluribus.mdc, rules.md, helper/,
                     skills/pluribus/SKILL.md, README.md
  claude-code/       CLAUDE.template.md (+ skills/pluribus/SKILL.md)
  claude-desktop/    custom-instructions.template.md
  openclaw/          policy.template.md
  opencode/          AGENTS.template.md (+ skills/pluribus/SKILL.md)
  continue/          rules/pluribus.md  → .continue/rules/pluribus.md
  zed/               agent-context.template.md
  vscode/            extension/ (TypeScript VS Code extension), github-copilot-instructions.template.md
  vscode-extension/  pointer README → builds from vscode/extension in CI
  generic-mcp/       (+ examples.json, skills/pluribus/SKILL.md)
  */snippets/context-prime.txt
```

**Never commit secrets.** Rename templates to your client’s expected paths.
