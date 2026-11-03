# Shared helpers for the Bash entry points in tooling/.
# Source this file; it never changes the caller's working directory.

# shellcheck shell=bash

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REQUIRED_GO_VERSION="1.27.3"

log_step() { printf '==> %s\n' "$*"; }
log_ok()   { printf 'OK   %s\n' "$*"; }
log_note() { printf '     %s\n' "$*"; }
log_warn() { printf 'WARN %s\n' "$*" >&2; }
log_fail() { printf 'FAIL %s\n' "$*" >&2; }

die() {
    log_fail "$*"
    exit 1
}

have() { command -v "$1" >/dev/null 2>&1; }

# See the note in _common.ps1: the module pins the toolchain, so allow Go to
# select it instead of failing on a stale host pin. Announced, process-scoped,
# and it never edits the user's global Go configuration.
use_pinned_go_toolchain() {
    have go || return 0
    local current
    current="$(go env GOTOOLCHAIN 2>/dev/null || true)"
    if [[ "${current}" =~ ^go[0-9]+\.[0-9]+\.[0-9]+$ ]] && [[ "${current}" != "go${REQUIRED_GO_VERSION}" ]]; then
        export GOTOOLCHAIN=auto
        log_note "[go] GOTOOLCHAIN '${current}' overridden to 'auto' for this command so backend/go.mod can select go${REQUIRED_GO_VERSION}"
    fi
}

# invoke_go runs `go` in a directory and returns its exit code.
invoke_go() {
    local dir="$1"
    shift
    use_pinned_go_toolchain
    ( cd "${dir}" && go "$@" )
}

# invoke_go_step runs one go command and exits the script on failure.
invoke_go_step() {
    local label="$1"
    local dir="$2"
    shift 2
    log_step "${label} (go $*)"
    local code=0
    invoke_go "${dir}" "$@" || code=$?
    if [[ ${code} -ne 0 ]]; then
        log_fail "${label} failed with exit code ${code}"
        exit "${code}"
    fi
    log_ok "${label}"
}

# confirm_destructive asks for an explicit confirmation on a TTY.
confirm_destructive() {
    local prompt="$1"
    if [[ ! -t 0 ]]; then
        log_fail "${prompt}: refusing to continue without an interactive confirmation"
        return 1
    fi
    local answer=""
    read -r -p "${prompt} [y/N] " answer
    [[ "${answer}" =~ ^(y|yes)$ ]]
}
