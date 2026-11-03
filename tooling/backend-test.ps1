#Requires -Version 7.0
<#
.SYNOPSIS
    Runs the backend quality gates: gofmt, go vet, unit tests and optionally race tests.

.DESCRIPTION
    Every step propagates its exit code; the script never reports success for a
    failing command.

    -Race requires cgo. On a Windows host without a C toolchain the step fails
    with the exact Go error, and the Linux CI job remains the authoritative race
    gate.

.PARAMETER Race
    Also run `go test -race ./...`.

.PARAMETER Short
    Pass -short to the test runner.

.EXAMPLE
    ./backend-test.ps1
    ./backend-test.ps1 -Race
#>
[CmdletBinding()]
param(
    [switch]$Race,
    [switch]$Short
)

. (Join-Path $PSScriptRoot '_common.ps1')

$root = Get-RepoRoot
$backend = Join-Path $root 'backend'

if ($null -eq (Get-CommandPath 'go')) {
    Write-Fail 'go is not installed or not on PATH'
    exit 1
}

Use-PinnedGoToolchain

# gofmt must be clean: formatting drift is a review failure, not a warning.
Write-Step 'gofmt check'
Push-Location $backend
try {
    $unformatted = & gofmt -l . 2>&1
    $code = $LASTEXITCODE
}
finally {
    Pop-Location
}
if ($code -ne 0) {
    Write-Fail "gofmt failed with exit code $code"
    exit $code
}
if (-not [string]::IsNullOrWhiteSpace(($unformatted | Out-String))) {
    Write-Fail "gofmt found unformatted files:`n$unformatted"
    exit 1
}
Write-Ok 'gofmt'

Invoke-GoStep -Label 'go vet' -Arguments @('vet', './...') -WorkingDirectory $backend

$testArgs = @('test')
if ($Short) { $testArgs += '-short' }
$testArgs += './...'
Invoke-GoStep -Label 'go test' -Arguments $testArgs -WorkingDirectory $backend

if ($Race) {
    $raceArgs = @('test', '-race')
    if ($Short) { $raceArgs += '-short' }
    $raceArgs += './...'
    Write-Step 'go test -race (requires cgo)'
    $code = Invoke-Go -Arguments $raceArgs -WorkingDirectory $backend
    if ($code -ne 0) {
        Write-Fail "go test -race failed with exit code $code (on Windows this usually means cgo/gcc is unavailable; the Linux CI job is the authoritative race gate)"
        exit $code
    }
    Write-Ok 'go test -race'
}

Write-Ok 'backend quality gates passed'
exit 0
