# Independently validates a downloaded Unity evidence bundle.
#
# The acceptance job must not trust a green job label: it re-parses the NUnit XML,
# re-checks the executed counts, re-reads the build summary and re-verifies that
# every artifact belongs to the approved revision and repository.
#
# Exit codes: 0 accepted, 1 rejected (reasons printed and, optionally, written to
# -OutFile as JSON).
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$ArtifactsRoot,
    [Parameter(Mandatory = $true)][string]$ExpectedRevision,
    [string]$ExpectedRepository = 'hung98dev/ttgqldl',
    [string]$ExpectedRef = 'refs/heads/main',
    [string]$ExpectedUnity = '6000.6.7f1',
    [int]$MinEditMode = 56,
    [int]$MinPlayMode = 7,
    [string]$OutFile = ''
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$scriptDir = $PSScriptRoot
. (Join-Path $scriptDir 'unity-results.ps1')
$sanitizer = Join-Path $scriptDir 'sanitize-unity-log.ps1'

$reasons = New-Object System.Collections.Generic.List[string]

function Add-Reason {
    param([string]$Message)
    $script:reasons.Add($Message)
}

$report = [ordered]@{
    revision   = $ExpectedRevision
    repository = $ExpectedRepository
    ref        = $ExpectedRef
    unity      = $ExpectedUnity
    editmode   = $null
    playmode   = $null
    build      = $null
    acceptedAt = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
    ok         = $false
    reasons    = @()
}

if ($ExpectedRevision -notmatch '^[0-9a-f]{40}$') {
    Add-Reason "expected revision '$ExpectedRevision' is not a full 40-character SHA"
}

# --- preflight decision -------------------------------------------------------
$gatePath = Join-Path $ArtifactsRoot 'unity-preflight/gate.json'
if (-not (Test-Path $gatePath)) {
    Add-Reason 'preflight gate.json is missing'
}
else {
    try {
        $gate = Get-Content $gatePath -Raw | ConvertFrom-Json
        if ($gate.ok -ne $true) { Add-Reason 'preflight gate did not accept the revision' }
        if ($gate.sha -ne $ExpectedRevision) { Add-Reason "preflight accepted $($gate.sha), expected $ExpectedRevision" }
    }
    catch {
        Add-Reason "preflight gate.json is not valid JSON: $($_.Exception.Message)"
    }
}

# --- per-platform evidence ----------------------------------------------------
foreach ($platform in @(
        @{ Name = 'editmode'; Dir = 'unity-editmode-results'; Xml = 'editmode.xml'; Min = $MinEditMode },
        @{ Name = 'playmode'; Dir = 'unity-playmode-results'; Xml = 'playmode.xml'; Min = $MinPlayMode }
    )) {
    $dir = Join-Path $ArtifactsRoot $platform.Dir
    $xmlPath = Join-Path $dir $platform.Xml
    $manifestPath = Join-Path $dir 'manifest.json'

    if (-not (Test-Path $xmlPath)) {
        Add-Reason "$($platform.Name): NUnit XML is missing"
        continue
    }
    if (-not (Test-Path $manifestPath)) {
        Add-Reason "$($platform.Name): manifest is missing (provenance unknown)"
    }
    else {
        try {
            $manifest = Get-Content $manifestPath -Raw | ConvertFrom-Json
            if ($manifest.revision -ne $ExpectedRevision) { Add-Reason "$($platform.Name): manifest revision $($manifest.revision) != $ExpectedRevision" }
            if ($manifest.repository -ne $ExpectedRepository) { Add-Reason "$($platform.Name): manifest repository $($manifest.repository)" }
            if ($manifest.ref -ne $ExpectedRef) { Add-Reason "$($platform.Name): manifest ref $($manifest.ref)" }
        }
        catch {
            Add-Reason "$($platform.Name): manifest is not valid JSON"
        }
    }

    & pwsh -NoProfile -File $sanitizer -InputPath $xmlPath -ScanOnly | Out-Null
    if ($LASTEXITCODE -ne 0) { Add-Reason "$($platform.Name): sensitive material found in NUnit XML" }

    try {
        $summary = Get-UnityTestSummary -XmlPath $xmlPath
    }
    catch {
        Add-Reason "$($platform.Name): NUnit XML is corrupt ($($_.Exception.Message))"
        continue
    }
    if ($summary.Problems.Count -gt 0) { foreach ($problem in $summary.Problems) { Add-Reason "$($platform.Name): $problem" } }
    if ($summary.Failed -gt 0) { Add-Reason "$($platform.Name): $($summary.Failed) failing test(s)" }
    if ($summary.Executed -lt $platform.Min) { Add-Reason "$($platform.Name): executed $($summary.Executed), baseline $($platform.Min)" }
    $report[$platform.Name] = [ordered]@{
        total = $summary.Total; passed = $summary.Passed; failed = $summary.Failed
        skipped = $summary.Skipped; executed = $summary.Executed
    }
}

# --- build evidence -----------------------------------------------------------
$buildDir = Join-Path $ArtifactsRoot 'unity-build-summary'
$buildPath = Join-Path $buildDir 'build-summary.json'
if (-not (Test-Path $buildPath)) {
    Add-Reason 'build summary is missing'
}
else {
    try {
        $build = Get-Content $buildPath -Raw | ConvertFrom-Json
        if ($build.revision -ne $ExpectedRevision) { Add-Reason "build summary revision $($build.revision) != $ExpectedRevision" }
        if ($build.unity -ne $ExpectedUnity) { Add-Reason "build summary unity version $($build.unity)" }
        if ($build.target -ne 'StandaloneWindows64') { Add-Reason "build summary target $($build.target)" }
        if ([int]$build.playerBytes -le 0) { Add-Reason 'build summary reports an empty player' }
        $report.build = $build
    }
    catch {
        Add-Reason "build summary is not valid JSON: $($_.Exception.Message)"
    }
}
$buildManifest = Join-Path $buildDir 'manifest.json'
if (-not (Test-Path $buildManifest)) {
    Add-Reason 'build manifest is missing (provenance unknown)'
}
else {
    try {
        $manifest = Get-Content $buildManifest -Raw | ConvertFrom-Json
        if ($manifest.revision -ne $ExpectedRevision) { Add-Reason "build manifest revision $($manifest.revision) != $ExpectedRevision" }
        if ($manifest.repository -ne $ExpectedRepository) { Add-Reason "build manifest repository $($manifest.repository)" }
    }
    catch {
        Add-Reason 'build manifest is not valid JSON'
    }
}

$report.reasons = @($reasons)
$report.ok = ($reasons.Count -eq 0)

$payload = $report | ConvertTo-Json -Depth 6
if ($OutFile -ne '') {
    $directory = Split-Path -Parent $OutFile
    if ($directory -and -not (Test-Path $directory)) { New-Item -ItemType Directory -Force -Path $directory | Out-Null }
    Set-Content -LiteralPath $OutFile -Value $payload -Encoding utf8
}
Write-Host $payload

if (-not $report.ok) {
    Write-Host ("acceptance: rejected with {0} reason(s)" -f $reasons.Count) -ForegroundColor Red
    exit 1
}
Write-Host 'acceptance: evidence bundle accepted'
exit 0
