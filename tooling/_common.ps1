# Shared helpers for the PowerShell entry points in tooling/.
# Dot-source this file; it never changes the caller's location.

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# The Go toolchain the backend module requires (backend/go.mod `go` directive).
$script:RequiredGoVersion = '1.27.3'

function Get-RepoRoot {
    return (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
}

function Write-Step { param([string]$Message) Write-Host "==> $Message" -ForegroundColor Cyan }
function Write-Ok { param([string]$Message) Write-Host "OK   $Message" -ForegroundColor Green }
function Write-WarnLine { param([string]$Message) Write-Host "WARN $Message" -ForegroundColor Yellow }
function Write-Fail { param([string]$Message) Write-Host "FAIL $Message" -ForegroundColor Red }
function Write-Note { param([string]$Message) Write-Host "     $Message" -ForegroundColor DarkGray }

function Get-CommandPath {
    param([string]$Name)
    $cmd = Get-Command $Name -ErrorAction SilentlyContinue
    if ($null -eq $cmd) { return $null }
    return $cmd.Source
}

# The host may pin an older Go through `go env -w GOTOOLCHAIN=goX`. The project
# pins the required toolchain in backend/go.mod, so allow Go to select it for the
# commands this tooling runs. The override is announced, scoped to this process,
# and never edits the user's global Go configuration.
function Use-PinnedGoToolchain {
    if ($null -eq (Get-CommandPath 'go')) { return }
    $current = (& go env GOTOOLCHAIN 2>$null)
    if ($LASTEXITCODE -ne 0 -or $null -eq $current) { return }
    $current = "$current".Trim()
    if ($current -match '^go[0-9]+\.[0-9]+\.[0-9]+$' -and $current -ne "go$script:RequiredGoVersion") {
        $env:GOTOOLCHAIN = 'auto'
        Write-Note "[go] GOTOOLCHAIN '$current' overridden to 'auto' for this command so backend/go.mod can select go$script:RequiredGoVersion"
    }
}

# Invoke-Go runs `go` in a directory and returns only its exit code.
# The command output goes to the host, never to the pipeline, so callers cannot
# accidentally capture it as part of the return value.
function Invoke-Go {
    param(
        [Parameter(Mandatory = $true)][string[]]$Arguments,
        [Parameter(Mandatory = $true)][string]$WorkingDirectory
    )
    Use-PinnedGoToolchain
    Push-Location $WorkingDirectory
    try {
        & go @Arguments | Out-Host
        $code = $LASTEXITCODE
    }
    finally {
        Pop-Location
    }
    return [int]$code
}

# Invoke-Step runs one go command and fails the script on a non-zero exit code.
function Invoke-GoStep {
    param(
        [Parameter(Mandatory = $true)][string]$Label,
        [Parameter(Mandatory = $true)][string[]]$Arguments,
        [Parameter(Mandatory = $true)][string]$WorkingDirectory
    )
    Write-Step "$Label (go $($Arguments -join ' '))"
    $code = Invoke-Go -Arguments $Arguments -WorkingDirectory $WorkingDirectory
    if ($code -ne 0) {
        Write-Fail "$Label failed with exit code $code"
        exit $code
    }
    Write-Ok $Label
}

function Confirm-DestructiveAction {
    param([string]$Prompt)
    if (-not [Environment]::UserInteractive) { return $false }
    $answer = Read-Host "$Prompt [y/N]"
    return ($answer -match '^(y|yes)$')
}
