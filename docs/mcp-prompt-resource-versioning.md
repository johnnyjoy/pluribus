# MCP prompt & resource versioning

**`SurfaceVersion`** (`control-plane/internal/mcp/surface_version.go`) versions the **bundle** of:

- Embedded prompt bodies: `task_start.md`, `pre_change.md`, `post_task.md`, `canon_vs_advisory.md` (filenames unchanged; MCP **names** are `pluribus_memory_grounding`, `pluribus_memory_curation`, etc. — see [pluribus-semantic-cutover-report.md](pluribus-semantic-cutover-report.md))
- Resource markdown in `resources.go`
- One-line **`description`** fields in `PromptDefinitions()` and `ResourceDefinitions()` (they include the same version string)

It is **independent** of the MCP protocol version and of the control-plane binary semver.

---

## When to bump

Bump **`SurfaceVersion`** when any of the following change **meaningfully** (operators or agents would need to re-read the surface):

- Wording in an embedded prompt or resource body
- Canonical lifecycle mnemonic or step order
- URI list or resource job descriptions
- Behavioral claims about tools (must stay true to [mcp-poc-contract.md](mcp-poc-contract.md))

Do **not** bump for unrelated Go refactors, comment-only changes outside MCP, or typo fixes that do not change meaning (use judgment; when unsure, bump patch).

**Suggested scheme:** semantic **major.minor.patch** (e.g. `1.1.0`): patch = copy edits; minor = new resource or prompt section; major = breaking rename of prompts or URIs (rare).

---

## Checklist before merge

1. Run `go test ./...` under `control-plane/` (L1 tests assert **`SurfaceVersion`** appears on every prompt/resource description and lock key substrings).
2. Update [mcp-prompt-resource-audit.md](mcp-prompt-resource-audit.md) if inventory or truth alignment changed.
3. Update [mcp-prompt-resource-proof.md](mcp-prompt-resource-proof.md) if new tests or scenarios were added.
4. Run `make regression` from repo root when touching control-plane behavior or docs that gate release.

---

## Optional: expose version in MCP

Exposing **`SurfaceVersion`** via `initialize` or `serverInfo` is **optional** and not required for v1; clients can rely on **`prompts/list`** / **`resources/list`** descriptions until product agrees on a stable field.
