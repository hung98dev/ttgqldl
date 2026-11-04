# Proves that raw Unity console output cannot leak credentials into the CI log.
#
# The wrapper captures native editor output to a protected temporary file and
# prints only a sanitized extract. This test runs the real wrapper against a fake
# editor that deliberately prints a credential, and asserts the credential never
# appears in the wrapper's own output.
#
# Usage: pwsh ./tooling/unity-safe-logging.test.ps1
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$scriptDir = $PSScriptRoot
$repoRoot = Split-Path -Parent $scriptDir
$wrapper = Join-Path $scriptDir 'unity-test.ps1'
$temp = Join-Path ([System.IO.Path]::GetTempPath()) ("ttg-safe-log-{0}" -f ([guid]::NewGuid().ToString('N').Substring(0, 8)))
New-Item -ItemType Directory -Force -Path $temp | Out-Null

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
$fake = Join-Path $temp $fakeName
Push-Location $scriptDir
try {
    & go build -o $fake ./cmd/unityfake
    if ($LASTEXITCODE -ne 0) {
        Write-Host 'FAIL could not build the fake editor' -ForegroundColor Red
        exit 1
    }
}
finally {
    Pop-Location
}

$validXml = @'
<?xml version="1.0" encoding="utf-8"?>
<test-run id="2" total="7" passed="7" failed="0" skipped="0" inconclusive="0" result="Passed">
  <test-case name="Suite.Passes" result="Passed" />
</test-run>
'@

# Credential-shaped strings are assembled at runtime so this file contains no
# token-shaped literal of its own.
$fakeToken = 'ghp_' + ('z' * 36)
$serialLabel = 'Unity_' + 'Serial'
$fakeSerial = 'XX-0000-1111-2222'
$noise = "License activated for $serialLabel=$fakeSerial`ncontact: leak@example.com"
$credentialNoise = "token: $fakeToken"

function Invoke-Wrapper {
    param([string]$Xml, [int]$EditorExit, [string]$Stdout, [string]$Stderr = '')
    $env:TTG_FAKE_XML = $Xml
    $env:TTG_FAKE_EXIT = "$EditorExit"
    $env:TTG_FAKE_STDOUT = $Stdout
    $env:TTG_FAKE_STDERR = $Stderr
    $results = Join-Path $temp ("results-{0}" -f ([guid]::NewGuid().ToString('N').Substring(0, 6)))
    New-Item -ItemType Directory -Force -Path $results | Out-Null
    $output = & pwsh -NoProfile -File $wrapper -Platform EditMode -EditorPath $fake -SkipVersionCheck -ResultsDirectory $results 2>&1
    return [pscustomobject]@{ ExitCode = $LASTEXITCODE; Output = ($output -join "`n") }
}

# 1. A passing run whose raw output contains a token must not echo the token.
$case = Invoke-Wrapper -Xml $validXml -EditorExit 0 -Stdout $noise -Stderr $credentialNoise
Assert-True ($case.ExitCode -eq 0) "passing run: expected exit 0, got $($case.ExitCode)"
Assert-True ($case.Output -notmatch [regex]::Escape($fakeToken)) 'passing run: github token is not printed'
Assert-True ($case.Output -notmatch [regex]::Escape($fakeSerial)) 'passing run: licence serial is not printed'
Assert-True ($case.Output -notmatch 'leak@example.com') 'passing run: email is not printed'
Assert-True ($case.Output -match 'EditMode: total=7') 'passing run: safe summary is still printed'

# 2. A failing run must keep the failure visible without leaking the token.
$case = Invoke-Wrapper -Xml $validXml -EditorExit 1 -Stdout $noise -Stderr $credentialNoise
Assert-True ($case.ExitCode -eq 1) "failing run: expected exit 1, got $($case.ExitCode)"
Assert-True ($case.Output -notmatch [regex]::Escape($fakeToken)) 'failing run: github token is not printed'
Assert-True ($case.Output -notmatch [regex]::Escape($fakeSerial)) 'failing run: licence serial is not printed'
Assert-True ($case.Output -match 'Unity exited with code 1') 'failing run: failure reason is visible'

# 3. No raw console capture may survive in the results directory.
$leftovers = @(Get-ChildItem -Path $temp -Recurse -File -Filter 'ttg-unity-console-*' -ErrorAction SilentlyContinue)
Assert-True ($leftovers.Count -eq 0) "raw console captures were removed ($($leftovers.Count) left)"
$staging = @(Get-ChildItem -Path ([System.IO.Path]::GetTempPath()) -File -Filter 'ttg-unity-safe-*' -ErrorAction SilentlyContinue)
Assert-True ($staging.Count -eq 0) "sanitized staging files were removed ($($staging.Count) left)"

Remove-Item Env:TTG_FAKE_XML -ErrorAction SilentlyContinue
Remove-Item Env:TTG_FAKE_EXIT -ErrorAction SilentlyContinue
Remove-Item Env:TTG_FAKE_STDOUT -ErrorAction SilentlyContinue
Remove-Item Env:TTG_FAKE_STDERR -ErrorAction SilentlyContinue
Remove-Item -Recurse -Force $temp -ErrorAction SilentlyContinue

if ($script:failures -gt 0) {
    Write-Host "FAIL unity-safe-logging: $script:failures of $script:checks checks failed" -ForegroundColor Red
    exit 1
}

Write-Host "OK   unity-safe-logging: $script:checks checks passed" -ForegroundColor Green
exit 0
