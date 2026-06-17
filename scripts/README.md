# Scripts

This directory holds **shell helpers**, **PRD/fixture text**, and **verification** scripts for Recall and Pluribus.

**Build and test:** use Go and Make from the repo root (for example `go test ./...`, `make regression`). There is no in-repo `package.json` or Node task runner.

**Task Master:** optional workflow tooling uses the **`task-master`** CLI (install globally or via `npx task-master-ai`) or the Taskmaster MCP integration described under `.cursor/rules/`. Configuration lives under `.taskmaster/` when present.

**L0 identity seed:** `seed-l0-identity-memories.sh` posts three minimal **`kind=state`** memories so **`wakeup_context`** / wake-up **`identity`** is non-empty on fresh pools. Requires `PLURIBUS_BASE_URL` and optional `PLURIBUS_API_KEY`. Used by the Claude Code plugin SessionStart path indirectly (server returns identity; plugin only displays it).
