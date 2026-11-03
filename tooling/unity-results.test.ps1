# Regression tests for tooling/unity-results.ps1.
#
# Runs without Unity: each case writes a synthetic NUnit result file and asserts
# the summary, including the zero-test and all-skipped refusals.
#
# Usage: pwsh ./tooling/unity-results.test.ps1
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

. (Join-Path $PSScriptRoot 'unity-results.ps1')

$script:failures = 0
$script:checks = 0

function Assert-True {
    param([bool]$Condition, [string]$Message)
    $script:checks++
    if (-not $Condition) {
        $script:failures++
        Write-Host "FAIL $Message" -ForegroundColor Red
    }
}

function New-ResultFile {
    param([string]$Name, [string]$Content)
    $path = Join-Path ([System.IO.Path]::GetTempPath()) ("ttg-unity-results-$Name-{0}.xml" -f [guid]::NewGuid())
    [System.IO.File]::WriteAllText($path, $Content)
    return $path
}

function Remove-ResultFile {
    param([string]$Path)
    if (Test-Path $Path) { Remove-Item -Path $Path -Force }
}

# 1. A normal green run.
$green = New-ResultFile 'green' @'
<?xml version="1.0" encoding="utf-8"?>
<test-run id="2" total="7" passed="7" failed="0" skipped="0" inconclusive="0" result="Passed">
  <test-suite type="Assembly" name="Ttgqldl.Tests.PlayMode" result="Passed">
    <test-case name="BroadcastFlowTests.BootstrapLoads" result="Passed" />
  </test-suite>
</test-run>
'@
$summary = Get-UnityTestSummary -XmlPath $green
Assert-True ($summary.Total -eq 7) 'green: total is read'
Assert-True ($summary.Executed -eq 7) 'green: executed equals total'
Assert-True ($summary.Problems.Count -eq 0) 'green: no problems'
Remove-ResultFile $green

# 2. Zero tests: a stale or empty result must be refused.
$empty = New-ResultFile 'empty' @'
<?xml version="1.0" encoding="utf-8"?>
<test-run id="2" total="0" passed="0" failed="0" skipped="0" inconclusive="0" result="Passed">
  <test-suite type="Assembly" name="Ttgqldl.Tests.PlayMode" result="Passed" />
</test-run>
'@
$summary = Get-UnityTestSummary -XmlPath $empty
Assert-True ($summary.Total -eq 0) 'empty: total is zero'
Assert-True ($summary.Problems.Count -gt 0) 'empty: reported as a problem'
Assert-True ($summary.Problems[0] -match 'no tests') 'empty: problem explains the zero-test run'
Remove-ResultFile $empty

# 3. Every test skipped: also refused.
$skipped = New-ResultFile 'skipped' @'
<?xml version="1.0" encoding="utf-8"?>
<test-run id="2" total="5" passed="0" failed="0" skipped="5" inconclusive="0" result="Skipped">
  <test-suite type="Assembly" name="Ttgqldl.Tests.EditMode" result="Skipped" />
</test-run>
'@
$summary = Get-UnityTestSummary -XmlPath $skipped
Assert-True ($summary.Executed -eq 0) 'skipped: executed is zero'
Assert-True ($summary.Problems.Count -gt 0) 'skipped: reported as a problem'
Assert-True ($summary.Problems[0] -match 'skipped') 'skipped: problem explains the skipped suite'
Remove-ResultFile $skipped

# 4. A mixed run with one real failure is a failure, and the case is named.
$failed = New-ResultFile 'failed' @'
<?xml version="1.0" encoding="utf-8"?>
<test-run id="2" total="3" passed="2" failed="1" skipped="0" inconclusive="0" result="Failed">
  <test-suite type="Assembly" name="Ttgqldl.Tests.EditMode" result="Failed">
    <test-case name="HudLayoutTests.CommandsStayInsideSafeArea" result="Failed">
      <failure><message>rect outside safe area</message></failure>
    </test-case>
  </test-suite>
</test-run>
'@
$summary = Get-UnityTestSummary -XmlPath $failed
Assert-True ($summary.Failed -eq 1) 'failed: failure count is read'
Assert-True ($summary.FailedCases.Count -eq 1) 'failed: failing case is named'
Assert-True ($summary.Problems.Count -eq 0) 'failed: problems stay empty (the failure itself is reported separately)'
Remove-ResultFile $failed

# 5. Missing file is an error, not a pass.
$threw = $false
try {
    $null = Get-UnityTestSummary -XmlPath (Join-Path ([System.IO.Path]::GetTempPath()) 'does-not-exist.xml')
}
catch {
    $threw = $true
}
Assert-True $threw 'missing result file throws'

if ($script:failures -gt 0) {
    Write-Host "FAIL unity-results: $script:failures of $script:checks checks failed" -ForegroundColor Red
    exit 1
}

Write-Host "OK   unity-results: $script:checks checks passed" -ForegroundColor Green
exit 0
