#!/usr/bin/env pwsh
# TTGQLDL — independent gameplay reference evaluator.
# Usage: ./reference-eval.ps1 [-check] [-sweep] [-write]
# Default (no flag) runs the assertions and rebuilds the vectors in memory.
# Exit codes: 0 = assertions hold, 1 = assertion or golden drift, 2 = IO error.
$ErrorActionPreference = 'Stop'
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
Push-Location $here
try {
    if ($args.Count -eq 0) {
        & go run ./refeval -check
    }
    else {
        & go run ./refeval @args
    }
    exit $LASTEXITCODE
}
finally {
    Pop-Location
}
