# Wrapper-level regression tests for tooling/unity-test.ps1.
#
# These tests execute the real wrapper (not just the shared summary helper) with
# a fake editor binary, and assert that deliberately invalid result files make
# the wrapper exit non-zero. They need Go to build the fake editor but no Unity
# licence.
#
# Usage: pwsh ./tooling/unity-test.wrapper.test.ps1
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$tooling = Join-Path $root 'tooling'
$wrapper = Join-Path $tooling 'unity-test.ps1'
$work = Join-Path ([System.IO.Path]::GetTempPath()) ("ttg-unity-wrapper-{0}" -f ([guid]::NewGuid().ToString('N').Substring(0, 8)))
New-Item -ItemType Directory -Force -Path $work | Out-Null

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

$fakeName = if ($IsWindows) { 'unityfake.exe' } else { 'unityfake' }
$fake = Join-Path $work $fakeName

Push-Location $tooling
try {
    & go build -o $fake ./cmd/unityfake
    if ($LASTEXITCODE -ne 0) {
        Write-Host 'FAIL could not build the fake Unity editor' -ForegroundColor Red
        exit 1
    }
}
finally {
    Pop-Location
}

function Invoke-WrapperCase {
    param(
        [string]$Name,
        [string]$XmlContent,
        [int]$EditorExit = 0,
        [string]$PreexistingXml = ''
    )

    $results = Join-Path $work ("results-$Name")
    New-Item -ItemType Directory -Force -Path $results | Out-Null
    if ($PreexistingXml -ne '') {
        Set-Content -Path (Join-Path $results 'editmode.xml') -Value $PreexistingXml -Encoding utf8
    }

    $env:TTG_FAKE_XML = $XmlContent
    $env:TTG_FAKE_EXIT = "$EditorExit"

    $output = & pwsh -NoProfile -File $wrapper -Platform EditMode -EditorPath $fake -SkipVersionCheck -ResultsDirectory $results 2>&1
    $code = $LASTEXITCODE
    return [pscustomobject]@{ ExitCode = $code; Output = ($output -join "`n") }
}

$greenXml = @'
<?xml version="1.0" encoding="utf-8"?>
<test-run id="2" total="7" passed="7" failed="0" skipped="0" inconclusive="0" result="Passed">
  <test-suite type="Assembly" name="Ttgqldl.Tests.PlayMode" result="Passed">
    <test-case name="BroadcastFlowTests.BootstrapLoads" result="Passed" />
  </test-suite>
</test-run>
'@

# 1. A real green suite passes.
$case = Invoke-WrapperCase -Name 'green' -XmlContent $greenXml
Assert-True ($case.ExitCode -eq 0) "green suite: expected exit 0, got $($case.ExitCode)"
Assert-True ($case.Output -match 'total=7') 'green suite: summary is reported'

# 2. Zero tests must fail.
$emptyXml = @'
<?xml version="1.0" encoding="utf-8"?>
<test-run id="2" total="0" passed="0" failed="0" skipped="0" inconclusive="0" result="Passed" />
'@
$case = Invoke-WrapperCase -Name 'empty' -XmlContent $emptyXml
Assert-True ($case.ExitCode -eq 1) "zero tests: expected exit 1, got $($case.ExitCode)"
Assert-True ($case.Output -match 'no tests were discovered') 'zero tests: reason is printed'

# 3. Every test skipped must fail.
$skippedXml = @'
<?xml version="1.0" encoding="utf-8"?>
<test-run id="2" total="5" passed="0" failed="0" skipped="5" inconclusive="0" result="Skipped" />
'@
$case = Invoke-WrapperCase -Name 'skipped' -XmlContent $skippedXml
Assert-True ($case.ExitCode -eq 1) "all skipped: expected exit 1, got $($case.ExitCode)"
Assert-True ($case.Output -match 'no test was actually executed') 'all skipped: reason is printed'

# 4. Every test inconclusive must fail.
$inconclusiveXml = @'
<?xml version="1.0" encoding="utf-8"?>
<test-run id="2" total="4" passed="0" failed="0" skipped="0" inconclusive="4" result="Inconclusive" />
'@
$case = Invoke-WrapperCase -Name 'inconclusive' -XmlContent $inconclusiveXml
Assert-True ($case.ExitCode -eq 1) "all inconclusive: expected exit 1, got $($case.ExitCode)"

# 5. Inconsistent counters must fail.
$inconsistentXml = @'
<?xml version="1.0" encoding="utf-8"?>
<test-run id="2" total="10" passed="3" failed="0" skipped="0" inconclusive="0" result="Passed" />
'@
$case = Invoke-WrapperCase -Name 'inconsistent' -XmlContent $inconsistentXml
Assert-True ($case.ExitCode -eq 1) "inconsistent counters: expected exit 1, got $($case.ExitCode)"
Assert-True ($case.Output -match 'inconsistent counters') 'inconsistent counters: reason is printed'

# 6. Missing counters must fail.
$missingCountersXml = @'
<?xml version="1.0" encoding="utf-8"?>
<test-run id="2" result="Passed" />
'@
$case = Invoke-WrapperCase -Name 'missing-counters' -XmlContent $missingCountersXml
Assert-True ($case.ExitCode -eq 1) "missing counters: expected exit 1, got $($case.ExitCode)"
Assert-True ($case.Output -match 'missing or non-integer') 'missing counters: reason is printed'

# 7. A failing test must fail, and the case must be named.
$failedXml = @'
<?xml version="1.0" encoding="utf-8"?>
<test-run id="2" total="3" passed="2" failed="1" skipped="0" inconclusive="0" result="Failed">
  <test-suite type="Assembly" name="Ttgqldl.Tests.EditMode" result="Failed">
    <test-case name="HudLayoutTests.CommandsStayInsideSafeArea" result="Failed" />
  </test-suite>
</test-run>
'@
$case = Invoke-WrapperCase -Name 'failed-case' -XmlContent $failedXml
Assert-True ($case.ExitCode -eq 1) "failed test: expected exit 1, got $($case.ExitCode)"
Assert-True ($case.Output -match 'HudLayoutTests.CommandsStayInsideSafeArea') 'failed test: case is named'

# 8. Corrupt XML must fail instead of throwing.
$case = Invoke-WrapperCase -Name 'corrupt' -XmlContent 'this is not xml at all'
Assert-True ($case.ExitCode -eq 1) "corrupt XML: expected exit 1, got $($case.ExitCode)"
Assert-True ($case.Output -match 'missing or corrupt') 'corrupt XML: reason is printed'

# 9. A missing result file must fail (editor wrote nothing).
$case = Invoke-WrapperCase -Name 'missing-file' -XmlContent ''
Assert-True ($case.ExitCode -eq 1) "missing result file: expected exit 1, got $($case.ExitCode)"
Assert-True ($case.Output -match 'produced no result file') 'missing result file: reason is printed'

# 10. A stale result file from a previous run must not be accepted.
$case = Invoke-WrapperCase -Name 'stale' -XmlContent '' -PreexistingXml $greenXml
Assert-True ($case.ExitCode -eq 1) "stale result file: expected exit 1, got $($case.ExitCode)"
Assert-True ($case.Output -match 'produced no result file') 'stale result file: reason is printed'

# 11. A non-zero editor exit code must fail even with a green result file.
$case = Invoke-WrapperCase -Name 'editor-exit' -XmlContent $greenXml -EditorExit 1
Assert-True ($case.ExitCode -eq 1) "editor exit code: expected exit 1, got $($case.ExitCode)"
Assert-True ($case.Output -match 'Unity exited with code 1') 'editor exit code: reason is printed'

Remove-Item Env:TTG_FAKE_XML -ErrorAction SilentlyContinue
Remove-Item Env:TTG_FAKE_EXIT -ErrorAction SilentlyContinue
Remove-Item -Recurse -Force $work -ErrorAction SilentlyContinue

if ($script:failures -gt 0) {
    Write-Host "FAIL unity-test wrapper: $script:failures of $script:checks checks failed" -ForegroundColor Red
    exit 1
}

Write-Host "OK   unity-test wrapper: $script:checks checks passed" -ForegroundColor Green
exit 0
