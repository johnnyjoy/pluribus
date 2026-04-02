#!/usr/bin/env bash
# Static checks for plugin layout (no Claude Code binary required).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "== JSON =="
for f in .claude-plugin/plugin.json hooks/hooks.json .mcp.json; do
  python3 -m json.tool "$f" >/dev/null
  echo "ok: $f"
done

echo "== Bash hook scripts =="
for s in hooks/*.sh; do
  bash -n "$s"
  echo "ok: $s"
done

echo "== Skills (SKILL.md) =="
count=0
while IFS= read -r -d '' md; do
  echo "ok: $md"
  count=$((count + 1))
done < <(find skills -name SKILL.md -print0 2>/dev/null || true)
echo "skill count: $count"

echo "== Agents =="
find agents -name '*.md' -print 2>/dev/null | while read -r a; do echo "ok: $a"; done

echo "verify-plugin: all checks passed"
