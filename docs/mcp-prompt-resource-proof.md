# MCP prompt & resource proof map (L2)

Links charter outcomes from [plan-pluribus-mcp-prompt-resource-hardening-20260325.md](../memory-bank/plans/plan-pluribus-mcp-prompt-resource-hardening-20260325.md) to **L1** automated checks and existing **proof-scenarios** receipts.

**Surface bundle:** **`SurfaceVersion`** in `control-plane/internal/mcp/surface_version.go` (currently **1.1.0**).

---

## Charter outcomes → evidence

| # | Outcome | L1 (`go test ./internal/mcp`) | Proof scenarios / notes |
|---|---------|-------------------------------|-------------------------|
| 1 | Full audit exists | N/A (this doc + [mcp-prompt-resource-audit.md](mcp-prompt-resource-audit.md)) | — |
| 2 | Lifecycle language explicit | `TestPromptMemoryGrounding_lifecycleMnemonicAndResources`, resource bodies via `TestResourceDefinitions_fiveCanonicalWithSurfaceVersion` | Lifecycle table in `resources.go` |
| 3 | Memory-grounding protocol | `TestPromptMemoryGrounding_body` (ontology + recall-first) | [continuity-second-step-from-first.yaml](../control-plane/proof-scenarios/continuity-second-step-from-first.yaml) (API continuity) |
| 4 | Pre-change routes risk | `TestPromptPreChange_enforcementGate` | [enforcement-sqlite-forbidden.yaml](../control-plane/proof-scenarios/enforcement-sqlite-forbidden.yaml), [enforcement-unrelated-allow.yaml](../control-plane/proof-scenarios/enforcement-unrelated-allow.yaml) |
| 5 | Memory curation avoids junk | `TestPromptMemoryCuration_candidatesNotCanon` | [curation-digest-materialize-durable.yaml](../control-plane/proof-scenarios/curation-digest-materialize-durable.yaml) |
| 6 | Authority classes | `TestPromptCanonVsAdvisory_authorityOnly`, `TestResourceText_canonVsAdvisoryAliases` | Recall binding: [recall-binding-constraint-surfaces.yaml](../control-plane/proof-scenarios/recall-binding-constraint-surfaces.yaml) |
| 7 | No instruction contradicts shipped behavior | B3 truth pass in audit doc | Contract: [mcp-poc-contract.md](mcp-poc-contract.md) |
| 8 | Proof scenarios (continuity, recall, enforcement, curation) | Substring tests + manual checklist | See table rows above + [mcp-prompt-resource-discipline-manual.yaml](../control-plane/proof-scenarios/mcp-prompt-resource-discipline-manual.yaml) |
| 9 | Versioning documented | `TestSurfaceVersion_nonEmpty`, `TestPromptDefinitions_includeSurfaceVersion`, `TestResourceDefinitions_sixCanonicalWithSurfaceVersion` | [mcp-prompt-resource-versioning.md](mcp-prompt-resource-versioning.md) |

---

## Release gate

1. `cd control-plane && go test ./...`
2. From repo root: `make regression`
3. Optional: run manual checklist steps in `mcp-prompt-resource-discipline-manual.yaml` for operator verification of prompt/resource copy against release notes.
