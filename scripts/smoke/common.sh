#!/usr/bin/env bash
# Shared helpers for local post-upgrade smoke scripts.
set -euo pipefail

usage_base() {
  cat <<EOF
Usage: $0 [options]

  --base-url URL     Control-plane base URL (default: http://127.0.0.1:8123)
  --api-key KEY      PLURIBUS API key if auth enabled (or set PLURIBUS_API_KEY)
  --help             Show this help

Non-destructive by default. Requires curl and jq.
EOF
}

parse_smoke_args() {
  BASE_URL="${PLURIBUS_BASE_URL:-http://127.0.0.1:8123}"
  API_KEY="${PLURIBUS_API_KEY:-}"
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --base-url) BASE_URL="$2"; shift 2 ;;
      --api-key) API_KEY="$2"; shift 2 ;;
      --help|-h) usage_base; exit 0 ;;
      *) echo "Unknown option: $1" >&2; usage_base; exit 2 ;;
    esac
  done
  BASE_URL="${BASE_URL%/}"
}

curl_json() {
  local method="$1" path="$2" body="${3:-}"
  local args=(-sS -f -X "$method" "${BASE_URL}${path}" -H "Content-Type: application/json")
  [[ -n "$API_KEY" ]] && args+=(-H "X-API-Key: ${API_KEY}")
  if [[ -n "$body" ]]; then
    args+=(-d "$body")
  fi
  curl "${args[@]}"
}

fail() { echo "FAIL: $*" >&2; exit 1; }
pass() { echo "PASS: $*"; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}
