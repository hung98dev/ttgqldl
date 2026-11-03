#Requires -Version 7.0
<#
.SYNOPSIS
    Validates the local TTGQLDL development environment against the pinned baseline.

.DESCRIPTION
    Prints one row per dependency and distinguishes:
      OK          correct version and usable
      FAIL        wrong version, missing required tool, or unusable install
      MISSING     optional for the current milestone (owner milestone named)
      OPTIONAL    present but not required yet

    Required for M01 Phase A: Go 1.27.3 (backend module), Docker with the Compose
    plugin, the pinned PostgreSQL 18.6 image digest, and >=5 GiB free disk.
    Unity (M06), Node (M01-W0105) and Bash (script parity) are reported but do
    not fail unless -Strict is passed.

.PARAMETER Strict
    Fail when an optional dependency for a later milestone is missing.

.EXAMPLE
    ./doctor.ps1
    ./doctor.ps1 -Strict
#>
[CmdletBinding()]
param(
    [switch]$Strict
)

. (Join-Path $PSScriptRoot '_common.ps1')

$root = Get-RepoRoot
$requiredGo = $script:RequiredGoVersion
$pinnedDigest = 'sha256:3a4264520e121176619f54ff867d12774bc81e95e7b84db58d63bccdb6144b81'
$requiredFreeBytes = 5GB

$rows = [System.Collections.Generic.List[object]]::new()
$failures = 0
$optionalMissing = 0

function Add-Row {
    param(
        [string]$Check,
        [string]$Status,
        [string]$Detail,
        [string]$Owner = 'M01',
        [bool]$Optional = $false
    )
    $rows.Add([pscustomobject]@{ Check = $Check; Status = $Status; Detail = $Detail; Owner = $Owner; Optional = $Optional })
    $bad = $Status -ne 'OK'
    if ($bad -and (-not $Optional -or $Strict)) { $script:failures++ }
    if ($bad -and $Optional) { $script:optionalMissing++ }
}

Write-Host ''
Write-Host 'TTGQLDL environment doctor' -ForegroundColor White
Write-Host "repository: $root"
Write-Host "baseline:   Go $requiredGo, PostgreSQL 18.6, Unity 6000.6.7f1"
Write-Host ''

# --- Go toolchain ------------------------------------------------------------
if ($null -eq (Get-CommandPath 'go')) {
    Add-Row 'go toolchain' 'FAIL' 'go not found on PATH; install Go 1.27.3 from https://go.dev/dl/' 'M01'
}
else {
    $ambient = (& go env GOTOOLCHAIN 2>$null | Out-String).Trim()
    Use-PinnedGoToolchain
    $backend = Join-Path $root 'backend'
    Push-Location $backend
    try {
        $out = (& go version 2>&1 | Out-String).Trim()
        $code = $LASTEXITCODE
    }
    finally {
        Pop-Location
    }
    if ($code -ne 0) {
        Add-Row 'go toolchain' 'FAIL' "cannot resolve the module toolchain: $out" 'M01'
    }
    elseif ($out -match 'go version go([0-9]+\.[0-9]+\.[0-9]+)') {
        $found = $Matches[1]
        if ($found -eq $requiredGo) {
            Add-Row 'go toolchain' 'OK' "go$found (effective in backend/)"
        }
        else {
            Add-Row 'go toolchain' 'FAIL' "found go$found, required go$requiredGo. Install it, or run: go env -w GOTOOLCHAIN=auto" 'M01'
        }
    }
    else {
        Add-Row 'go toolchain' 'FAIL' "unrecognised go version output: $out" 'M01'
    }
    if ($ambient -match '^go[0-9]+\.[0-9]+\.[0-9]+$' -and $ambient -ne "go$requiredGo") {
        Add-Row 'GOTOOLCHAIN' 'OK' "host pin '$ambient' (tooling passes auto automatically); permanent fix: go env -w GOTOOLCHAIN=auto" 'M01'
    }
    else {
        Add-Row 'GOTOOLCHAIN' 'OK' "$ambient"
    }
}

# --- Docker ------------------------------------------------------------------
$dockerPath = Get-CommandPath 'docker'
if ($null -eq $dockerPath) {
    Add-Row 'docker' 'FAIL' 'docker not found on PATH; install Docker Desktop or Docker Engine' 'M01'
}
else {
    $server = (& docker version --format '{{.Server.Version}}' 2>&1 | Out-String).Trim()
    if ($LASTEXITCODE -ne 0) {
        $status = if ($server -match 'permission denied|access is denied') { 'BLOCKED' } else { 'FAIL' }
        Add-Row 'docker' $status "daemon not reachable: $server" 'M01'
    }
    else {
        Add-Row 'docker' 'OK' "server $server"
    }
    $compose = (& docker compose version --short 2>&1 | Out-String).Trim()
    if ($LASTEXITCODE -ne 0) {
        Add-Row 'docker compose' 'MISSING' "compose plugin missing: $compose" 'M01'
    }
    else {
        Add-Row 'docker compose' 'OK' "v$compose"
    }
}

# --- Pinned PostgreSQL image -------------------------------------------------
if ($null -ne $dockerPath) {
    $digests = (& docker image inspect 'postgres:18.6' --format '{{json .RepoDigests}}' 2>$null | Out-String).Trim()
    if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($digests)) {
        Add-Row 'postgres image' 'MISSING' 'postgres:18.6@sha256:3a42645... not present locally; run ./dev-up.ps1' 'M01'
    }
    elseif ($digests -match [regex]::Escape($pinnedDigest)) {
        Add-Row 'postgres image' 'OK' "postgres:18.6@$($pinnedDigest.Substring(0, 19))..."
    }
    else {
        Add-Row 'postgres image' 'FAIL' "local postgres:18.6 digest does not match the pin: $digests" 'M01'
    }
}

# --- Disk space --------------------------------------------------------------
$driveName = (Split-Path -Qualifier $root).TrimEnd(':')
$drive = Get-PSDrive -Name $driveName -ErrorAction SilentlyContinue
if ($null -eq $drive) {
    Add-Row 'free disk' 'FAIL' "cannot inspect drive $driveName" 'M01'
}
else {
    $freeGb = [math]::Round($drive.Free / 1GB, 1)
    if ($drive.Free -ge $requiredFreeBytes) {
        Add-Row 'free disk' 'OK' "$freeGb GiB free on ${driveName}:"
    }
    else {
        Add-Row 'free disk' 'FAIL' "$freeGb GiB free on ${driveName}:, need at least 5 GiB" 'M01'
    }
}

# --- Local .env --------------------------------------------------------------
$envPath = Join-Path $root '.env'
if (Test-Path $envPath) {
    Add-Row 'env file' 'OK' '.env present (gitignored)'
}
else {
    Add-Row 'env file' 'MISSING' '.env absent; ./dev-up.ps1 creates it from .env.example' 'M01' $true
}

# --- Optional: Node (operator tooling, W0105) --------------------------------
if ($null -eq (Get-CommandPath 'node')) {
    Add-Row 'node' 'MISSING' 'node not found; required by M01-W0105 (operator toolchain)' 'W0105' $true
}
else {
    Add-Row 'node' 'OK' (& node --version)
}

# --- Optional: Unity (client, M06) -------------------------------------------
# A Unity license and graphics capability can only be proven by launching the
# editor (M06). This check distinguishes "not installed" from "wrong version".
$unityCandidates = @()
$unityCmd = Get-CommandPath 'unity'
if ($null -ne $unityCmd) { $unityCandidates += $unityCmd }
foreach ($base in @((Join-Path $env:ProgramFiles 'Unity\Hub\Editor'), (Join-Path ${env:ProgramFiles(x86)} 'Unity\Hub\Editor'))) {
    if ($base -and (Test-Path $base)) {
        $unityCandidates += (Get-ChildItem -Path $base -Directory -ErrorAction SilentlyContinue |
            ForEach-Object { Join-Path $_.FullName 'Editor\Unity.exe' })
    }
}
$unityMatch = $null
$unityWrong = @()
foreach ($candidate in $unityCandidates) {
    if (-not (Test-Path $candidate)) { continue }
    $version = (Get-Item $candidate).VersionInfo.ProductVersion
    if ($version -like '6000.6.7f1*') { $unityMatch = "$version ($candidate)"; break }
    $unityWrong += "$version ($candidate)"
}
if ($null -ne $unityMatch) {
    Add-Row 'unity' 'OK' $unityMatch 'M06' $true
}
elseif ($unityWrong.Count -gt 0) {
    Add-Row 'unity' 'MISMATCH' "found $($unityWrong -join '; '), required 6000.6.7f1" 'M06' $true
}
else {
    Add-Row 'unity' 'MISSING' 'Unity 6000.6.7f1 not found; required by M06, not by M01' 'M06' $true
}

# --- Optional: Bash (shell script parity) ------------------------------------
$bashPath = $null
foreach ($candidate in @('C:\Program Files\Git\bin\bash.exe', 'C:\Program Files\Git\usr\bin\bash.exe')) {
    if (Test-Path $candidate) { $bashPath = $candidate; break }
}
if ($null -eq $bashPath) {
    $found = Get-CommandPath 'bash'
    if ($null -ne $found) { $bashPath = $found }
}
if ($null -eq $bashPath) {
    Add-Row 'bash' 'MISSING' 'no usable bash; the .sh entry points cannot be validated on this host' 'M01' $true
}
else {
    Add-Row 'bash' 'OK' $bashPath
}

# --- Report ------------------------------------------------------------------
$width = ($rows | ForEach-Object { $_.Check.Length } | Measure-Object -Maximum).Maximum
foreach ($row in $rows) {
    $colour = switch ($row.Status) {
        'OK' { 'Green' }
        'MISSING' { if ($row.Optional) { 'Yellow' } else { 'Red' } }
        'MISMATCH' { if ($row.Optional) { 'Yellow' } else { 'Red' } }
        'BLOCKED' { 'Red' }
        default { 'Red' }
    }
    $status = "[{0}]" -f $row.Status
    Write-Host ("{0,-10} {1,-$($width + 2)} {2}" -f $status, $row.Check, $row.Detail) -ForegroundColor $colour
}

Write-Host ''
if ($failures -gt 0) {
    Write-Fail "doctor: $failures required check(s) failed"
    exit 1
}
$note = if ($optionalMissing -gt 0) { " ($optionalMissing optional dependency gap(s) recorded for later milestones)" } else { '' }
Write-Ok "doctor: all required checks passed$note"
exit 0
