# Builds the Windows x86_64 standalone player with the pinned editor.
#
# Exit codes:
#   0  player built and verified on disk
#   1  build failed
#   2  BLOCKED: editor missing, wrong version, missing Windows module or licence
#
# Local usage:
#   pwsh ./tooling/unity-build.ps1
# Output: unity/Build/Windows/TTGQLDL.exe (gitignored).
[CmdletBinding()]
param()

. (Join-Path $PSScriptRoot '_common.ps1')
. (Join-Path $PSScriptRoot 'unity-safe-console.ps1')

$script:RequiredUnityVersion = '6000.6.7f1'
$root = Get-RepoRoot
$project = Join-Path $root 'unity'
$logs = Join-Path $project 'TestResults'
$player = Join-Path $project 'Build\Windows\TTGQLDL.exe'
$log = Join-Path $logs 'build.log'

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

$version = Get-UnityVersion $editor
$version = if ($null -eq $version) { '' } else { "$version".Trim() }
if ($version -notlike "$script:RequiredUnityVersion*") {
    Write-Fail "BLOCKED: editor version '$version' does not match the pinned $script:RequiredUnityVersion"
    exit 2
}

$editorRoot = Split-Path (Split-Path $editor -Parent) -Parent
$windowsModule = Join-Path $editorRoot 'Editor\Data\PlaybackEngines\windowsstandalonesupport'
if (-not (Test-Path $windowsModule)) {
    Write-Fail 'BLOCKED: the Windows standalone build module is not installed for this editor'
    exit 2
}

New-Item -ItemType Directory -Force -Path $logs | Out-Null
Remove-Item -Path $log -ErrorAction SilentlyContinue
Remove-Item -Path (Join-Path $project 'Build') -Recurse -Force -ErrorAction SilentlyContinue

Write-Step "building Windows player with Unity $version"
$started = Get-Date
# Raw editor output is captured to a protected temporary file and never streamed
# into a CI log; only a sanitized extract is printed.
$rawConsole = Join-Path ([System.IO.Path]::GetTempPath()) ("ttg-unity-build-console-{0}.log" -f ([guid]::NewGuid().ToString('N').Substring(0, 8)))
$previous = $ErrorActionPreference
$ErrorActionPreference = 'Continue'
try {
    & $editor -batchmode -nographics -quit -projectPath $project `
        -executeMethod Ttgqldl.Editor.BuildWindows.PerformBuild -logFile $log 2>&1 | Out-File -FilePath $rawConsole -Encoding utf8
    $code = $LASTEXITCODE
}
finally {
    $ErrorActionPreference = $previous
}
Publish-SafeConsoleOutput -RawPath $rawConsole -Label 'player build'
Remove-Item -LiteralPath $rawConsole -Force -ErrorAction SilentlyContinue
$elapsed = [math]::Round(((Get-Date) - $started).TotalSeconds)

if ($code -ne 0) {
    Write-Fail "build failed with exit code $code after ${elapsed}s"
    if (Test-Path $log) {
        Select-String -Path $log -Pattern 'error CS|Error building|\[TTGQLDL\] build failed' |
            Select-Object -First 10 | ForEach-Object { Write-Fail $_.Line }
    }
    exit 1
}

if (-not (Test-Path $player)) {
    Write-Fail "build reported success but $player is missing"
    exit 1
}

$sizeMb = [math]::Round((Get-ChildItem (Split-Path $player -Parent) -Recurse -File |
    Measure-Object Length -Sum).Sum / 1MB, 1)
Write-Ok "player built in ${elapsed}s: $player ($sizeMb MB)"
exit 0
