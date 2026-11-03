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
[CmdletBinding()]
param(
    [string]$BinDirectory = '',
    [string]$ReportsDirectory = ''
)

. (Join-Path $PSScriptRoot '_common.ps1')

$root = Get-RepoRoot
$reports = if ($ReportsDirectory -ne '') { $ReportsDirectory } else { Join-Path $root 'security-reports' }
$toolDir = if ($BinDirectory -ne '') { $BinDirectory } elseif ($env:TTG_SECURITY_BIN) { $env:TTG_SECURITY_BIN } else { '' }

New-Item -ItemType Directory -Force -Path $reports | Out-Null
$env:GOTOOLCHAIN = 'auto'

function Resolve-Tool {
    param([string]$Name)
    if ($toolDir -ne '') {
        foreach ($candidate in @((Join-Path $toolDir $Name), (Join-Path $toolDir "$Name.exe"))) {
            if (Test-Path $candidate) { return $candidate }
        }
    }
    return (Get-CommandPath $Name)
}

$tools = @{}
foreach ($name in @('govulncheck', 'gosec', 'gitleaks', 'actionlint', 'npm')) {
    $resolved = Resolve-Tool $name
    if ($null -eq $resolved) {
        Write-Fail "BLOCKED: $name is not installed. Install the pinned tool set (see tooling/README.md)."
        exit 2
    }
    $tools[$name] = $resolved
    Write-Note "$name -> $resolved"
}

$failures = @()

# --- 1. govulncheck -----------------------------------------------------------
Write-Step 'govulncheck (Go vulnerabilities)'
Push-Location (Join-Path $root 'backend')
try {
    $previous = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        $vulnOutput = & $tools['govulncheck'] ./... 2>&1
        $vulnCode = $LASTEXITCODE
    }
    finally {
        $ErrorActionPreference = $previous
    }
}
finally {
    Pop-Location
}
$vulnText = ($vulnOutput | Out-String)
Set-Content -Path (Join-Path $reports 'govulncheck.txt') -Value $vulnText
if ($vulnCode -ne 0 -or $vulnText -match 'Vulnerability #') {
    Write-Fail 'govulncheck reported vulnerabilities'
    $failures += 'govulncheck'
}
else {
    Write-Ok 'govulncheck: no vulnerabilities'
}

# --- 2. gosec -----------------------------------------------------------------
Write-Step 'gosec (Go static analysis)'
$gosecReport = Join-Path $reports 'gosec.json'
Push-Location (Join-Path $root 'backend')
try {
    $previous = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        & $tools['gosec'] -fmt=json -out $gosecReport ./... 2>&1 | Out-Null
        $gosecCode = $LASTEXITCODE
    }
    finally {
        $ErrorActionPreference = $previous
    }
}
finally {
    Pop-Location
}
if (-not (Test-Path $gosecReport)) {
    Write-Fail 'gosec produced no report; refusing to treat that as a pass'
    $failures += 'gosec'
}
else {
    $report = Get-Content -Path $gosecReport -Raw | ConvertFrom-Json -AsHashtable
    $issues = @($report['Issues'])
    $loadErrors = @($report['Golang errors'].Keys)
    if ($issues.Count -gt 0) {
        Write-Fail "gosec: $($issues.Count) finding(s)"
        foreach ($issue in $issues) {
            Write-Fail ("  [{0}] {1} {2}:{3}" -f $issue['severity'], $issue['rule_id'], $issue['file'], $issue['line'])
        }
        $failures += 'gosec'
    }
    elseif ($loadErrors.Count -gt 0) {
        Write-Fail "gosec could not load $($loadErrors.Count) package(s); the scan was incomplete"
        $failures += 'gosec'
    }
    elseif ($gosecCode -ne 0) {
        Write-Fail "gosec exited with $gosecCode but reported no parsed findings"
        $failures += 'gosec'
    }
    else {
        Write-Ok 'gosec: no findings'
    }
}

# --- 3. npm audit -------------------------------------------------------------
Write-Step 'npm audit (dependency vulnerabilities)'
$auditReport = Join-Path $reports 'npm-audit.json'
Push-Location (Join-Path $root 'operator')
try {
    $previous = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        & $tools['npm'] audit --json 2>$null | Set-Content -Path $auditReport
        & $tools['npm'] audit --audit-level=high --omit=dev 2>&1 | Out-Null
        $auditCode = $LASTEXITCODE
    }
    finally {
        $ErrorActionPreference = $previous
    }
}
finally {
    Pop-Location
}
$audit = Get-Content -Path $auditReport -Raw | ConvertFrom-Json -AsHashtable
$counts = $audit['metadata']['vulnerabilities']
Write-Note ("severity counts: critical={0} high={1} moderate={2} low={3}" -f $counts['critical'], $counts['high'], $counts['moderate'], $counts['low'])
if ($auditCode -ne 0 -or $counts['critical'] -gt 0 -or $counts['high'] -gt 0) {
    Write-Fail 'npm audit found high or critical vulnerabilities in runtime dependencies'
    $failures += 'npm-audit'
}
else {
    Write-Ok "npm audit: no high/critical runtime findings ($($counts['moderate']) moderate, dev-only)"
}

# --- 4. gitleaks --------------------------------------------------------------
Write-Step 'gitleaks (secret scan)'
$leaksReport = Join-Path $reports 'gitleaks.json'
& $tools['gitleaks'] detect --source $root --no-git --redact --config (Join-Path $root '.gitleaks.toml') `
    --report-format json --report-path $leaksReport 2>&1 | Select-Object -Last 2 | Out-Host
$leaksCode = $LASTEXITCODE
if ($leaksCode -ne 0) {
    Write-Fail 'gitleaks found potential secrets'
    $failures += 'gitleaks'
}
else {
    Write-Ok 'gitleaks: no leaks'
}

# --- 5. actionlint ------------------------------------------------------------
Write-Step 'actionlint (GitHub Actions)'
$workflows = Get-ChildItem -Path (Join-Path $root '.github/workflows') -Filter *.yml
& $tools['actionlint'] @($workflows.FullName) 2>&1 | Out-Host
$lintCode = $LASTEXITCODE
if ($lintCode -ne 0) {
    Write-Fail 'actionlint reported workflow problems'
    $failures += 'actionlint'
}
else {
    Write-Ok 'actionlint: workflows are valid'
}

# --- 6. SBOM and container pinning -------------------------------------------
Write-Step 'sbom (inventory + container pinning)'
Push-Location (Join-Path $root 'tooling')
try {
    & go run ./sbom -root $root -check-images 2>&1 | Out-Host
    $pinCode = $LASTEXITCODE
    & go run ./sbom -root $root -out (Join-Path $reports 'sbom.cdx.json') 2>&1 | Out-Host
    $sbomCode = $LASTEXITCODE
}
finally {
    Pop-Location
}
if ($pinCode -ne 0 -or $sbomCode -ne 0) {
    Write-Fail 'container pinning or SBOM generation failed'
    $failures += 'sbom'
}
else {
    Write-Ok 'sbom: inventory written and every container image is digest-pinned'
}

if ($failures.Count -gt 0) {
    Write-Fail ("security: failed gate(s): " + ($failures -join ', '))
    exit 1
}

Write-Ok "security: all gates passed (reports in $reports)"
exit 0
