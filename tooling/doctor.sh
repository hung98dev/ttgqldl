#!/usr/bin/env bash
# Validates the local TTGQLDL development environment against the pinned baseline.
#
# Statuses:
#   OK        correct version and usable
#   FAIL      wrong version, missing required tool, or unusable install
#   MISSING   not installed
#   MISMATCH  installed but not the pinned version
#   BLOCKED   present but unusable because of permissions/licence
#
# Required for M01 Phase A: Go 1.27.3, Docker + Compose plugin, the pinned
# PostgreSQL 18.6 image digest and >=5 GiB free disk. Unity (M06), Node
# (M01-W0105) and a local .env are reported but do not fail unless --strict.
set -uo pipefail

# shellcheck source=_common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

STRICT=0
for arg in "$@"; do
    case "${arg}" in
        --strict) STRICT=1 ;;
        -h | --help)
            sed -n '2,16p' "$0"
            exit 0
            ;;
        *) die "unknown argument: ${arg}" ;;
    esac
done

PINNED_DIGEST="sha256:3a4264520e121176619f54ff867d12774bc81e95e7b84db58d63bccdb6144b81"
REQUIRED_FREE_KB=5242880 # 5 GiB
FAILURES=0
GAPS=0
ROWS=()

add_row() {
    local check="$1" status="$2" detail="$3" optional="${4:-0}"
    ROWS+=("${status}|${check}|${detail}")
    if [[ "${status}" != "OK" ]]; then
        if [[ "${optional}" == "0" || ${STRICT} -eq 1 ]]; then
            FAILURES=$((FAILURES + 1))
        fi
        if [[ "${optional}" == "1" ]]; then
            GAPS=$((GAPS + 1))
        fi
    fi
}

printf '\nTTGQLDL environment doctor\n'
printf 'repository: %s\n' "${REPO_ROOT}"
printf 'baseline:   Go %s, PostgreSQL 18.6, Unity 6000.6.7f1\n\n' "${REQUIRED_GO_VERSION}"

# --- Go toolchain ------------------------------------------------------------
if ! have go; then
    add_row "go toolchain" "FAIL" "go not found on PATH; install Go ${REQUIRED_GO_VERSION} from https://go.dev/dl/"
else
    AMBIENT="$(go env GOTOOLCHAIN 2>/dev/null || true)"
    use_pinned_go_toolchain
    GO_OUT="$(cd "${REPO_ROOT}/backend" && go version 2>&1)"
    GO_CODE=$?
    if [[ ${GO_CODE} -ne 0 ]]; then
        add_row "go toolchain" "FAIL" "cannot resolve the module toolchain: ${GO_OUT}"
    elif [[ "${GO_OUT}" =~ go\ version\ go([0-9]+\.[0-9]+\.[0-9]+) ]]; then
        if [[ "${BASH_REMATCH[1]}" == "${REQUIRED_GO_VERSION}" ]]; then
            add_row "go toolchain" "OK" "go${BASH_REMATCH[1]} (effective in backend/)"
        else
            add_row "go toolchain" "MISMATCH" "found go${BASH_REMATCH[1]}, required go${REQUIRED_GO_VERSION}; install it or run: go env -w GOTOOLCHAIN=auto"
        fi
    else
        add_row "go toolchain" "FAIL" "unrecognised go version output: ${GO_OUT}"
    fi
    if [[ "${AMBIENT}" =~ ^go[0-9]+\.[0-9]+\.[0-9]+$ && "${AMBIENT}" != "go${REQUIRED_GO_VERSION}" ]]; then
        add_row "GOTOOLCHAIN" "OK" "host pin '${AMBIENT}' (tooling passes auto automatically); permanent fix: go env -w GOTOOLCHAIN=auto"
    else
        add_row "GOTOOLCHAIN" "OK" "${AMBIENT:-auto}"
    fi
fi

# --- Docker ------------------------------------------------------------------
if ! have docker; then
    add_row "docker" "FAIL" "docker not found on PATH; install Docker Engine or Docker Desktop"
else
    SERVER="$(docker version --format '{{.Server.Version}}' 2>&1)"
    if [[ $? -ne 0 ]]; then
        if [[ "${SERVER}" == *"permission denied"* ]]; then
            add_row "docker" "BLOCKED" "daemon not reachable (permission denied): ${SERVER}"
        else
            add_row "docker" "FAIL" "daemon not reachable: ${SERVER}"
        fi
    else
        add_row "docker" "OK" "server ${SERVER}"
    fi
    COMPOSE="$(docker compose version --short 2>&1)"
    if [[ $? -ne 0 ]]; then
        add_row "docker compose" "MISSING" "compose plugin missing: ${COMPOSE}"
    else
        add_row "docker compose" "OK" "v${COMPOSE}"
    fi

    DIGESTS="$(docker image inspect 'postgres:18.6' --format '{{json .RepoDigests}}' 2>/dev/null || true)"
    if [[ -z "${DIGESTS}" ]]; then
        add_row "postgres image" "MISSING" "postgres:18.6@sha256:3a42645... not present locally; run ./dev-up.sh"
    elif [[ "${DIGESTS}" == *"${PINNED_DIGEST}"* ]]; then
        add_row "postgres image" "OK" "postgres:18.6@sha256:3a4264520e12..."
    else
        add_row "postgres image" "MISMATCH" "local postgres:18.6 digest does not match the pin: ${DIGESTS}"
    fi
fi

# --- Disk space --------------------------------------------------------------
if FREE_KB="$(df -Pk "${REPO_ROOT}" 2>/dev/null | awk 'NR==2 {print $4}')" && [[ -n "${FREE_KB}" ]]; then
    FREE_GB=$((FREE_KB / 1024 / 1024))
    if [[ ${FREE_KB} -ge ${REQUIRED_FREE_KB} ]]; then
        add_row "free disk" "OK" "${FREE_GB} GiB free"
    else
        add_row "free disk" "FAIL" "${FREE_GB} GiB free, need at least 5 GiB"
    fi
else
    add_row "free disk" "MISSING" "cannot determine free disk space"
fi

# --- Local .env --------------------------------------------------------------
if [[ -f "${REPO_ROOT}/.env" ]]; then
    add_row "env file" "OK" ".env present (gitignored)"
else
    add_row "env file" "MISSING" ".env absent; ./dev-up.sh creates it from .env.example" 1
fi

# --- Optional: Node (operator tooling, W0105) --------------------------------
if ! have node; then
    add_row "node" "MISSING" "node not found; required by M01-W0105 (operator toolchain)" 1
else
    add_row "node" "OK" "$(node --version)"
fi

# --- Optional: Unity (client, M06) -------------------------------------------
# The pinned editor only exists on the Windows capture host; licensing and
# graphics capability are proven by M06, never by this check.
UNITY_DIR=""
for base in "/c/Program Files/Unity/Hub/Editor" "/mnt/c/Program Files/Unity/Hub/Editor" \
    "${HOME}/Unity/Hub/Editor" "/opt/unity/editors"; do
    if [[ -d "${base}/6000.6.7f1" ]]; then
        UNITY_DIR="${base}/6000.6.7f1"
        break
    fi
done
if [[ -n "${UNITY_DIR}" ]]; then
    add_row "unity" "OK" "6000.6.7f1 present at ${UNITY_DIR}"
elif have unity; then
    add_row "unity" "MISMATCH" "found $(command -v unity) but no 6000.6.7f1 editor directory; required by M06" 1
else
    add_row "unity" "MISSING" "Unity 6000.6.7f1 not found in this environment; required by M06 on the Windows host" 1
fi

# --- Optional: Git Bash / shell parity ---------------------------------------
add_row "bash" "OK" "${BASH_VERSION}"

# --- Report ------------------------------------------------------------------
WIDTH=0
for row in "${ROWS[@]}"; do
    check="${row#*|}"
    check="${check%%|*}"
    (( ${#check} > WIDTH )) && WIDTH=${#check}
done
for row in "${ROWS[@]}"; do
    status="${row%%|*}"
    rest="${row#*|}"
    check="${rest%%|*}"
    detail="${rest#*|}"
    printf '[%-8s] %-*s %s\n' "${status}" "$((WIDTH + 2))" "${check}" "${detail}"
done

printf '\n'
if [[ ${FAILURES} -gt 0 ]]; then
    log_fail "doctor: ${FAILURES} required check(s) failed"
    exit 1
fi
if [[ ${GAPS} -gt 0 ]]; then
    log_ok "doctor: all required checks passed (${GAPS} optional dependency gap(s) recorded for later milestones)"
else
    log_ok "doctor: all required checks passed"
fi
exit 0
