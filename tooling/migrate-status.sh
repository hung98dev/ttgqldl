#!/usr/bin/env bash
# Shows which migrations are applied and which are pending.
set -euo pipefail

# shellcheck source=_common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

ENV_FILE="${REPO_ROOT}/.env"
[[ -f "${ENV_FILE}" ]] || die ".env is missing; run ./dev-up.sh first"

export TTG_ENV_FILE="${ENV_FILE}"
log_step "migration status"
code=0
invoke_go "${REPO_ROOT}/backend" run ./cmd/migrate -direction status || code=$?
if [[ ${code} -ne 0 ]]; then
    log_fail "migrate status failed with exit code ${code}"
    exit "${code}"
fi
