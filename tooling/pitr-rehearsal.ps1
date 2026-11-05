param(
    [switch]$Help
)
$ErrorActionPreference = 'Stop'
if ($Help) {
    Write-Output 'Usage: tooling/pitr-rehearsal.ps1'
    Write-Output 'Runs the isolated encrypted physical backup + WAL archive + PITR rehearsal.'
    Write-Output 'Requires docker and a disposable local environment; synthetic data only.'
    exit 0
}
Push-Location (Join-Path $PSScriptRoot '..\backend')
try {
    $env:TTG_PITR_TEST = '1'
    & go test -count=1 -timeout 10m -v ./test/pitr/...
    exit $LASTEXITCODE
} finally {
    Pop-Location
}
