param([switch]$Help)
$ErrorActionPreference = 'Stop'
if ($Help) {
    Write-Output 'Usage: tooling/restore-verify.ps1'
    Write-Output 'Runs the isolated encrypted PITR restore rehearsal. Synthetic data only.'
    exit 0
}
& (Join-Path $PSScriptRoot 'pitr-rehearsal.ps1')
exit $LASTEXITCODE
