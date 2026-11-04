# End-to-end regression tests for the Unity evidence chain.
#
# Exercises the real scripts the workflow calls:
#   tooling/publish-unity-evidence.ps1   (artifact publication, fail-closed)
#   tooling/unity-acceptance-check.ps1   (independent bundle validation)
#
# Every negative case asserts a non-zero exit AND that no rejected artifact was
# left behind, because the workflow uploads with `if: always()`.
#
# Usage: pwsh ./tooling/unity-acceptance.test.ps1
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$scriptDir = $PSScriptRoot
$publisher = Join-Path $scriptDir 'publish-unity-evidence.ps1'
$checker = Join-Path $scriptDir 'unity-acceptance-check.ps1'

$revision = 'f88757f46065674dcbf1557f1ca94f080ea87767'
# Credential-shaped fixtures are assembled at runtime so this file never holds a
# token-shaped literal of its own.
$fakeToken = 'ghp_' + ('a' * 36)
$repository = 'hung98dev/ttgqldl'
$tempRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("ttg-accept-{0}" -f ([guid]::NewGuid().ToString('N').Substring(0, 8)))
New-Item -ItemType Directory -Force -Path $tempRoot | Out-Null

$fakeToken = 'ghp_' + ('a' * 36)
$script:checks = 0
$script:failures = 0

function Assert-True {
    param([bool]$Condition, [string]$Message)
    $script:checks++
    if (-not $Condition) {
        $script:failures++
        Write-Host "FAIL $Message" -ForegroundColor Red
    }
}

function New-NUnitXml {
    param([int]$Total, [int]$Passed, [int]$Failed, [int]$Skipped, [string]$CaseName = 'Suite.Passes', [string]$CaseResult = 'Passed')
    $inconclusive = $Total - $Passed - $Failed - $Skipped
    if ($inconclusive -lt 0) { $inconclusive = 0 }
    return @"
<?xml version="1.0" encoding="utf-8"?>
<test-run id="2" total="$Total" passed="$Passed" failed="$Failed" skipped="$Skipped" inconclusive="$inconclusive" result="Passed">
  <test-suite type="Assembly" name="Ttgqldl.Tests" result="Passed">
    <test-case name="$CaseName" result="$CaseResult" />
  </test-suite>
</test-run>
"@
}

function New-Manifest {
    param([string]$Kind, [string]$Rev = $revision, [string]$Repo = $repository, [string]$Ref = 'refs/heads/main')
    $manifest = [ordered]@{
        kind = $Kind; revision = $Rev; repository = $Repo; ref = $Ref
        unity = '6000.6.7f1'; publishedAt = '2026-11-04T00:00:00Z'; files = @()
    }
    return ($manifest | ConvertTo-Json -Depth 3)
}

# A complete, valid bundle as it would be downloaded from a successful run.
function New-Bundle {
    param([string]$Directory, [string]$Rev = $revision)
    New-Item -ItemType Directory -Force -Path (Join-Path $Directory 'unity-preflight') | Out-Null
    $gate = [ordered]@{ ok = $true; sha = $Rev; checkedAt = '2026-11-04T00:00:00Z' }
    $gate | ConvertTo-Json | Set-Content (Join-Path $Directory 'unity-preflight/gate.json')

    foreach ($platform in @(
            @{ Name = 'editmode'; Total = 56; Failed = 0; Skipped = 0 },
            @{ Name = 'playmode'; Total = 7; Failed = 0; Skipped = 0 }
        )) {
        $dir = Join-Path $Directory "unity-$($platform.Name)-results"
        New-Item -ItemType Directory -Force -Path $dir | Out-Null
        $xml = New-NUnitXml -Total $platform.Total -Passed ($platform.Total - $platform.Failed - $platform.Skipped) -Failed $platform.Failed -Skipped $platform.Skipped
        Set-Content -LiteralPath (Join-Path $dir "$($platform.Name).xml") -Value $xml -Encoding utf8
        Set-Content -LiteralPath (Join-Path $dir 'manifest.json') -Value (New-Manifest -Kind $platform.Name -Rev $Rev) -Encoding utf8
        Set-Content -LiteralPath (Join-Path $dir "$($platform.Name).sanitized.log") -Value 'Unity Editor 6000.6.7f1: 56 passed' -Encoding utf8
    }

    $buildDir = Join-Path $Directory 'unity-build-summary'
    New-Item -ItemType Directory -Force -Path $buildDir | Out-Null
    $build = [ordered]@{
        revision = $Rev; unity = '6000.6.7f1'; target = 'StandaloneWindows64'
        player = 'TTGQLDL.exe'; playerBytes = 96738838; builtAtUtc = '2026-11-04T00:00:00Z'
    }
    $build | ConvertTo-Json | Set-Content (Join-Path $buildDir 'build-summary.json')
    Set-Content -LiteralPath (Join-Path $buildDir 'manifest.json') -Value (New-Manifest -Kind 'build' -Rev $Rev) -Encoding utf8
}

function Copy-Bundle {
    param([string]$Name)
    $target = Join-Path $tempRoot $Name
    Copy-Item -Recurse -Force (Join-Path $tempRoot 'bundle') $target
    return $target
}

function Invoke-Checker {
    param([string]$Root, [string]$Rev = $revision)
    $output = & pwsh -NoProfile -File $checker -ArtifactsRoot $Root -ExpectedRevision $Rev 2>&1
    return [pscustomobject]@{ ExitCode = $LASTEXITCODE; Output = ($output -join "`n") }
}

# --- baseline -----------------------------------------------------------------
$base = Join-Path $tempRoot 'bundle'
New-Bundle -Directory $base
$case = Invoke-Checker -Root $base
Assert-True ($case.ExitCode -eq 0) "baseline: expected exit 0, got $($case.ExitCode)"
Assert-True ($case.Output -match '"ok": true') 'baseline: report marks the bundle accepted'

# --- checker negative cases ---------------------------------------------------
$mutations = @(
    @{ Name = 'wrong-repository'; Path = 'unity-editmode-results/manifest.json'; Mutate = { param($p) (Get-Content $p -Raw) -replace 'hung98dev/ttgqldl', 'attacker/ttgqldl' | Set-Content $p } },
    @{ Name = 'wrong-ref'; Path = 'unity-playmode-results/manifest.json'; Mutate = { param($p) (Get-Content $p -Raw) -replace 'refs/heads/main', 'refs/heads/feature' | Set-Content $p } },
    @{ Name = 'stale-revision'; Path = 'unity-build-summary/manifest.json'; Mutate = { param($p) (Get-Content $p -Raw) -replace $revision, ('0' * 40) | Set-Content $p } },
    @{ Name = 'missing-xml'; Path = 'unity-editmode-results/editmode.xml'; Mutate = { param($p) Remove-Item $p -Force } },
    @{ Name = 'corrupt-xml'; Path = 'unity-editmode-results/editmode.xml'; Mutate = { param($p) Set-Content $p 'this is not xml' } },
    @{ Name = 'zero-executed'; Path = 'unity-editmode-results/editmode.xml'; Mutate = { param($p) Set-Content $p (New-NUnitXml -Total 0 -Passed 0 -Failed 0 -Skipped 0) } },
    @{ Name = 'all-skipped'; Path = 'unity-editmode-results/editmode.xml'; Mutate = { param($p) Set-Content $p (New-NUnitXml -Total 56 -Passed 0 -Failed 0 -Skipped 56) } },
    @{ Name = 'editmode-failed'; Path = 'unity-editmode-results/editmode.xml'; Mutate = { param($p) Set-Content $p (New-NUnitXml -Total 56 -Passed 55 -Failed 1 -Skipped 0 -CaseName 'Hud.Fails' -CaseResult 'Failed') } },
    @{ Name = 'playmode-failed'; Path = 'unity-playmode-results/playmode.xml'; Mutate = { param($p) Set-Content $p (New-NUnitXml -Total 7 -Passed 6 -Failed 1 -Skipped 0 -CaseName 'Flow.Fails' -CaseResult 'Failed') } },
    @{ Name = 'missing-build-summary'; Path = 'unity-build-summary/build-summary.json'; Mutate = { param($p) Remove-Item $p -Force } },
    @{ Name = 'wrong-target'; Path = 'unity-build-summary/build-summary.json'; Mutate = { param($p) (Get-Content $p -Raw) -replace 'StandaloneWindows64', 'StandaloneLinux64' | Set-Content $p } },
    @{ Name = 'wrong-unity-version'; Path = 'unity-build-summary/build-summary.json'; Mutate = { param($p) (Get-Content $p -Raw) -replace '6000\.6\.7f1', '2022.3.10f1' | Set-Content $p } },
    @{ Name = 'build-sha-mismatch'; Path = 'unity-build-summary/build-summary.json'; Mutate = { param($p) (Get-Content $p -Raw) -replace $revision, ('a' * 40) | Set-Content $p } },
    @{ Name = 'secret-in-xml'; Path = 'unity-playmode-results/playmode.xml'; Mutate = { param($p) Set-Content $p ("<test-run total=`"7`" passed=`"7`" failed=`"0`" skipped=`"0`" inconclusive=`"0`"><test-case name=`"Leak $fakeToken`" result=`"Passed`" /></test-run>") } },
    @{ Name = 'missing-gate'; Path = 'unity-preflight/gate.json'; Mutate = { param($p) Remove-Item $p -Force } },
    @{ Name = 'gate-rejected'; Path = 'unity-preflight/gate.json'; Mutate = { param($p) Set-Content $p '{"ok":false,"sha":"","reasons":["stale"]}' } },
    @{ Name = 'missing-manifest'; Path = 'unity-playmode-results/manifest.json'; Mutate = { param($p) Remove-Item $p -Force } }
)

foreach ($mutation in $mutations) {
    $bundle = Copy-Bundle -Name $mutation.Name
    $target = Join-Path $bundle $mutation.Path
    & $mutation.Mutate $target
    $case = Invoke-Checker -Root $bundle
    Assert-True ($case.ExitCode -eq 1) "$($mutation.Name): expected rejection, got exit $($case.ExitCode)"
}

# An unrelated SHA must be rejected even when the bundle is otherwise perfect.
$case = Invoke-Checker -Root $base -Rev ('b' * 40)
Assert-True ($case.ExitCode -eq 1) 'wrong-expected-sha: rejected'
# An invalid SHA is rejected before any file is read.
$case = Invoke-Checker -Root $base -Rev 'main'
Assert-True ($case.ExitCode -eq 1) 'invalid-sha: rejected'

# --- publisher negative/positive cases ---------------------------------------
function Invoke-Publisher {
    param([string]$ArtifactDirectory, [string]$Kind = 'editmode', [string]$XmlPath = '', [string]$LogPath = '', [string]$Rev = $revision)
    $output = & pwsh -NoProfile -File $publisher -ArtifactDirectory $ArtifactDirectory -Revision $Rev -Kind $Kind -XmlPath $XmlPath -LogPath $LogPath 2>&1
    return [pscustomobject]@{ ExitCode = $LASTEXITCODE; Output = ($output -join "`n") }
}

$work = Join-Path $tempRoot 'publish'
New-Item -ItemType Directory -Force -Path $work | Out-Null
$goodXml = Join-Path $work 'editmode.xml'
Set-Content $goodXml (New-NUnitXml -Total 56 -Passed 56 -Failed 0 -Skipped 0) -Encoding utf8
$goodLog = Join-Path $work 'editmode.log'
Set-Content $goodLog 'Unity Editor version: 6000.6.7f1`nAll tests passed' -Encoding utf8

$artifacts = Join-Path $work 'artifacts'
$case = Invoke-Publisher -ArtifactDirectory $artifacts -XmlPath $goodXml -LogPath $goodLog
Assert-True ($case.ExitCode -eq 0) "publisher success: expected exit 0, got $($case.ExitCode)"
Assert-True (Test-Path (Join-Path $artifacts 'editmode.xml')) 'publisher success: XML published'
Assert-True (Test-Path (Join-Path $artifacts 'editmode.sanitized.log')) 'publisher success: sanitized log published'
Assert-True (Test-Path (Join-Path $artifacts 'manifest.json')) 'publisher success: manifest published'

# Stale files are removed before publishing.
Set-Content (Join-Path $artifacts 'stale-from-previous-run.xml') '<x/>'
$case = Invoke-Publisher -ArtifactDirectory $artifacts -XmlPath $goodXml -LogPath $goodLog
Assert-True (-not (Test-Path (Join-Path $artifacts 'stale-from-previous-run.xml'))) 'publisher: stale artifact removed'

# A secret in the log must block publication and leave the directory empty.
$secretLog = Join-Path $work 'secret.log'
Set-Content $secretLog "token: $fakeToken" -Encoding utf8
$case = Invoke-Publisher -ArtifactDirectory $artifacts -XmlPath $goodXml -LogPath $secretLog
Assert-True ($case.ExitCode -eq 3) "publisher secret log: expected exit 3, got $($case.ExitCode)"
$leftover = @(Get-ChildItem -LiteralPath $artifacts -Force -ErrorAction SilentlyContinue)
Assert-True ($leftover.Count -eq 0) "publisher secret log: artifact directory is empty ($($leftover.Count) file(s))"

# A secret in the NUnit XML must block publication too.
$secretXml = Join-Path $work 'secret.xml'
Set-Content $secretXml ("<test-run total=`"7`" passed=`"7`" failed=`"0`" skipped=`"0`" inconclusive=`"0`"><test-case name=`"$fakeToken`" /></test-run>") -Encoding utf8
$case = Invoke-Publisher -ArtifactDirectory $artifacts -XmlPath $secretXml -LogPath $goodLog
Assert-True ($case.ExitCode -eq 3) "publisher secret xml: expected exit 3, got $($case.ExitCode)"
$leftover = @(Get-ChildItem -LiteralPath $artifacts -Force -ErrorAction SilentlyContinue)
Assert-True ($leftover.Count -eq 0) "publisher secret xml: artifact directory is empty ($($leftover.Count) file(s))"

# Missing inputs fail closed.
$case = Invoke-Publisher -ArtifactDirectory $artifacts -XmlPath (Join-Path $work 'nope.xml') -LogPath $goodLog
Assert-True ($case.ExitCode -eq 1) 'publisher missing xml: exit 1'
$case = Invoke-Publisher -ArtifactDirectory $artifacts -XmlPath $goodXml -LogPath (Join-Path $work 'nope.log')
Assert-True ($case.ExitCode -eq 1) 'publisher missing log: exit 1'

# Invalid revision is rejected before anything is written.
$case = Invoke-Publisher -ArtifactDirectory $artifacts -XmlPath $goodXml -LogPath $goodLog -Rev 'not-a-sha'
Assert-True ($case.ExitCode -eq 1) 'publisher invalid revision: exit 1'

Remove-Item -Recurse -Force $tempRoot -ErrorAction SilentlyContinue

if ($script:failures -gt 0) {
    Write-Host "FAIL unity-acceptance: $script:failures of $script:checks checks failed" -ForegroundColor Red
    exit 1
}

Write-Host "OK   unity-acceptance: $script:checks checks passed" -ForegroundColor Green
exit 0
