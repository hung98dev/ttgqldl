#!/usr/bin/env bash
# Runs the backend quality gates: gofmt, go vet, unit tests, optional race tests.
#
#   ./backend-test.sh            gofmt + vet + test
#   ./backend-test.sh --race     also go test -race (requires cgo)
#   ./backend-test.sh -short     pass -short to the test runner
#
# Every step propagates its exit code; the script never reports success for a
# failing command.
set -uo pipefail

# shellcheck source=_common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

BACKEND="${REPO_ROOT}/backend"
RACE=0
SHORT_ARG=""

for arg in "$@"; do
    case "${arg}" in
        --race) RACE=1 ;;
        -short | --short) SHORT_ARG="-short" ;;
        -h | --help)
            sed -n '2,10p' "$0"
            exit 0
            ;;
        *) die "unknown argument: ${arg}" ;;
    esac
done

have go || die "go is not installed or not on PATH"
[[ -d "${BACKEND}" ]] || die "backend/ directory not found"

use_pinned_go_toolchain

log_step "gofmt check"
UNFORMATTED="$(cd "${BACKEND}" && gofmt -l .)"
if [[ -n "${UNFORMATTED}" ]]; then
    log_fail "gofmt found unformatted files:"
    printf '%s\n' "${UNFORMATTED}" >&2
    exit 1
fi
log_ok "gofmt"

if [[ -n "${SHORT_ARG}" ]]; then
    invoke_go_step "go vet" "${BACKEND}" vet ./...
    invoke_go_step "go test" "${BACKEND}" test -short ./...
else
    invoke_go_step "go vet" "${BACKEND}" vet ./...
    invoke_go_step "go test" "${BACKEND}" test ./...
fi

if [[ ${RACE} -eq 1 ]]; then
    log_step "go test -race (requires cgo)"
    code=0
    if [[ -n "${SHORT_ARG}" ]]; then
        invoke_go "${BACKEND}" test -race -short ./... || code=$?
    else
        invoke_go "${BACKEND}" test -race ./... || code=$?
    fi
    if [[ ${code} -ne 0 ]]; then
        log_fail "go test -race failed with exit code ${code} (on Windows this usually means cgo/gcc is unavailable; the Linux CI job is the authoritative race gate)"
        exit "${code}"
    fi
    log_ok "go test -race"
fi

log_ok "backend quality gates passed"
