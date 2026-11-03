#!/usr/bin/env bash
# Starts the operator console against the local Go backend.
#
# Vite proxies /healthz, /readyz and /admin/v1 to TTG_OPERATOR_BACKEND_URL
# (default http://127.0.0.1:8080). No database is exposed to the browser.
#
# Usage:
#   ./tooling/operator-run.sh            # real-service mode
#   ./tooling/operator-run.sh --demo     # fixed fixtures + MOCK banner
set -euo pipefail

# shellcheck source=_common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

PROJECT="${REPO_ROOT}/operator"
DEMO="false"

for argument in "$@"; do
    case "${argument}" in
        --demo) DEMO="true" ;;
        *)
            log_fail "unknown option '${argument}' (expected --demo)"
            exit 2
            ;;
    esac
done

if ! have node; then
    log_fail "node was not found on PATH; Node 24 is required for the operator console"
    exit 2
fi

NODE_VERSION="$(node --version | sed 's/^v//')"
NODE_MAJOR="${NODE_VERSION%%.*}"
if [[ "${NODE_MAJOR}" -lt 24 ]]; then
    log_fail "node ${NODE_VERSION} is too old; the operator requires Node 24 LTS"
    exit 2
fi
log_ok "node ${NODE_VERSION}"

if [[ ! -d "${PROJECT}/node_modules" ]]; then
    log_step "installing operator dependencies (npm ci)"
    ( cd "${PROJECT}" && npm ci --no-audit --no-fund )
fi

log_note "backend proxy -> ${TTG_OPERATOR_BACKEND_URL:-http://127.0.0.1:8080}"

export VITE_OPERATOR_DEMO="${DEMO}"
if [[ "${DEMO}" == "true" ]]; then
    log_warn "DEMO mode: the console will show fixed fixtures and a MOCK banner"
else
    log_note "real-service mode: no fabricated data"
fi

log_step "starting Vite on http://127.0.0.1:5173"
cd "${PROJECT}"
exec npm run dev
