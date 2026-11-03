#!/usr/bin/env bash
# Starts the TTGQLDL development dependencies (PostgreSQL 18.6).
#
# Creates .env from .env.example when missing, then runs
#   docker compose --env-file .env -f compose.dev.yaml up -d --wait
# It never deletes data and never modifies an existing .env.
set -euo pipefail

# shellcheck source=_common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

ENV_FILE="${REPO_ROOT}/.env"
COMPOSE_FILE="${REPO_ROOT}/compose.dev.yaml"

have docker || die "docker is not installed or not on PATH"
[[ -f "${COMPOSE_FILE}" ]] || die "compose file not found: ${COMPOSE_FILE}"

if [[ ! -f "${ENV_FILE}" ]]; then
    [[ -f "${REPO_ROOT}/.env.example" ]] || die ".env is missing and .env.example is not present to copy from"
    cp "${REPO_ROOT}/.env.example" "${ENV_FILE}"
    log_warn "created .env from .env.example (development placeholder secrets, loopback only)"
    log_note "edit .env if you need different ports; never commit it"
fi

log_step "starting development dependencies"
cd "${REPO_ROOT}"
code=0
docker compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" up -d --wait || code=$?
if [[ ${code} -ne 0 ]]; then
    log_fail "docker compose up failed with exit code ${code}"
    exit "${code}"
fi
docker compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" ps || true

log_ok "development dependencies are up (data volume preserved across restarts)"
log_note "connection: postgres://ttg_app@127.0.0.1:5432/ttgqldl (least-privileged app role)"
