#!/usr/bin/env bash
set -euo pipefail

: "${MAX_API_BASE_URL:=https://platform-api.max.ru}"
: "${MAX_BOT_TOKEN:?MAX_BOT_TOKEN is required}"

echo "== MAX /me =="
curl -fsS -H "Authorization: ${MAX_BOT_TOKEN}" "${MAX_API_BASE_URL}/me"
echo

echo "== MAX /subscriptions =="
curl -fsS -H "Authorization: ${MAX_BOT_TOKEN}" "${MAX_API_BASE_URL}/subscriptions"
echo

echo "== MAX /updates dry poll =="
curl -fsS -H "Authorization: ${MAX_BOT_TOKEN}" \
  "${MAX_API_BASE_URL}/updates?types=message_created,bot_started&timeout=1&limit=5"
echo
