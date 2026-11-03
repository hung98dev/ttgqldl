# Starts the operator console against the local Go backend.
#
# The Vite dev server proxies /healthz, /readyz and /admin/v1 to
# TTG_OPERATOR_BACKEND_URL (default http://127.0.0.1:8080); no database is ever
# exposed to the browser.
[CmdletBinding()]
param(
    [switch]$Demo,
    [string]$BackendUrl = ''
)

. (Join-Path $PSScriptRoot '_common.ps1')

$root = Get-RepoRoot
$project = Join-Path $root 'operator'

if ($null -eq (Get-CommandPath 'node')) {
    Write-Fail 'node was not found on PATH; Node 24 is required for the operator console'
    exit 2
}

$nodeVersion = (& node --version).TrimStart('v')
$major = [int]($nodeVersion.Split('.')[0])
if ($major -lt 24) {
    Write-Fail "node $nodeVersion is too old; the operator requires Node 24 LTS"
    exit 2
}
Write-Ok "node $nodeVersion"

if (-not (Test-Path (Join-Path $project 'node_modules'))) {
    Write-Step 'installing operator dependencies (npm ci)'
    Push-Location $project
    try {
        & npm ci --no-audit --no-fund
        if ($LASTEXITCODE -ne 0) {
            Write-Fail "npm ci failed with exit code $LASTEXITCODE"
            exit 1
        }
    }
    finally {
        Pop-Location
    }
}

if ($BackendUrl -ne '') {
    $env:TTG_OPERATOR_BACKEND_URL = $BackendUrl
    Write-Note "backend proxy -> $BackendUrl"
}
else {
    Write-Note 'backend proxy -> http://127.0.0.1:8080 (set TTG_OPERATOR_BACKEND_URL to override)'
}

if ($Demo) {
    $env:VITE_OPERATOR_DEMO = 'true'
    Write-WarnLine 'DEMO mode: the console will show fixed fixtures and a MOCK banner'
}
else {
    $env:VITE_OPERATOR_DEMO = 'false'
    Write-Note 'real-service mode: no fabricated data'
}

Write-Step 'starting Vite on http://127.0.0.1:5173'
Push-Location $project
try {
    $previous = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        & npm run dev 2>&1 | Out-Host
        $code = $LASTEXITCODE
    }
    finally {
        $ErrorActionPreference = $previous
    }
}
finally {
    Pop-Location
}

exit $code
