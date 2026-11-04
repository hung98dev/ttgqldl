# Shared safe-console helper for the Unity wrappers.
#
# Unity output can contain licence channels, OS account names and machine paths.
# The wrappers capture native output to a protected temporary file and print only
# a sanitized extract; this module implements that extract.
# Prints a sanitized extract of captured editor output. When sanitization cannot
# prove the text is clean, nothing is printed except a warning.
function Publish-SafeConsoleOutput {
    param([string]$RawPath, [string]$Label)

    if (-not (Test-Path -LiteralPath $RawPath)) { return }

    $staging = Join-Path ([System.IO.Path]::GetTempPath()) ("ttg-unity-safe-{0}.log" -f ([guid]::NewGuid().ToString('N').Substring(0, 8)))
    & pwsh -NoProfile -File (Join-Path $PSScriptRoot 'sanitize-unity-log.ps1') -InputPath $RawPath -OutputPath $staging | Out-Null
    $sanitizeExit = $LASTEXITCODE
    if ($sanitizeExit -ne 0 -or -not (Test-Path -LiteralPath $staging)) {
        Write-WarnLine "$Label editor output withheld: sanitizer refused it (exit $sanitizeExit)"
        Remove-Item -LiteralPath $staging -Force -ErrorAction SilentlyContinue
        return
    }

    $lines = Get-Content -LiteralPath $staging
    $interesting = @($lines | Where-Object { $_ -match '(?i)unity editor version|licensing|error CS|compilation failed|test run|passed|failed|exception|player build|build finished' })
    if ($env:GITHUB_ACTIONS -eq 'true') {
        # CI: a short, sanitized summary only.
        $interesting | Select-Object -First 20 | ForEach-Object { Write-Note "  $_" }
    }
    else {
        # Local development: full sanitized output is useful and safe enough.
        $lines | Select-Object -Last 200 | ForEach-Object { Write-Note "  $_" }
    }
    Remove-Item -LiteralPath $staging -Force -ErrorAction SilentlyContinue
}
