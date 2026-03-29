# LSP features (Tasks 100–101; recall enrichment)

> **Boundary:** These features use an **LSP client to gopls inside the control-plane process**. Pluribus is **not** an LSP server for the editor. Cursor and other editors keep using **gopls** locally; agent integration stays **MCP/HTTP**. See [Pluribus — LSP vs MCP boundary](../../docs/pluribus-lsp-mcp-boundary.md).

The control-plane can use LSP (Language Server Protocol) data to improve **recall** (symbol-overlap scoring, optional auto symbols and **`reference_count`**) and **drift** (reference-count risk). These features are gated by config so you can enable or disable them without code changes. **Quick curls:** [README §Recall](../README.md) (LSP-backed recall examples).

## Configuration

In your config file loaded via **`CONFIG`** (copy from `config.example.yaml` to `configs/config.local.yaml` for local runs), use the **`lsp`** section:

```yaml
lsp:
  enabled: false
  recall_symbol_boost: 0.5
  reference_expansion_limit: 0
  high_risk_reference_threshold: 0
```

- **`enabled`** (default: `false`)  
  When `false`, no LSP-based behavior runs: drift does not use reference counts, and recall does not apply a symbol-overlap boost even if the client sends `symbols` in the compile request. Set to `true` to turn on LSP features.

- **`recall_symbol_boost`**  
  Weight for the symbol-overlap factor in recall ranking. When LSP is enabled and this is &gt; 0, it overrides `recall.ranking.weight_symbol_overlap`. When 0, the recall ranking config is used (if present).

- **`reference_expansion_limit`**  
  When computing **`reference_count`** on the recall bundle (see below), each symbol’s effective reference count is capped at this value before taking the **max** across matched symbols. **0** = no cap.

- **`high_risk_reference_threshold`**  
  Used by **drift**: if a symbol touched by the proposal has more than this many references (from the LSP), drift sets risk to **high** and can block execution. 0 = off. When LSP is enabled and this is set, the control-plane uses the gopls-based LSP client for drift checks when the request includes `repo_root` and `touched_symbols`.

- **`auto_symbol_max`** (default **64** when `lsp` is present)  
  Caps how many distinct symbol **names** are taken from **documentSymbol** when the server auto-fills recall `symbols`.

## Recall: symbol overlap

- The **compile** (and **compile-multi**) request may include **`symbols`** (e.g. from an LSP document-symbols call). When LSP is enabled and ranking is configured with a symbol-overlap weight &gt; 0, pattern memories whose payload lists overlapping symbols get a score boost.

### Auto symbols

When **`lsp.enabled`** is **true** and the client **does not** send `symbols` (empty or omitted), the server can auto-fill them by calling **gopls** **`documentSymbol`** if the request includes:

- **`repo_root`** — repository root (same idea as drift / tool API), and  
- **`lsp_focus_path`** — file path (relative to `repo_root` or absolute).

If the client sends a non-empty `symbols` list, the server **does not** replace it. Optional **`lsp_focus_line`** / **`lsp_focus_column`** are accepted for future narrowing; they participate in the **compile cache key** so different cursor positions do not share a stale bundle.

The **Redis recall bundle cache key** includes `symbols`, `repo_root`, and the LSP focus fields so different files or symbol sets cannot collide.

### Run-multi

**`POST /v1/recall/run-multi`** accepts the same optional **`repo_root`**, **`lsp_focus_path`**, **`lsp_focus_line`**, and **`lsp_focus_column`** fields as compile / compile-multi. The server-side runner forwards them on the **compile-multi** HTTP body so LSP auto symbols and **`reference_count`** behave consistently. Response **`debug.orchestration`** includes boolean flags (**`lsp_recall_repo_root_set`**, **`lsp_recall_focus_path_set`**, **`lsp_recall_focus_position_set`**) without echoing raw paths.

### Bundle diagnostics (Task 101)

The **recall bundle** can include:

- **`matched_symbols`**: request symbols that matched at least one pattern payload symbol.
- **`symbol_relevance_reason`**: short text when symbol overlap was applied (e.g. `"1 request symbol matched pattern symbols"`).
- **`reference_count`**: when **`lsp.enabled`**, there is at least one **`matched_symbols`** entry, and the compile request includes **`repo_root`** and **`lsp_focus_path`**, the compiler runs **`documentSymbol`** then **`textDocument/references`** for each **unique** matched name (first definition position in the symbol tree). **`reference_count`** is the **maximum** effective reference count across those symbols (each count optionally capped by **`reference_expansion_limit`**). **0** if LSP is off, there is no overlap, or positions could not be resolved.

## Drift: reference-count risk

- The **drift check** request may include `repo_root` and `touched_symbols` (path, line, column per symbol). When LSP is enabled and `high_risk_reference_threshold` &gt; 0, the service calls the LSP (gopls) to get reference counts for each touched symbol. If any symbol has more references than the threshold, risk is set to **high** and execution can be blocked; a warning is added to the result.

## Enabling / disabling

- **Disable all LSP behavior:** set `lsp.enabled: false` (or omit the `lsp` section and rely on drift/recall defaults). No LSP client is wired for drift or recall enrichment.
- **Enable:** set `lsp.enabled: true`. Optionally set `recall_symbol_boost`, **`reference_expansion_limit`**, and/or `high_risk_reference_threshold`. **gopls** is used for recall auto-symbols / **`reference_count`** and for drift reference checks when those features are exercised.

## Dependencies

- **Recall** symbol overlap only needs the compile request to include `symbols` (or auto-filled symbols from 8.1); **no gopls** is required unless you want **`reference_count`** (8.2) or auto symbols (8.1).
- **Drift** reference-count risk requires the control-plane to run **gopls** (or another LSP-compatible server on the subprocess the client uses) when the request includes `repo_root` and `touched_symbols`; the **in-process LSP client** invokes it — there is **no** separate public HTTP “LSP API” for editors.
