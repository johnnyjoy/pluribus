ARCHIVED — NOT ACTIVE SYSTEM TRUTH

---

# RC1 — operator checklist (zero guesswork)

Use this for a **clean bring-up** and validation of **Pluribus control-plane** on a developer machine. Commands assume a POSIX shell and repo root **`recall/`** (or adjust paths).

---

## 1. Bring-up

1. **Clone**

   ```bash
   git clone <repository-url> recall
   cd recall
   ```

2. **Postgres**  
   You need a reachable Postgres and a database (e.g. `controlplane`). Example DSN format:  
   `postgres://USER:PASS@localhost:5432/controlplane?sslmode=disable`

3. **Config**  
   Copy the example and set DSN:

   ```bash
   cp control-plane/configs/config.example.yaml control-plane/configs/config.local.yaml
   # Edit postgres.dsn in config.local.yaml
   ```

4. **Build**

   ```bash
   cd control-plane
   go build -o controlplane ./cmd/controlplane
   ```

5. **Run** (schema reconciliation runs on boot)

   ```bash
   CONFIG=configs/config.local.yaml ./controlplane
   ```

   Server listens on **`server.bind`** (default **`:8123`** from the example).

**Minimal config required:** `server`, `postgres` (valid DSN). Other sections may use defaults from `LoadConfig`.

---

## 2. Configuration validation

### Valid — synthesis **disabled** (default)

```yaml
synthesis:
  enabled: false
```

Expect: process starts; **`POST /v1/recall/run-multi`** returns **`503`** (runner not configured).

### Valid — synthesis **enabled** (example: Ollama)

```yaml
synthesis:
  enabled: true
  provider: ollama
  model: "qwen2.5-coder:latest"
  timeout_seconds: 120
  base_url: "http://127.0.0.1:11434"
```

Expect: process starts **only if** Ollama is reachable when run-multi runs (runtime HTTP errors possible if down).

### Valid — OpenAI (requires key)

```yaml
synthesis:
  enabled: true
  provider: openai
  model: "gpt-4o-mini"
  api_key_env: "OPENAI_API_KEY"
```

Expect: set **`OPENAI_API_KEY`** in the environment before starting **`controlplane`**.

### Invalid — must **fail at startup** (`LoadConfig`)

| Config | Expected |
|--------|----------|
| `synthesis.enabled: true` and missing `provider` | Process exits with error |
| `synthesis.enabled: true` and empty `model` | Process exits with error |
| `synthesis.enabled: true`, `provider: openai`, no API key / env | Process exits with error |

---

## 3. Health checks

```bash
curl -sS http://127.0.0.1:8123/healthz
# Expect: 200, body: ok

curl -sS -o /dev/null -w "%{http_code}" http://127.0.0.1:8123/readyz
# Expect: 200 when DB is up and schema current; 503 otherwise
```

- **`/healthz`:** liveness only — does **not** check the database.
- **`/readyz`:** DB ping + core tables + migration state **current**.

---

## 4. Core workflow validation

Set **`BASE=http://127.0.0.1:8123`**. If the server has **`PLURIBUS_API_KEY`** set, add **`-H "X-API-Key: <key>"`** to every request below.

### Recall — compile

```bash
curl -sS -X POST "$BASE/v1/recall/compile" \
  -H 'Content-Type: application/json' \
  -d '{"tags":["demo"],"retrieval_query":"RC1 checklist"}'
```

**Expect:** **`200`**, JSON **`RecallBundle`** with keys such as **`governing_constraints`**, **`decisions`**, **`known_failures`**, **`applicable_patterns`** (arrays, may be empty).

### Enforcement — evaluate

```bash
curl -sS -X POST "$BASE/v1/enforcement/evaluate" \
  -H 'Content-Type: application/json' \
  -d '{"proposal_text":"Use SQLite for production storage.","tags":["demo"]}'
```

**Expect:** **`200`**, JSON with **`decision`**, **`explanation`**, **`triggered_memories`**.  
If enforcement is disabled in config: **`403`**.

### Run-multi — synthesis **disabled**

With default **`synthesis.enabled: false`**:

```bash
curl -sS -X POST "$BASE/v1/recall/run-multi" \
  -H 'Content-Type: application/json' \
  -d '{"query":"Plan the migration","tags":["demo"]}'
```

**Expect:** **`503`**, body **`{"error":"run-multi is unavailable: the server-side runner is not configured (enable synthesis in config or use client-side run-multi)"}`**.

### Run-multi — synthesis **enabled**

With valid **`synthesis`** and a running provider (e.g. Ollama), the same request should return **`200`** and a **`RunMultiResponse`** including **`debug`** and scores.

---

## 5. Regression

From **`control-plane/`**:

```bash
go test ./...
```

From **repo root** (if Makefile exists):

```bash
make regression
```

**Expect:** **`go test ./...`** passes. **`make regression`** requires Docker and **`TEST_PG_DSN`** per project docs — follow **`INSTALL.md`** / **`docs/pluribus-operational-guide.md`** if used.

---

## Canonical references

- **HTTP contract (this release):** [api-contract.md](api-contract.md)
- **Backend synthesis:** [../control-plane/docs/backend-synthesis.md](../control-plane/docs/backend-synthesis.md)
- **Example config:** `control-plane/configs/config.example.yaml`
