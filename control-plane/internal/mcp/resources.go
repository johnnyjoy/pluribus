package mcp

import "strings"

// Resource URIs for prompts/resources/list.
const (
	URIDisciplineDoctrine            = "pluribus://discipline/doctrine"
	URIDisciplineLifecycle           = "pluribus://discipline/lifecycle"
	URIDisciplineCanonAdvisory       = "pluribus://discipline/canon-vs-advisory" // canonical URI (name kept for minimal export churn)
	URIDisciplineHistoryNotMemory    = "pluribus://discipline/history-not-memory"
	URIArchitectureActiveVsDurable   = "pluribus://architecture/active-context-vs-durable-store"
)

// ResourceDefinitions returns resources/list entries (six canonical surfaces + SurfaceVersion in descriptions).
func ResourceDefinitions() []map[string]any {
	sv := SurfaceVersion
	return []map[string]any{
		{
			"uri":         URIDisciplineDoctrine,
			"name":        "Discipline doctrine (compact)",
			"description": "Strong defaults, anti-skip, avoid bogus ritual; pointers to lifecycle and authority resources. Surface " + sv + ".",
			"mimeType":    "text/markdown",
		},
		{
			"uri":         URIDisciplineLifecycle,
			"name":        "Canonical lifecycle (phase map)",
			"description": "Recall → Act → Validate → Update → Repeat (enforce risky proposals). Surface " + sv + ".",
			"mimeType":    "text/markdown",
		},
		{
			"uri":         URIDisciplineCanonAdvisory,
			"name":        "Canon vs advisory vs candidate vs evidence",
			"description": "Authority classes for gates and curation. Surface " + sv + ".",
			"mimeType":    "text/markdown",
		},
		{
			"uri":         URIArchitectureActiveVsDurable,
			"name":        "Active context vs durable store (+ reference compact)",
			"description": "Recall bundle vs DB rows; reference large artifacts, do not inline. Surface " + sv + ".",
			"mimeType":    "text/markdown",
		},
		{
			"uri":         URIDisciplineHistoryNotMemory,
			"name":        "History is not memory",
			"description": "Transcript/log ≠ durable operational knowledge. Surface " + sv + ".",
			"mimeType":    "text/markdown",
		},
	}
}

// ResourceText returns markdown body for resources/read, or empty if unknown URI.
func ResourceText(uri string) (string, bool) {
	switch strings.TrimSpace(uri) {
	case URIDisciplineDoctrine:
		return resourceDoctrineMarkdown, true
	case URIDisciplineLifecycle:
		return resourceLifecycleMarkdown, true
	case URIDisciplineCanonAdvisory:
		return resourceCanonMarkdown, true
	case "pluribus://discipline/canon-advisory":
		return resourceCanonMarkdown, true
	case URIDisciplineHistoryNotMemory:
		return resourceHistoryNotMemoryMarkdown, true
	case URIArchitectureActiveVsDurable:
		return resourceArchNotesMarkdown, true
	case "pluribus://discipline/architecture-notes":
		return resourceArchNotesMarkdown, true
	default:
		return "", false
	}
}

const resourceDoctrineMarkdown = `# Pluribus discipline doctrine (compact)

**Pluribus is a global memory system.** Durable memory is keyed by **kind**, **statement**, **tags**, and server-side authority — not by hidden partition or ownership containers. **Recall** selects a situation-shaped slice from that shared pool using **retrieval_query** and **tags**.

Treat prompts/resources as a **behavioral contract**, not decorative help. (Version is carried in MCP **resources/list** descriptions and **SurfaceVersion** in code.)

## Strong defaults
- **Recall** (**recall_get**) before substantive work; **recall_compile** when tags/intent improve retrieval.
- **Enforce** (**enforcement_evaluate**) before **risky** change — not for trivia.
- **Digest** (**curation_digest**) after **meaningful** learning; output is **candidate**, not canon.
- **Materialize** (**curation_materialize**) only **validated** candidates.
- **Refresh** recall when context **materially** shifts.
- **Never guess missing context**; if continuity/constraints/experience are insufficient, recall again.

## Anti-skip cues
- Substantive work → recall before recommendations.
- Risky proposal → enforcement before you endorse.
- Real lesson → digest; materialize only what passes review.

## Avoid bogus ritual
- Do not gate typos.
- Do not digest noise.

## Pointers
- Phase map: **pluribus://discipline/lifecycle**
- Authority: **pluribus://discipline/canon-vs-advisory**
- Canonical product model: [memory-doctrine.md](../../../docs/memory-doctrine.md) (Recall repository)
- Longer: [mcp-discipline-doctrine.md](../../docs/mcp-discipline-doctrine.md)
`

const resourceLifecycleMarkdown = `# Canonical lifecycle (Pluribus)

Use this **same ordering** in planning and tool choice.

| Step | Name | Tool(s) | Notes |
|------|------|---------|--------|
| 1 | **Recall** | **recall_get** (default); **recall_compile** when tags/intent help | Load continuity/constraints/experience first; do not guess missing context |
| 2 | **Act** | Normal work | Draft/implement proposal |
| 3 | **Validate** | **enforcement_evaluate** (risky proposals) | Follow ` + "`validation.next_action`" + ` = proceed/revise/reject |
| 4 | **Update** | **curation_digest** then **curation_materialize** | Persist validated learning to durable memory |
| 5 | **Repeat** | **recall_get** / compile | Re-enter loop after update/context shifts |

**Mnemonic:** *Recall → Act → Validate → Update → Repeat.*

Recall output is structured:
` + "```json" + `
{
  "continuity": [...],
  "constraints": [...],
  "experience": [...]
}
` + "```" + `

Prompts: **pluribus_memory_grounding**, **pluribus_pre_change_enforcement**, **pluribus_memory_curation**, **pluribus_canon_vs_advisory**.
`

const resourceCanonMarkdown = `# Canon vs advisory vs candidate vs evidence

| Class | Role |
|-------|------|
| **Governing / binding** | Participates in **enforcement**; can block or require review (server decides membership). |
| **Advisory** | Context; does **not** replace binding in gates. |
| **Candidates** | Output of **curation_digest** — pending until **curation_materialize**. |
| **Evidence** | Receipts; support scoring; **not** policy alone. |
| **Transcript / chat** | Not durable memory until promoted through server rules. |

**Digest → candidate.** **Materialize → durable** (subject to server policy).

See pluribus://discipline/history-not-memory for transcript vs memory.
`

const resourceHistoryNotMemoryMarkdown = `# History is not memory

- **Transcripts, logs, and chat** are **not** the same as **durable operational memory** in Pluribus.
- **Recall** returns a **working slice** of what the server stores — not your full conversation.
- Promote only **validated** content through **curation_digest** then **curation_materialize** (or other server paths).

**Default:** cite or summarize; **do not** paste raw history into digest as a substitute for a clear statement.

See pluribus://discipline/doctrine and pluribus://architecture/active-context-vs-durable-store.
`

const resourceArchNotesMarkdown = `# Active context vs durable store

## Active context
- The **recall bundle** for this request: **selected** rows and fields for the current question.
- **Assembled** per request — not a dump of the entire database.

## Durable store
- Rows the **server** persists (memory, evidence links, etc.).
- **Authority** and relevance selection are determined by the service, not by chat.

## Reference, do not inline
- Large files, diffs, and evidence blobs: **cite path or URI**; keep **proposal_text** and **work_summary** **bounded** per server limits.
- Prefer **structured** statements over pasting **raw** logs into **curation_digest**.

## Why recall is selective
- Keeps context **situation-shaped** (focused on the current question) and within token limits.
- Avoids treating **history** as **authority** by accident.

These notes describe behavior; they do not change server semantics by themselves.
`
