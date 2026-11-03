#!/usr/bin/env bash
# Runs the TTGQLDL backend server from source with the repository .env.
set -euo pipefail

# shellcheck source=_common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

ENV_FILE="${REPO_ROOT}/.env"

[[ -f "${ENV_FILE}" ]] || die ".env is missing; run ./dev-up.sh (it creates .env from .env.example)"
have go || die "go is not installed or not on PATH"
[[ -d "${REPO_ROOT}/backend" ]] || die "backend/ directory not found"

export TTG_ENV_FILE="${ENV_FILE}"
log_step "starting backend (Ctrl+C to stop gracefully)"
log_note "TTG_ENV_FILE=${ENV_FILE}"

use_pinned_go_toolchain
cd "${REPO_ROOT}/backend"
exec go run ./cmd/server
