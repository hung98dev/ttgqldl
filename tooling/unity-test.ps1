# Runs the real Unity EditMode and/or PlayMode test suites and reports from the
# produced NUnit XML.
#
# Exit codes:
#   0  every requested platform passed
#   1  at least one test failed (or Unity reported a non-zero exit code)
#   2  BLOCKED: editor missing, wrong version, or no usable licence
#   3  harness error: Unity exited without producing a result file
#
# This script never reports PASS because Unity merely exited: a missing XML file
# is always a failure. Local usage:
#   pwsh ./tooling/unity-test.ps1              # both platforms
#   pwsh ./tooling/unity-test.ps1 -Platform EditMode
[CmdletBinding()]
param(
    [ValidateSet('EditMode', 'PlayMode', 'All')]
    [string]$Platform = 'All'
)

. (Join-Path $PSScriptRoot '_common.ps1')

$script:RequiredUnityVersion = '6000.6.7f1'
$root = Get-RepoRoot
$project = Join-Path $root 'unity'
$results = Join-Path $project 'TestResults'

function Get-UnityVersion {
    param([string]$EditorPath)
    # Unity prints -version straight to the console, bypassing PowerShell
    # redirection, so the executable metadata is the reliable source on Windows.
    try {
        $info = (Get-Item -LiteralPath $EditorPath).VersionInfo
        if ($info -and $info.ProductVersion) { return ($info.ProductVersion -split '_')[0] }
    }
    catch { }
    return $null
}
function Resolve-UnityEditor {
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

New-Item -ItemType Directory -Force -Path $results | Out-Null
$platforms = if ($Platform -eq 'All') { @('EditMode', 'PlayMode') } else { @($Platform) }
$failedPlatforms = @()
$blocked = $false

foreach ($current in $platforms) {
    $xml = Join-Path $results ("{0}.xml" -f $current.ToLowerInvariant())
    $log = Join-Path $results ("{0}.log" -f $current.ToLowerInvariant())
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
            $licenceProblem = $null -ne (Select-String -Path $log -Pattern 'No valid Unity Editor license|License is not active|licensing.*error' -Quiet)
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

    [xml]$doc = Get-Content -Path $xml
    $run = $doc.'test-run'
    $total = [int]$run.total
    $passed = [int]$run.passed
    $failed = [int]$run.failed
    $skipped = [int]$run.skipped

    if ($failed -gt 0) {
        Write-Fail "$current`: $failed of $total failed"
        $doc.SelectNodes("//test-case[@result='Failed']") | ForEach-Object {
            Write-Fail "  $($_.name)"
            $message = $_.failure.message.'#cdata-section'
            if ($message) { Write-Note (($message -split "`n")[0]) }
        }
        $failedPlatforms += $current
        continue
    }

    if ($code -ne 0) {
        Write-Fail "${current}: Unity exited with $code although the result file reports no failures"
        $failedPlatforms += $current
        continue
    }

    Write-Ok "$current`: total=$total passed=$passed failed=$failed skipped=$skipped"
}

if ($blocked) { exit 2 }
if ($failedPlatforms.Count -gt 0) {
    Write-Fail ("unity test: failed platform(s) " + ($failedPlatforms -join ', '))
    exit 1
}
Write-Ok "unity test: all requested platforms passed"
exit 0
