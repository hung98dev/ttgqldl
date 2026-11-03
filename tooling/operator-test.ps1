# Runs the operator console quality gates in order.
#
# Exit codes: 0 all gates passed, 1 a gate failed, 2 the toolchain is missing.
[CmdletBinding()]
param()

. (Join-Path $PSScriptRoot '_common.ps1')

$root = Get-RepoRoot
$project = Join-Path $root 'operator'

if ($null -eq (Get-CommandPath 'node')) {
    Write-Fail 'BLOCKED: node was not found on PATH; Node 24 is required for the operator console'
    exit 2
}

$nodeVersion = (& node --version).TrimStart('v')
$major = [int]($nodeVersion.Split('.')[0])
if ($major -lt 24) {
    Write-Fail "BLOCKED: node $nodeVersion is too old; the operator requires Node 24 LTS"
    exit 2
}
Write-Ok "node $nodeVersion"

function Invoke-Npm {
    param([string[]]$NpmArguments)

    Push-Location $project
    try {
        $previous = $ErrorActionPreference
        $ErrorActionPreference = 'Continue'
        try {
            & npm @NpmArguments 2>&1 | Out-Host
            return $LASTEXITCODE
        }
        finally {
            $ErrorActionPreference = $previous
        }
    }
    finally {
        Pop-Location
    }
}

Write-Step 'installing locked dependencies (npm ci)'
$code = Invoke-Npm @('ci', '--no-audit', '--no-fund')
if ($code -ne 0) {
    Write-Fail "npm ci failed with exit code $code"
    exit 1
}

$steps = @(
    @{ Name = 'typecheck'; Args = @('run', 'typecheck') },
    @{ Name = 'lint'; Args = @('run', 'lint') },
    @{ Name = 'tests'; Args = @('run', 'test', '--', '--run') },
    @{ Name = 'production build'; Args = @('run', 'build') }
)

foreach ($step in $steps) {
    Write-Step "$($step.Name)"
    $code = Invoke-Npm $step.Args
    if ($code -ne 0) {
        Write-Fail "$($step.Name) failed with exit code $code"
        exit 1
    }
    Write-Ok "$($step.Name) passed"
}

Write-Ok 'operator: all gates passed'
exit 0
