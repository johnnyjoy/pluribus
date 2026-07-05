#!/usr/bin/env bash
# Deployed benefit receipts: run the integration proof-scenario suite against a
# live control-plane (real formation/recall policy — not proof-friendly defaults).
#
# Usage:
#   CONTROL_PLANE_URL=http://host:8123 ./scripts/proof-deployed-benefit-receipts.sh
#   PLURIBUS_PROOF_BASE_URL=http://host:8123 ./scripts/proof-deployed-benefit-receipts.sh
#
# Optional:
#   CONTROL_PLANE_API_KEY / PLURIBUS_API_KEY — X-API-Key when auth is enabled
#   RECALL_PROOF_RESULTS_OUT — markdown summary path (default: artifacts/deployed-benefit-receipts-latest.md)
#
# Exit: 0 if all scenarios pass; non-zero otherwise.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BASE="${PLURIBUS_PROOF_BASE_URL:-${CONTROL_PLANE_URL:-}}"
BASE="${BASE%/}"
if [[ -z "$BASE" ]]; then
  echo "usage: CONTROL_PLANE_URL=http://host:8123 $0" >&2
  echo "   or: PLURIBUS_PROOF_BASE_URL=http://host:8123 $0" >&2
  exit 2
fi

OUT="${RECALL_PROOF_RESULTS_OUT:-$ROOT/artifacts/deployed-benefit-receipts-latest.md}"
mkdir -p "$(dirname "$OUT")"

echo "== deployed benefit receipts =="
echo "base: $BASE"
echo "results: $OUT"

# Preflight: health
code="$(curl -sS -o /tmp/pluribus-deployed-health.txt -w '%{http_code}' "${BASE}/healthz" || true)"
if [[ "$code" != "200" ]]; then
  echo "FAIL: healthz HTTP $code (body: $(head -c 200 /tmp/pluribus-deployed-health.txt))" >&2
  exit 1
fi
echo "healthz: ok"

export PLURIBUS_PROOF_BASE_URL="$BASE"
export RECALL_PROOF_RESULTS_OUT="$OUT"
# Do not require TEST_PG_DSN for deployed mode.
unset TEST_PG_DSN || true

set +e
(
  cd "$ROOT/control-plane"
  go test -tags=integration -count=1 -v ./cmd/controlplane -run 'TestIntegration_proofScenarioSuite'
)
rc=$?
set -e

if [[ -f "$OUT" ]]; then
  echo "== results written: $OUT =="
  cat "$OUT"
else
  echo "WARN: no results file at $OUT (suite may have failed before summary write)" >&2
fi

# Append root-cause note when any scenario failed (formation/pending is the common deploy gap).
if [[ $rc -ne 0 ]]; then
  {
    echo ""
    echo "## Deployed-policy notes"
    echo ""
    echo "Local/CI receipts use proof-friendly formation defaults (seeded memories land **active**)."
    echo "Deployed servers use real formation/recall policy (active at capped authority). Failures usually mean"
    echo "warehouse defaults, pending sinks, consolidate traps, or unverifiable writes — not flaky tests."
    echo ""
    echo "Generated: $(date -Iseconds 2>/dev/null || date)"
    echo "Base URL: \`$BASE\`"
  } >>"$OUT"
fi

exit "$rc"
