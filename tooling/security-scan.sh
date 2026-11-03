#!/usr/bin/env bash
# Runs the security gates against the actual repository.
#
# Gates (all required; none may be skipped silently):
#   1. govulncheck   Go vulnerability scan of the backend module
#   2. gosec         Go static analysis (fails on any finding or package load error)
#   3. npm audit     high/critical dependency vulnerabilities (dev moderates reported)
#   4. gitleaks      secret scan with the reviewed .gitleaks.toml config
#   5. actionlint    GitHub Actions workflow validation
#   6. sbom          container digest pinning + CycloneDX inventory
#
# Exit codes: 0 all gates passed, 1 a gate failed, 2 BLOCKED (tool missing).
set -euo pipefail

# shellcheck source=_common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

REPORTS="${TTG_SECURITY_REPORTS:-${REPO_ROOT}/security-reports}"
TOOL_DIR="${TTG_SECURITY_BIN:-}"
mkdir -p "${REPORTS}"
export GOTOOLCHAIN=auto

resolve_tool() {
    local name="$1"
    if [[ -n "${TOOL_DIR}" ]]; then
        if [[ -x "${TOOL_DIR}/${name}" ]]; then
            printf '%s\n' "${TOOL_DIR}/${name}"
            return 0
        fi
        if [[ -x "${TOOL_DIR}/${name}.exe" ]]; then
            printf '%s\n' "${TOOL_DIR}/${name}.exe"
            return 0
        fi
    fi
    if have "${name}"; then
        command -v "${name}"
        return 0
    fi
    return 1
}

declare -A TOOLS=()
for tool in govulncheck gosec gitleaks actionlint npm; do
    if ! resolved="$(resolve_tool "${tool}")"; then
        log_fail "BLOCKED: ${tool} is not installed. Install the pinned tool set (see tooling/README.md)."
        exit 2
    fi
    TOOLS["${tool}"]="${resolved}"
    log_note "${tool} -> ${resolved}"
done

FAILURES=()

# --- 1. govulncheck -----------------------------------------------------------
log_step "govulncheck (Go vulnerabilities)"
set +e
( cd "${REPO_ROOT}/backend" && "${TOOLS[govulncheck]}" ./... ) >"${REPORTS}/govulncheck.txt" 2>&1
vuln_code=$?
set -e
if [[ ${vuln_code} -ne 0 ]] || grep -q 'Vulnerability #' "${REPORTS}/govulncheck.txt"; then
    log_fail "govulncheck reported vulnerabilities"
    FAILURES+=("govulncheck")
else
    log_ok "govulncheck: no vulnerabilities"
fi

# --- 2. gosec -----------------------------------------------------------------
log_step "gosec (Go static analysis)"
GOSEC_REPORT="${REPORTS}/gosec.json"
rm -f "${GOSEC_REPORT}"
set +e
( cd "${REPO_ROOT}/backend" && "${TOOLS[gosec]}" -fmt=json -out "${GOSEC_REPORT}" ./... ) >/dev/null 2>&1
gosec_code=$?
set -e
if [[ ! -f "${GOSEC_REPORT}" ]]; then
    log_fail "gosec produced no report; refusing to treat that as a pass"
    FAILURES+=("gosec")
else
    issues="$(grep -c '"rule_id"' "${GOSEC_REPORT}" || true)"
    load_errors="$(grep -c 'Golang errors": {' "${GOSEC_REPORT}" || true)"
    golang_error_entries="$(grep -c '"error": "loading files' "${GOSEC_REPORT}" || true)"
    if [[ "${issues}" -gt 0 ]]; then
        log_fail "gosec: ${issues} finding(s); see ${GOSEC_REPORT}"
        FAILURES+=("gosec")
    elif [[ "${golang_error_entries}" -gt 0 ]]; then
        log_fail "gosec could not load ${golang_error_entries} package(s); the scan was incomplete"
        FAILURES+=("gosec")
    elif [[ ${gosec_code} -ne 0 ]]; then
        log_fail "gosec exited with ${gosec_code} but reported no parsed findings"
        FAILURES+=("gosec")
    else
        log_ok "gosec: no findings"
    fi
    : "${load_errors}"
fi

# --- 3. npm audit -------------------------------------------------------------
log_step "npm audit (dependency vulnerabilities)"
AUDIT_REPORT="${REPORTS}/npm-audit.json"
set +e
( cd "${REPO_ROOT}/operator" && "${TOOLS[npm]}" audit --json ) >"${AUDIT_REPORT}" 2>/dev/null
( cd "${REPO_ROOT}/operator" && "${TOOLS[npm]}" audit --audit-level=high --omit=dev ) >/dev/null 2>&1
audit_code=$?
set -e
count_of() {
    local key="$1"
    if [[ -f "${AUDIT_REPORT}" ]]; then
        local value
        value="$(grep -oE "\"${key}\"[[:space:]]*:[[:space:]]*[0-9]+" "${AUDIT_REPORT}" | head -n 1 | grep -oE '[0-9]+' || true)"
        printf '%s' "${value:-0}"
        return 0
    fi
    printf '0'
}
moderate="$(count_of moderate)"
high="$(count_of high)"
critical="$(count_of critical)"
log_note "severity counts: critical=${critical} high=${high} moderate=${moderate}"
if [[ ${audit_code} -ne 0 ]] || [[ "${high}" -gt 0 ]] || [[ "${critical}" -gt 0 ]]; then
    log_fail "npm audit found high or critical vulnerabilities in runtime dependencies"
    FAILURES+=("npm-audit")
else
    log_ok "npm audit: no high/critical runtime findings (${moderate} moderate, dev-only)"
fi

# --- 4. gitleaks --------------------------------------------------------------
log_step "gitleaks (secret scan)"
set +e
"${TOOLS[gitleaks]}" detect --source "${REPO_ROOT}" --no-git --redact \
    --config "${REPO_ROOT}/.gitleaks.toml" \
    --report-format json --report-path "${REPORTS}/gitleaks.json" >/dev/null 2>&1
leaks_code=$?
set -e
if [[ ${leaks_code} -ne 0 ]]; then
    log_fail "gitleaks found potential secrets"
    FAILURES+=("gitleaks")
else
    log_ok "gitleaks: no leaks"
fi

# --- 5. actionlint ------------------------------------------------------------
log_step "actionlint (GitHub Actions)"
set +e
"${TOOLS[actionlint]}" "${REPO_ROOT}"/.github/workflows/*.yml
lint_code=$?
set -e
if [[ ${lint_code} -ne 0 ]]; then
    log_fail "actionlint reported workflow problems"
    FAILURES+=("actionlint")
else
    log_ok "actionlint: workflows are valid"
fi

# --- 6. SBOM and container pinning -------------------------------------------
log_step "sbom (inventory + container pinning)"
set +e
( cd "${REPO_ROOT}/tooling" && go run ./sbom -root "${REPO_ROOT}" -check-images )
pin_code=$?
( cd "${REPO_ROOT}/tooling" && go run ./sbom -root "${REPO_ROOT}" -out "${REPORTS}/sbom.cdx.json" )
sbom_code=$?
set -e
if [[ ${pin_code} -ne 0 ]] || [[ ${sbom_code} -ne 0 ]]; then
    log_fail "container pinning or SBOM generation failed"
    FAILURES+=("sbom")
else
    log_ok "sbom: inventory written and every container image is digest-pinned"
fi

if [[ ${#FAILURES[@]} -gt 0 ]]; then
    log_fail "security: failed gate(s): ${FAILURES[*]}"
    exit 1
fi

log_ok "security: all gates passed (reports in ${REPORTS})"
exit 0
