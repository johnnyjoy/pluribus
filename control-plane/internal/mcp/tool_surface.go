package mcp

// MCP tool surface — behavior-first names and compatibility aliases
// =============================================================================
// Preferred names (agents should pattern-match these first in tools/list):
//   recall_context      — before substantive work; same handler as memory_context_resolve
//   record_experience   — after meaningful work; same handler as mcp_episode_ingest
//
// Stable compatibility (do not remove; existing clients and scripts rely on them):
//   memory_context_resolve, mcp_episode_ingest
//
// Ordering: primary loop tools first, then L1 opportunistic ingest, then gates and L2 tools,
// compatibility aliases near the end, health last.
