#Requires -Version 7.0
<#
.SYNOPSIS
    Rolls back PostgreSQL migrations.

.DESCRIPTION
    Without -To, rolls back exactly one migration. With -To N, rolls back to
    version N; -To 0 removes every migration (destructive: all local data is
    dropped). Data loss does not happen without an explicit -To value.

    -To 0 asks for an interactive confirmation.

.EXAMPLE
    ./migrate-down.ps1
    ./migrate-down.ps1 -To 0
#>
[CmdletBinding()]
param(
    [int]$To = -1
)

. (Join-Path $PSScriptRoot '_common.ps1')

$root = Get-RepoRoot
$envPath = Join-Path $root '.env'

if (-not (Test-Path $envPath)) {
    Write-Fail '.env is missing; run ./dev-up.ps1 first'
    exit 1
}
if ($To -eq 0) {
    Write-WarnLine 'rolling back to version 0 drops every migration table and its local data'
    if (-not (Confirm-DestructiveAction -Prompt 'Roll back all migrations and delete local data?')) {
        Write-Fail 'refusing to roll back everything without confirmation'
        exit 1
    }
}
$env:TTG_ENV_FILE = $envPath

$arguments = @('run', './cmd/migrate', '-direction', 'down')
if ($To -ge 0) {
    $arguments += @('-to', "$To")
}

Write-Step 'rolling back migrations'
$code = Invoke-Go -Arguments $arguments -WorkingDirectory (Join-Path $root 'backend')
if ($code -ne 0) {
    Write-Fail "migrate down failed with exit code $code"
    exit $code
}
Write-Ok 'rollback complete'
exit 0
