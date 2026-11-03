#!/usr/bin/env bash
# Runs the M00 baseline tooling gates: plan validator and golden vectors.
#
#   go vet ./...  &&  go test ./...
#   go run ./planvalidate -root ..
#   go run ./refeval -check
#
# The validator is strict by design: a status claim that outruns its evidence
# fails the run. The reference evaluator is independent of the backend and must
# never be edited to make backend tests pass.
set -uo pipefail

# shellcheck source=_common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

TOOLING="${REPO_ROOT}/tooling"
have go || die "go is not installed or not on PATH"

invoke_go_step "tooling go vet" "${TOOLING}" vet ./...
invoke_go_step "tooling go test" "${TOOLING}" test ./...

log_step "plan validator (go run ./planvalidate -root ..)"
code=0
invoke_go "${TOOLING}" run ./planvalidate -root "${REPO_ROOT}" || code=$?
if [[ ${code} -ne 0 ]]; then
    log_fail "plan validator failed with exit code ${code}"
    exit "${code}"
fi
log_ok "plan validator"

log_step "golden vectors (go run ./refeval -check)"
code=0
invoke_go "${TOOLING}" run ./refeval -check || code=$?
if [[ ${code} -ne 0 ]]; then
    log_fail "reference evaluator failed with exit code ${code}"
    exit "${code}"
fi
log_ok "golden vectors"

log_ok "baseline tooling gates passed"
