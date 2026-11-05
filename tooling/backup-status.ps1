param(
    [Parameter(Mandatory = $true)][string]$Manifest,
    [switch]$Help
)
$ErrorActionPreference = 'Stop'
if ($Help) {
    Write-Output 'Usage: tooling/backup-status.ps1 -Manifest <manifest.json>'
    exit 0
}
if (-not (Test-Path -LiteralPath $Manifest)) {
    Write-Error 'manifest not found'
    exit 2
}
$m = Get-Content -LiteralPath $Manifest -Raw | ConvertFrom-Json
$ok = ($m.format -eq 'ttg-pitr-backup-v1' -and $m.verified -eq $true -and $m.restore_eligible -eq $true -and -not $m.incomplete)
[pscustomobject]@{
    backup_id = $m.backup_id
    postgres_version = $m.postgres_version
    system_identifier = $m.system_identifier
    start_wal = $m.start_wal
    end_wal = $m.end_wal
    timeline = $m.timeline
    encryption = $m.encryption
    restore_eligible = $ok
    blocker = if ($ok) { $null } else { 'manifest is incomplete, unverified or ineligible' }
} | ConvertTo-Json -Compress
if (-not $ok) { exit 1 }
