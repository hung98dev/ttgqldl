#Requires -Version 7.0
<#
.SYNOPSIS
    Runs the TTGQLDL backend server from source.

.DESCRIPTION
    Points TTG_ENV_FILE at the repository .env, then runs `go run ./cmd/server`
    from backend/. The server refuses to start on invalid configuration and
    reports PostgreSQL availability through /readyz instead of crashing.

.EXAMPLE
    ./backend-run.ps1
#>
[CmdletBinding()]
param()

. (Join-Path $PSScriptRoot '_common.ps1')

$root = Get-RepoRoot
$envPath = Join-Path $root '.env'
$backend = Join-Path $root 'backend'

if (-not (Test-Path $envPath)) {
    Write-Fail '.env is missing; run ./dev-up.ps1 (it creates .env from .env.example)'
    exit 1
}
if ($null -eq (Get-CommandPath 'go')) {
    Write-Fail 'go is not installed or not on PATH'
    exit 1
}

Write-Step 'starting backend (Ctrl+C to stop gracefully)'
$env:TTG_ENV_FILE = $envPath
Write-Note "TTG_ENV_FILE=$envPath"

Use-PinnedGoToolchain
Push-Location $backend
try {
    & go run ./cmd/server
    $code = $LASTEXITCODE
}
finally {
    Pop-Location
}

if ($code -ne 0) {
    Write-Fail "backend exited with code $code"
    exit $code
}
Write-Ok 'backend stopped'
exit 0
