#Requires -Version 7.0
<#
.SYNOPSIS
    Starts the TTGQLDL development dependencies (PostgreSQL 18.6).

.DESCRIPTION
    Creates .env from .env.example when it is missing, then runs
        docker compose --env-file .env -f compose.dev.yaml up -d --wait
    and waits for the container health check. It never deletes data and never
    modifies an existing .env.

.EXAMPLE
    ./dev-up.ps1
#>
[CmdletBinding()]
param()

. (Join-Path $PSScriptRoot '_common.ps1')

$root = Get-RepoRoot
$envPath = Join-Path $root '.env'
$templatePath = Join-Path $root '.env.example'
$composePath = Join-Path $root 'compose.dev.yaml'

if ($null -eq (Get-CommandPath 'docker')) {
    Write-Fail 'docker is not installed or not on PATH'
    exit 1
}
if (-not (Test-Path $composePath)) {
    Write-Fail "compose file not found: $composePath"
    exit 1
}

if (-not (Test-Path $envPath)) {
    if (-not (Test-Path $templatePath)) {
        Write-Fail ".env is missing and .env.example is not present to copy from"
        exit 1
    }
    Copy-Item -Path $templatePath -Destination $envPath
    Write-WarnLine 'created .env from .env.example (development placeholder secrets, loopback only)'
    Write-Note 'edit .env if you need different ports; never commit it'
}

Write-Step 'starting development dependencies'
Push-Location $root
try {
    & docker compose --env-file $envPath -f $composePath up -d --wait
    $code = $LASTEXITCODE
    if ($code -eq 0) {
        & docker compose --env-file $envPath -f $composePath ps
    }
}
finally {
    Pop-Location
}

if ($code -ne 0) {
    Write-Fail "docker compose up failed with exit code $code"
    exit $code
}

Write-Ok 'development dependencies are up (data volume preserved across restarts)'
Write-Note 'connection: postgres://ttg_app@127.0.0.1:5432/ttgqldl (least-privileged app role)'
exit 0
