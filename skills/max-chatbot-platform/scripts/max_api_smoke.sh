#!/usr/bin/env bash
set -euo pipefail

: "${MAX_API_BASE_URL:=https://platform-api.max.ru}"
: "${MAX_BOT_TOKEN:?MAX_BOT_TOKEN is required}"

AUTH_HEADER="Authorization: ${MAX_BOT_TOKEN}"

curl -fsS -H "$AUTH_HEADER" "${MAX_API_BASE_URL}/updates" >/dev/null || {
  echo "[FAIL] /updates" >&2
  exit 1
}

echo "[OK] MAX API auth + /updates reachable"
