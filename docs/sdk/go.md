# Go SDK (v0)

Path: `sdk/go/pluribus`

Focus: **RecallContext → plan/act → RecordExperience** (same four-step loop as MCP). Curation is optional.

## Naming

Go exports **PascalCase**; intent matches the MCP-oriented names **`recall_context`** and **`record_experience`**.

## Install (local)

```bash
cd sdk/go
go test ./...
```

## Minimal loop

```go
import "os"

client := pluribus.NewClient("http://127.0.0.1:8123", os.Getenv("PLURIBUS_API_KEY"))

bundle, err := client.RecallContext(ctx, "Your task in natural language", pluribus.RecallContextOpts{
    Tags: []string{"my-app"},
})
// plan + act using bundle ...
_, err = client.RecordExperience(ctx, "Short summary of what happened", pluribus.RecordExperienceOpts{
    Tags: []string{"my-app"},
})
```

Full example: `examples/go/minimal_loop/` (`go.mod` uses `replace` → `sdk/go` so you can `go run .` there).

## API (v0)

- `NewClient(baseURL, apiKey string) *Client` — empty `apiKey` when the server has no key
- `RecallContext(ctx, query string, opts RecallContextOpts) (*RecallBundle, error)`
- `RecordExperience(ctx, summary string, opts RecordExperienceOpts) (*AdvisoryEpisode, error)`
- `ListPendingCandidates(ctx) ([]CandidateEvent, error)`
- `ReviewCandidate(ctx, candidateID string) (*CandidateReview, error)`
- `PromoteCandidate(ctx, candidateID string) (*MaterializeOutcome, error)`

`RecallContextOpts`: `Tags`, `CorrelationID`  
`RecordExperienceOpts`: `Tags`, `Entities`, `CorrelationID`  
Record requests always use ingest channel **`source: mcp`** (agent-shaped default).

## Endpoints

- `RecallContext` → `POST /v1/recall/compile`
- `RecordExperience` → `POST /v1/advisory-episodes`
- `ListPendingCandidates` → `GET /v1/curation/pending`
- `ReviewCandidate` → `GET /v1/curation/candidates/{id}/review`
- `PromoteCandidate` → `POST /v1/curation/candidates/{id}/materialize`

## Errors

Non-2xx responses return `*APIError` (method, path, status, body snippet).
