#!/usr/bin/env pwsh
# TTGQLDL — canonical plan validator.
# Usage: ./validate-plan.ps1 [-json] [-v]
# Exit codes: 0 = clean, 1 = errors found, 2 = usage/IO error.
$ErrorActionPreference = 'Stop'
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
$root = (Resolve-Path (Join-Path $here '..')).Path
Push-Location $here
try {
    & go run ./planvalidate -root $root @args
    exit $LASTEXITCODE
}
finally {
    Pop-Location
}
