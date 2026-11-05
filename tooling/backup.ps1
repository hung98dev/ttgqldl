param([switch]$Help)
$ErrorActionPreference = 'Stop'
if ($Help) {
    Write-Output 'Usage: tooling/backup.ps1'
    Write-Output 'Runs the real encrypted physical backup + PITR rehearsal. Synthetic data only.'
    exit 0
}
& (Join-Path $PSScriptRoot 'pitr-rehearsal.ps1')
exit $LASTEXITCODE
