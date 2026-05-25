#!/usr/bin/env bash
set -euo pipefail

: "${PUBLIC_BASE_URL:=http://localhost}"
: "${CORE_API_INTERNAL_TOKEN:?CORE_API_INTERNAL_TOKEN is required}"

base="${PUBLIC_BASE_URL%/}"

echo "== public bot health =="
curl -fsS "${base}/healthz"
echo

echo "== mini app =="
curl -fsSI "${base}/max-miniapp/" | head -n 1

echo "== citizen API =="
curl -fsS "${base}/v2/citizen/categories"
echo

echo "== internal API auth check =="
curl -fsS -H "x-internal-token: ${CORE_API_INTERNAL_TOKEN}" "${base}/internal/v2/events/pending?limit=1"
echo
