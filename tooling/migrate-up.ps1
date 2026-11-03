#Requires -Version 7.0
<#
.SYNOPSIS
    Applies all pending PostgreSQL migrations.

.DESCRIPTION
    Runs backend/cmd/migrate with the migration DSN (the schema-owner role).
    The server itself connects as the least-privileged application role and
    cannot run DDL.

.EXAMPLE
    ./migrate-up.ps1
#>
[CmdletBinding()]
param()

. (Join-Path $PSScriptRoot '_common.ps1')

$root = Get-RepoRoot
$envPath = Join-Path $root '.env'

if (-not (Test-Path $envPath)) {
    Write-Fail '.env is missing; run ./dev-up.ps1 (it creates .env from .env.example)'
    exit 1
}
$env:TTG_ENV_FILE = $envPath

Write-Step 'applying migrations'
$code = Invoke-Go -Arguments @('run', './cmd/migrate', '-direction', 'up') -WorkingDirectory (Join-Path $root 'backend')
if ($code -ne 0) {
    Write-Fail "migrate up failed with exit code $code"
    exit $code
}
Write-Ok 'migrations are up to date'
exit 0
