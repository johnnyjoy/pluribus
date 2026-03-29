# Backend synthesis (optional)

Pluribus **does not** run a separate “reasoner” service. Optional **backend synthesis** is a narrow, in-process capability used only for **run-multi candidate text generation** when operators explicitly enable it.

## Doctrine

- **Default:** the **client / agent LLM** performs synthesis when possible.
- **Pluribus** remembers, enforces drift, curates, and guides — it is not a second general-purpose brain.
- Backend synthesis is **optional**, **explicit**, and **replaceable** (swap provider via config).

## Where it is used

| Feature | Uses backend synthesis? |
|--------|-------------------------|
| `POST /v1/recall/run-multi` | **Yes**, when `synthesis.enabled: true` and a valid provider is configured. Otherwise the handler returns an error indicating run-multi is not configured for server-side synthesis. |
| Recall compile, drift, enforcement, MCP | **No** |

No other HTTP routes call the synthesis backends. Ingest `reasoning_trace` is unrelated (MCL validation only).

## Configuration

See `configs/config.example.yaml` under `synthesis:`.

- **`enabled`** — `false` by default. Must be `true` to wire run-multi with a backend LLM.
- **`provider`** — `ollama` | `openai` | `anthropic`
- **`model`** — provider-specific model id (required when enabled).
- **`timeout_seconds`** — HTTP client timeout (default 120).
- **`base_url`** — optional; each provider has a documented default (Ollama local, OpenAI and Anthropic APIs).
- **`api_key`** — optional inline key (development only).
- **`api_key_env`** — environment variable name for the key; if unset, `OPENAI_API_KEY` or `ANTHROPIC_API_KEY` is used for those providers.

Ollama does not require an API key. OpenAI and Anthropic require a key at runtime when enabled.

Invalid combinations fail at **config load** with a clear error (no silent empty success).

## Providers

- **Ollama** — OpenAI-compatible `POST …/v1/chat/completions` at `base_url` (default `http://127.0.0.1:11434`).
- **OpenAI** — Chat Completions API (`/v1/chat/completions` under the configured base, default `https://api.openai.com/v1`).
- **Anthropic** — Messages API (`/v1/messages`), `anthropic-version` header set; default base `https://api.anthropic.com/v1`.

## Operational notes

- Single **controlplane** process only; no sidecar or extra binary.
- Misconfiguration surfaces at startup (YAML validation) or as provider HTTP errors during run-multi (no fake success with empty output).

## Migration from `reasoning:`

Older configs used a flat `reasoning:` block with `endpoint` (full OpenAI-compatible chat-completions URL). That key is **removed**. Replace it with `synthesis:` — set **`enabled: true`**, **`provider`** (`ollama` | `openai` | `anthropic`), **`model`**, and derive **`base_url`** from the old host (e.g. Ollama: `http://127.0.0.1:11434`; OpenAI: `https://api.openai.com/v1`). Use **`api_key`** / **`api_key_env`** for OpenAI and Anthropic.
