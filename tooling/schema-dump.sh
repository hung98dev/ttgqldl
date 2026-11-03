#!/usr/bin/env bash
# Regenerates backend/db/schema.sql from the migrated development database.
#
# sqlc cannot parse goose migration files (they contain down migrations and
# markers), so it reads a schema snapshot. Run this after every migration change
# and commit the result; `/tmp` is used for the intermediate file.
set -euo pipefail

# shellcheck source=_common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

ENV_FILE="${REPO_ROOT}/.env"
COMPOSE_FILE="${REPO_ROOT}/compose.dev.yaml"
TARGET="${REPO_ROOT}/backend/db/schema.sql"

[[ -f "${ENV_FILE}" ]] || die ".env is missing; run ./dev-up.sh first"
have docker || die "docker is not installed or not on PATH"

CONTAINER_ID="$(cd "${REPO_ROOT}" && docker compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" ps -q postgres 2>/dev/null | tr -d '[:space:]')"
[[ -n "${CONTAINER_ID}" ]] || die "the postgres container is not running; run ./dev-up.sh"

log_step "dumping schema ttg from container ${CONTAINER_ID:0:12}"
TMP_FILE="$(mktemp)"
trap 'rm -f "${TMP_FILE}"' EXIT

docker exec "${CONTAINER_ID}" pg_dump -U ttg_owner -d ttgqldl \
    --schema-only --no-owner --no-privileges --schema=ttg --exclude-table=ttg.goose_db_version |
    grep -Ev '^\\|^SET |^SELECT pg_catalog\.set_config' >"${TMP_FILE}"

{
    echo '-- GENERATED FILE - do not edit by hand.'
    echo '-- Snapshot of the schema produced by backend/db/migrations, used only by sqlc:'
    echo '-- the goose files carry down migrations and markers that sqlc cannot parse.'
    echo '-- Regenerate with tooling/schema-dump.ps1 (PowerShell) or schema-dump.sh after'
    echo '-- changing a migration; CI fails if the snapshot drifts from the migrations.'
    echo
    cat "${TMP_FILE}"
} >"${TARGET}"

log_ok "wrote ${TARGET} ($(wc -c <"${TARGET}" | tr -d ' ') bytes)"
log_note "next: cd backend && sqlc generate"
