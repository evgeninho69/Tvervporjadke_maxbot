#!/usr/bin/env bash
set -euo pipefail

: "${MAX_API_BASE_URL:=https://platform-api.max.ru}"
: "${MAX_BOT_TOKEN:?MAX_BOT_TOKEN is required}"
: "${MAX_WEBHOOK_URL:?MAX_WEBHOOK_URL is required}"
: "${MAX_WEBHOOK_SECRET:?MAX_WEBHOOK_SECRET is required}"

curl -fsS -X POST "${MAX_API_BASE_URL}/subscriptions" \
  -H "Authorization: ${MAX_BOT_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d "{\"type\":\"Webhook\",\"url\":\"${MAX_WEBHOOK_URL}\",\"secret\":\"${MAX_WEBHOOK_SECRET}\",\"update_types\":[\"message_created\",\"message_callback\",\"bot_started\"]}"

echo "Webhook subscription request sent"
