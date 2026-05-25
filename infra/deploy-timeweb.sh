#!/usr/bin/env bash
set -euo pipefail

: "${COMPOSE_FILE:=docker-compose.timeweb.yaml}"
: "${PUBLIC_BASE_URL:=https://tver4ever.ru}"
: "${BUILDX_NO_DEFAULT_ATTESTATIONS:=1}"
: "${DOCKER_BUILDKIT:=0}"
: "${COMPOSE_DOCKER_CLI_BUILD:=0}"
: "${COMPOSE_PARALLEL_LIMIT:=1}"

export BUILDX_NO_DEFAULT_ATTESTATIONS
export DOCKER_BUILDKIT
export COMPOSE_DOCKER_CLI_BUILD
export COMPOSE_PARALLEL_LIMIT

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

if [[ -f .env.timeweb ]]; then
  load_env_file ./.env.timeweb
fi

if [[ -f core-api/.env ]]; then
  load_env_file ./core-api/.env
fi

if [[ -f max-bot-service/.env ]]; then
  load_env_file ./max-bot-service/.env
fi

docker compose -f "${COMPOSE_FILE}" build core-api
docker compose -f "${COMPOSE_FILE}" build max-bot-service
docker compose -f "${COMPOSE_FILE}" build max-miniapp
docker compose -f "${COMPOSE_FILE}" up -d postgres

if [[ -f ./max-bot-service/data/tveractive.db ]]; then
  docker compose -f "${COMPOSE_FILE}" run --rm \
    -v "${PWD}/core-api:/work" \
    -v "${PWD}/max-bot-service/data:/bot-data" \
    -w /work \
    -e SQLITE_DB_PATH=/bot-data/tveractive.db \
    core-api sh -lc 'npm ci --include=dev --no-audit --no-fund && npm run migrate:sqlite-to-postgres'
  rm -rf ./core-api/node_modules
else
  echo "SQLite database not found; skipped SQLite -> Postgres migration"
fi

docker compose -f "${COMPOSE_FILE}" up -d
docker compose -f "${COMPOSE_FILE}" ps

if [[ -n "${CORE_API_INTERNAL_TOKEN:-}" ]]; then
  PUBLIC_BASE_URL="${PUBLIC_BASE_URL}" CORE_API_INTERNAL_TOKEN="${CORE_API_INTERNAL_TOKEN}" ./infra/smoke-deploy.sh
else
  echo "CORE_API_INTERNAL_TOKEN is not exported; skipped internal smoke"
fi
