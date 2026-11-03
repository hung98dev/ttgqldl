#!/usr/bin/env bash
# Builds the Windows x86_64 standalone player with the pinned editor.
#
# Exit codes:
#   0  player built and verified on disk
#   1  build failed
#   2  BLOCKED: editor missing, wrong version, missing Windows module or licence
#
# Usage:
#   ./tooling/unity-build.sh
# Output: unity/Build/Windows/TTGQLDL.exe (gitignored).
set -euo pipefail

# shellcheck source=_common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

REQUIRED_UNITY_VERSION="6000.6.7f1"
PROJECT="${REPO_ROOT}/unity"
LOGS="${PROJECT}/TestResults"
PLAYER="${PROJECT}/Build/Windows/TTGQLDL.exe"
LOGFILE="${LOGS}/build.log"

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

VERSION="$("${EDITOR}" -version 2>&1 | head -n 1 || true)"
if [[ ! "${VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+[abfpx][0-9]+ ]]; then
    log_fail "BLOCKED: '${EDITOR}' is not a Unity editor (version output: ${VERSION:-<empty>})"
    exit 2
fi
if [[ "${VERSION}" != ${REQUIRED_UNITY_VERSION}* ]]; then
    log_fail "BLOCKED: editor version '${VERSION}' does not match the pinned ${REQUIRED_UNITY_VERSION}"
    exit 2
fi

EDITOR_ROOT="$(cd "$(dirname "${EDITOR}")/../.." && pwd)"
if [[ ! -d "${EDITOR_ROOT}/Editor/Data/PlaybackEngines/windowsstandalonesupport" ]]; then
    log_fail "BLOCKED: the Windows standalone build module is not installed for this editor"
    exit 2
fi

mkdir -p "${LOGS}"
rm -f "${LOGFILE}"
rm -rf "${PROJECT}/Build"

log_step "building Windows player with Unity ${VERSION}"
set +e
"${EDITOR}" -batchmode -nographics -quit -projectPath "${PROJECT}" \
    -executeMethod Ttgqldl.Editor.BuildWindows.PerformBuild -logFile "${LOGFILE}"
code=$?
set -e

if [[ "${code}" -ne 0 ]]; then
    log_fail "build failed with exit code ${code}"
    if [[ -f "${LOGFILE}" ]]; then
        grep -E 'error CS|Error building|\[TTGQLDL\] build failed' "${LOGFILE}" | head -n 10 || true
    fi
    exit 1
fi

if [[ ! -f "${PLAYER}" ]]; then
    log_fail "build reported success but ${PLAYER} is missing"
    exit 1
fi

size_mb="$(du -sm "${PROJECT}/Build/Windows" 2>/dev/null | awk '{print $1}')"
log_ok "player built: ${PLAYER} (${size_mb:-?} MB)"
exit 0
