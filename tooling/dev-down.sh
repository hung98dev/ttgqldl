#!/usr/bin/env bash
# Stops the TTGQLDL development dependencies without deleting data.
#
#   ./dev-down.sh              stop, keep the PostgreSQL volume
#   ./dev-down.sh --wipe-data  stop and delete the volume (asks for confirmation)
set -euo pipefail

# shellcheck source=_common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

ENV_FILE="${REPO_ROOT}/.env"
COMPOSE_FILE="${REPO_ROOT}/compose.dev.yaml"
WIPE=0

for arg in "$@"; do
    case "${arg}" in
        --wipe-data) WIPE=1 ;;
        -h | --help)
            sed -n '2,6p' "$0"
            exit 0
            ;;
        *) die "unknown argument: ${arg}" ;;
    esac
done

[[ -f "${COMPOSE_FILE}" ]] || die "compose file not found: ${COMPOSE_FILE}"
[[ -f "${ENV_FILE}" ]] || die ".env is missing; run ./dev-up.sh first (nothing to stop)"

ARGS=(compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" down)
if [[ ${WIPE} -eq 1 ]]; then
    log_warn "this deletes the development PostgreSQL volume and all local data"
    confirm_destructive "Delete the development database volume?" || die "refusing to delete data without confirmation"
    ARGS+=(--volumes)
fi

log_step "stopping development dependencies"
cd "${REPO_ROOT}"
code=0
docker "${ARGS[@]}" || code=$?
if [[ ${code} -ne 0 ]]; then
    log_fail "docker compose down failed with exit code ${code}"
    exit "${code}"
fi
if [[ ${WIPE} -eq 1 ]]; then
    log_ok "development dependencies stopped and the data volume was removed"
else
    log_ok "development dependencies stopped; the data volume was preserved"
fi
