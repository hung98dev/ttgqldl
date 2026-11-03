#!/usr/bin/env bash
# Runs the operator console quality gates in order.
#
# Exit codes: 0 all gates passed, 1 a gate failed, 2 the toolchain is missing.
set -euo pipefail

# shellcheck source=_common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

PROJECT="${REPO_ROOT}/operator"

if ! have node; then
    log_fail "BLOCKED: node was not found on PATH; Node 24 is required for the operator console"
    exit 2
fi

NODE_VERSION="$(node --version | sed 's/^v//')"
NODE_MAJOR="${NODE_VERSION%%.*}"
if [[ "${NODE_MAJOR}" -lt 24 ]]; then
    log_fail "BLOCKED: node ${NODE_VERSION} is too old; the operator requires Node 24 LTS"
    exit 2
fi
log_ok "node ${NODE_VERSION}"

run_step() {
    local name="$1"
    shift
    log_step "${name}"
    if ! ( cd "${PROJECT}" && "$@" ); then
        log_fail "${name} failed"
        exit 1
    fi
    log_ok "${name} passed"
}

log_step "installing locked dependencies (npm ci)"
if ! ( cd "${PROJECT}" && npm ci --no-audit --no-fund ); then
    log_fail "npm ci failed"
    exit 1
fi

run_step "typecheck" npm run typecheck
run_step "lint" npm run lint
run_step "tests" npm run test -- --run
run_step "production build" npm run build

log_ok "operator: all gates passed"
exit 0
