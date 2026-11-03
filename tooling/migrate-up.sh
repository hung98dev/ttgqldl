#!/usr/bin/env bash
# Applies all pending PostgreSQL migrations with the schema-owner DSN.
set -euo pipefail

# shellcheck source=_common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

ENV_FILE="${REPO_ROOT}/.env"
[[ -f "${ENV_FILE}" ]] || die ".env is missing; run ./dev-up.sh (it creates .env from .env.example)"

export TTG_ENV_FILE="${ENV_FILE}"
log_step "applying migrations"
code=0
invoke_go "${REPO_ROOT}/backend" run ./cmd/migrate -direction up || code=$?
if [[ ${code} -ne 0 ]]; then
    log_fail "migrate up failed with exit code ${code}"
    exit "${code}"
fi
log_ok "migrations are up to date"
