# Decision entry format

## Authority note

For authoritative system behavior, decisions are promoted via control-plane (`POST /v1/memory/promote`) and stored in Postgres-backed memory.

The JSON shape below matches **`CreateRequest` / promoted memory objects** in the API. Use it when authoring payloads or reviewing examples — not as an on-disk repo layout (this repo does not ship a root `memory/` tree).

## Memory JSON object (API shape)

Decision-like memory objects use fields aligned with **`control-plane/internal/memory/types.go`**. A minimal **decision** payload often includes:

| Field       | Required | Description |
|------------|----------|-------------|
| `id`       | Varies   | Server-assigned or client-supplied per API rules. |
| `type` / `kind` | Yes | Maps to memory kind (e.g. decision). |
| `project`  | Context  | Scope / project identifier as required by the request. |
| `statement`| Yes      | One concise sentence: what was decided. |
| `authority`| Often    | Numeric authority where applicable. |
| `tags`     | No       | Array of strings for recall filtering. |
| `data` / payload | No | Optional structured fields (`reason`, links, etc.). |

Example (illustrative): [docs/templates/decision-memory.example.json](templates/decision-memory.example.json).

Optional: some workflows keep an append-only **`decisions.md`** for human audit; **authoritative** recall still comes from **control-plane** memory rows and recall APIs.
