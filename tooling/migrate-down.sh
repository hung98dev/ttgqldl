#!/usr/bin/env bash
# Rolls back PostgreSQL migrations.
#
#   ./migrate-down.sh          roll back exactly one migration
#   ./migrate-down.sh --to 0   roll back everything (asks for confirmation)
set -euo pipefail

# shellcheck source=_common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

ENV_FILE="${REPO_ROOT}/.env"
TO=-1

while [[ $# -gt 0 ]]; do
    case "$1" in
        --to)
            [[ $# -ge 2 ]] || die "--to needs a version number"
            TO="$2"
            shift 2
            ;;
        -h | --help)
            sed -n '2,6p' "$0"
            exit 0
            ;;
        *) die "unknown argument: $1" ;;
    esac
done

[[ -f "${ENV_FILE}" ]] || die ".env is missing; run ./dev-up.sh first"

ARGS=(run ./cmd/migrate -direction down)
if [[ ${TO} -ge 0 ]]; then
    if [[ ${TO} -eq 0 ]]; then
        log_warn "rolling back to version 0 drops every migration table and its local data"
        confirm_destructive "Roll back all migrations and delete local data?" ||
            die "refusing to roll back everything without confirmation"
    fi
    ARGS+=(-to "${TO}")
fi

export TTG_ENV_FILE="${ENV_FILE}"
log_step "rolling back migrations"
code=0
invoke_go "${REPO_ROOT}/backend" "${ARGS[@]}" || code=$?
if [[ ${code} -ne 0 ]]; then
    log_fail "migrate down failed with exit code ${code}"
    exit "${code}"
fi
log_ok "rollback complete"
