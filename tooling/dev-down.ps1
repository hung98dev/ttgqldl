#Requires -Version 7.0
<#
.SYNOPSIS
    Stops the TTGQLDL development dependencies without deleting data.

.DESCRIPTION
    Runs `docker compose down` against compose.dev.yaml. The named PostgreSQL
    volume is preserved, so the next ./dev-up.ps1 resumes the same cluster.

    With -WipeData the volume is removed as well. That is destructive and asks
    for an interactive confirmation first; non-interactive runs are refused.

.EXAMPLE
    ./dev-down.ps1
    ./dev-down.ps1 -WipeData
#>
[CmdletBinding()]
param(
    [switch]$WipeData
)

. (Join-Path $PSScriptRoot '_common.ps1')

$root = Get-RepoRoot
$envPath = Join-Path $root '.env'
$composePath = Join-Path $root 'compose.dev.yaml'

if (-not (Test-Path $composePath)) {
    Write-Fail "compose file not found: $composePath"
    exit 1
}
if (-not (Test-Path $envPath)) {
    Write-Fail '.env is missing; run ./dev-up.ps1 first (nothing to stop)'
    exit 1
}

$composeArgs = @('compose', '--env-file', $envPath, '-f', $composePath, 'down')
if ($WipeData) {
    Write-WarnLine 'this deletes the development PostgreSQL volume and all local data'
    if (-not (Confirm-DestructiveAction -Prompt 'Delete the development database volume?')) {
        Write-Fail 'refusing to delete data without confirmation'
        exit 1
    }
    $composeArgs += '--volumes'
}

Write-Step 'stopping development dependencies'
Push-Location $root
try {
    & docker @composeArgs
    $code = $LASTEXITCODE
}
finally {
    Pop-Location
}

if ($code -ne 0) {
    Write-Fail "docker compose down failed with exit code $code"
    exit $code
}
if ($WipeData) {
    Write-Ok 'development dependencies stopped and the data volume was removed'
}
else {
    Write-Ok 'development dependencies stopped; the data volume was preserved'
}
exit 0
