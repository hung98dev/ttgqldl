# Publishes one job's Unity evidence into the artifact directory - or nothing.
#
# Fail-closed contract:
#   - the artifact directory is emptied first, so a stale file from an earlier
#     run can never be uploaded as current evidence;
#   - NUnit XML is scanned for sensitive material and is never rewritten (a
#     secret in test output is a failure, not something to redact);
#   - the editor log is sanitized into a temporary file OUTSIDE the artifact
#     directory and copied in only after the sanitizer exits 0;
#   - any failure removes every partial file, so an `if: always()` upload step
#     finds nothing to publish.
#
# Exit codes: 0 published, 1 invalid input, 3 sensitive material found.
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$ArtifactDirectory,
    [Parameter(Mandatory = $true)][string]$Revision,
    [Parameter(Mandatory = $true)][ValidateSet('editmode', 'playmode', 'build')][string]$Kind,
    [string]$XmlPath = '',
    [string]$LogPath = '',
    [string]$Repository = 'hung98dev/ttgqldl',
    [string]$Ref = 'refs/heads/main',
    [string]$UnityVersion = '6000.6.7f1'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$scriptDir = $PSScriptRoot
$sanitizer = Join-Path $scriptDir 'sanitize-unity-log.ps1'

function Clear-Artifacts {
    param([string]$Directory)
    if (Test-Path $Directory) {
        Get-ChildItem -LiteralPath $Directory -Force | Remove-Item -Recurse -Force -ErrorAction SilentlyContinue
    }
}

function Fail-Publish {
    param([string]$Message, [int]$Code)
    Clear-Artifacts -Directory $ArtifactDirectory
    Write-Host "publish($Kind): $Message" -ForegroundColor Red
    Write-Host "publish($Kind): artifact directory left empty; nothing may be uploaded" -ForegroundColor Red
    exit $Code
}

if ($Revision -notmatch '^[0-9a-f]{40}$') {
    Fail-Publish -Message "invalid revision '$Revision'" -Code 1
}

New-Item -ItemType Directory -Force -Path $ArtifactDirectory | Out-Null
Clear-Artifacts -Directory $ArtifactDirectory

$published = @()

# NUnit XML: scan, never rewrite.
if ($XmlPath -ne '') {
    if (-not (Test-Path -LiteralPath $XmlPath)) {
        Fail-Publish -Message "NUnit XML not found: $XmlPath" -Code 1
    }
    & pwsh -NoProfile -File $sanitizer -InputPath $XmlPath -ScanOnly | Out-Host
    if ($LASTEXITCODE -ne 0) {
        Fail-Publish -Message "sensitive material found in $XmlPath" -Code 3
    }
    $target = Join-Path $ArtifactDirectory (Split-Path -Leaf $XmlPath)
    Copy-Item -LiteralPath $XmlPath -Destination $target -Force
    $published += [pscustomobject]@{ name = (Split-Path -Leaf $target); bytes = (Get-Item $target).Length }
}

# Editor log: sanitize outside the artifact directory, publish only on success.
if ($LogPath -ne '') {
    if (-not (Test-Path -LiteralPath $LogPath)) {
        Fail-Publish -Message "editor log not found: $LogPath" -Code 1
    }
    $staging = Join-Path ([System.IO.Path]::GetTempPath()) ("ttg-publish-{0}" -f ([guid]::NewGuid().ToString('N').Substring(0, 8)))
    New-Item -ItemType Directory -Force -Path $staging | Out-Null
    $stagedLog = Join-Path $staging ("{0}.sanitized.log" -f $Kind)
    & pwsh -NoProfile -File $sanitizer -InputPath $LogPath -OutputPath $stagedLog -BlockOnSecrets | Out-Host
    if ($LASTEXITCODE -ne 0 -or -not (Test-Path $stagedLog)) {
        Remove-Item -Recurse -Force $staging -ErrorAction SilentlyContinue
        Fail-Publish -Message "sanitizer rejected the editor log (exit $LASTEXITCODE)" -Code 3
    }
    $target = Join-Path $ArtifactDirectory ("{0}.sanitized.log" -f $Kind)
    Copy-Item -LiteralPath $stagedLog -Destination $target -Force
    Remove-Item -Recurse -Force $staging -ErrorAction SilentlyContinue
    $published += [pscustomobject]@{ name = (Split-Path -Leaf $target); bytes = (Get-Item $target).Length }
}

$manifest = [ordered]@{
    kind        = $Kind
    revision    = $Revision
    repository  = $Repository
    ref         = $Ref
    unity       = $UnityVersion
    publishedAt = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
    files       = @($published)
}
$manifestPath = Join-Path $ArtifactDirectory 'manifest.json'
$manifest | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $manifestPath -Encoding utf8
Write-Host ("publish($Kind): {0} file(s) published for revision {1}" -f $published.Count, $Revision)
exit 0
