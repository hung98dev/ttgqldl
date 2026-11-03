#!/usr/bin/env bash
# Runs the real Unity EditMode and/or PlayMode test suites and reports from the
# produced NUnit XML.
#
# Acceptance rules (identical to tooling/unity-test.ps1):
#   - the result file must exist and contain a <test-run> element
#   - total, passed, failed, skipped and inconclusive must be present, integers
#     and non-negative
#   - total must equal passed+failed+skipped+inconclusive
#   - at least one test must have actually executed (not skipped/inconclusive)
#   - zero failures, and Unity must exit 0
#   - a stale result file cannot be reused: it is deleted before the run
#
# Exit codes:
#   0  every requested platform passed
#   1  at least one test failed, or the result was empty/inconsistent/missing
#   2  BLOCKED: editor missing, wrong version, or no usable licence
#   3  harness error
#
# Usage:
#   ./tooling/unity-test.sh              # both platforms
#   ./tooling/unity-test.sh EditMode
#   TTG_UNITY_EDITOR=/path/to/Unity ./tooling/unity-test.sh
set -euo pipefail

# shellcheck source=_common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

REQUIRED_UNITY_VERSION="6000.6.7f1"
PROJECT="${REPO_ROOT}/unity"
RESULTS="${PROJECT}/TestResults"
PLATFORM="${1:-All}"

resolve_editor() {
    if [[ -n "${TTG_UNITY_EDITOR:-}" ]]; then
        if [[ -x "${TTG_UNITY_EDITOR}" ]]; then
            printf '%s\n' "${TTG_UNITY_EDITOR}"
            return 0
        fi
        log_warn "TTG_UNITY_EDITOR points to a missing or non-executable path: ${TTG_UNITY_EDITOR}"
    fi
    local candidates=(
        "/opt/unity/editors/${REQUIRED_UNITY_VERSION}/Editor/Unity"
        "/opt/unity/Editor/Unity"
        "/Applications/Unity/Hub/Editor/${REQUIRED_UNITY_VERSION}/Unity.app/Contents/MacOS/Unity"
    )
    local candidate
    for candidate in "${candidates[@]}"; do
        if [[ -x "${candidate}" ]]; then
            printf '%s\n' "${candidate}"
            return 0
        fi
    done
    if have Unity; then
        command -v Unity
        return 0
    fi
    return 1
}

if ! EDITOR="$(resolve_editor)"; then
    log_fail "BLOCKED: Unity ${REQUIRED_UNITY_VERSION} was not found. Set TTG_UNITY_EDITOR to the Unity executable."
    exit 2
fi

log_step "using editor ${EDITOR}"
VERSION="$("${EDITOR}" -version 2>&1 | head -n 1 || true)"
# A bare "Unity" on PATH may be the Hub CLI wrapper, which rejects -version.
# Require a real version string before trusting the binary.
if [[ ! "${VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+[abfpx][0-9]+ ]]; then
    log_fail "BLOCKED: '${EDITOR}' is not a Unity editor (version output: ${VERSION:-<empty>})"
    exit 2
fi
if [[ "${VERSION}" != ${REQUIRED_UNITY_VERSION}* ]]; then
    log_fail "BLOCKED: editor version ${VERSION} does not match the pinned ${REQUIRED_UNITY_VERSION}"
    exit 2
fi
log_ok "editor version ${VERSION}"

mkdir -p "${RESULTS}"
case "${PLATFORM}" in
    All) PLATFORMS=("EditMode" "PlayMode") ;;
    EditMode|PlayMode) PLATFORMS=("${PLATFORM}") ;;
    *)
        log_fail "unknown platform '${PLATFORM}' (expected EditMode, PlayMode or All)"
        exit 3
        ;;
esac

counter_of() {
    local xml="$1" name="$2"
    grep -o "${name}=\"[0-9]*\"" "${xml}" | head -n 1 | grep -o '[0-9]*' || true
}

FAILED=()
BLOCKED=0

for current in "${PLATFORMS[@]}"; do
    xml="${RESULTS}/$(printf '%s' "${current}" | tr '[:upper:]' '[:lower:]').xml"
    logfile="${RESULTS}/$(printf '%s' "${current}" | tr '[:upper:]' '[:lower:]').log"
    # Removing the previous result makes a stale file impossible.
    rm -f "${xml}" "${logfile}"

    log_step "running ${current} tests"
    set +e
    "${EDITOR}" -batchmode -nographics -projectPath "${PROJECT}" \
        -runTests -testPlatform "${current}" -testResults "${xml}" -logFile "${logfile}"
    code=$?
    set -e

    if [[ ! -f "${xml}" ]]; then
        if [[ -f "${logfile}" ]] && grep -Eq 'No valid Unity Editor license|License is not active|licensing.*error' "${logfile}"; then
            log_fail "BLOCKED: ${current} tests could not run because no usable Unity licence was found"
            BLOCKED=1
            continue
        fi
        log_fail "${current}: Unity exited with ${code} and produced no result file; refusing to report a result"
        FAILED+=("${current}")
        continue
    fi

    if ! grep -q '<test-run' "${xml}"; then
        log_fail "${current}: the result file has no <test-run> element; refusing to report a result"
        FAILED+=("${current}")
        continue
    fi

    total="$(counter_of "${xml}" total)"
    passed="$(counter_of "${xml}" passed)"
    failed_count="$(counter_of "${xml}" failed)"
    skipped="$(counter_of "${xml}" skipped)"
    inconclusive="$(counter_of "${xml}" inconclusive)"

    if [[ -z "${total}" || -z "${passed}" || -z "${failed_count}" || -z "${skipped}" || -z "${inconclusive}" ]]; then
        log_fail "${current}: missing or non-integer NUnit counters; refusing to report a result"
        FAILED+=("${current}")
        continue
    fi

    executed=$((total - skipped - inconclusive))
    counted=$((passed + failed_count + skipped + inconclusive))
    problem=""
    if [[ "${total}" -eq 0 ]]; then
        problem="no tests were discovered or executed"
    elif [[ "${total}" -ne "${counted}" ]]; then
        problem="inconsistent counters: total=${total} but passed+failed+skipped+inconclusive=${counted}"
    elif [[ "${executed}" -le 0 ]]; then
        problem="no test was actually executed (${skipped} skipped, ${inconclusive} inconclusive of ${total})"
    elif [[ "${passed}" -eq 0 && "${failed_count}" -eq 0 ]]; then
        problem="no passing or failing test reported (total=${total})"
    fi

    if [[ -n "${problem}" ]]; then
        log_fail "${current}: ${problem}"
        FAILED+=("${current}")
        continue
    fi

    if [[ "${failed_count}" -gt 0 ]]; then
        log_fail "${current}: ${failed_count} of ${total} failed"
        FAILED+=("${current}")
        continue
    fi

    if [[ "${code}" -ne 0 ]]; then
        log_fail "${current}: Unity exited with ${code} although the result file reports no failures"
        FAILED+=("${current}")
        continue
    fi

    log_ok "${current}: total=${total} passed=${passed} failed=${failed_count} skipped=${skipped} executed=${executed}"
done

if [[ "${BLOCKED}" -eq 1 ]]; then
    exit 2
fi
if [[ ${#FAILED[@]} -gt 0 ]]; then
    log_fail "unity test: failed platform(s) ${FAILED[*]}"
    exit 1
fi
log_ok "unity test: all requested platforms passed"
exit 0
