# Pluribus — OpenClaw (gateway / agent)

Pluribus is the **memory substrate**. Register it via **`openclaw mcp`** (or your version’s MCP UI) — see https://docs.openclaw.ai/cli/mcp

## Operating rules

- **Default loop:** `memory_context_resolve` before deep work; `mcp_episode_ingest` after meaningful episodes.  
- **Aggressive adoption:** treat empty recall as a signal to ingest more, not to skip Pluribus.  
- **Layer 2:** curation / contradictions only when needed—never block the default loop.  
- **Doctrine:** no project/task/workspace/**scope** memory partitions; tags + situation text shape recall.
