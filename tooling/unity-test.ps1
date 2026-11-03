# Runs the real Unity EditMode and/or PlayMode test suites and reports from the
# produced NUnit XML.
#
# Acceptance rules (identical to tooling/unity-test.sh):
#   - the result file must exist and be valid XML with a <test-run> element
#   - total, passed, failed, skipped and inconclusive must be present, integers
#     and non-negative
#   - total must equal passed+failed+skipped+inconclusive
#   - at least one test must have actually executed (not skipped/inconclusive)
#   - zero failures, and Unity must exit 0
#   - a stale result file cannot be reused: it is deleted before the run
#
# Exit codes:
#   0  every requested platform passed
#   1  at least one test failed, or the result was empty/inconsistent/missing
#   2  BLOCKED: editor missing, wrong version, or no usable licence
#   3  harness error (reserved for unexpected script failures)
[CmdletBinding()]
param(
    [ValidateSet('EditMode', 'PlayMode', 'All')]
    [string]$Platform = 'All',

    # Regression-test seams. Never set these for a normal run.
    [string]$EditorPath = '',
    [string]$ResultsDirectory = '',
    [switch]$SkipVersionCheck
)

. (Join-Path $PSScriptRoot '_common.ps1')
. (Join-Path $PSScriptRoot 'unity-results.ps1')

$script:RequiredUnityVersion = '6000.6.7f1'
$root = Get-RepoRoot
$project = Join-Path $root 'unity'
$results = if ($ResultsDirectory -ne '') { $ResultsDirectory } else { Join-Path $project 'TestResults' }

function Get-UnityVersion {
    param([string]$EditorExecutable)
    # Unity prints -version straight to the console, bypassing PowerShell
    # redirection, so the executable metadata is the reliable source on Windows.
    try {
        $info = (Get-Item -LiteralPath $EditorExecutable).VersionInfo
        if ($info -and $info.ProductVersion) { return ($info.ProductVersion -split '_')[0] }
    }
    catch { }
    return $null
}

function Resolve-UnityEditor {
    if ($EditorPath -ne '') {
        if (Test-Path $EditorPath) { return (Resolve-Path $EditorPath).Path }
        return $null
    }
    if ($env:TTG_UNITY_EDITOR) {
        if (Test-Path $env:TTG_UNITY_EDITOR) { return (Resolve-Path $env:TTG_UNITY_EDITOR).Path }
        Write-WarnLine "TTG_UNITY_EDITOR points to a missing path: $env:TTG_UNITY_EDITOR"
    }
    $candidates = @(
        (Join-Path $env:ProgramFiles "Unity\Hub\Editor\$script:RequiredUnityVersion\Editor\Unity.exe"),
        (Join-Path ${env:ProgramFiles(x86)} "Unity\Hub\Editor\$script:RequiredUnityVersion\Editor\Unity.exe"),
        (Join-Path $env:LOCALAPPDATA "Unity\Hub\Editor\$script:RequiredUnityVersion\Editor\Unity.exe")
    )
    foreach ($candidate in $candidates) {
        if ($candidate -and (Test-Path $candidate)) { return $candidate }
    }
    $command = Get-CommandPath 'Unity'
    if ($null -ne $command) { return $command }
    return $null
}

$editor = Resolve-UnityEditor
if ($null -eq $editor) {
    Write-Fail "BLOCKED: Unity $script:RequiredUnityVersion was not found. Set TTG_UNITY_EDITOR to the Unity executable."
    exit 2
}

Write-Step "using editor $editor"
if ($SkipVersionCheck) {
    Write-WarnLine 'version verification skipped (regression-test mode only)'
}
else {
    $version = Get-UnityVersion $editor
    if ($null -eq $version) {
        Write-Fail 'BLOCKED: the editor did not report a version'
        exit 2
    }
    $version = "$version".Trim()
    if ($version -notlike "$script:RequiredUnityVersion*") {
        Write-Fail "BLOCKED: editor version $version does not match the pinned $script:RequiredUnityVersion"
        exit 2
    }
    Write-Ok "editor version $version"
}

New-Item -ItemType Directory -Force -Path $results | Out-Null
$platforms = if ($Platform -eq 'All') { @('EditMode', 'PlayMode') } else { @($Platform) }
$failedPlatforms = @()
$blocked = $false

foreach ($current in $platforms) {
    $xml = Join-Path $results ("{0}.xml" -f $current.ToLowerInvariant())
    $log = Join-Path $results ("{0}.log" -f $current.ToLowerInvariant())

    # Removing the previous result is what makes a stale file impossible: if the
    # editor does not write a new one, the platform fails below.
    Remove-Item -Path $xml, $log -ErrorAction SilentlyContinue

    Write-Step "running $current tests"
    # Unity writes progress to stderr; merging it keeps ErrorActionPreference
    # from turning a healthy run into a terminating NativeCommandError, and
    # Out-Host keeps $LASTEXITCODE intact for the caller.
    $previous = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        & $editor -batchmode -nographics -projectPath $project -runTests `
            -testPlatform $current -testResults $xml -logFile $log 2>&1 | Out-Host
        $code = $LASTEXITCODE
    }
    finally {
        $ErrorActionPreference = $previous
    }

    if (-not (Test-Path $xml)) {
        $licenceProblem = $false
        if (Test-Path $log) {
            $licenceProblem = [bool](Select-String -Path $log -Pattern 'No valid Unity Editor license|License is not active|licensing.*error' -Quiet)
        }
        if ($licenceProblem) {
            Write-Fail "BLOCKED: $current tests could not run because no usable Unity licence was found"
            $blocked = $true
        }
        else {
            Write-Fail "${current}: Unity exited with $code and produced no result file; refusing to report a result"
            $failedPlatforms += $current
        }
        continue
    }

    try {
        $summary = Get-UnityTestSummary -XmlPath $xml
    }
    catch {
        Write-Fail "${current}: the result file is missing or corrupt ($($_.Exception.Message))"
        $failedPlatforms += $current
        continue
    }

    $problems = @($summary.Problems)
    if ($code -ne 0) {
        $problems += "Unity exited with code $code"
    }
    if ($summary.Failed -gt 0) {
        $problems += "$($summary.Failed) of $($summary.Total) tests failed"
    }

    if ($problems.Count -gt 0) {
        foreach ($problem in $problems) {
            Write-Fail "${current}: $problem"
        }
        foreach ($case in $summary.FailedCases) {
            Write-Fail "  failed case: $case"
        }
        $failedPlatforms += $current
        continue
    }

    Write-Ok "${current}: total=$($summary.Total) passed=$($summary.Passed) failed=$($summary.Failed) skipped=$($summary.Skipped) executed=$($summary.Executed)"
}

if ($blocked) { exit 2 }
if ($failedPlatforms.Count -gt 0) {
    Write-Fail ("unity test: failed platform(s) " + ($failedPlatforms -join ', '))
    exit 1
}
Write-Ok "unity test: all requested platforms passed"
exit 0
