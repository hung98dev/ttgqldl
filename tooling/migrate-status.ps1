#Requires -Version 7.0
<#
.SYNOPSIS
    Shows which migrations are applied and which are pending.

.EXAMPLE
    ./migrate-status.ps1
#>
[CmdletBinding()]
param()

. (Join-Path $PSScriptRoot '_common.ps1')

$root = Get-RepoRoot
$envPath = Join-Path $root '.env'

if (-not (Test-Path $envPath)) {
    Write-Fail '.env is missing; run ./dev-up.ps1 first'
    exit 1
}
$env:TTG_ENV_FILE = $envPath

Write-Step 'migration status'
$code = Invoke-Go -Arguments @('run', './cmd/migrate', '-direction', 'status') -WorkingDirectory (Join-Path $root 'backend')
if ($code -ne 0) {
    Write-Fail "migrate status failed with exit code $code"
    exit $code
}
exit 0
