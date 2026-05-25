#!/usr/bin/env bash
set -euo pipefail

: "${COMPOSE_FILE:=docker-compose.iinnovator.yaml}"
: "${PUBLIC_BASE_URL:=https://iinnovator.website}"
: "${IINNOVATOR_HTTP_BIND:=127.0.0.1}"
: "${IINNOVATOR_HTTP_PORT:=18080}"

load_env_file() {
  local file="$1"
  [[ -f "${file}" ]] || return 0
  while IFS= read -r line || [[ -n "${line}" ]]; do
    [[ -z "${line}" || "${line}" =~ ^[[:space:]]*# ]] && continue
    [[ "${line}" == *=* ]] || continue
    local key="${line%%=*}"
    local value="${line#*=}"
    key="$(echo "${key}" | tr -d '[:space:]')"
    [[ "${key}" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]] || continue
    value="${value%$'\r'}"
    value="${value%\"}"
    value="${value#\"}"
    export "${key}=${value}"
  done < "${file}"
}

load_env_file ./.env.timeweb
load_env_file ./core-api/.env
load_env_file ./max-bot-service/.env

export COMPOSE_PROJECT_NAME=tveractive_iinnovator
export IINNOVATOR_HTTP_BIND
export IINNOVATOR_HTTP_PORT

docker compose -f "${COMPOSE_FILE}" build
docker compose -f "${COMPOSE_FILE}" up -d
docker compose -f "${COMPOSE_FILE}" ps

echo "== local isolated contour health =="
curl -fsS -H 'Host: iinnovator.website' "http://127.0.0.1:${IINNOVATOR_HTTP_PORT}/healthz"
echo

if [[ -n "${CORE_API_INTERNAL_TOKEN:-}" ]]; then
  echo "== local isolated contour internal API =="
  curl -fsS -H 'Host: iinnovator.website' -H "x-internal-token: ${CORE_API_INTERNAL_TOKEN}" \
    "http://127.0.0.1:${IINNOVATOR_HTTP_PORT}/internal/v2/events/pending?limit=1"
  echo
fi
