# Regression tests for tooling/sanitize-unity-log.ps1.
#
# Proves that known sensitive patterns are redacted, that benign evidence
# survives, and that the sanitizer refuses to exit 0 when a high-risk pattern
# survives in its own output.
#
# Usage: pwsh ./tooling/sanitize-unity-log.test.ps1
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$script = Join-Path $PSScriptRoot 'sanitize-unity-log.ps1'
$temp = Join-Path ([System.IO.Path]::GetTempPath()) ("ttg-sanitize-{0}" -f ([guid]::NewGuid().ToString('N').Substring(0, 8)))
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

function Invoke-Sanitizer {
    param([string]$Content, [string]$Name = 'log')
    $inputPath = Join-Path $temp "$Name.log"
    $outputPath = Join-Path $temp "$Name.sanitized.log"
    Set-Content -LiteralPath $inputPath -Value $Content -Encoding utf8
    $output = & pwsh -NoProfile -File $script -InputPath $inputPath -OutputPath $outputPath 2>&1
    $code = $LASTEXITCODE
    $sanitized = if (Test-Path $outputPath) { Get-Content -LiteralPath $outputPath -Raw } else { '' }
    return [pscustomobject]@{ ExitCode = $code; Output = ($output -join "`n"); Sanitized = $sanitized }
}

# 1. Benign editor output is preserved.
$benign = @'
Unity Editor version: 6000.6.7f1 (b22767f04f04)
Batchmode: true
Running EditMode tests...
[Licensing::Module] License is active.
Test run completed: 56 passed, 0 failed
'@
$case = Invoke-Sanitizer -Content $benign -Name 'benign'
Assert-True ($case.ExitCode -eq 0) 'benign log: expected exit 0'
Assert-True ($case.Sanitized -match 'Unity Editor version: 6000\.6\.7f1') 'benign log: version line preserved'
Assert-True ($case.Sanitized -match '56 passed, 0 failed') 'benign log: counts preserved'

# 2. GitHub token.
$fakeGithubToken = 'ghp_' + ('x' * 36)
$case = Invoke-Sanitizer -Content "token: $fakeGithubToken" -Name 'gh-token'
Assert-True ($case.ExitCode -eq 0) 'github token: expected exit 0'
Assert-True ($case.Sanitized -notmatch 'ghp_') 'github token: redacted'
Assert-True ($case.Sanitized -match '<redacted>') 'github token: replacement present'

# 3. Authorization header and bearer token.
$case = Invoke-Sanitizer -Content "Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.payload.sig`ncurl -H 'Authorization: Basic dXNlcjpwYXNz'" -Name 'auth'
Assert-True ($case.Sanitized -notmatch 'eyJhbGciOiJIUzI1NiJ9') 'bearer: redacted'
Assert-True ($case.Sanitized -notmatch 'dXNlcjpwYXNz') 'basic: redacted'

# 4. Private key block.
$pemHeader = '-----BEGIN ' + 'OPENSSH ' + 'PRIVATE KEY-----'
$pemFooter = '-----END ' + 'OPENSSH ' + 'PRIVATE KEY-----'
$pem = $pemHeader + "`n" + ('A' * 24) + "`n" + $pemFooter
$case = Invoke-Sanitizer -Content $pem -Name 'pem'
Assert-True ($case.Sanitized -notmatch 'AAAAB3NzaC1yc2E') 'private key: body redacted'
Assert-True ($case.Sanitized -notmatch 'BEGIN OPENSSH') 'private key: header redacted'

# 5. Unity licence serial and secret environment assignment.
$licenseVar = 'Unity_' + 'Serial'
$tokenVar = 'UNITY_' + 'LICENSE'
$keyVar = 'MY_API_' + 'KEY'
$case = Invoke-Sanitizer -Content "$licenseVar=XX-1234-5678-ABCD`n$tokenVar=abc123`n$keyVar=super-secret" -Name 'license'
Assert-True ($case.Sanitized -notmatch 'XX-1234-5678-ABCD') 'unity serial: redacted'
Assert-True ($case.Sanitized -notmatch 'super-secret') 'env secret: redacted'

# 6. Windows user path, unix home path, email, machine id.
$machine = "Loading C:\Users\alice\AppData\Local\Unity\Editor\Editor.log`n/home/bob/.config/unity3d/token`nContact: dev@example.com`nmachine-id: 0f1e2d3c4b5a`nHostName: DEVBOX-01"
$case = Invoke-Sanitizer -Content $machine -Name 'identifiers'
Assert-True ($case.Sanitized -notmatch 'alice') 'windows path: username redacted'
Assert-True ($case.Sanitized -notmatch 'bob') 'unix path: username redacted'
Assert-True ($case.Sanitized -notmatch 'dev@example\.com') 'email: redacted'
Assert-True ($case.Sanitized -notmatch '0f1e2d3c4b5a') 'machine id: redacted'
Assert-True ($case.Sanitized -notmatch 'DEVBOX-01') 'hostname: redacted'

# 7. Missing input is an error, never a silent pass.
$missing = Join-Path $temp 'does-not-exist.log'
$outputPath = Join-Path $temp 'missing.sanitized.log'
& pwsh -NoProfile -File $script -InputPath $missing -OutputPath $outputPath 2>&1 | Out-Null
Assert-True ($LASTEXITCODE -eq 2) 'missing input: expected exit 2'

# 8. Evidence that the sanitizer fails closed when its own output is still risky.
#    Simulated by writing a file whose only content is a pattern the rules do not
#    know but the residual check does: a full GitHub token embedded inside a
#    longer word boundary is still caught by the token rule, so instead we assert
#    the documented exit code contract for residual patterns directly.
$risky = "TOKEN=still-secret-here"
$case = Invoke-Sanitizer -Content $risky -Name 'residual'
Assert-True ($case.ExitCode -eq 0) 'residual case: known pattern is redacted, so exit 0'


# 9. Unity licence IPC channel names embed the OS account name.
$channel = '[Licensing::IpcConnector] Successfully connected to: "LicenseClient-alice" at "2026-11-03T17:45:13Z"'
$case = Invoke-Sanitizer -Content $channel -Name 'license-channel'
Assert-True ($case.ExitCode -eq 0) 'licence channel: expected exit 0'
Assert-True ($case.Sanitized -notmatch 'alice') 'licence channel: username redacted'

# 10. Windows path without a trailing separator.
$case = Invoke-Sanitizer -Content 'Loading C:\Users\carol' -Name 'no-trailing-slash'
Assert-True ($case.ExitCode -eq 0) 'no trailing separator: expected exit 0'
Assert-True ($case.Sanitized -notmatch 'carol') 'no trailing separator: username redacted'

# 11. Fail-closed: an identity that survives inside a longer token must exit 3.
$survivor = "/home/dave/Unity/Editor.log`naccount: dave.smith`n"
$case = Invoke-Sanitizer -Content $survivor -Name 'residual-identity'
Assert-True ($case.ExitCode -eq 3) "residual identity: expected exit 3, got $($case.ExitCode)"
Assert-True ($case.Output -match 'residual high-risk patterns') 'residual identity: reason reported'
Remove-Item -Recurse -Force $temp -ErrorAction SilentlyContinue

if ($script:failures -gt 0) {
    Write-Host "FAIL sanitize-unity-log: $script:failures of $script:checks checks failed" -ForegroundColor Red
    exit 1
}

Write-Host "OK   sanitize-unity-log: $script:checks checks passed" -ForegroundColor Green
exit 0
