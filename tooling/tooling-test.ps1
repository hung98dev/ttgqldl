#Requires -Version 7.0
<#
.SYNOPSIS
    Runs the M00 baseline tooling gates: plan validator and golden vectors.

.DESCRIPTION
    Executes, from tooling/:
      go vet ./...
      go test ./...
      go run ./planvalidate -root ..
      go run ./refeval -check

    The validator is strict by design: a status claim that outruns its evidence
    fails the run. The reference evaluator is independent of the backend and must
    never be edited to make backend tests pass.

.EXAMPLE
    ./tooling-test.ps1
#>
[CmdletBinding()]
param()

. (Join-Path $PSScriptRoot '_common.ps1')

$root = Get-RepoRoot
$tooling = Join-Path $root 'tooling'

if ($null -eq (Get-CommandPath 'go')) {
    Write-Fail 'go is not installed or not on PATH'
    exit 1
}

Invoke-GoStep -Label 'tooling go vet' -Arguments @('vet', './...') -WorkingDirectory $tooling
Invoke-GoStep -Label 'tooling go test' -Arguments @('test', './...') -WorkingDirectory $tooling

Write-Step 'plan validator (go run ./planvalidate -root ..)'
$code = Invoke-Go -Arguments @('run', './planvalidate', '-root', $root) -WorkingDirectory $tooling
if ($code -ne 0) {
    Write-Fail "plan validator failed with exit code $code"
    exit $code
}
Write-Ok 'plan validator'

Write-Step 'golden vectors (go run ./refeval -check)'
$code = Invoke-Go -Arguments @('run', './refeval', '-check') -WorkingDirectory $tooling
if ($code -ne 0) {
    Write-Fail "reference evaluator failed with exit code $code"
    exit $code
}
Write-Ok 'golden vectors'

Write-Ok 'baseline tooling gates passed'
exit 0
