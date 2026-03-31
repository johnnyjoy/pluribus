# Python SDK (v0)

Path: `sdk/python/pluribus`

Focus: **`recall_context` → plan/act → `record_experience`**. Curation is optional.

## Install (local)

```bash
cd sdk/python
python3 -m pip install -e .
```

## Minimal loop

```python
from pluribus import PluribusClient

client = PluribusClient("http://127.0.0.1:8123")

bundle = client.recall_context("Your task in natural language", tags=["my-app"])
# plan + act using bundle ...
client.record_experience("Short summary of what happened", tags=["my-app"])
```

Full example: `examples/python/minimal_loop.py`

## API (v0)

- `PluribusClient(base_url, api_key=None, timeout=15.0)`
- `recall_context(query, *, tags=None, correlation_id=None)` → `dict`
- `record_experience(summary, *, tags=None, entities=None, correlation_id=None)` → `dict`  
  Always sends **`source: mcp`**.
- `list_pending_candidates()` → `list`
- `review_candidate(candidate_id)` → `dict`
- `promote_candidate(candidate_id)` → `dict`

## Endpoints

Same HTTP mapping as the Go SDK “Endpoints” section in [go.md](go.md#endpoints).

## Errors

Non-2xx responses raise `PluribusAPIError` (method, path, status, body snippet).
