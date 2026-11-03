# Sanitizes a Unity editor log before it may be uploaded as a CI artifact.
#
# CI logs are evidence, not a place to leak credentials or machine identity. The
# script rewrites a copy of the log, never the original, and then re-scans its own
# output for the identities it discovered (local usernames, licence-client channel
# names, machine ids). If anything survives, the exit code is non-zero and the
# workflow must not upload the file.
#
# Exit codes:
#   0  sanitized and clean
#   2  input unreadable
#   3  output still contains a discovered identity or high-risk pattern
#
# Usage:
#   pwsh ./tooling/sanitize-unity-log.ps1 -InputPath log -OutputPath out
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$InputPath,
    [Parameter(Mandatory = $true)][string]$OutputPath,
    [string]$Replacement = '<redacted>'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

if (-not (Test-Path -LiteralPath $InputPath)) {
    Write-Host "sanitize: input not found: $InputPath" -ForegroundColor Red
    exit 2
}

$text = Get-Content -LiteralPath $InputPath -Raw -ErrorAction Stop
if ($null -eq $text) { $text = '' }

# Ordered: broad secrets first, then identifiers, then machine paths.
$rules = @(
    @{ Name = 'private-key'; Pattern = '(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----' }
    @{ Name = 'github-token'; Pattern = '\b(?:gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,})\b' }
    @{ Name = 'authorization-header'; Pattern = '(?i)(authorization\s*:\s*(?:bearer|basic)\s+)\S+' }
    @{ Name = 'bearer-token'; Pattern = '(?i)\b(bearer\s+)[A-Za-z0-9._\-]{16,}' }
    @{ Name = 'unity-license'; Pattern = '(?i)\b(?:unity[_-]?serial|serial[_-]?number|license[_-]?key)\s*[:=]\s*\S+' }
    @{ Name = 'secret-env'; Pattern = '\b([A-Z0-9_]*(?:TOKEN|SECRET|PASSWORD|PASSWD|LICENSE|SERIAL|API_KEY)[A-Z0-9_]*)\s*=\s*\S+' }
    @{ Name = 'machine-id'; Pattern = '(?i)\b(?:machine[_-]?id|hostname)\s*[:=]\s*\S+' }
    # Windows paths with either separator; no trailing separator is required, so
    # "C:\Users\alice" and "C:/Users/alice/AppData" are both covered.
    @{ Name = 'windows-user-path'; Pattern = '(?i)([A-Z]:[\\/]Users[\\/])[^\\/\s"'']+' }
    @{ Name = 'unix-home-path'; Pattern = '(?i)(/(?:home|Users)/)[^/\s"'']+' }
    # Unity licensing IPC channel names embed the OS account name.
    @{ Name = 'license-client-channel'; Pattern = '(?i)(LicenseClient-)(?:notifications-)?[A-Za-z0-9._\-]+' }
    @{ Name = 'email'; Pattern = '[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}' }
    @{ Name = 'private-ip'; Pattern = '\b(?:10\.\d{1,3}\.\d{1,3}\.\d{1,3}|192\.168\.\d{1,3}\.\d{1,3}|172\.(?:1[6-9]|2\d|3[01])\.\d{1,3}\.\d{1,3})\b' }
)

# Discover the identities that appear in THIS log before rewriting anything, so
# the verification step checks exactly the values that were at risk.
$identities = New-Object System.Collections.Generic.HashSet[string]
foreach ($candidate in @($env:USERNAME, $env:USER, $env:LOGNAME)) {
    if ($candidate -and $candidate.Length -ge 4) { [void]$identities.Add($candidate) }
}
foreach ($pattern in @(
        '(?i)[A-Z]:[\\/]Users[\\/]([^\\/\s"'']+)',
        '(?i)/(?:home|Users)/([^/\s"'']+)',
        '(?i)LicenseClient-(?:notifications-)?([A-Za-z0-9._\-]+)'
    )) {
    foreach ($match in [regex]::Matches($text, $pattern)) {
        $identity = $match.Groups[1].Value
        if ($identity.Length -ge 4 -and $identity -notin @('Application', 'Default', 'Public')) {
            [void]$identities.Add($identity)
        }
    }
}

$counts = [ordered]@{}
$sanitized = $text
foreach ($rule in $rules) {
    $matches = [regex]::Matches($sanitized, $rule.Pattern)
    if ($matches.Count -gt 0) {
        $counts[$rule.Name] = $matches.Count
        $sanitized = [regex]::Replace($sanitized, $rule.Pattern, $Replacement)
    }
}

# Second pass: every discovered identity is removed wherever it still appears.
$identityHits = 0
foreach ($identity in $identities) {
    $escaped = [regex]::Escape($identity)
    $pattern = "(?i)(?<![A-Za-z0-9._\-])${escaped}(?![A-Za-z0-9._\-])"
    $hits = [regex]::Matches($sanitized, $pattern).Count
    if ($hits -gt 0) {
        $identityHits += $hits
        $sanitized = [regex]::Replace($sanitized, $pattern, $Replacement)
    }
}
if ($identityHits -gt 0) { $counts['identity'] = $identityHits }

$directory = Split-Path -Parent $OutputPath
if ($directory -and -not (Test-Path $directory)) {
    New-Item -ItemType Directory -Force -Path $directory | Out-Null
}
Set-Content -LiteralPath $OutputPath -Value $sanitized -NoNewline -Encoding utf8

# Verify our own output. Any surviving identity or high-risk pattern is a failure.
$residual = @()
foreach ($identity in $identities) {
    # Substring, not word-boundary: an identity embedded in a longer token
    # ("alice.smith") is still a disclosure, so it must fail closed.
    if ($sanitized.IndexOf($identity, [System.StringComparison]::OrdinalIgnoreCase) -ge 0) {
        $residual += "identity"
    }
}
foreach ($rule in $rules) {
    if ($rule.Name -eq 'email') { continue } # may legitimately appear in package metadata
    if ([regex]::IsMatch($sanitized, $rule.Pattern)) { $residual += $rule.Name }
}

$summary = ($counts.GetEnumerator() | ForEach-Object { "$($_.Key)=$($_.Value)" }) -join ', '
if ($summary -eq '') { $summary = 'none' }
Write-Host "sanitize: redactions [$summary]"
Write-Host ("sanitize: identities discovered={0}, wrote {1} ({2} bytes)" -f $identities.Count, $OutputPath, (Get-Item -LiteralPath $OutputPath).Length)

if ($residual.Count -gt 0) {
    Write-Host ("sanitize: residual high-risk patterns: " + (($residual | Select-Object -Unique) -join ', ')) -ForegroundColor Red
    exit 3
}

exit 0
