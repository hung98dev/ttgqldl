#Requires -Version 7.0
<#
.SYNOPSIS
    Regenerates backend/db/schema.sql from the migrated development database.

.DESCRIPTION
    sqlc cannot parse goose migration files (they contain down migrations and
    markers), so it reads a schema snapshot. This script runs pg_dump inside the
    running PostgreSQL container, strips session/meta commands and writes the
    snapshot with its provenance header.

    Run it after every migration change and commit the result.

.EXAMPLE
    ./migrate-up.ps1 ; ./schema-dump.ps1
#>
[CmdletBinding()]
param()

. (Join-Path $PSScriptRoot '_common.ps1')

$root = Get-RepoRoot
$envPath = Join-Path $root '.env'
$composePath = Join-Path $root 'compose.dev.yaml'
$target = Join-Path $root 'backend\db\schema.sql'

if (-not (Test-Path $envPath)) {
    Write-Fail '.env is missing; run ./dev-up.ps1 first'
    exit 1
}

Push-Location $root
try {
    $containerId = (& docker compose --env-file $envPath -f $composePath ps -q postgres 2>$null | Out-String).Trim()
}
finally {
    Pop-Location
}
if ([string]::IsNullOrWhiteSpace($containerId)) {
    Write-Fail 'the postgres container is not running; run ./dev-up.ps1'
    exit 1
}

Write-Step "dumping schema ttg from container $($containerId.Substring(0, 12))"
$raw = & docker exec $containerId pg_dump -U ttg_owner -d ttgqldl `
    --schema-only --no-owner --no-privileges --schema=ttg --exclude-table=ttg.goose_db_version
if ($LASTEXITCODE -ne 0) {
    Write-Fail "pg_dump failed with exit code $LASTEXITCODE"
    exit $LASTEXITCODE
}

$filtered = $raw | Where-Object {
    $_ -notmatch '^\\.' -and
    $_ -notmatch '^SET ' -and
    $_ -notmatch '^SELECT pg_catalog\.set_config'
}
$header = @(
    '-- GENERATED FILE - do not edit by hand.',
    '-- Snapshot of the schema produced by backend/db/migrations, used only by sqlc:',
    '-- the goose files carry down migrations and markers that sqlc cannot parse.',
    '-- Regenerate with tooling/schema-dump.ps1 (PowerShell) or schema-dump.sh after',
    '-- changing a migration; CI fails if the snapshot drifts from the migrations.',
    ''
)
# Always write LF: the repository stores SQL with eol=lf and a CRLF working copy
# would show up as a spurious diff on every run.
$text = (($header + $filtered) -join "`n") + "`n"
[System.IO.File]::WriteAllText($target, $text, [System.Text.UTF8Encoding]::new($false))
Write-Ok "wrote $target ($((Get-Item $target).Length) bytes)"
Write-Note 'next: cd backend && sqlc generate'
exit 0
