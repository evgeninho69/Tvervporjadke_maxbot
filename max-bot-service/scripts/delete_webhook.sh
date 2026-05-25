#!/usr/bin/env bash
set -euo pipefail

: "${MAX_API_BASE_URL:=https://platform-api.max.ru}"
: "${MAX_BOT_TOKEN:?MAX_BOT_TOKEN is required}"
: "${MAX_WEBHOOK_URL:?MAX_WEBHOOK_URL is required}"

encoded_url="$(python3 - <<'PY'
import os
import urllib.parse
print(urllib.parse.quote(os.environ["MAX_WEBHOOK_URL"], safe=""))
PY
)"

curl -fsS -X DELETE "${MAX_API_BASE_URL}/subscriptions?url=${encoded_url}" \
  -H "Authorization: ${MAX_BOT_TOKEN}"

echo "Webhook subscription delete request sent"
